package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/wsserver"
)

// Session is one authenticated connection.
type Session struct {
	hub    *Hub
	conn   *wsserver.Conn
	client Client
	out    chan wsserver.Outbound
	done   chan struct{}
	reset  chan struct{}

	closeOnce sync.Once

	mu      sync.Mutex
	playing bool
	pending int // pings sent without a pong
	rtts    []int
	weak    bool
	expires time.Time
}

func (s *Session) enqueue(f wsserver.Outbound) {
	select {
	case <-s.done:
	case s.out <- f:
	default:
		s.close(wsserver.CloseProtocol, "client too slow")
	}
}

// close ends the session; the close handshake runs in the background so
// callers (another session's handshake, the heartbeat) never wait for it.
func (s *Session) close(code wsserver.CloseCode, reason string) {
	s.closeOnce.Do(func() {
		close(s.done)
		go s.conn.Close(code, reason)
	})
}

func (s *Session) setPlaying(p bool) {
	s.mu.Lock()
	changed := s.playing != p
	s.playing = p
	s.mu.Unlock()
	if changed {
		select {
		case s.reset <- struct{}{}:
		default:
		}
	}
}

// pong records an RTT sample; it returns the new connection quality when
// it changed (websocket.md §4).
func (s *Session) pong(rtt int) (median int, changed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = 0
	s.rtts = append(s.rtts, rtt)
	if len(s.rtts) > rttSamples {
		s.rtts = s.rtts[1:]
	}
	sorted := slices.Clone(s.rtts)
	slices.Sort(sorted)
	median = sorted[len(sorted)/2]
	weak := median > WeakRTTMS
	changed, s.weak = weak != s.weak, weak
	return median, changed
}

// authRequest is the payload of sys.auth (websocket.md §2).
type authRequest struct {
	Token          string `json:"token"`
	InstallationID string `json:"installation_id"`
	Resume         *struct {
		SessionID   uuid.UUID        `json:"session_id"`
		ResumeToken string           `json:"resume_token"`
		Streams     map[string]int64 `json:"streams"`
	} `json:"resume"`
}

// authCloseCode maps an authentication failure to its close code.
func authCloseCode(err error) (wsserver.CloseCode, string) {
	e := apierror.From(err)
	switch {
	case e == nil:
		return 1011, "authentication failed"
	case e.Code == apierror.TokenExpired:
		return wsserver.CloseTokenExpired, "token expired"
	case e.Code == apierror.Unauthenticated && e.Details["reason"] == "account_deleted":
		return wsserver.CloseAccountBlocked, "account deleted"
	case e.Code == apierror.Unauthenticated:
		return wsserver.CloseInvalidToken, "invalid token"
	default:
		return 1011, "authentication unavailable"
	}
}

// Serve runs one connection: handshake, then messages until it closes.
func (h *Hub) Serve(ctx context.Context, conn *wsserver.Conn, locale i18n.Locale) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s, streams, ok := h.handshake(ctx, conn, locale)
	if !ok {
		return
	}
	go s.writeLoop(ctx)
	go s.heartbeat(ctx)
	for stream, seq := range streams {
		h.resume(ctx, s, stream, seq, "")
	}
	for {
		m, err := conn.Read(ctx, h.d.Clock.Now())
		if err != nil {
			var ce *wsserver.CloseError
			if errors.As(err, &ce) {
				s.close(ce.Code, ce.Reason)
				for _, fn := range h.onViolation {
					fn(context.WithoutCancel(ctx), s.client.UserID, ce.Reason)
				}
			} else {
				s.close(wsserver.CloseNormal, "")
			}
			break
		}
		h.dispatch(ctx, s, m)
	}
	h.unregister(context.WithoutCancel(ctx), s)
}

func (h *Hub) handshake(ctx context.Context, conn *wsserver.Conn, locale i18n.Locale) (*Session, map[string]int64, bool) {
	authCtx, cancel := context.WithTimeout(ctx, AuthTimeout)
	m, err := conn.Read(authCtx, h.d.Clock.Now())
	cancel()
	var ce *wsserver.CloseError
	switch {
	case errors.As(err, &ce):
		conn.Close(ce.Code, ce.Reason)
		return nil, nil, false
	case err != nil:
		conn.Close(wsserver.CloseAuthTimeout, "sys.auth expected within 5 s")
		return nil, nil, false
	case m.T != "sys.auth":
		conn.Close(wsserver.CloseAuthTimeout, "sys.auth expected first")
		return nil, nil, false
	}
	var a authRequest
	if json.Unmarshal(m.D, &a) != nil || a.InstallationID == "" || len(a.InstallationID) > maxInstallationID {
		conn.Close(wsserver.CloseProtocol, "malformed sys.auth")
		return nil, nil, false
	}
	p, expires, err := h.d.Auth.Authenticate(ctx, a.Token)
	if err != nil {
		code, reason := authCloseCode(err)
		conn.Close(code, reason)
		return nil, nil, false
	}
	if p.Status == identity.StatusSuspended {
		conn.Close(wsserver.CloseAccountBlocked, "account suspended")
		return nil, nil, false
	}

	sessionID := ids.New()
	var streams map[string]int64
	if a.Resume != nil && h.resumable(ctx, p.UserID, a.Resume.SessionID, a.Resume.ResumeToken) {
		sessionID, streams = a.Resume.SessionID, a.Resume.Streams
	}
	token, hash := newResumeToken()
	s := &Session{
		hub: h, conn: conn, out: make(chan wsserver.Outbound, sendBuffer), done: make(chan struct{}), reset: make(chan struct{}, 1),
		client:  Client{UserID: p.UserID, SessionID: sessionID, InstallationID: a.InstallationID, Locale: locale},
		expires: expires,
	}
	if err := h.register(ctx, s, hash); err != nil {
		h.d.Logger.ErrorContext(ctx, "realtime: registering a session failed", "error", err)
		h.unregister(context.WithoutCancel(ctx), s)
		conn.Close(1011, "try again")
		return nil, nil, false
	}
	live := map[string]any{}
	for _, fn := range h.live {
		v, err := fn(ctx, p.UserID)
		if err != nil {
			h.d.Logger.WarnContext(ctx, "realtime: live lookup failed", "error", err)
			continue
		}
		for k, x := range v {
			live[k] = x
		}
	}
	now := h.d.Clock.Now()
	s.enqueue(wsserver.Outbound{T: "sys.welcome", TS: now.UnixMilli(), Re: m.ID, D: map[string]any{
		"session_id": sessionID, "resume_token": token, "server_time": now.UnixMilli(),
		"heartbeat_ms": PingIdle.Milliseconds(), "user_id": p.UserID, "live": live,
	}})
	return s, streams, true
}

func (s *Session) writeLoop(ctx context.Context) {
	for {
		select {
		case <-s.done:
			return
		case f := <-s.out:
			if err := s.conn.Write(ctx, f); err != nil {
				s.close(wsserver.CloseGoingAway, "write failed")
				return
			}
		}
	}
}

// heartbeat pings every 2 s in play and 20 s otherwise; three unanswered
// pings or an expired token end the session (websocket.md §2, §4).
func (s *Session) heartbeat(ctx context.Context) {
	h := s.hub
	for {
		s.mu.Lock()
		interval := PingIdle
		if s.playing {
			interval = PingInPlay
		}
		s.mu.Unlock()
		t := h.d.Clock.NewTimer(interval)
		select {
		case <-s.done:
			t.Stop()
			return
		case <-s.reset:
			t.Stop()
			continue
		case <-t.C():
		}
		now := h.d.Clock.Now()
		s.mu.Lock()
		dead := s.pending >= MissedPongs
		expired := !s.expires.IsZero() && now.After(s.expires)
		s.pending++
		s.mu.Unlock()
		switch {
		case dead:
			s.close(wsserver.CloseGoingAway, "heartbeat timeout")
			return
		case expired:
			s.close(wsserver.CloseTokenExpired, "token expired")
			return
		}
		s.enqueue(wsserver.Outbound{T: "sys.ping", TS: now.UnixMilli(), D: map[string]int64{"ts": now.UnixMilli()}})
		if err := h.touch(ctx, s); err != nil {
			h.d.Logger.WarnContext(ctx, "realtime: refreshing presence failed", "error", err)
		}
	}
}

func (h *Hub) dispatch(ctx context.Context, s *Session, m wsserver.Inbound) {
	switch m.T {
	case "sys.auth":
		s.close(wsserver.CloseProtocol, "already authenticated")
	case "sys.pong":
		var p struct {
			TS int64 `json:"ts"`
		}
		if json.Unmarshal(m.D, &p) != nil {
			return
		}
		rtt := int(h.d.Clock.Now().UnixMilli() - p.TS)
		if rtt < 0 || rtt > 60_000 {
			return
		}
		median, changed := s.pong(rtt)
		if changed {
			quality := "good"
			if median > WeakRTTMS {
				quality = "weak"
			}
			s.enqueue(h.frame("sys.connection", "", map[string]any{"rtt_ms": median, "quality": quality}))
		}
		for _, fn := range h.onPong {
			fn(s.client, median)
		}
	case "sys.reauth":
		var p struct {
			Token string `json:"token"`
		}
		if json.Unmarshal(m.D, &p) != nil {
			h.reply(ctx, s, m.ID, apierror.New(apierror.MalformedRequest))
			return
		}
		pr, expires, err := h.d.Auth.Authenticate(ctx, p.Token)
		if err == nil && pr.UserID != s.client.UserID {
			err = apierror.New(apierror.Unauthenticated)
		}
		if err == nil {
			s.mu.Lock()
			s.expires = expires
			s.mu.Unlock()
		}
		h.reply(ctx, s, m.ID, err)
	case "sys.subscribe":
		var p struct {
			Stream  string `json:"stream"`
			FromSeq int64  `json:"from_seq"`
		}
		if json.Unmarshal(m.D, &p) != nil || p.Stream == "" {
			h.reply(ctx, s, m.ID, apierror.New(apierror.MalformedRequest))
			return
		}
		h.resume(ctx, s, p.Stream, max(p.FromSeq-1, 0), m.ID)
	case "sys.unsubscribe", "presence.set":
		h.reply(ctx, s, m.ID, nil)
	default:
		fn, ok := h.handlers[prefixOf(m.T)]
		if !ok {
			h.d.Logger.DebugContext(ctx, "realtime: unknown message type ignored", "type", m.T)
			return
		}
		h.reply(ctx, s, m.ID, fn(ctx, s.client, Message{Type: m.T, ID: m.ID, Data: m.D}))
	}
}

// resume asks the stream's owner to replay it; re answers a sys.subscribe.
func (h *Hub) resume(ctx context.Context, s *Session, stream string, lastSeq int64, re string) {
	prefix, _, _ := strings.Cut(stream, ":")
	fn, ok := h.resumers[prefix]
	var err error
	if !ok {
		err = apierror.New(apierror.NotFound).With("stream", stream)
	} else {
		err = fn(ctx, s.client, stream, lastSeq)
	}
	if re != "" {
		h.reply(ctx, s, re, err)
	} else if err != nil {
		h.d.Logger.DebugContext(ctx, "realtime: resume failed", "stream", stream, "error", err)
	}
}

// reply answers an intent with sys.ack or sys.error (websocket.md §3);
// ErrDeferred leaves the answer to the handler.
func (h *Hub) reply(ctx context.Context, s *Session, re string, err error) {
	if re == "" || errors.Is(err, ErrDeferred) {
		return
	}
	s.enqueue(h.replyFrame(ctx, s.client.Locale, re, err))
}

func (h *Hub) replyFrame(ctx context.Context, locale i18n.Locale, re string, err error) wsserver.Outbound {
	if err == nil {
		return h.frame("sys.ack", re, nil)
	}
	e := apierror.From(err)
	if e == nil || !e.Code.Known() {
		h.d.Logger.ErrorContext(ctx, "realtime: handler failed", "error", err)
		e = apierror.Wrap(apierror.Internal, err)
	}
	d := map[string]any{"code": e.Code, "message": e.Message(locale)}
	if len(e.Details) > 0 {
		d["details"] = e.Details
	}
	return h.frame("sys.error", re, d)
}

func (h *Hub) frame(t, re string, d any) wsserver.Outbound {
	return wsserver.Outbound{T: t, TS: h.d.Clock.Now().UnixMilli(), Re: re, D: d}
}
