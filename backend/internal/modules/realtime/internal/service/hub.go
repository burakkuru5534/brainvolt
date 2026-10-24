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
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/wsserver"
)

// Timing of the protocol (websocket.md §2, §4, D-032, D-051).
const (
	AuthTimeout = 5 * time.Second
	PingInPlay  = 2 * time.Second
	PingIdle    = 10 * time.Second
	// IdleTimeout closes a session (4010) that had no intent and no event
	// for this long while its user is in no match, room or queue (D-097):
	// an open WebSocket keeps Cloud Run billing CPU.
	IdleTimeout       = 3 * time.Minute
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
	// ViolationFunc runs when a session is closed for a protocol violation
	// or flooding (anti-cheat signal, D-084).
	ViolationFunc func(ctx context.Context, user uuid.UUID, reason string)
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
	Auth     Authenticator
	Clock    clock.Clock
	Logger   *slog.Logger
	Instance string // this API instance's id
	// PingIdle and IdleTimeout override the defaults (tests).
	PingIdle    time.Duration
	IdleTimeout time.Duration
}

type resumableEntry struct {
	userID    uuid.UUID
	tokenHash string
	expiresAt time.Time
}

// Hub holds this instance's sessions.
type Hub struct {
	d Deps

	mu             sync.Mutex
	sessions       map[uuid.UUID]*Session
	byUser         map[uuid.UUID]map[uuid.UUID]*Session
	byInstallation map[string]*Session
	resumables     map[uuid.UUID]resumableEntry

	handlers     map[string]Handler
	resumers     map[string]ResumeFunc
	onDisconnect []DisconnectFunc
	onPong       []PongFunc
	onViolation  []ViolationFunc
	live         []LiveFunc
}

// New returns a hub.
func New(d Deps) *Hub {
	if d.PingIdle <= 0 {
		d.PingIdle = PingIdle
	}
	if d.IdleTimeout <= 0 {
		d.IdleTimeout = IdleTimeout
	}
	return &Hub{
		d: d, sessions: map[uuid.UUID]*Session{}, byUser: map[uuid.UUID]map[uuid.UUID]*Session{},
		byInstallation: map[string]*Session{}, resumables: map[uuid.UUID]resumableEntry{},
		handlers: map[string]Handler{}, resumers: map[string]ResumeFunc{},
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

// OnViolation registers a protocol violation listener.
func (h *Hub) OnViolation(fn ViolationFunc) { h.onViolation = append(h.onViolation, fn) }

// OnLive registers a contributor to sys.welcome.live.
func (h *Hub) OnLive(fn LiveFunc) { h.live = append(h.live, fn) }

// ---------- Delivery ----------

// ErrDeferred is returned by a handler that answers the intent itself
// later with Reply (e.g. after the match owner processed it).
var ErrDeferred = errors.New("realtime: reply deferred")

// Send delivers a frame to the target's sessions on this instance. A
// session that cannot keep up is closed; delivery is best effort, the
// client recovers missed stream events by resume (websocket.md §6).
func (h *Hub) Send(ctx context.Context, t Target, f wsserver.Outbound) error {
	if f.TS == 0 {
		f.TS = h.d.Clock.Now().UnixMilli()
	}
	h.deliverLocal(t, f)
	return nil
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

// Run waits until ctx ends, then closes all sessions.
func (h *Hub) Run(ctx context.Context) error {
	<-ctx.Done()
	h.closeAll(wsserver.CloseGoingAway, "server shutting down")
	return nil
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
	h.mu.Lock()
	defer h.mu.Unlock()
	entry, ok := h.resumables[session]
	if !ok {
		return false
	}
	if h.d.Clock.Now().After(entry.expiresAt) {
		delete(h.resumables, session)
		return false
	}
	return entry.userID == user && entry.tokenHash != "" && entry.tokenHash == hashToken(token)
}

// register adds a session, replacing the installation's previous one
// (close 4009, websocket.md §2).
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
	h.resumables[c.SessionID] = resumableEntry{
		userID:    c.UserID,
		tokenHash: tokenHash,
		expiresAt: h.d.Clock.Now().Add(ResumeWindow),
	}
	h.mu.Unlock()
	if old != nil && old != s {
		old.close(wsserver.CloseReplaced, "replaced by a newer session")
	}
	return nil
}

// touch refreshes the session's resume window.
func (h *Hub) touch(ctx context.Context, s *Session) error {
	c := s.client
	h.mu.Lock()
	defer h.mu.Unlock()
	if entry, ok := h.resumables[c.SessionID]; ok {
		entry.expiresAt = h.d.Clock.Now().Add(ResumeWindow)
		h.resumables[c.SessionID] = entry
	}
	return nil
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
	h.mu.Unlock()

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
// 20 s (websocket.md §4).
func (h *Hub) SetPlaying(ctx context.Context, t Target, playing bool) error {
	if s := h.session(t.Session); s != nil {
		s.setPlaying(playing)
	}
	return nil
}

// Reply answers an intent later (after ErrDeferred) on the client's session.
func (h *Hub) Reply(ctx context.Context, c Client, re string, err error) error {
	if re == "" {
		return nil
	}
	return h.Send(ctx, Target{User: c.UserID, Session: c.SessionID}, h.replyFrame(ctx, c.Locale, re, err))
}

// Online reports whether the user has a session seen in the last 60 s.
func (h *Hub) Online(ctx context.Context, user uuid.UUID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.byUser[user]) > 0
}

// prefixOf returns the routing prefix of a message type ("match.answer" → "match").
func prefixOf(t string) string {
	p, _, _ := strings.Cut(t, ".")
	return p
}
