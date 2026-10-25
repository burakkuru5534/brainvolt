package service

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// RoundScheduled is the payload of tournament.RoundScheduled: a bracket
// match is ready and its players should show up.
type RoundScheduled struct {
	TournamentID uuid.UUID   `json:"tournament_id"`
	RoundNo      int         `json:"round_no"`
	Users        []uuid.UUID `json:"user_ids"`
	ReadyUntil   time.Time   `json:"ready_until"`
}

// Finished is the payload of tournament.TournamentFinished.
type Finished struct {
	TournamentID uuid.UUID   `json:"tournament_id"`
	Placements   []Placement `json:"placements"`
	// Kind is official or friends; friend tournaments earn nothing (D-109).
	Kind string `json:"kind"`
}

// Placement is a player's final place and XP reward.
type Placement struct {
	UserID uuid.UUID `json:"user_id"`
	Place  int       `json:"place"`
	XP     int       `json:"xp"`
}

// round is one tournament's state inside a transaction.
type round struct {
	s       *Service
	tx      pgx.Tx
	q       *sqlc.Queries
	t       sqlc.TournamentTournament
	entries map[uuid.UUID]*sqlc.TournamentEntry
	rounds  int
	now     time.Time
	touched bool // the bracket changed: tell the players
}

func (s *Service) load(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*round, error) {
	q := sqlc.New(tx)
	t, err := q.LockTournament(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := q.Entries(ctx, id)
	if err != nil {
		return nil, err
	}
	r := &round{s: s, tx: tx, q: q, t: t, entries: map[uuid.UUID]*sqlc.TournamentEntry{}, now: s.d.Clock.Now()}
	seeded := 0
	for i := range rows {
		r.entries[rows[i].ID] = &rows[i]
		if rows[i].Seed != nil {
			seeded++
		}
	}
	if seeded > 0 {
		r.rounds = domain.Rounds(domain.Size(seeded))
	}
	return r, nil
}

// inTournament runs fn on a locked tournament and then tells its players
// when the bracket changed.
func (s *Service) inTournament(ctx context.Context, id uuid.UUID, fn func(r *round) error) error {
	var users []uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		r, err := s.load(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := fn(r); err != nil {
			return err
		}
		if r.touched {
			users, err = lineups(ctx, r.q, id)
		}
		return err
	})
	if err == nil {
		s.pushUpdated(ctx, id, users)
	}
	return err
}

func (r *round) publish(ctx context.Context, name string, payload any) error {
	if r.s.d.Events == nil {
		return nil
	}
	return r.s.d.Events.PublishTx(ctx, r.tx, name, r.now, payload)
}

// start seeds the checked-in entries at random and builds the bracket; too
// few entries cancel the tournament.
func (r *round) start(ctx context.Context) error {
	if r.t.Status != StatusRegistration {
		return nil
	}
	r.touched = true
	var in []*sqlc.TournamentEntry
	for _, e := range r.entries {
		if e.Status == "checked_in" {
			in = append(in, e)
		}
	}
	if err := r.q.WithdrawUnchecked(ctx, r.t.ID); err != nil {
		return err
	}
	if len(in) < int(r.t.MinEntrants) {
		if err := r.q.SetTournamentStatus(ctx, sqlc.SetTournamentStatusParams{ID: r.t.ID, Status: StatusCancelled, Now: r.now}); err != nil {
			return err
		}
		return r.s.refundFee(ctx, r.tx, &r.t, r.now)
	}
	rand.Shuffle(len(in), func(i, j int) { in[i], in[j] = in[j], in[i] }) //nolint:gosec // G404: seeding is not security-sensitive
	for i, e := range in {
		seed := int32(i + 1)
		e.Seed, e.Status = &seed, "active"
		if err := r.q.SeedEntry(ctx, sqlc.SeedEntryParams{ID: e.ID, Seed: &seed}); err != nil {
			return err
		}
	}
	size := domain.Size(len(in))
	r.rounds = domain.Rounds(size)
	for rn := 1; rn <= r.rounds; rn++ {
		for slot := 1; slot <= size>>rn; slot++ {
			p := sqlc.InsertSlotParams{TournamentID: r.t.ID, RoundNo: int16(rn), SlotNo: int16(slot)}
			if rn == 1 {
				pair := domain.FirstRound(len(in))[slot-1]
				p.EntryAID = &in[pair.A-1].ID
				if pair.B > 0 {
					p.EntryBID = &in[pair.B-1].ID
				}
			}
			if err := r.q.InsertSlot(ctx, p); err != nil {
				return err
			}
		}
	}
	if err := r.q.SetTournamentStatus(ctx, sqlc.SetTournamentStatusParams{ID: r.t.ID, Status: StatusRunning, Now: r.now}); err != nil {
		return err
	}
	r.t.Status = StatusRunning
	for slot, pair := range domain.FirstRound(len(in)) {
		m, err := r.q.LockSlot(ctx, sqlc.LockSlotParams{TournamentID: r.t.ID, RoundNo: 1, SlotNo: int16(slot + 1)})
		if err != nil {
			return err
		}
		if pair.B == 0 {
			if err := r.advance(ctx, &m, *m.EntryAID, "walkover"); err != nil {
				return err
			}
			continue
		}
		if err := r.open(ctx, &m); err != nil {
			return err
		}
	}
	return nil
}

// open starts a slot's ready window and calls its players.
func (r *round) open(ctx context.Context, m *sqlc.TournamentBracketMatch) error {
	until := r.now.Add(time.Duration(r.t.RoundReadyWindowS) * time.Second)
	if err := r.q.OpenSlot(ctx, sqlc.OpenSlotParams{TournamentID: r.t.ID, RoundNo: m.RoundNo, SlotNo: m.SlotNo, Now: &r.now, ReadyUntil: &until}); err != nil {
		return err
	}
	users := slices.Concat(r.entries[*m.EntryAID].Lineup, r.entries[*m.EntryBID].Lineup)
	return r.publish(ctx, EventRoundScheduled, RoundScheduled{TournamentID: r.t.ID, RoundNo: int(m.RoundNo), Users: users, ReadyUntil: until})
}

// advance records a slot's winner, places the loser and moves the winner
// on; the final ends the tournament.
func (r *round) advance(ctx context.Context, m *sqlc.TournamentBracketMatch, winner uuid.UUID, status string) error {
	r.touched = true
	if err := r.q.FinishSlot(ctx, sqlc.FinishSlotParams{TournamentID: r.t.ID, RoundNo: m.RoundNo, SlotNo: m.SlotNo, Status: status, WinnerEntryID: &winner}); err != nil {
		return err
	}
	for _, id := range []*uuid.UUID{m.EntryAID, m.EntryBID} {
		if id != nil && *id != winner {
			place := int32(domain.LoserPlace(int(m.RoundNo), r.rounds)) //nolint:gosec // small
			if err := r.place(ctx, *id, "eliminated", place); err != nil {
				return err
			}
		}
	}
	if int(m.RoundNo) == r.rounds {
		if err := r.place(ctx, winner, "winner", 1); err != nil {
			return err
		}
		return r.finish(ctx)
	}
	nr, ns, sideA := domain.Next(int(m.RoundNo), int(m.SlotNo))
	p := sqlc.SetSlotEntryParams{TournamentID: r.t.ID, RoundNo: int16(nr), SlotNo: int16(ns), SideA: sideA, EntryID: winner} //nolint:gosec // ≤ 64
	if err := r.q.SetSlotEntry(ctx, p); err != nil {
		return err
	}
	next, err := r.q.LockSlot(ctx, sqlc.LockSlotParams{TournamentID: r.t.ID, RoundNo: p.RoundNo, SlotNo: p.SlotNo})
	if err != nil {
		return err
	}
	if next.EntryAID != nil && next.EntryBID != nil {
		return r.open(ctx, &next)
	}
	return nil
}

func (r *round) place(ctx context.Context, entry uuid.UUID, status string, place int32) error {
	e := r.entries[entry]
	e.Status, e.FinalPlace = status, &place
	return r.q.PlaceEntry(ctx, sqlc.PlaceEntryParams{ID: entry, Status: status, FinalPlace: &place})
}

// finish closes the tournament and hands out the rewards (progression).
func (r *round) finish(ctx context.Context) error {
	if err := r.q.SetTournamentStatus(ctx, sqlc.SetTournamentStatusParams{ID: r.t.ID, Status: StatusFinished, Now: r.now}); err != nil {
		return err
	}
	var rewards TournamentRewards
	if err := json.Unmarshal(r.t.Rewards, &rewards); err != nil {
		return err
	}
	f := Finished{TournamentID: r.t.ID, Placements: []Placement{}, Kind: r.t.Kind}
	for _, e := range r.entries {
		if e.FinalPlace == nil {
			continue
		}
		for _, u := range e.Lineup {
			f.Placements = append(f.Placements, Placement{UserID: u, Place: int(*e.FinalPlace), XP: rewards.XPFor(int(*e.FinalPlace))})
		}
	}
	slices.SortFunc(f.Placements, func(a, b Placement) int { return a.Place - b.Place })
	return r.publish(ctx, EventFinished, f)
}

// higherSeed is the better-seeded entry of a slot (draws, D-090).
func (r *round) higherSeed(m *sqlc.TournamentBracketMatch) uuid.UUID {
	a, b := r.entries[*m.EntryAID], r.entries[*m.EntryBID]
	if b.Seed != nil && (a.Seed == nil || *b.Seed < *a.Seed) {
		return b.ID
	}
	return a.ID
}

// byPresence decides a slot whose window closed: the entry with all its
// players present wins, else the one with more, else the higher seed.
func (r *round) byPresence(m *sqlc.TournamentBracketMatch) uuid.UUID {
	count := func(id uuid.UUID) int {
		n := 0
		for _, u := range r.entries[id].Lineup {
			if slices.Contains(m.Present, u) {
				n++
			}
		}
		return n
	}
	a, b := count(*m.EntryAID), count(*m.EntryBID)
	switch {
	case a > b:
		return *m.EntryAID
	case b > a:
		return *m.EntryBID
	}
	return r.higherSeed(m)
}

func (r *round) fullyPresent(id uuid.UUID, present []uuid.UUID) bool {
	for _, u := range r.entries[id].Lineup {
		if !slices.Contains(present, u) {
			return false
		}
	}
	return true
}

// TournamentPlay is the answer of POST /v1/tournaments/{id}/play.
type TournamentPlay struct {
	// MatchID is set once both entries are present and the match exists.
	MatchID *uuid.UUID            `json:"match_id"`
	Present []uuid.UUID           `json:"present"`
	Match   TournamentBracketSlot `json:"bracket_match"`
}

// Play marks the caller present for their bracket match; when every
// player of both entries is there, the match is created and they are
// sent to it (tournament.match_ready).
func (s *Service) Play(ctx context.Context, user, id uuid.UUID) (TournamentPlay, error) {
	var out TournamentPlay
	var notify []uuid.UUID
	err := s.inTournament(ctx, id, func(r *round) error {
		if r.t.Status != StatusRunning {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "tournament_"+r.t.Status)
		}
		e, err := r.q.EntryOf(ctx, sqlc.EntryOfParams{TournamentID: id, UserID: user})
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return err
		}
		m, err := r.q.CurrentSlotOf(ctx, sqlc.CurrentSlotOfParams{TournamentID: id, EntryID: &e.ID})
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "no_match_now")
		}
		if err != nil {
			return err
		}
		if m.Status == "in_progress" {
			out = TournamentPlay{MatchID: m.MatchID, Present: m.Present, Match: toSlot(&m)}
			return nil
		}
		if m.ReadyUntil != nil && !r.now.Before(*m.ReadyUntil) {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "ready_window_closed")
		}
		r.touched = true
		if !slices.Contains(m.Present, user) {
			m.Present = append(m.Present, user)
		}
		if !r.fullyPresent(*m.EntryAID, m.Present) || !r.fullyPresent(*m.EntryBID, m.Present) {
			out = TournamentPlay{Present: m.Present, Match: toSlot(&m)}
			return r.q.SetPresent(ctx, sqlc.SetPresentParams{TournamentID: id, RoundNo: m.RoundNo, SlotNo: m.SlotNo, Present: m.Present})
		}
		matchID, err := s.createMatch(ctx, r, &m)
		if err != nil {
			return err
		}
		if err := r.q.SetPresent(ctx, sqlc.SetPresentParams{TournamentID: id, RoundNo: m.RoundNo, SlotNo: m.SlotNo, Present: m.Present}); err != nil {
			return err
		}
		if err := r.q.StartSlot(ctx, sqlc.StartSlotParams{TournamentID: id, RoundNo: m.RoundNo, SlotNo: m.SlotNo, MatchID: &matchID}); err != nil {
			return err
		}
		m.Status, m.MatchID = "in_progress", &matchID
		out = TournamentPlay{MatchID: &matchID, Present: m.Present, Match: toSlot(&m)}
		notify = m.Present
		return nil
	})
	if err != nil {
		return TournamentPlay{}, err
	}
	if s.d.Pusher != nil {
		for _, u := range notify {
			if err := s.d.Pusher.Push(ctx, u, "tournament.match_ready", map[string]any{"tournament_id": id, "match_id": out.MatchID}); err != nil {
				s.d.Logger.DebugContext(ctx, "tournament: match_ready not delivered", "error", err)
			}
		}
	}
	return out, nil
}

// createMatch starts the game match of a slot: side 1 is entry A.
func (s *Service) createMatch(ctx context.Context, r *round, m *sqlc.TournamentBracketMatch) (uuid.UUID, error) {
	if s.d.Games == nil {
		return uuid.Nil, errors.New("tournament: no game in this process")
	}
	var cfg game.Config
	if err := json.Unmarshal(r.t.MatchConfig, &cfg); err != nil {
		return uuid.Nil, err
	}
	a, b := r.entries[*m.EntryAID], r.entries[*m.EntryBID]
	locales, err := s.d.Cards.LocalesOf(ctx, slices.Concat(a.Lineup, b.Lineup))
	if err != nil {
		return uuid.Nil, err
	}
	req := game.CreateRequest{Config: cfg, OriginRef: &r.t.ID}
	for i, e := range []*sqlc.TournamentEntry{a, b} {
		for _, u := range e.Lineup {
			req.Sides[i] = append(req.Sides[i], game.Member{UserID: u, Locale: locales[u]})
		}
		switch e.EntityKind {
		case "team":
			req.Teams[i] = &e.EntityID
		case "couple":
			req.Couples[i] = &e.EntityID
		}
	}
	c, err := s.d.Games.Start(ctx, req)
	if err != nil {
		return uuid.Nil, err
	}
	return c.Match.ID, nil
}

// matchEnded is what tournament reads from game.MatchCompleted and
// game.MatchCancelled.
type matchEnded struct {
	MatchID    uuid.UUID   `json:"match_id"`
	Origin     string      `json:"origin"`
	WinnerSide int         `json:"winner_side_no"`
	IsDraw     bool        `json:"is_draw"`
	Reason     string      `json:"reason"`
	ReadyIDs   []uuid.UUID `json:"ready_ids"`
	StartedAt  *time.Time  `json:"started_at"`
}

// slotOf finds the bracket match of a game match and runs fn on it.
func (s *Service) slotOf(ctx context.Context, e events.Envelope, fn func(r *round, m *sqlc.TournamentBracketMatch, p *matchEnded) error) error {
	var p matchEnded
	if err := e.Decode(&p); err != nil {
		return err
	}
	if p.Origin != string(game.OriginTournament) {
		return nil
	}
	m, err := sqlc.New(s.d.DB).SlotOfMatch(ctx, &p.MatchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.inTournament(ctx, m.TournamentID, func(r *round) error {
		m, err := r.q.SlotOfMatch(ctx, &p.MatchID)
		if err != nil || m.Status != "in_progress" {
			return err // decided already (redelivery)
		}
		return fn(r, &m, &p)
	})
}

// OnMatchCompleted moves the winner on; a draw goes to the higher seed.
func (s *Service) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	return s.slotOf(ctx, e, func(r *round, m *sqlc.TournamentBracketMatch, p *matchEnded) error {
		winner := r.higherSeed(m)
		switch {
		case p.IsDraw:
		case p.WinnerSide == 1:
			winner = *m.EntryAID
		case p.WinnerSide == 2:
			winner = *m.EntryBID
		}
		return r.advance(ctx, m, winner, "done")
	})
}

// OnMatchCancelled handles a tournament match that did not finish: before
// the start the players who readied count as present (an entry fully
// ready against one that is not wins by walkover); otherwise the slot
// opens again for a replay.
func (s *Service) OnMatchCancelled(ctx context.Context, e events.Envelope) error {
	return s.slotOf(ctx, e, func(r *round, m *sqlc.TournamentBracketMatch, p *matchEnded) error {
		r.touched = true
		until := r.now.Add(time.Duration(r.t.RoundReadyWindowS) * time.Second)
		present := []uuid.UUID{}
		if p.StartedAt == nil && p.Reason == string(game.CancelNotAllReady) {
			present = p.ReadyIDs
			a, b := r.fullyPresent(*m.EntryAID, present), r.fullyPresent(*m.EntryBID, present)
			if a != b {
				m.Present = present
				return r.advance(ctx, m, r.byPresence(m), "walkover")
			}
			if m.ReadyUntil != nil && m.ReadyUntil.Before(until) {
				until = *m.ReadyUntil
			}
		}
		if err := r.q.OpenSlot(ctx, sqlc.OpenSlotParams{TournamentID: r.t.ID, RoundNo: m.RoundNo, SlotNo: m.SlotNo, Now: &r.now, ReadyUntil: &until}); err != nil {
			return err
		}
		return r.q.SetPresent(ctx, sqlc.SetPresentParams{TournamentID: r.t.ID, RoundNo: m.RoundNo, SlotNo: m.SlotNo, Present: present})
	})
}

// Tick starts tournaments whose start time came and decides bracket
// matches whose ready window closed (worker, every minute).
func (s *Service) Tick(ctx context.Context) error {
	now := s.d.Clock.Now()
	q := sqlc.New(s.d.DB)
	due, err := q.DueToStart(ctx, now)
	if err != nil {
		return err
	}
	for _, id := range due {
		if err := s.inTournament(ctx, id, func(r *round) error { return r.start(ctx) }); err != nil {
			return err
		}
	}
	expired, err := q.ExpiredSlots(ctx, &now)
	if err != nil {
		return err
	}
	for _, x := range expired {
		err := s.inTournament(ctx, x.TournamentID, func(r *round) error {
			m, err := r.q.LockSlot(ctx, sqlc.LockSlotParams(x))
			if err != nil || m.Status != "ready" || m.ReadyUntil == nil || m.ReadyUntil.After(r.now) {
				return err
			}
			return r.advance(ctx, &m, r.byPresence(&m), "walkover")
		})
		if err != nil {
			return err
		}
	}
	return nil
}
