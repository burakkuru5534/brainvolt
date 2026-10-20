// Package service holds tournament's use cases (schema-platform §3, D-031,
// D-090): admins create single-elimination tournaments; full accounts
// register as a user, couple or team and check in; at the start the
// checked-in entries are seeded into a bracket; each round's players show
// up within the ready window and the match is created through game;
// no-shows lose by walkover; the final places earn the tournament's XP.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/team"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Events (backend-modules §3.2).
const (
	EventRoundScheduled = "tournament.RoundScheduled"
	EventFinished       = "tournament.TournamentFinished"
)

// CheckInOpens is how long before the start check-in opens.
const CheckInOpens = time.Hour

// Statuses.
const (
	StatusDraft        = "draft"
	StatusRegistration = "registration"
	StatusRunning      = "running"
	StatusFinished     = "finished"
	StatusCancelled    = "cancelled"
)

// Games creates tournament matches (game).
type Games interface {
	Start(ctx context.Context, r game.CreateRequest) (*game.Created, error)
}

// Cards reads players' cards and languages (profile).
type Cards interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
	LocalesOf(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// Teams reads teams and couples (team).
type Teams interface {
	TeamInfo(ctx context.Context, id uuid.UUID) (team.Info, error)
	ActiveCoupleOf(ctx context.Context, user uuid.UUID) (team.CoupleInfo, bool, error)
	TeamLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]team.Label, error)
	CoupleLabels(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]team.Label, error)
}

// Pusher sends a realtime message to a user's open sessions.
type Pusher interface {
	Push(ctx context.Context, user uuid.UUID, typ string, data any) error
}

// Deps are the service's collaborators; Games is nil in the worker.
type Deps struct {
	DB     *pgxpool.Pool
	Games  Games
	Cards  Cards
	Teams  Teams
	Pusher Pusher
	Events *events.Bus
	Clock  clock.Clock
	Logger *slog.Logger
}

// Service implements tournament.
type Service struct{ d Deps }

// New builds the service.
func New(d Deps) *Service {
	if d.Logger == nil {
		d.Logger = slog.New(slog.DiscardHandler)
	}
	return &Service{d: d}
}

// TournamentRewards are XP per final place, and for everyone else who played.
type TournamentRewards struct {
	Places          []TournamentPlaceReward `json:"places"`
	ParticipationXP int                     `json:"participation_xp"`
}

// TournamentPlaceReward is the XP of a final place.
type TournamentPlaceReward struct {
	Place int `json:"place"`
	XP    int `json:"xp"`
}

// XPFor is the XP of a final place.
func (r TournamentRewards) XPFor(place int) int {
	for _, p := range r.Places {
		if p.Place == place {
			return p.XP
		}
	}
	return r.ParticipationXP
}

// TournamentText is a tournament's name and description in one language.
type TournamentText struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// TournamentInput is the body of POST /v1/admin/tournaments.
type TournamentInput struct {
	// Texts by locale (tr required, en optional).
	Texts                map[string]TournamentText `json:"texts"`
	EntityKind           string                    `json:"entity_kind"`
	MatchConfig          game.Config               `json:"match_config"`
	MinEntrants          int                       `json:"min_entrants"`
	MaxEntrants          int                       `json:"max_entrants"`
	RegistrationOpensAt  time.Time                 `json:"registration_opens_at"`
	RegistrationClosesAt time.Time                 `json:"registration_closes_at"`
	StartsAt             time.Time                 `json:"starts_at"`
	RoundReadyWindowS    int                       `json:"round_ready_window_s,omitempty"`
	Rewards              TournamentRewards         `json:"rewards"`
	// Publish opens it at once (status registration); otherwise a draft.
	Publish bool `json:"publish"`
}

// TournamentUpdate is the body of PATCH /v1/admin/tournaments/{id}.
type TournamentUpdate struct {
	// Status: registration (publish a draft) or cancelled.
	Status string `json:"status"`
}

// Tournament is a tournament as players see it.
type Tournament struct {
	ID                   uuid.UUID         `json:"id"`
	Name                 string            `json:"name"`
	Description          *string           `json:"description,omitempty"`
	Status               string            `json:"status"`
	BracketType          string            `json:"bracket_type"`
	EntityKind           string            `json:"entity_kind"`
	MatchConfig          game.Config       `json:"match_config"`
	MinEntrants          int               `json:"min_entrants"`
	MaxEntrants          int               `json:"max_entrants"`
	Entrants             int               `json:"entrants"`
	RegistrationOpensAt  time.Time         `json:"registration_opens_at"`
	RegistrationClosesAt time.Time         `json:"registration_closes_at"`
	CheckInOpensAt       time.Time         `json:"check_in_opens_at"`
	StartsAt             time.Time         `json:"starts_at"`
	RoundReadyWindowS    int               `json:"round_ready_window_s"`
	Rewards              TournamentRewards `json:"rewards"`
	FinishedAt           *time.Time        `json:"finished_at,omitempty"`
	MyEntry              *TournamentEntry  `json:"my_entry,omitempty"`
}

// TournamentEntry is a registered user, couple or team.
type TournamentEntry struct {
	ID         uuid.UUID   `json:"id"`
	EntityKind string      `json:"entity_kind"`
	EntityID   uuid.UUID   `json:"entity_id"`
	Name       string      `json:"name"`
	Image      string      `json:"image,omitempty"` // avatar or emblem id
	Lineup     []uuid.UUID `json:"lineup"`
	Seed       *int        `json:"seed,omitempty"`
	Status     string      `json:"status"`
	FinalPlace *int        `json:"final_place,omitempty"`
}

// TournamentBracketSlot is a slot of the bracket.
type TournamentBracketSlot struct {
	RoundNo       int         `json:"round_no"`
	SlotNo        int         `json:"slot_no"`
	EntryAID      *uuid.UUID  `json:"entry_a_id"`
	EntryBID      *uuid.UUID  `json:"entry_b_id"`
	WinnerEntryID *uuid.UUID  `json:"winner_entry_id"`
	Status        string      `json:"status"`
	MatchID       *uuid.UUID  `json:"match_id"`
	ReadyUntil    *time.Time  `json:"ready_until"`
	Present       []uuid.UUID `json:"present"`
}

// TournamentDetail is GET /v1/tournaments/{id}: the tournament, its entries and
// bracket, and the caller's current bracket match.
type TournamentDetail struct {
	Tournament
	Rounds  int                     `json:"rounds"`
	Entries []TournamentEntry       `json:"entries"`
	Bracket []TournamentBracketSlot `json:"bracket"`
	MyMatch *TournamentBracketSlot  `json:"my_match,omitempty"`
}

func notFound() error { return apierror.New(apierror.NotFound) }

// sideSize is the lineup size of an entry kind's format.
func sideSize(kind string, f game.Format) (int, bool) {
	switch kind {
	case "user":
		return 1, f == "1v1"
	case "couple":
		return 2, f == "couple"
	case "team":
		return f.SideSize(), slices.Contains([]game.Format{"2v2", "3v3", "4v4"}, f)
	}
	return 0, false
}

// Create stores a tournament (draft, or open for registration).
func (s *Service) Create(ctx context.Context, admin uuid.UUID, in TournamentInput) (Tournament, error) {
	bad := func(field, rule string) error {
		return apierror.Validation(apierror.FieldError{Field: field, Rule: rule})
	}
	for loc, t := range in.Texts {
		if loc != "tr" && loc != "en" {
			return Tournament{}, bad("texts", "locale")
		}
		if n := len([]rune(strings.TrimSpace(t.Name))); n < 3 || n > 60 {
			return Tournament{}, bad("texts."+loc+".name", "length")
		}
		if t.Description != nil && len([]rune(*t.Description)) > 500 {
			return Tournament{}, bad("texts."+loc+".description", "length")
		}
	}
	if _, ok := in.Texts["tr"]; !ok {
		return Tournament{}, bad("texts.tr", "required")
	}
	in.MatchConfig.Origin = game.OriginTournament
	if _, ok := sideSize(in.EntityKind, in.MatchConfig.Format); !ok {
		return Tournament{}, bad("entity_kind", "format")
	}
	if vs := game.ValidateConfig(in.MatchConfig); len(vs) > 0 {
		return Tournament{}, apierror.New(apierror.ModeNotAllowed).With("rule_id", vs[0].RuleID).With("violations", vs)
	}
	switch {
	case in.MinEntrants < 2 || in.MaxEntrants < in.MinEntrants || in.MaxEntrants > domain.MaxEntrants:
		return Tournament{}, bad("max_entrants", "range")
	case !in.RegistrationOpensAt.Before(in.RegistrationClosesAt) || in.RegistrationClosesAt.After(in.StartsAt):
		return Tournament{}, bad("starts_at", "order")
	case !in.StartsAt.After(s.d.Clock.Now()):
		return Tournament{}, bad("starts_at", "future")
	}
	if in.RoundReadyWindowS == 0 {
		in.RoundReadyWindowS = 300
	}
	if in.RoundReadyWindowS < 60 || in.RoundReadyWindowS > 3600 {
		return Tournament{}, bad("round_ready_window_s", "range")
	}
	for _, p := range in.Rewards.Places {
		if p.Place < 1 || p.XP < 0 || p.XP > 5000 {
			return Tournament{}, bad("rewards", "range")
		}
	}
	if in.Rewards.ParticipationXP < 0 || in.Rewards.ParticipationXP > 1000 {
		return Tournament{}, bad("rewards", "range")
	}
	cfg, err := json.Marshal(in.MatchConfig)
	if err != nil {
		return Tournament{}, err
	}
	rewards, err := json.Marshal(in.Rewards)
	if err != nil {
		return Tournament{}, err
	}
	status := StatusDraft
	if in.Publish {
		status = StatusRegistration
	}
	id := ids.New()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.InsertTournament(ctx, sqlc.InsertTournamentParams{
			ID: id, Status: status, EntityKind: in.EntityKind, MatchConfig: cfg, MinEntrants: int32(in.MinEntrants), //nolint:gosec // range checked
			MaxEntrants: int32(in.MaxEntrants), RegistrationOpensAt: in.RegistrationOpensAt, RegistrationClosesAt: in.RegistrationClosesAt, //nolint:gosec // range checked
			StartsAt: in.StartsAt, RoundReadyWindowS: int32(in.RoundReadyWindowS), Rewards: rewards, CreatedBy: admin, Now: s.d.Clock.Now(), //nolint:gosec // range checked
		}); err != nil {
			return err
		}
		for loc, t := range in.Texts {
			if err := q.InsertTranslation(ctx, sqlc.InsertTranslationParams{TournamentID: id, Locale: loc, Name: strings.TrimSpace(t.Name), Description: t.Description}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Tournament{}, err
	}
	d, err := s.Get(ctx, admin, id, "tr", true)
	return d.Tournament, err
}

// Update publishes a draft or cancels a tournament that has not finished.
func (s *Service) Update(ctx context.Context, admin, id uuid.UUID, in TournamentUpdate) (Tournament, error) {
	var users []uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		t, err := q.LockTournament(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return notFound()
		}
		if err != nil {
			return err
		}
		switch {
		case in.Status == StatusRegistration && t.Status == StatusDraft:
		case in.Status == StatusCancelled && t.Status != StatusFinished && t.Status != StatusCancelled:
			if users, err = lineups(ctx, q, id); err != nil {
				return err
			}
		case in.Status != StatusRegistration && in.Status != StatusCancelled:
			return apierror.Validation(apierror.FieldError{Field: "status", Rule: "one_of"})
		default:
			return apierror.New(apierror.MatchStateInvalid).With("reason", "tournament_"+t.Status)
		}
		return q.SetTournamentStatus(ctx, sqlc.SetTournamentStatusParams{ID: id, Status: in.Status, Now: s.d.Clock.Now()})
	})
	if err != nil {
		return Tournament{}, err
	}
	s.pushUpdated(ctx, id, users)
	d, err := s.Get(ctx, admin, id, "tr", true)
	return d.Tournament, err
}

// lineups is every player of a tournament's live entries.
func lineups(ctx context.Context, q *sqlc.Queries, id uuid.UUID) ([]uuid.UUID, error) {
	es, err := q.Entries(ctx, id)
	if err != nil {
		return nil, err
	}
	var out []uuid.UUID
	for i := range es {
		if es[i].Status != "withdrawn" && es[i].Status != "disqualified" {
			out = append(out, es[i].Lineup...)
		}
	}
	return out, nil
}

// pushUpdated tells players to reload the tournament.
func (s *Service) pushUpdated(ctx context.Context, id uuid.UUID, users []uuid.UUID) {
	if s.d.Pusher == nil {
		return
	}
	for _, u := range users {
		if err := s.d.Pusher.Push(ctx, u, "tournament.updated", map[string]any{"tournament_id": id}); err != nil {
			s.d.Logger.DebugContext(ctx, "tournament: update not delivered", "error", err)
		}
	}
}

// texts picks locale's text, falling back to Turkish.
func texts(rows []sqlc.TournamentTournamentTranslation, id uuid.UUID, locale string) TournamentText {
	var tr *TournamentText
	for _, r := range rows {
		if r.TournamentID != id {
			continue
		}
		t := TournamentText{Name: r.Name, Description: r.Description}
		if r.Locale == locale {
			return t
		}
		if r.Locale == "tr" {
			tr = &t
		}
	}
	if tr == nil {
		return TournamentText{}
	}
	return *tr
}

func toTournament(t *sqlc.TournamentTournament, entrants int, text TournamentText) (Tournament, error) {
	out := Tournament{
		ID: t.ID, Name: text.Name, Description: text.Description, Status: t.Status, BracketType: t.BracketType, EntityKind: t.EntityKind,
		MinEntrants: int(t.MinEntrants), MaxEntrants: int(t.MaxEntrants), Entrants: entrants, RegistrationOpensAt: t.RegistrationOpensAt,
		RegistrationClosesAt: t.RegistrationClosesAt, CheckInOpensAt: t.StartsAt.Add(-CheckInOpens), StartsAt: t.StartsAt,
		RoundReadyWindowS: int(t.RoundReadyWindowS), FinishedAt: t.FinishedAt,
	}
	if err := json.Unmarshal(t.MatchConfig, &out.MatchConfig); err != nil {
		return out, err
	}
	if err := json.Unmarshal(t.Rewards, &out.Rewards); err != nil {
		return out, err
	}
	if out.Rewards.Places == nil {
		out.Rewards.Places = []TournamentPlaceReward{}
	}
	return out, nil
}

// List returns tournaments, newest start first; drafts only for staff.
func (s *Service) List(ctx context.Context, user uuid.UUID, status *string, locale string, staff bool) ([]Tournament, error) {
	q := sqlc.New(s.d.DB)
	rows, err := q.ListTournaments(ctx, sqlc.ListTournamentsParams{Status: status, WithDrafts: staff})
	if err != nil {
		return nil, err
	}
	idList := make([]uuid.UUID, len(rows))
	for i := range rows {
		idList[i] = rows[i].ID
	}
	trs, err := q.Translations(ctx, idList)
	if err != nil {
		return nil, err
	}
	mine, err := q.EntriesOfUser(ctx, sqlc.EntriesOfUserParams{Ids: idList, UserID: user})
	if err != nil {
		return nil, err
	}
	out := make([]Tournament, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		text := texts(trs, r.ID, locale)
		t, err := toTournament(&sqlc.TournamentTournament{
			ID: r.ID, Status: r.Status, BracketType: r.BracketType, EntityKind: r.EntityKind, MatchConfig: r.MatchConfig, MinEntrants: r.MinEntrants,
			MaxEntrants: r.MaxEntrants, RegistrationOpensAt: r.RegistrationOpensAt, RegistrationClosesAt: r.RegistrationClosesAt, StartsAt: r.StartsAt,
			RoundReadyWindowS: r.RoundReadyWindowS, Rewards: r.Rewards, FinishedAt: r.FinishedAt,
		}, int(r.Entrants), text)
		if err != nil {
			return nil, err
		}
		for j := range mine {
			if mine[j].TournamentID == r.ID {
				e := toEntry(&mine[j])
				t.MyEntry = &e
			}
		}
		out = append(out, t)
	}
	return out, nil
}

func toEntry(e *sqlc.TournamentEntry) TournamentEntry {
	out := TournamentEntry{ID: e.ID, EntityKind: e.EntityKind, EntityID: e.EntityID, Lineup: e.Lineup, Status: e.Status}
	if e.Seed != nil {
		v := int(*e.Seed)
		out.Seed = &v
	}
	if e.FinalPlace != nil {
		v := int(*e.FinalPlace)
		out.FinalPlace = &v
	}
	return out
}

func toSlot(m *sqlc.TournamentBracketMatch) TournamentBracketSlot {
	return TournamentBracketSlot{
		RoundNo: int(m.RoundNo), SlotNo: int(m.SlotNo), EntryAID: m.EntryAID, EntryBID: m.EntryBID, WinnerEntryID: m.WinnerEntryID,
		Status: m.Status, MatchID: m.MatchID, ReadyUntil: m.ReadyUntil, Present: m.Present,
	}
}

// Get returns a tournament with its entries and bracket.
func (s *Service) Get(ctx context.Context, user, id uuid.UUID, locale string, staff bool) (TournamentDetail, error) {
	q := sqlc.New(s.d.DB)
	t, err := q.GetTournament(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && t.Status == StatusDraft && !staff) {
		return TournamentDetail{}, notFound()
	}
	if err != nil {
		return TournamentDetail{}, err
	}
	trs, err := q.Translations(ctx, []uuid.UUID{id})
	if err != nil {
		return TournamentDetail{}, err
	}
	text := texts(trs, id, locale)
	rows, err := q.Entries(ctx, id)
	if err != nil {
		return TournamentDetail{}, err
	}
	entries, err := s.named(ctx, t.EntityKind, rows)
	if err != nil {
		return TournamentDetail{}, err
	}
	live := 0
	for i := range entries {
		if entries[i].Status != "withdrawn" && entries[i].Status != "disqualified" {
			live++
		}
	}
	tour, err := toTournament(&t, live, text)
	if err != nil {
		return TournamentDetail{}, err
	}
	d := TournamentDetail{Tournament: tour, Entries: entries, Bracket: []TournamentBracketSlot{}}
	for i := range entries {
		if entries[i].Status != "withdrawn" && entries[i].Status != "disqualified" && slices.Contains(entries[i].Lineup, user) {
			d.MyEntry = &entries[i]
		}
	}
	slots, err := q.Slots(ctx, id)
	if err != nil {
		return TournamentDetail{}, err
	}
	for i := range slots {
		b := toSlot(&slots[i])
		d.Bracket = append(d.Bracket, b)
		d.Rounds = max(d.Rounds, b.RoundNo)
		if d.MyEntry != nil && (b.Status == "ready" || b.Status == "in_progress") &&
			((b.EntryAID != nil && *b.EntryAID == d.MyEntry.ID) || (b.EntryBID != nil && *b.EntryBID == d.MyEntry.ID)) {
			d.MyMatch = &b
		}
	}
	return d, nil
}

// named adds names and images to entries.
func (s *Service) named(ctx context.Context, kind string, rows []sqlc.TournamentEntry) ([]TournamentEntry, error) {
	out := make([]TournamentEntry, len(rows))
	entities := make([]uuid.UUID, len(rows))
	for i := range rows {
		out[i] = toEntry(&rows[i])
		entities[i] = rows[i].EntityID
	}
	switch kind {
	case "user":
		cards, err := s.d.Cards.Cards(ctx, entities)
		if err != nil {
			return nil, err
		}
		for i := range out {
			out[i].Name, out[i].Image = cards[out[i].EntityID].DisplayName, cards[out[i].EntityID].AvatarID
		}
	default:
		read := s.d.Teams.TeamLabels
		if kind == "couple" {
			read = s.d.Teams.CoupleLabels
		}
		labels, err := read(ctx, entities)
		if err != nil {
			return nil, err
		}
		for i := range out {
			out[i].Name, out[i].Image = labels[out[i].EntityID].Name, labels[out[i].EntityID].EmblemID
		}
	}
	return out, nil
}
