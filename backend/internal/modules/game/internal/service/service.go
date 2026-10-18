// Package service holds the game module's use cases around the pure engine
// (docs/game/match-lifecycle.md T1, schema-game §1): creating a match with
// every guard, and persisting what the engine produces — transitions,
// revealed rounds, the final result with its outbox event — plus the
// one-live-match-per-user locks. The live match owner (Phase 4) runs the
// engine and calls Apply after each command.
package service

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/matchtype"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/scoring"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
)

// Event names (backend-modules.md §3.2).
const (
	EventMatchStarted   = "game.MatchStarted"
	EventMatchCompleted = "game.MatchCompleted"
	EventMatchCancelled = "game.MatchCancelled"
)

// Timing of the guards.
const (
	// NoRepeatWindow: questions seen in the last 30 days are avoided (D-032).
	NoRepeatWindow = 30 * 24 * time.Hour
	// liveLockTTL is the safety TTL of game:user:<id>:live (redis-keys §2).
	liveLockTTL = 3 * time.Hour
	// RevengeWindow and RevengeChain limit revenge challenges (D-045/D-046).
	RevengeWindow = 24 * time.Hour
	RevengeChain  = 3
)

// PreseasonID is the season of ranked matches until the ranking module
// runs real seasons (Phase 7, D-064).
var PreseasonID = uuid.MustParse("01a14f8e-0000-7000-8000-000000000001")

// Questions is what game needs from content.
type Questions interface {
	Select(ctx context.Context, r content.SelectRequest) ([]content.SelectedQuestion, error)
	CategoryExists(ctx context.Context, id uuid.UUID) (bool, error)
}

// Users is what game needs from identity.
type Users interface {
	User(ctx context.Context, id uuid.UUID) (identity.User, error)
}

// SeasonProvider returns the active ranked season (ranking, Phase 7);
// ok = false → NO_ACTIVE_SEASON.
type SeasonProvider interface {
	ActiveSeason(ctx context.Context) (id uuid.UUID, ok bool, err error)
}

// SanctionChecker reports a blocking sanction for a user and match type
// (moderation, Phase 8); kind is "" when the user may play.
type SanctionChecker interface {
	Blocking(ctx context.Context, user uuid.UUID, t domain.MatchType, o domain.Origin) (kind string, err error)
}

// Preseason is the SeasonProvider until ranking exists.
type Preseason struct{}

// ActiveSeason implements SeasonProvider.
func (Preseason) ActiveSeason(context.Context) (uuid.UUID, bool, error) {
	return PreseasonID, true, nil
}

// NoSanctions is the SanctionChecker until moderation exists.
type NoSanctions struct{}

// Blocking implements SanctionChecker.
func (NoSanctions) Blocking(context.Context, uuid.UUID, domain.MatchType, domain.Origin) (string, error) {
	return "", nil
}

// Deps are the service's collaborators.
type Deps struct {
	DB        *pgxpool.Pool
	Redis     *redis.Client
	Questions Questions
	Users     Users
	Seasons   SeasonProvider
	Sanctions SanctionChecker
	Events    *events.Bus
	Clock     clock.Clock
	Logger    *slog.Logger
	// Random is the engine's random source (crypto-random when nil).
	Random engine.Random
}

// Service implements game's use cases.
type Service struct{ d Deps }

// New returns a service.
func New(d Deps) *Service {
	if d.Random == nil {
		d.Random = cryptoRandom{}
	}
	if d.Seasons == nil {
		d.Seasons = Preseason{}
	}
	if d.Sanctions == nil {
		d.Sanctions = NoSanctions{}
	}
	return &Service{d: d}
}

// cryptoRandom shuffles options with crypto/rand (round-lifecycle §5).
type cryptoRandom struct{}

func (cryptoRandom) Intn(n int) int {
	v, err := crand.Int(crand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	return int(v.Int64())
}

// Member is one participant of a new match.
type Member struct {
	UserID uuid.UUID
	Locale string
}

// CreateRequest creates a match (match-lifecycle T1).
type CreateRequest struct {
	Config    config.Config
	Sides     [2][]Member
	OriginRef *uuid.UUID
	// Parent is the previous match of a rematch or revenge.
	Parent *uuid.UUID
}

// Created is a new match in `waiting` with the engine output of T1.
type Created struct {
	Match  *engine.Match
	Output engine.Output
}

func modeNotAllowed(vs []config.Violation) error {
	return apierror.New(apierror.ModeNotAllowed).With("rule_id", vs[0].RuleID).With("violations", vs)
}

// Create checks every T1 guard and stores the match: configuration
// (mode matrix), sides, participants (active, no blocking sanction, not in
// another live match), enough questions, an active season for Ranked.
// Nothing is written when a guard fails, and no live lock is left behind.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Created, error) {
	now := s.d.Clock.Now().UTC()
	var parent *sqlc.GameMatch
	if req.Parent != nil {
		p, err := sqlc.New(s.d.DB).GetMatch(ctx, *req.Parent)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierror.New(apierror.MatchNotFound)
		}
		if err != nil {
			return nil, err
		}
		parent = &p
	}
	cfg := req.Config
	if parent != nil {
		pc, err := matchConfig(*parent)
		if err != nil {
			return nil, err
		}
		cfg = config.Transform(cfg, &pc)
	} else {
		cfg = config.Transform(cfg, nil)
	}
	if vs := config.Validate(cfg); len(vs) > 0 {
		return nil, modeNotAllowed(vs)
	}
	if err := s.contextRules(ctx, cfg, req, parent, now); err != nil {
		return nil, err
	}
	d := config.Derive(cfg)

	// Sides and participants.
	size := cfg.Format.SideSize()
	seen := map[uuid.UUID]bool{}
	var seats []engine.Seat
	var locales []string
	tiers := map[uuid.UUID]identity.Tier{}
	for side, members := range req.Sides {
		if len(members) != size {
			return nil, apierror.New(apierror.ValidationFailed).With("reason", fmt.Sprintf("side %d has %d players, format %s needs %d", side+1, len(members), cfg.Format, size))
		}
		for seat, m := range members {
			if seen[m.UserID] {
				return nil, apierror.New(apierror.ValidationFailed).With("reason", "duplicate participant")
			}
			seen[m.UserID] = true
			if m.Locale != "tr" && m.Locale != "en" {
				return nil, apierror.New(apierror.ValidationFailed).With("reason", "locale")
			}
			u, err := s.d.Users.User(ctx, m.UserID)
			if err != nil {
				return nil, err
			}
			if u.Status != identity.StatusActive || u.DeletionRequestedAt != nil {
				return nil, apierror.New(apierror.Forbidden).With("reason", "participant_not_active").With("user_id", m.UserID)
			}
			kind, err := s.d.Sanctions.Blocking(ctx, m.UserID, cfg.MatchType, cfg.Origin)
			if err != nil {
				return nil, err
			}
			if kind != "" {
				return nil, apierror.New(apierror.Sanctioned).With("kind", kind).With("user_id", m.UserID)
			}
			tiers[m.UserID] = u.Tier
			seats = append(seats, engine.Seat{UserID: m.UserID, Side: side + 1, Seat: seat + 1, Locale: m.Locale})
			if !slices.Contains(locales, m.Locale) {
				locales = append(locales, m.Locale)
			}
		}
	}
	slices.Sort(locales)

	var season *uuid.UUID
	if d.IsRated {
		id, ok, err := s.d.Seasons.ActiveSeason(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierror.New(apierror.NoActiveSeason)
		}
		season = &id
	}

	questions, err := s.selectQuestions(ctx, cfg, d, locales, seats, now)
	if err != nil {
		return nil, err
	}

	matchID := ids.New()
	if err := s.lockUsers(ctx, matchID, seats); err != nil {
		return nil, err
	}
	m, out, err := engine.New(engine.Setup{
		MatchID: matchID, Config: cfg, Seats: seats, Questions: questions,
		Rules: scoring.Current, Random: s.d.Random, NowMS: now.UnixMilli(),
	})
	if err == nil {
		err = s.insert(ctx, m, cfg, d, season, locales, tiers, req, out, now)
	}
	if err != nil {
		s.unlockUsers(context.WithoutCancel(ctx), matchID, seats)
		return nil, err
	}
	return &Created{Match: m, Output: out}, nil
}

// contextRules checks the context rules that need data (mode-matrix §3.2)
// available in this phase; the rest arrive with their modules.
func (s *Service) contextRules(ctx context.Context, cfg config.Config, req CreateRequest, parent *sqlc.GameMatch, now time.Time) error {
	fail := func(id, desc string) error {
		return modeNotAllowed([]config.Violation{{RuleID: id, Description: desc}})
	}
	if cfg.Format == domain.FormatCouple {
		return fail("R16", "couple sides need active couples (Couple Contest arrives in Phase 6)")
	}
	if cfg.CategoryMode == domain.CreatorQuiz {
		return fail("R14", "creator quizzes arrive in Phase 8")
	}
	if (cfg.Origin == domain.Rematch || cfg.Origin == domain.Revenge) && parent == nil {
		return fail("R15", "rematch and revenge need the parent match")
	}
	if cfg.Origin == domain.Revenge {
		if err := s.revengeRule(ctx, req, parent, now); err != nil {
			return err
		}
	}
	for _, id := range cfg.CategoryIDs {
		ok, err := s.d.Questions.CategoryExists(ctx, id)
		if err != nil {
			return err
		}
		if !ok {
			return fail("R12", "unknown or inactive category")
		}
	}
	return nil
}

// revengeRule is R15 (D-045/D-046): the challenger side lost the parent,
// the parent ended less than 24 h ago, and the chain has fewer than 3
// revenge matches.
func (s *Service) revengeRule(ctx context.Context, req CreateRequest, parent *sqlc.GameMatch, now time.Time) error {
	fail := func(desc string) error {
		return modeNotAllowed([]config.Violation{{RuleID: "R15", Description: desc}})
	}
	if parent.Status != string(domain.StatusCompleted) || parent.EndedAt == nil || now.Sub(*parent.EndedAt) > RevengeWindow {
		return fail("revenge is offered for 24 h after a completed match")
	}
	if parent.IsDraw || parent.WinnerSideNo == nil {
		return fail("revenge needs a losing side")
	}
	ps, err := sqlc.New(s.d.DB).ListParticipants(ctx, parent.ID)
	if err != nil {
		return err
	}
	loser := 3 - *parent.WinnerSideNo
	for i := range ps {
		if ps[i].UserID == req.Sides[0][0].UserID && ps[i].SideNo != loser {
			return fail("revenge is issued by the losing side (side 1 of the request)")
		}
	}
	chain, cur := 0, parent
	for cur != nil && cur.Origin == string(domain.Revenge) {
		chain++
		if cur.ParentMatchID == nil {
			break
		}
		next, err := sqlc.New(s.d.DB).GetMatch(ctx, *cur.ParentMatchID)
		if err != nil {
			return err
		}
		cur = &next
	}
	if chain >= RevengeChain {
		return fail("at most 3 revenge matches per chain")
	}
	return nil
}

// selectQuestions turns the configuration into a content selection
// (match-types §2, category-modes §2–§7).
func (s *Service) selectQuestions(ctx context.Context, cfg config.Config, d config.Derived, locales []string, seats []engine.Seat, now time.Time) ([]engine.Question, error) {
	t := matchtype.For(cfg.MatchType)
	if cfg.RankedLength {
		t = matchtype.RankedLengthQuickDuel()
	}
	n := d.QuestionCount
	if n == 0 {
		n = matchtype.SelectionBatch
	}
	curve := func(k int) []int {
		out := make([]int, k)
		for i := range out {
			out[i] = t.Difficulty(i + 1)
		}
		return out
	}
	users := make([]uuid.UUID, len(seats))
	for i, st := range seats {
		users[i] = st.UserID
	}
	recent, err := sqlc.New(s.d.DB).RecentQuestionIDs(ctx, sqlc.RecentQuestionIDsParams{UserIds: users, Since: now.Add(-NoRepeatWindow)})
	if err != nil {
		return nil, err
	}
	base := content.SelectRequest{
		Locales: locales, MultipleChoiceOnly: d.QuestionRules.MultipleChoiceOnly, Exclude: recent,
		Rules: content.SelectRules{Difficulties: curve(n), MaxPerCategory: d.QuestionRules.MaxPerCategory, MaxTrueFalsePercent: d.QuestionRules.MaxTrueFalsePercent},
	}
	var picked []content.SelectedQuestion
	switch cfg.CategoryMode {
	case domain.CategoryMaster:
		base.CategoryIDs, base.SubcategoryID = cfg.CategoryIDs, cfg.SubcategoryID
		base.Rules.MaxPerCategory = 0 // one category: caps do not apply (D-042)
		picked, err = s.d.Questions.Select(ctx, base)
	case domain.MixedBattle:
		// A pool per category; picks decide the order (D-043). Each needs
		// enough for half the match, so any pick sequence can be served.
		per := (n + 1) / 2
		if cfg.MatchType == domain.SuddenDeath {
			per = matchtype.SelectionBatch
		}
		for _, cat := range cfg.CategoryIDs {
			r := base
			r.CategoryIDs, r.Rules.MaxPerCategory, r.Rules.Difficulties = []uuid.UUID{cat}, 0, curve(per)
			qs, err := s.d.Questions.Select(ctx, r)
			if err != nil {
				return nil, err
			}
			picked = append(picked, qs...)
		}
	case domain.RevengeMatch:
		base.Rules.HomeCategory, base.Rules.MaxRun, base.Rules.MaxPerCategory = cfg.CategoryIDs[0], 3, 0
		picked, err = s.d.Questions.Select(ctx, base)
	default:
		picked, err = s.d.Questions.Select(ctx, base)
	}
	if err != nil {
		return nil, err
	}
	out := make([]engine.Question, len(picked))
	for i, q := range picked {
		out[i] = engine.Question{
			ID: q.ID, Revision: q.Revision, Type: domain.QuestionType(q.Type), Difficulty: q.Difficulty,
			CategoryID: q.CategoryID, SubcategoryID: q.SubcategoryID, Options: q.Options, Correct: q.Correct,
		}
	}
	return out, nil
}

func liveKey(user uuid.UUID) string { return redis.Key("game", "user", user.String(), "live") }

// lockUsers takes game:user:<id>:live for every participant (one live
// match per user); on any failure the acquired locks are released.
func (s *Service) lockUsers(ctx context.Context, match uuid.UUID, seats []engine.Seat) error {
	var taken []engine.Seat
	for _, st := range seats {
		ok, err := s.d.Redis.SetNX(ctx, liveKey(st.UserID), match.String(), liveLockTTL).Result()
		if err == nil && !ok {
			other, _ := s.d.Redis.Get(ctx, liveKey(st.UserID)).Result()
			s.unlockUsers(context.WithoutCancel(ctx), match, taken)
			return apierror.New(apierror.AlreadyInMatch).With("match_id", other).With("user_id", st.UserID)
		}
		if err != nil {
			s.unlockUsers(context.WithoutCancel(ctx), match, taken)
			return err
		}
		taken = append(taken, st)
	}
	return nil
}

// unlockUsers releases the locks that still point at match.
func (s *Service) unlockUsers(ctx context.Context, match uuid.UUID, seats []engine.Seat) {
	for _, st := range seats {
		key := liveKey(st.UserID)
		if v, err := s.d.Redis.Get(ctx, key).Result(); err == nil && v == match.String() {
			if err := s.d.Redis.Del(ctx, key).Err(); err != nil {
				s.d.Logger.WarnContext(ctx, "live lock release failed", "user_id", st.UserID, "error", err)
			}
		}
	}
}

// extra is game.matches.config_extra.
type extra struct {
	Aggregation   domain.Aggregation `json:"aggregation,omitempty"`
	RankedLength  bool               `json:"ranked_length,omitempty"`
	CategoryIDs   []uuid.UUID        `json:"category_ids,omitempty"`
	SubcategoryID *uuid.UUID         `json:"subcategory_id,omitempty"`
	QuizLength    int                `json:"quiz_length,omitempty"`
	QuizWindowMS  int                `json:"quiz_window_ms,omitempty"`
}

func optional[T ~string](v T) *string {
	if v == "" {
		return nil
	}
	s := string(v)
	return &s
}

func (s *Service) insert(ctx context.Context, m *engine.Match, cfg config.Config, d config.Derived, season *uuid.UUID,
	locales []string, tiers map[uuid.UUID]identity.Tier, req CreateRequest, out engine.Output, now time.Time,
) error {
	raw, err := json.Marshal(extra{
		Aggregation: cfg.Aggregation, RankedLength: cfg.RankedLength, CategoryIDs: cfg.CategoryIDs,
		SubcategoryID: cfg.SubcategoryID, QuizLength: cfg.QuizLength, QuizWindowMS: cfg.QuizWindowMS,
	})
	if err != nil {
		return err
	}
	var count *int16
	if d.QuestionCount > 0 {
		c := int16(d.QuestionCount) //nolint:gosec // G115: ≤ 30
		count = &c
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.InsertMatch(ctx, sqlc.InsertMatchParams{
			ID: m.ID, MatchType: string(cfg.MatchType), Format: string(cfg.Format),
			TeamAnswerMode: optional(cfg.TeamMode), CategoryMode: optional(cfg.CategoryMode),
			IsRated: d.IsRated, RatingBucket: optional(d.RatingBucket), SeasonID: season, QuestionCount: count,
			AnswerWindowMs: int32(d.AnswerWindowMS), Locales: locales, QuestionSetID: cfg.QuestionSetID, //nolint:gosec // G115: ≤ 60 000
			ConfigExtra: raw, RulesetVersion: int32(m.Rules.Version), Origin: string(cfg.Origin), //nolint:gosec // G115: small
			OriginRef: req.OriginRef, ParentMatchID: req.Parent, Now: now,
		}); err != nil {
			return err
		}
		for side := int16(1); side <= 2; side++ {
			if err := q.InsertSide(ctx, sqlc.InsertSideParams{MatchID: m.ID, SideNo: side}); err != nil {
				return err
			}
		}
		for _, p := range m.Participants {
			if err := q.InsertParticipant(ctx, sqlc.InsertParticipantParams{
				MatchID: m.ID, UserID: p.UserID, SideNo: int16(p.Side), SeatNo: int16(p.Seat.Seat), //nolint:gosec // G115: 1–4
				Locale: p.Locale, AccountTierAtStart: string(tiers[p.UserID]), JoinedAt: now,
			}); err != nil {
				return err
			}
		}
		return insertTransitions(ctx, q, m.ID, out.Transitions)
	})
}

func insertTransitions(ctx context.Context, q *sqlc.Queries, match uuid.UUID, ts []engine.Transition) error {
	for _, t := range ts {
		var actorID *uuid.UUID
		if t.ActorID != uuid.Nil {
			id := t.ActorID
			actorID = &id
		}
		var from *string
		if t.From != "" {
			f := string(t.From)
			from = &f
		}
		reason := t.Reason
		if err := q.InsertTransition(ctx, sqlc.InsertTransitionParams{
			MatchID: match, Seq: int32(t.Seq), FromStatus: from, ToStatus: string(t.To), //nolint:gosec // G115: small
			Reason: &reason, Actor: t.Actor, ActorID: actorID, At: time.UnixMilli(t.AtMS).UTC(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// matchConfig rebuilds a stored match's configuration (rematch/revenge).
func matchConfig(m sqlc.GameMatch) (config.Config, error) {
	var ex extra
	if len(m.ConfigExtra) > 0 {
		if err := json.Unmarshal(m.ConfigExtra, &ex); err != nil {
			return config.Config{}, err
		}
	}
	c := config.Config{
		MatchType: domain.MatchType(m.MatchType), Format: domain.Format(m.Format), Origin: domain.Origin(m.Origin),
		Aggregation: ex.Aggregation, RankedLength: ex.RankedLength, CategoryIDs: ex.CategoryIDs,
		SubcategoryID: ex.SubcategoryID, QuestionSetID: m.QuestionSetID, QuizLength: ex.QuizLength, QuizWindowMS: ex.QuizWindowMS,
	}
	if m.TeamAnswerMode != nil {
		c.TeamMode = domain.TeamMode(*m.TeamAnswerMode)
	}
	if m.CategoryMode != nil {
		c.CategoryMode = domain.CategoryMode(*m.CategoryMode)
	}
	return c, nil
}
