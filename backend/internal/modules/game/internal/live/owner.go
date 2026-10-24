package live

import (
	"context"
	"math/rand/v2"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/realtime"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
)

// Command kinds of an owner's mailbox.
const (
	kindJoin       = "join"
	kindReady      = "ready"
	kindAnswer     = "answer"
	kindPick       = "pick"
	kindPropose    = "propose"
	kindLock       = "lock"
	kindLeave      = "leave"
	kindDisconnect = "disconnect"
	kindRTT        = "rtt"
	kindFire       = "fire"
	kindQuick      = "quick"
)

// QuickPhrases are the preset in-match phrases (translated on the device;
// the same set as team chat, D-088).
var QuickPhrases = []string{"good_luck", "well_played", "nice", "thanks", "oops", "lets_go", "gg", "rematch"}

// quickGapMS spaces one player's quick phrases.
const quickGapMS = 3000

// command is one entry of a match owner's mailbox; it travels between
// instances as JSON when the owner runs elsewhere.
type command struct {
	Kind   string          `json:"kind"`
	Match  uuid.UUID       `json:"match"`
	Client realtime.Client `json:"client"`
	Re     string          `json:"re,omitempty"`
	Round  int             `json:"round,omitempty"`
	// Option is the answer option, or the category of a pick.
	Option uuid.UUID `json:"option,omitempty"`
	// AtMS is when the intent was read from the socket (timing.md §5).
	AtMS  int64        `json:"at_ms,omitempty"`
	RTT   int          `json:"rtt,omitempty"`
	Timer engine.Timer `json:"timer"`
	// Phrase is a quick-chat phrase id.
	Phrase string `json:"phrase,omitempty"`
}

// owner runs one live match: the single writer of its state.
type owner struct {
	m     *Manager
	match *engine.Match
	meta  *meta
	seats map[uuid.UUID]uuid.UUID // user → session holding the seat
	// quick is each player's last quick phrase (ms), for spacing.
	quick  map[uuid.UUID]int64
	inbox  chan command
	stop   chan struct{}
	timers []clock.Timer
	events []stored

	stopOnce sync.Once
}

const (
	inboxSize     = 64
	postTimeout   = 2 * time.Second
	persistTries  = 3
	persistBackof = 200 * time.Millisecond
)

func (o *owner) post(c command) error {
	t := o.m.d.Clock.NewTimer(postTimeout)
	defer t.Stop()
	select {
	case o.inbox <- c:
		return nil
	case <-o.stop:
		return apierror.New(apierror.MatchStateInvalid).With("reason", "match_over")
	case <-t.C():
		return apierror.New(apierror.Unavailable).WithRetryAfter(time.Second)
	}
}

func (o *owner) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			o.halt()
			return
		case c := <-o.inbox:
			o.handle(ctx, c)
			if o.match.Status.Terminal() {
				o.finish(ctx)
				return
			}
		}
	}
}

func (o *owner) handle(ctx context.Context, c command) {
	now := c.AtMS
	if now == 0 {
		now = o.m.d.Clock.Now().UnixMilli()
	}
	user := c.Client.UserID
	if c.Kind == kindQuick {
		err := o.quickChat(ctx, user, c.Phrase, now)
		if c.Re != "" {
			if rerr := o.m.d.RT.Reply(ctx, c.Client, c.Re, err); rerr != nil {
				o.m.d.Logger.DebugContext(ctx, "game: reply failed", "error", rerr)
			}
		}
		return
	}
	var out engine.Output
	var err error
	switch c.Kind {
	case kindJoin:
		if o.match.Participant(user) == nil {
			err = apierror.New(apierror.NotAParticipant)
			break
		}
		if prev := o.seats[user]; prev != uuid.Nil && prev != c.Client.SessionID {
			o.send(ctx, realtime.Target{User: user, Session: prev}, realtime.Frame{T: "match.seat_moved", S: streamOf(o.match.ID), D: map[string]any{"match_id": o.match.ID}})
		}
		o.seats[user] = c.Client.SessionID
		out = o.match.Reconnect(user, now)
	case kindReady:
		out, err = o.match.Ready(user, now)
	case kindAnswer:
		out, err = o.match.Answer(user, c.Round, c.Option, now)
	case kindPick:
		out, err = o.match.Pick(user, c.Round, c.Option, now)
	case kindPropose:
		out, err = o.match.Propose(user, c.Round, c.Option, now)
	case kindLock:
		out, err = o.match.Lock(user, c.Round, c.Option, now)
	case kindLeave:
		out, err = o.match.Leave(user, now)
	case kindDisconnect:
		if o.seats[user] != c.Client.SessionID {
			return // an older socket of a player who already joined again
		}
		out = o.match.Disconnect(user, now)
	case kindRTT:
		o.match.RTT(user, c.RTT)
		return
	case kindFire:
		out = o.match.Fire(c.Timer, now)
	}
	if err == nil {
		err = o.commit(ctx, out, now)
	} else if c.Kind == kindAnswer || c.Kind == kindPick || c.Kind == kindPropose || c.Kind == kindLock {
		o.match.Rejected(user, err)
	}
	if c.Re != "" {
		if rerr := o.m.d.RT.Reply(ctx, c.Client, c.Re, err); rerr != nil {
			o.m.d.Logger.DebugContext(ctx, "game: reply failed", "error", rerr)
		}
	}
	if c.Kind == kindJoin && err == nil {
		t := realtime.Target{User: user, Session: c.Client.SessionID}
		if o.match.Status.Live() {
			if perr := o.m.d.RT.SetPlaying(ctx, t, true); perr != nil {
				o.m.d.Logger.DebugContext(ctx, "game: heartbeat switch failed", "error", perr)
			}
		}
		o.send(ctx, t, o.meta.snapshot(o.match, o.recipient(user), o.m.d.Clock.Now().UnixMilli()))
	}
}

// quickChat sends a preset phrase to everyone in the match (all sides,
// D-031): participants only, one every 3 s; nothing is stored.
func (o *owner) quickChat(ctx context.Context, user uuid.UUID, phrase string, now int64) error {
	if o.match.Participant(user) == nil {
		return apierror.New(apierror.NotAParticipant)
	}
	if !slices.Contains(QuickPhrases, phrase) {
		return apierror.Validation(apierror.FieldError{Field: "phrase_id", Rule: "one_of"})
	}
	if o.quick == nil {
		o.quick = map[uuid.UUID]int64{}
	}
	if last, ok := o.quick[user]; ok && now-last < quickGapMS {
		return apierror.New(apierror.RateLimited).WithRetryAfter(time.Duration(quickGapMS-(now-last)) * time.Millisecond)
	}
	o.quick[user] = now
	f := realtime.Frame{T: "match.quick_chat", S: streamOf(o.match.ID), TS: now, D: map[string]any{
		"match_id": o.match.ID, "user_id": user, "phrase_id": phrase,
	}}
	for _, p := range o.match.Participants {
		o.send(ctx, realtime.Target{User: p.UserID}, f)
	}
	return nil
}

// commit persists an output, saves the state, publishes its events and
// arms its timers (match-lifecycle §5). A persistence failure that keeps
// failing cancels the match with server_error (round-lifecycle §5).
func (o *owner) commit(ctx context.Context, out engine.Output, now int64) error {
	if err := o.persist(ctx, out); err != nil {
		o.m.d.Logger.ErrorContext(ctx, "game: persisting a match step failed; cancelling", "match_id", o.match.ID, "error", err)
		cancelled := o.match.Cancel(domain.CancelServerError, uuid.Nil, now)
		if perr := o.persist(ctx, cancelled); perr != nil {
			o.m.d.Logger.ErrorContext(ctx, "game: persisting the cancellation failed", "match_id", o.match.ID, "error", perr)
		}
		out.Events = append(out.Events, cancelled.Events...)
		out.Timers = nil
		if serr := o.save(ctx, out.Events, now); serr != nil {
			o.m.d.Logger.ErrorContext(ctx, "game: saving match state failed", "match_id", o.match.ID, "error", serr)
		}
		o.deliver(ctx, out.Events, now)
		return apierror.Wrap(apierror.Internal, err)
	}
	if err := o.save(ctx, out.Events, now); err != nil {
		// Redis is the live copy; Postgres already holds the step. Clients
		// still get the events; a later save or a snapshot catches up.
		o.m.d.Logger.ErrorContext(ctx, "game: saving match state failed", "match_id", o.match.ID, "error", err)
	}
	o.deliver(ctx, out.Events, now)
	o.arm(out.Timers, now)
	o.playBots(out.Events)
	if out.NeedQuestions > 0 {
		return o.commit(ctx, o.more(ctx, now), now)
	}
	return nil
}

// more answers NeedQuestions (Sudden Death): the next batch with its
// texts, or none when the pool ran dry or failed, which ends the match by
// score (match-types §3.3).
func (o *owner) more(ctx context.Context, now int64) engine.Output {
	qs, err := o.m.d.Persister.More(ctx, o.match)
	if err == nil {
		err = o.m.addTexts(ctx, o.meta, o.match, qs)
	}
	if err != nil {
		o.m.d.Logger.ErrorContext(ctx, "game: loading more questions failed", "match_id", o.match.ID, "error", err)
		qs = nil
	}
	return o.match.AddQuestions(qs, now)
}

func (o *owner) persist(ctx context.Context, out engine.Output) error {
	var err error
	for i := range persistTries {
		if err = o.m.d.Persister.Apply(ctx, o.match, out); err == nil {
			return nil
		}
		if i+1 < persistTries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(persistBackof << i):
			}
		}
	}
	return err
}

// save records new events in memory.
func (o *owner) save(ctx context.Context, events []engine.Event, now int64) error {
	for _, e := range events {
		s, err := toStored(e, now)
		if err != nil {
			return err
		}
		o.events = append(o.events, s)
		if len(o.events) > maxEvents {
			o.events = o.events[len(o.events)-maxEvents:]
		}
	}
	return nil
}

func (o *owner) recipient(user uuid.UUID) recipient {
	r := recipient{UserID: user}
	if p := o.match.Participant(user); p != nil {
		r.Side, r.Locale = p.Side, p.Locale
	}
	return r
}

// deliver renders each event for each participant and sends it to the
// session holding their seat (all their sessions before they joined).
func (o *owner) deliver(ctx context.Context, events []engine.Event, now int64) {
	for _, e := range events {
		s, err := toStored(e, now)
		if err != nil {
			o.m.d.Logger.ErrorContext(ctx, "game: encoding an event failed", "type", e.Type, "error", err)
			continue
		}
		for _, p := range o.match.Participants {
			o.send(ctx, realtime.Target{User: p.UserID, Session: o.seats[p.UserID]}, o.meta.render(s, o.recipient(p.UserID)))
		}
	}
}

func (o *owner) send(ctx context.Context, t realtime.Target, f realtime.Frame) {
	if err := o.m.d.RT.Send(ctx, t, f); err != nil {
		o.m.d.Logger.WarnContext(ctx, "game: delivering an event failed", "type", f.T, "error", err)
	}
}

// arm schedules the output's timers; the engine ignores stale ones.
func (o *owner) arm(timers []engine.Timer, now int64) {
	for _, t := range timers {
		o.later(time.Duration(t.AtMS-now)*time.Millisecond, command{Kind: kindFire, Match: o.match.ID, Timer: t})
	}
}

// later posts a command to the mailbox after d.
func (o *owner) later(d time.Duration, c command) {
	o.timers = append(o.timers, o.m.d.Clock.AfterFunc(max(d, 0), func() {
		select {
		case o.inbox <- c:
		case <-o.stop:
		}
	}))
}

// Practice bot behaviour (development only, D-066).
const (
	botReadyAfter   = 800 * time.Millisecond
	botMinAnswerMS  = 1_500
	botAnswerSpread = 4_500
	botCorrectPct   = 60
)

// playBots makes the practice bots ready after creation and answer every
// round they are expected in, right about 60 % of the time.
func (o *owner) playBots(events []engine.Event) {
	if len(o.meta.Bots) == 0 {
		return
	}
	now := o.m.d.Clock.Now().UnixMilli()
	for _, e := range events {
		switch e.Type {
		case "match.created":
			for _, b := range o.meta.Bots {
				o.later(botReadyAfter, command{Kind: kindReady, Match: o.match.ID, Client: realtime.Client{UserID: b}})
			}
		case "round.started":
			r := o.match.Round
			if r == nil {
				continue
			}
			for _, b := range o.meta.Bots {
				opt := r.Question.Correct
				if rand.IntN(100) >= botCorrectPct { //nolint:gosec // G404: practice bot, not security relevant
					for _, x := range r.Question.Options {
						if x != r.Question.Correct {
							opt = x
							break
						}
					}
				}
				at := r.ShowAtMS + botMinAnswerMS + int64(rand.IntN(botAnswerSpread)) //nolint:gosec // G404: as above
				o.later(time.Duration(at-now)*time.Millisecond, command{
					Kind: kindAnswer, Match: o.match.ID, Client: realtime.Client{UserID: b}, Round: r.No, Option: opt,
				})
			}
		}
	}
}

// halt stops the owner's timers and refuses further commands.
func (o *owner) halt() {
	o.stopOnce.Do(func() { close(o.stop) })
	for _, t := range o.timers {
		t.Stop()
	}
	o.timers = nil
}

// finish ends a terminal match: timers stop, participants set to not playing.
func (o *owner) finish(ctx context.Context) {
	o.halt()
	ctx = context.WithoutCancel(ctx)
	for user, session := range o.seats {
		if err := o.m.d.RT.SetPlaying(ctx, realtime.Target{User: user, Session: session}, false); err != nil {
			o.m.d.Logger.DebugContext(ctx, "game: heartbeat switch failed", "error", err)
		}
	}
	o.m.forget(o)
	o.m.notifyEnded(ctx, o.match)
}
