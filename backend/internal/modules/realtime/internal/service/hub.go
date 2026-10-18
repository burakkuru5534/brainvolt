// Package service is the realtime gateway (docs/api/websocket.md): the
// sessions connected to this instance, routing of client intents to the
// handlers other modules register, delivery of server events to users on
// any instance (Redis pub/sub), presence and resume sessions
// (redis-keys §4).
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/wsserver"
)

// Timing of the protocol (websocket.md §2, §4, D-032, D-051).
const (
	AuthTimeout       = 5 * time.Second
	PingInPlay        = 2 * time.Second
	PingIdle          = 20 * time.Second
	MissedPongs       = 3
	ResumeWindow      = 2 * time.Minute
	PresenceTTL       = 60 * time.Second
	WeakRTTMS         = 400
	rttSamples        = 10
	sendBuffer        = 256
	maxInstallationID = 100
)

// Client is the authenticated side of one session.
type Client struct {
	UserID         uuid.UUID
	SessionID      uuid.UUID
	InstallationID string
	Locale         i18n.Locale
}

// Message is a routed client intent.
type Message struct {
	Type string
	ID   string
	Data json.RawMessage
}

// Handler processes one intent; nil → sys.ack, an error → sys.error.
type Handler func(ctx context.Context, c Client, m Message) error

// Callbacks other modules register at start-up.
type (
	// DisconnectFunc runs when a session's socket closes.
	DisconnectFunc func(ctx context.Context, c Client)
	// PongFunc receives a session's RTT estimate (median of its last 10
	// samples) after every pong (timing.md §3).
	PongFunc func(c Client, rttMS int)
	// LiveFunc adds what the user is part of to sys.welcome.live.
	LiveFunc func(ctx context.Context, user uuid.UUID) (map[string]any, error)
	// ResumeFunc sends a resuming session the events of stream after
	// lastSeq, or a snapshot (websocket.md §6).
	ResumeFunc func(ctx context.Context, c Client, stream string, lastSeq int64) error
)

// Authenticator verifies the token of sys.auth / sys.reauth (identity).
type Authenticator interface {
	Authenticate(ctx context.Context, raw string) (identity.Principal, time.Time, error)
}

// Target addresses a delivery: every session of a user, or one session.
type Target struct {
	User    uuid.UUID
	Session uuid.UUID // optional
}

// Deps are the hub's collaborators.
type Deps struct {
	Redis    *redis.Client
	Auth     Authenticator
	Clock    clock.Clock
	Logger   *slog.Logger
	Instance string // this API instance's id
}

// Hub holds this instance's sessions.
type Hub struct {
	d Deps

	mu             sync.Mutex
	sessions       map[uuid.UUID]*Session
	byUser         map[uuid.UUID]map[uuid.UUID]*Session
	byInstallation map[string]*Session

	handlers     map[string]Handler
	resumers     map[string]ResumeFunc
	onDisconnect []DisconnectFunc
	onPong       []PongFunc
	live         []LiveFunc
}

// New returns a hub.
func New(d Deps) *Hub {
	return &Hub{
		d: d, sessions: map[uuid.UUID]*Session{}, byUser: map[uuid.UUID]map[uuid.UUID]*Session{},
		byInstallation: map[string]*Session{}, handlers: map[string]Handler{}, resumers: map[string]ResumeFunc{},
	}
}

// Handle routes intents whose type starts with prefix + "." to h
// (backend-modules §3.3). Registration happens before serving.
func (h *Hub) Handle(prefix string, fn Handler) { h.handlers[prefix] = fn }

// OnResume registers the resumer of streams "<prefix>:<id>".
func (h *Hub) OnResume(prefix string, fn ResumeFunc) { h.resumers[prefix] = fn }

// OnDisconnect registers a disconnect listener.
func (h *Hub) OnDisconnect(fn DisconnectFunc) { h.onDisconnect = append(h.onDisconnect, fn) }

// OnPong registers an RTT listener.
func (h *Hub) OnPong(fn PongFunc) { h.onPong = append(h.onPong, fn) }

// OnLive registers a contributor to sys.welcome.live.
func (h *Hub) OnLive(fn LiveFunc) { h.live = append(h.live, fn) }

// ---------- Delivery ----------

// ErrDeferred is returned by a handler that answers the intent itself
// later with Reply (e.g. after the match owner processed it).
var ErrDeferred = errors.New("realtime: reply deferred")

// remote is a pub/sub message between instances.
type remote struct {
	Kind    string             `json:"kind"` // deliver | replace | playing
	Playing bool               `json:"playing,omitempty"`
	Target  Target             `json:"target"`
	Frame   *wsserver.Outbound `json:"frame,omitempty"`
}

func instChannel(instance string) string { return redis.Key("rt", "ch", "inst", instance) }
func presenceKey(user uuid.UUID) string  { return redis.Key("rt", "presence", user.String()) }
func sessionKey(id uuid.UUID) string     { return redis.Key("rt", "sess", id.String()) }
func installationKey(id string) string   { return redis.Key("rt", "installation", id) }

// Send delivers a frame to the target's sessions on every instance. A
// session that cannot keep up is closed; delivery is best effort, the
// client recovers missed stream events by resume (websocket.md §6).
func (h *Hub) Send(ctx context.Context, t Target, f wsserver.Outbound) error {
	if f.TS == 0 {
		f.TS = h.d.Clock.Now().UnixMilli()
	}
	h.deliverLocal(t, f)
	instances, err := h.instancesOf(ctx, t)
	if err != nil {
		return err
	}
	if len(instances) == 0 {
		return nil
	}
	msg, err := json.Marshal(remote{Kind: "deliver", Target: t, Frame: &f})
	if err != nil {
		return err
	}
	for _, inst := range instances {
		if err := h.d.Redis.Publish(ctx, instChannel(inst), msg).Err(); err != nil {
			return fmt.Errorf("realtime: publish to %s: %w", inst, err)
		}
	}
	return nil
}

// instancesOf returns the other instances holding sessions of the target.
func (h *Hub) instancesOf(ctx context.Context, t Target) ([]string, error) {
	if t.Session != uuid.Nil {
		inst, err := h.d.Redis.HGet(ctx, sessionKey(t.Session), "instance").Result()
		if errors.Is(err, goredis.Nil) || inst == h.d.Instance {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return []string{inst}, nil
	}
	all, err := h.d.Redis.HGetAll(ctx, presenceKey(t.User)).Result()
	if err != nil {
		return nil, err
	}
	cutoff := h.d.Clock.Now().Add(-PresenceTTL).UnixMilli()
	var out []string
	for inst, seen := range all {
		ms, _ := strconv.ParseInt(seen, 10, 64)
		if inst != h.d.Instance && ms >= cutoff {
			out = append(out, inst)
		}
	}
	return out, nil
}

func (h *Hub) deliverLocal(t Target, f wsserver.Outbound) {
	h.mu.Lock()
	var targets []*Session
	if t.Session != uuid.Nil {
		if s, ok := h.sessions[t.Session]; ok && s.client.UserID == t.User {
			targets = append(targets, s)
		}
	} else {
		for _, s := range h.byUser[t.User] {
			targets = append(targets, s)
		}
	}
	h.mu.Unlock()
	for _, s := range targets {
		s.enqueue(f)
	}
}

// Run receives deliveries for this instance until ctx ends.
func (h *Hub) Run(ctx context.Context) error {
	sub := h.d.Redis.Subscribe(ctx, instChannel(h.d.Instance))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("realtime: subscribe: %w", err)
	}
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			h.closeAll(wsserver.CloseGoingAway, "server shutting down")
			return nil
		case m, ok := <-ch:
			if !ok {
				return nil
			}
			var r remote
			if err := json.Unmarshal([]byte(m.Payload), &r); err != nil {
				h.d.Logger.WarnContext(ctx, "realtime: bad pub/sub message", "error", err)
				continue
			}
			switch r.Kind {
			case "deliver":
				if r.Frame != nil {
					h.deliverLocal(r.Target, *r.Frame)
				}
			case "replace":
				if s := h.session(r.Target.Session); s != nil {
					s.close(wsserver.CloseReplaced, "replaced by a newer session")
				}
			case "playing":
				if s := h.session(r.Target.Session); s != nil {
					s.setPlaying(r.Playing)
				}
			}
		}
	}
}

func (h *Hub) closeAll(code wsserver.CloseCode, reason string) {
	h.mu.Lock()
	all := make([]*Session, 0, len(h.sessions))
	for _, s := range h.sessions {
		all = append(all, s)
	}
	h.mu.Unlock()
	for _, s := range all {
		s.close(code, reason)
	}
}

// ---------- Sessions ----------

// newResumeToken returns a token and its stored hash.
func newResumeToken() (token, hash string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand never fails on supported platforms
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token)
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

// resumable checks a resume request: the session exists (within the
// 2 min window), belongs to the user and the token matches.
func (h *Hub) resumable(ctx context.Context, user, session uuid.UUID, token string) bool {
	v, err := h.d.Redis.HMGet(ctx, sessionKey(session), "user_id", "token_hash").Result()
	if err != nil || len(v) != 2 {
		return false
	}
	u, _ := v[0].(string)
	th, _ := v[1].(string)
	return u == user.String() && th != "" && th == hashToken(token)
}

// register adds a session, replacing the installation's previous one
// (close 4009, websocket.md §2), and writes its Redis records.
func (h *Hub) register(ctx context.Context, s *Session, tokenHash string) error {
	c := s.client
	h.mu.Lock()
	old := h.byInstallation[c.InstallationID]
	h.sessions[c.SessionID] = s
	if h.byUser[c.UserID] == nil {
		h.byUser[c.UserID] = map[uuid.UUID]*Session{}
	}
	h.byUser[c.UserID][c.SessionID] = s
	h.byInstallation[c.InstallationID] = s
	h.mu.Unlock()
	if old != nil && old != s {
		old.close(wsserver.CloseReplaced, "replaced by a newer session")
	}

	prev, err := h.d.Redis.SetArgs(ctx, installationKey(c.InstallationID), c.SessionID.String(),
		goredis.SetArgs{Get: true, TTL: ResumeWindow}).Result()
	if err != nil && !errors.Is(err, goredis.Nil) {
		return err
	}
	if prevID, perr := uuid.Parse(prev); perr == nil && prevID != c.SessionID && (old == nil || old.client.SessionID != prevID) {
		if inst, ierr := h.d.Redis.HGet(ctx, sessionKey(prevID), "instance").Result(); ierr == nil && inst != h.d.Instance {
			msg, _ := json.Marshal(remote{Kind: "replace", Target: Target{User: c.UserID, Session: prevID}})
			h.d.Redis.Publish(ctx, instChannel(inst), msg)
		}
	}
	pipe := h.d.Redis.TxPipeline()
	pipe.HSet(ctx, sessionKey(c.SessionID), "user_id", c.UserID.String(), "token_hash", tokenHash,
		"installation_id", c.InstallationID, "instance", h.d.Instance)
	pipe.Expire(ctx, sessionKey(c.SessionID), ResumeWindow)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return err
	}
	return h.touch(ctx, s)
}

// touch refreshes the session's resume window and the user's presence.
func (h *Hub) touch(ctx context.Context, s *Session) error {
	c := s.client
	pipe := h.d.Redis.Pipeline()
	pipe.Expire(ctx, sessionKey(c.SessionID), ResumeWindow)
	pipe.Expire(ctx, installationKey(c.InstallationID), ResumeWindow)
	pipe.HSet(ctx, presenceKey(c.UserID), h.d.Instance, h.d.Clock.Now().UnixMilli())
	pipe.Expire(ctx, presenceKey(c.UserID), PresenceTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// unregister removes a closed session; its resume record stays for the
// 2 min window.
func (h *Hub) unregister(ctx context.Context, s *Session) {
	c := s.client
	h.mu.Lock()
	if h.sessions[c.SessionID] == s {
		delete(h.sessions, c.SessionID)
	}
	if m := h.byUser[c.UserID]; m != nil && m[c.SessionID] == s {
		delete(m, c.SessionID)
		if len(m) == 0 {
			delete(h.byUser, c.UserID)
		}
	}
	if h.byInstallation[c.InstallationID] == s {
		delete(h.byInstallation, c.InstallationID)
	}
	userGone := h.byUser[c.UserID] == nil
	h.mu.Unlock()
	if userGone {
		if err := h.d.Redis.HDel(ctx, presenceKey(c.UserID), h.d.Instance).Err(); err != nil {
			h.d.Logger.WarnContext(ctx, "realtime: presence cleanup failed", "error", err)
		}
	}
	for _, fn := range h.onDisconnect {
		fn(ctx, c)
	}
}

// Session returns a connected local session (tests, SetPlaying).
func (h *Hub) session(id uuid.UUID) *Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[id]
}

// SetPlaying switches a session's heartbeat between 2 s (in a match) and
// 20 s (websocket.md §4), on whichever instance holds it.
func (h *Hub) SetPlaying(ctx context.Context, t Target, playing bool) error {
	if s := h.session(t.Session); s != nil {
		s.setPlaying(playing)
		return nil
	}
	instances, err := h.instancesOf(ctx, t)
	if err != nil || len(instances) == 0 {
		return err
	}
	msg, err := json.Marshal(remote{Kind: "playing", Target: t, Playing: playing})
	if err != nil {
		return err
	}
	return h.d.Redis.Publish(ctx, instChannel(instances[0]), msg).Err()
}

// Reply answers an intent later (after ErrDeferred) on the client's session.
func (h *Hub) Reply(ctx context.Context, c Client, re string, err error) error {
	if re == "" {
		return nil
	}
	return h.Send(ctx, Target{User: c.UserID, Session: c.SessionID}, h.replyFrame(ctx, c.Locale, re, err))
}

// Online reports whether the user has a session on this instance.
func (h *Hub) Online(user uuid.UUID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byUser[user]) > 0
}

// prefixOf returns the routing prefix of a message type ("match.answer" → "match").
func prefixOf(t string) string {
	p, _, _ := strings.Cut(t, ".")
	return p
}
