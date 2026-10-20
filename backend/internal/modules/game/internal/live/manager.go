// Package live runs live matches (backend-modules §3.4, match-lifecycle
// §5): one owner goroutine per match on the instance holding its Redis
// lease, intents from the WebSocket routed to it from any instance, every
// step persisted, saved to Redis and delivered as sequenced events; and
// reconnect replays or snapshots built from Redis on any instance.
package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Lease and retention (redis-keys §2, match-lifecycle §5).
const (
	LeaseTTL    = 10 * time.Second
	LeaseRenew  = 3 * time.Second
	FinishedTTL = time.Hour
	maxEvents   = 1000
	// SweepEvery is how often each instance looks for matches whose owner
	// lease expired; TakeoverGiveUp cancels a match nobody resumed in time.
	SweepEvery     = 2 * time.Second
	TakeoverGiveUp = 30 * time.Second
)

var liveKey = redis.Key("game", "live")

func ownerKey(m uuid.UUID) string  { return "game:{m:" + m.String() + "}:owner" }
func stateKey(m uuid.UUID) string  { return "game:{m:" + m.String() + "}:state" }
func eventsKey(m uuid.UUID) string { return "game:{m:" + m.String() + "}:events" }
func userLiveKey(u uuid.UUID) string {
	return redis.Key("game", "user", u.String(), "live")
}
func instChannel(instance string) string { return redis.Key("game", "ch", "inst", instance) }

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
	Redis     *redis.Client
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

	mu     sync.Mutex
	owners map[uuid.UUID]*owner
	ended  []EndFunc
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
	return &Manager{d: d, ctx: ctx, cancel: cancel, owners: map[uuid.UUID]*owner{}}
}

// Close stops every local owner (process shutdown); their leases expire
// and another instance takes the matches over (edge-cases §4).
func (m *Manager) Close() { m.cancel() }

var renewScript = goredis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
  redis.call('ZADD', KEYS[2], ARGV[3], ARGV[4])
  return 1
end
return 0`)

var releaseScript = goredis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then return redis.call('DEL', KEYS[1]) end
return 0`)

func (m *Manager) renewLease(ctx context.Context, id uuid.UUID) bool {
	n, err := renewScript.Run(ctx, m.d.Redis, []string{ownerKey(id), liveKey},
		m.d.Instance, LeaseTTL.Milliseconds(), m.d.Clock.Now().UnixMilli(), id.String()).Int()
	return err == nil && n == 1
}

func (m *Manager) releaseLease(ctx context.Context, id uuid.UUID) {
	if err := releaseScript.Run(ctx, m.d.Redis, []string{ownerKey(id)}, m.d.Instance).Err(); err != nil {
		m.d.Logger.WarnContext(ctx, "game: releasing a match lease failed", "match_id", id, "error", err)
	}
}

func (m *Manager) local(id uuid.UUID) *owner {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.owners[id]
}

func (m *Manager) forget(o *owner) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.owners[o.match.ID] == o {
		delete(m.owners, o.match.ID)
	}
}

// Start takes ownership of a match just created in `waiting`, saves its
// live state and delivers the creation events (T1).
// bots are development practice opponents the owner plays (D-066).
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
	metaJSON, err := json.Marshal(mt)
	if err != nil {
		return err
	}
	ok, err := m.d.Redis.SetNX(ctx, ownerKey(match.ID), m.d.Instance, LeaseTTL).Result()
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("game: match %s already has an owner", match.ID)
	}
	if err := m.d.Redis.HSet(ctx, stateKey(match.ID), "meta", metaJSON).Err(); err != nil {
		return err
	}
	m.d.Redis.ZAdd(ctx, liveKey, goredis.Z{Score: float64(m.d.Clock.Now().UnixMilli()), Member: match.ID.String()})
	o := m.newOwner(match, mt, map[uuid.UUID]uuid.UUID{})
	// The owner outlives the request that created the match: it runs on
	// the manager's context, stopped by Close.
	if err := o.commit(m.ctx, out, m.d.Clock.Now().UnixMilli()); err != nil { //nolint:contextcheck // see above
		o.halt()
		m.forget(o)
		m.releaseLease(ctx, match.ID)
		return err
	}
	go o.run(m.ctx) //nolint:contextcheck // the owner runs on the manager's context
	return nil
}

func (m *Manager) newOwner(match *engine.Match, mt *meta, seats map[uuid.UUID]uuid.UUID) *owner {
	o := &owner{m: m, match: match, meta: mt, seats: seats, inbox: make(chan command, inboxSize), stop: make(chan struct{})}
	m.mu.Lock()
	m.owners[match.ID] = o
	m.mu.Unlock()
	return o
}

// route delivers a command to the match's owner, here or on its instance.
func (m *Manager) route(ctx context.Context, c command) error {
	if o := m.local(c.Match); o != nil {
		return o.post(c)
	}
	inst, err := m.d.Redis.Get(ctx, ownerKey(c.Match)).Result()
	if errors.Is(err, goredis.Nil) {
		return apierror.New(apierror.MatchNotFound)
	}
	if err != nil {
		return err
	}
	if inst == m.d.Instance {
		return apierror.New(apierror.MatchNotFound) // lease left over from a stopped owner
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return m.d.Redis.Publish(ctx, instChannel(inst), b).Err()
}

// Run receives commands forwarded from other instances until ctx ends.
func (m *Manager) Run(ctx context.Context) error {
	defer m.Close()
	sub := m.d.Redis.Subscribe(ctx, instChannel(m.d.Instance))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("game: subscribe: %w", err)
	}
	ch := sub.Channel()
	sweep := m.d.Clock.NewTicker(SweepEvery)
	defer sweep.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-sweep.C():
			m.sweep(ctx)
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			var c command
			if err := json.Unmarshal([]byte(msg.Payload), &c); err != nil {
				m.d.Logger.WarnContext(ctx, "game: bad forwarded command", "error", err)
				continue
			}
			var err error = apierror.New(apierror.MatchNotFound)
			if o := m.local(c.Match); o != nil {
				err = o.post(c)
			}
			if err != nil && c.Re != "" {
				_ = m.d.RT.Reply(ctx, c.Client, c.Re, err)
			}
		}
	}
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

// liveMatch returns the user's live match, if any.
func (m *Manager) liveMatch(ctx context.Context, user uuid.UUID) (uuid.UUID, bool) {
	v, err := m.d.Redis.Get(ctx, userLiveKey(user)).Result()
	if err != nil {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(v)
	return id, err == nil
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
	vals, err := m.d.Redis.HMGet(ctx, stateKey(id), "engine", "meta").Result()
	if err != nil {
		return err
	}
	engineJSON, _ := vals[0].(string)
	metaJSON, _ := vals[1].(string)
	if engineJSON == "" || metaJSON == "" {
		return apierror.New(apierror.MatchNotFound)
	}
	match, err := engine.Restore([]byte(engineJSON), m.d.Random)
	if err != nil {
		return err
	}
	var mt meta
	if err := json.Unmarshal([]byte(metaJSON), &mt); err != nil {
		return err
	}
	p := match.Participant(c.UserID)
	if p == nil {
		return apierror.New(apierror.NotAParticipant)
	}
	r := recipient{UserID: c.UserID, Side: p.Side, Locale: p.Locale}
	t := realtime.Target{User: c.UserID, Session: c.SessionID}
	if lastSeq > 0 {
		entries, err := m.d.Redis.XRange(ctx, eventsKey(id), "0-"+strconv.FormatInt(lastSeq+1, 10), "+").Result()
		if err != nil {
			return err
		}
		complete := (len(entries) == 0 && int64(match.Seq()) == lastSeq) ||
			(len(entries) > 0 && entries[0].ID == "0-"+strconv.FormatInt(lastSeq+1, 10))
		if complete {
			for _, e := range entries {
				raw, _ := e.Values["e"].(string)
				var s stored
				if err := json.Unmarshal([]byte(raw), &s); err != nil {
					return err
				}
				if err := m.d.RT.Send(ctx, t, mt.render(s, r)); err != nil {
					return err
				}
			}
			return nil
		}
	}
	return m.d.RT.Send(ctx, t, mt.snapshot(match, r, m.d.Clock.Now().UnixMilli()))
}

// ---------- Owner loss & takeover (edge-cases §4) ----------

// sweep takes over matches whose owner stopped renewing its lease.
func (m *Manager) sweep(ctx context.Context) {
	now := m.d.Clock.Now()
	stale, err := m.d.Redis.ZRangeByScoreWithScores(ctx, liveKey, &goredis.ZRangeBy{
		Min: "-inf", Max: strconv.FormatInt(now.Add(-LeaseTTL).UnixMilli(), 10),
	}).Result()
	if err != nil {
		m.d.Logger.WarnContext(ctx, "game: sweeping live matches failed", "error", err)
		return
	}
	for _, z := range stale {
		id, err := uuid.Parse(fmt.Sprint(z.Member))
		if err != nil || m.local(id) != nil {
			continue
		}
		ok, err := m.d.Redis.SetNX(ctx, ownerKey(id), m.d.Instance, LeaseTTL).Result()
		if err != nil || !ok {
			continue // another instance owns or is taking it
		}
		lost := now.Sub(time.UnixMilli(int64(z.Score)))
		if err := m.takeover(ctx, id, lost); err != nil {
			m.d.Logger.ErrorContext(ctx, "game: taking over a match failed", "match_id", id, "error", err)
			m.releaseLease(ctx, id)
		}
	}
}

// takeover resumes a match from its Redis state on this instance.
func (m *Manager) takeover(ctx context.Context, id uuid.UUID, lost time.Duration) error {
	vals, err := m.d.Redis.HMGet(ctx, stateKey(id), "engine", "meta", "seats").Result()
	if err != nil {
		return err
	}
	engineJSON, _ := vals[0].(string)
	metaJSON, _ := vals[1].(string)
	seatsJSON, _ := vals[2].(string)
	if engineJSON == "" || metaJSON == "" {
		// Redis lost the live state: Postgres decides (edge-cases §4).
		if err := m.d.Persister.CancelOrphan(ctx, id); err != nil {
			return err
		}
		m.d.Redis.ZRem(ctx, liveKey, id.String())
		m.releaseLease(ctx, id)
		return nil
	}
	match, err := engine.Restore([]byte(engineJSON), m.d.Random)
	if err != nil {
		return err
	}
	if match.Status.Terminal() {
		m.d.Redis.ZRem(ctx, liveKey, id.String())
		m.releaseLease(ctx, id)
		return nil
	}
	var mt meta
	if err := json.Unmarshal([]byte(metaJSON), &mt); err != nil {
		return err
	}
	seats := map[uuid.UUID]uuid.UUID{}
	if seatsJSON != "" {
		if err := json.Unmarshal([]byte(seatsJSON), &seats); err != nil {
			return err
		}
	}
	now := m.d.Clock.Now().UnixMilli()
	var out engine.Output
	if lost > TakeoverGiveUp+LeaseTTL {
		out = match.Cancel(domain.CancelServerError, uuid.Nil, now)
	} else {
		spare, err := m.d.Persister.Spare(ctx, match)
		if err != nil {
			m.d.Logger.WarnContext(ctx, "game: no replacement question for a voided round", "match_id", id, "error", err)
		}
		if err := m.addTexts(ctx, &mt, match, spare); err != nil {
			return err
		}
		out = match.Takeover(spare, now)
	}
	m.d.Redis.ZAdd(ctx, liveKey, goredis.Z{Score: float64(now), Member: id.String()})
	o := m.newOwner(match, &mt, seats)
	m.d.Logger.InfoContext(ctx, "game: match taken over", "match_id", id, "lost_ms", lost.Milliseconds())
	if err := o.commit(m.ctx, out, now); err != nil { //nolint:contextcheck // the owner runs on the manager's context
		return err
	}
	if match.Status.Terminal() {
		o.finish(m.ctx) //nolint:contextcheck // as above
		return nil
	}
	go o.run(m.ctx) //nolint:contextcheck // as above
	return nil
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
	b, err := json.Marshal(mt)
	if err != nil {
		return err
	}
	return m.d.Redis.HSet(ctx, stateKey(match.ID), "meta", b).Err()
}
