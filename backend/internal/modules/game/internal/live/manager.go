// Package live runs live matches: one owner goroutine per match, intents
// from the WebSocket routed to it, every step persisted to Postgres and
// delivered as sequenced events; and reconnect replays or snapshots built
// from in-memory match state.
package live

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
)

// Lease and retention.
const (
	FinishedTTL = time.Hour
	maxEvents   = 1000
)

// Persister is the game service: it writes engine outputs to Postgres,
// picks replacement questions and cancels matches whose live state is gone.
type Persister interface {
	Apply(ctx context.Context, m *engine.Match, out engine.Output) error
	Spare(ctx context.Context, m *engine.Match) ([]engine.Question, error)
	More(ctx context.Context, m *engine.Match) ([]engine.Question, error)
	CancelOrphan(ctx context.Context, id uuid.UUID) error
}

// Realtime is what the owner needs from the gateway.
type Realtime interface {
	Send(ctx context.Context, t realtime.Target, f realtime.Frame) error
	Reply(ctx context.Context, c realtime.Client, re string, err error) error
	SetPlaying(ctx context.Context, t realtime.Target, playing bool) error
}

// Texts returns question wording per locale (content).
type Texts interface {
	Texts(ctx context.Context, ids []uuid.UUID, locales []string) (map[uuid.UUID]map[string]content.QuestionText, error)
}

// Cards returns players' public cards (profile).
type Cards interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
}

// Deps are the manager's collaborators.
type Deps struct {
	Persister Persister
	RT        Realtime
	Texts     Texts
	Cards     Cards
	Clock     clock.Clock
	Logger    *slog.Logger
	Instance  string
	Random    engine.Random
}

// Manager holds the matches owned by this instance.
type Manager struct {
	d      Deps
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	owners   map[uuid.UUID]*owner
	userLive map[uuid.UUID]uuid.UUID
	ended    []EndFunc
}

// Ended is a match that just reached completed or cancelled, as the
// owning instance saw it (matchmaking: re-queue, cooldowns).
type Ended struct {
	MatchID      uuid.UUID
	Origin       domain.Origin
	Status       domain.Status
	CancelReason domain.CancelReason
	Participants []EndedParticipant
}

// EndedParticipant is a participant's final state.
type EndedParticipant struct {
	UserID uuid.UUID
	Locale string
	Status domain.ParticipantStatus
}

// EndFunc is called once per finished match on the instance that owned it.
type EndFunc func(ctx context.Context, e Ended)

// OnEnd registers fn for finished matches; register before matches start.
func (m *Manager) OnEnd(fn EndFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ended = append(m.ended, fn)
}

func (m *Manager) notifyEnded(ctx context.Context, match *engine.Match) {
	m.mu.Lock()
	fns := m.ended
	m.mu.Unlock()
	if len(fns) == 0 {
		return
	}
	e := Ended{MatchID: match.ID, Origin: match.Config.Origin, Status: match.Status}
	if r := match.Result(); r != nil {
		e.CancelReason = r.CancelReason
	}
	for _, p := range match.Participants {
		e.Participants = append(e.Participants, EndedParticipant{UserID: p.UserID, Locale: p.Locale, Status: p.Status})
	}
	for _, fn := range fns {
		fn(ctx, e)
	}
}

// New returns a manager; its owners run until Close.
func New(d Deps) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		d: d, ctx: ctx, cancel: cancel,
		owners:   map[uuid.UUID]*owner{},
		userLive: map[uuid.UUID]uuid.UUID{},
	}
}

// Close stops every local owner (process shutdown).
func (m *Manager) Close() { m.cancel() }

// Start creates an owner for match with out as its creation output.
func (m *Manager) Start(ctx context.Context, match *engine.Match, out engine.Output, bots ...uuid.UUID) error {
	users := make([]uuid.UUID, 0, len(match.Participants))
	locales := map[string]bool{}
	seats := make([]engine.Seat, 0, len(match.Participants))
	for _, p := range match.Participants {
		users = append(users, p.UserID)
		locales[p.Locale] = true
		seats = append(seats, p.Seat)
	}
	qs := match.Questions()
	ids := make([]uuid.UUID, 0, len(qs))
	for _, q := range qs {
		ids = append(ids, q.ID)
	}
	ls := make([]string, 0, len(locales))
	for l := range locales {
		ls = append(ls, l)
	}
	texts, err := m.d.Texts.Texts(ctx, ids, ls)
	if err != nil {
		return err
	}
	cards, err := m.d.Cards.Cards(ctx, users)
	if err != nil {
		return err
	}
	mt := &meta{MatchID: match.ID, Texts: texts, Cards: cards, Seats: seats, Summary: map[string]any{
		"match_type": match.Config.MatchType, "format": match.Config.Format, "category_mode": match.Config.CategoryMode,
		"team_answer_mode": match.Config.TeamMode,
		"question_count":   match.Derived.QuestionCount, "answer_window_ms": match.Derived.AnswerWindowMS,
	}, Bots: bots}
	mt.addCategories(qs)

	m.mu.Lock()
	if _, exists := m.owners[match.ID]; exists {
		m.mu.Unlock()
		return fmt.Errorf("game: match %s already has an owner", match.ID)
	}
	for _, p := range match.Participants {
		m.userLive[p.UserID] = match.ID
	}
	m.mu.Unlock()

	o := m.newOwner(match, mt, map[uuid.UUID]uuid.UUID{})
	if err := o.commit(m.ctx, out, m.d.Clock.Now().UnixMilli()); err != nil {
		o.halt()
		m.forget(o)
		return err
	}
	go o.run(m.ctx)
	return nil
}

func (m *Manager) newOwner(match *engine.Match, mt *meta, seats map[uuid.UUID]uuid.UUID) *owner {
	o := &owner{m: m, match: match, meta: mt, seats: seats, inbox: make(chan command, inboxSize), stop: make(chan struct{})}
	m.mu.Lock()
	m.owners[match.ID] = o
	m.mu.Unlock()
	return o
}

func (m *Manager) local(id uuid.UUID) *owner {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.owners[id]
}

func (m *Manager) forget(o *owner) {
	m.mu.Lock()
	delete(m.owners, o.match.ID)
	for _, p := range o.match.Participants {
		if m.userLive[p.UserID] == o.match.ID {
			delete(m.userLive, p.UserID)
		}
	}
	m.mu.Unlock()
}

// route delivers a command to the match's owner.
func (m *Manager) route(ctx context.Context, c command) error {
	if o := m.local(c.Match); o != nil {
		return o.post(c)
	}
	return apierror.New(apierror.MatchNotFound)
}

// Run waits until ctx ends.
func (m *Manager) Run(ctx context.Context) error {
	defer m.Close()
	<-ctx.Done()
	return nil
}

// ---------- WebSocket entry points ----------

// intent is the payload of match.* intents (websocket.md §5.1).
type intent struct {
	MatchID    uuid.UUID `json:"match_id"`
	RoundNo    int       `json:"round_no"`
	OptionID   uuid.UUID `json:"option_id"`
	CategoryID uuid.UUID `json:"category_id"`
	PhraseID   string    `json:"phrase_id"`
}

// Handle routes a match.* intent to the owner; the owner answers it.
func (m *Manager) Handle(ctx context.Context, c realtime.Client, msg realtime.Message) error {
	now := m.d.Clock.Now().UnixMilli()
	var p intent
	if err := json.Unmarshal(msg.Data, &p); err != nil || p.MatchID == uuid.Nil {
		return apierror.New(apierror.MalformedRequest)
	}
	cmd := command{Match: p.MatchID, Client: c, Re: msg.ID, AtMS: now}
	switch msg.Type {
	case "match.join":
		cmd.Kind = kindJoin
	case "match.ready":
		cmd.Kind = kindReady
	case "match.leave":
		cmd.Kind = kindLeave
	case "match.answer", "match.propose", "match.lock":
		if p.RoundNo <= 0 || p.OptionID == uuid.Nil {
			return apierror.New(apierror.MalformedRequest)
		}
		cmd.Kind = map[string]string{"match.answer": kindAnswer, "match.propose": kindPropose, "match.lock": kindLock}[msg.Type]
		cmd.Round, cmd.Option = p.RoundNo, p.OptionID
	case "match.pick_category":
		if p.RoundNo <= 0 || p.CategoryID == uuid.Nil {
			return apierror.New(apierror.MalformedRequest)
		}
		cmd.Kind, cmd.Round, cmd.Option = kindPick, p.RoundNo, p.CategoryID
	case "match.quick_chat":
		cmd.Kind, cmd.Phrase = kindQuick, p.PhraseID
	default:
		return apierror.New(apierror.MalformedRequest).With("reason", "unknown_intent")
	}
	if err := m.route(ctx, cmd); err != nil {
		return err
	}
	return realtime.ErrDeferred
}

// LiveMatch returns the user's live match, if any.
func (m *Manager) LiveMatch(ctx context.Context, user uuid.UUID) (uuid.UUID, bool) {
	return m.liveMatch(ctx, user)
}

func (m *Manager) liveMatch(ctx context.Context, user uuid.UUID) (uuid.UUID, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.userLive[user]
	return id, ok
}

// OnDisconnect tells the owner a seat's socket closed.
func (m *Manager) OnDisconnect(ctx context.Context, c realtime.Client) {
	if id, ok := m.liveMatch(ctx, c.UserID); ok {
		_ = m.route(ctx, command{Kind: kindDisconnect, Match: id, Client: c, AtMS: m.d.Clock.Now().UnixMilli()})
	}
}

// OnPong feeds a participant's RTT estimate to the owner (timing.md §3).
func (m *Manager) OnPong(c realtime.Client, rtt int) {
	if id, ok := m.liveMatch(m.ctx, c.UserID); ok {
		_ = m.route(m.ctx, command{Kind: kindRTT, Match: id, Client: c, RTT: rtt})
	}
}

// OnLive adds the user's live match to sys.welcome.live.
func (m *Manager) OnLive(ctx context.Context, user uuid.UUID) (map[string]any, error) {
	if id, ok := m.liveMatch(ctx, user); ok {
		return map[string]any{"match_id": id}, nil
	}
	return nil, nil
}

// Resume replays the match stream after lastSeq, or sends a snapshot when
// the events are gone or nothing was seen yet (websocket.md §6).
func (m *Manager) Resume(ctx context.Context, c realtime.Client, stream string, lastSeq int64) error {
	id, err := uuid.Parse(strings.TrimPrefix(stream, "match:"))
	if err != nil {
		return apierror.New(apierror.MatchNotFound)
	}
	o := m.local(id)
	if o == nil {
		return apierror.New(apierror.MatchNotFound)
	}
	p := o.match.Participant(c.UserID)
	if p == nil {
		return apierror.New(apierror.NotAParticipant)
	}
	r := recipient{UserID: c.UserID, Side: p.Side, Locale: p.Locale}
	t := realtime.Target{User: c.UserID, Session: c.SessionID}
	if lastSeq > 0 {
		var replayed []stored
		for _, s := range o.events {
			if int64(s.Seq) > lastSeq {
				replayed = append(replayed, s)
			}
		}
		if len(replayed) > 0 && int64(replayed[0].Seq) == lastSeq+1 {
			for _, s := range replayed {
				if err := m.d.RT.Send(ctx, t, o.meta.render(s, r)); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return m.d.RT.Send(ctx, t, o.meta.snapshot(o.match, r, m.d.Clock.Now().UnixMilli()))
}

// addTexts loads the wording of replacement questions into meta.
func (m *Manager) addTexts(ctx context.Context, mt *meta, match *engine.Match, qs []engine.Question) error {
	if len(qs) == 0 {
		return nil
	}
	locales := map[string]bool{}
	for _, p := range match.Participants {
		locales[p.Locale] = true
	}
	ls := make([]string, 0, len(locales))
	for l := range locales {
		ls = append(ls, l)
	}
	ids := make([]uuid.UUID, len(qs))
	for i, q := range qs {
		ids[i] = q.ID
	}
	texts, err := m.d.Texts.Texts(ctx, ids, ls)
	if err != nil {
		return err
	}
	if mt.Texts == nil {
		mt.Texts = map[uuid.UUID]map[string]content.QuestionText{}
	}
	maps.Copy(mt.Texts, texts)
	mt.addCategories(qs)
	return nil
}
