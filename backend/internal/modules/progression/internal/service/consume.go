package service

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// matchCompleted is the part of game.MatchCompleted progression reads
// (the event's JSON contract; progression does not import game).
type matchCompleted struct {
	MatchID       uuid.UUID     `json:"match_id"`
	MatchType     string        `json:"match_type"`
	Format        string        `json:"format"`
	RoundsPlayed  int           `json:"rounds_played"`
	EndedAt       time.Time     `json:"ended_at"`
	Sides         []side        `json:"sides"`
	Participants  []participant `json:"participants"`
	Rounds        []round       `json:"rounds"`
	CategoryMode  string        `json:"category_mode"`
	QuizAuthorID  *uuid.UUID    `json:"quiz_author_id"`
	QuizPublished bool          `json:"quiz_published"`
}

// earns reports whether a participant gets anything from the match: a
// Creator Quiz awards the players of a published quiz only, never its
// author (D-023, D-044).
func (p *matchCompleted) earns(user uuid.UUID) bool {
	if p.CategoryMode != "creator_quiz" {
		return true
	}
	return p.QuizPublished && (p.QuizAuthorID == nil || *p.QuizAuthorID != user)
}

type side struct {
	SideNo int    `json:"side_no"`
	Result string `json:"result"`
}

type participant struct {
	UserID  uuid.UUID `json:"user_id"`
	SideNo  int       `json:"side_no"`
	Status  string    `json:"status"`
	Correct int       `json:"correct"`
}

type round struct {
	RoundNo        int         `json:"round_no"`
	CategoryID     uuid.UUID   `json:"category_id"`
	CorrectUserIDs []uuid.UUID `json:"correct_user_ids"`
}

// MsgAwards is the personal message with a match's progression awards
// (websocket.md: match.awards; ranking sends its part separately).
const MsgAwards = "match.awards"

// Awards is progression's part of match.awards.
type Awards struct {
	MatchID uuid.UUID       `json:"match_id"`
	XP      int             `json:"xp"`
	XPTotal int64           `json:"xp_total"`
	Level   int             `json:"level"`
	LevelUp bool            `json:"level_up"`
	Streak  int             `json:"streak"`
	Badges  []EarnedBadge   `json:"badges"`
	Quests  []QuestProgress `json:"quests"`
}

// OnMatchCompleted consumes game.MatchCompleted once per match: match
// XP, play streaks, badges and quest progress for every participant,
// then match.awards to each of them.
func (s *Service) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	var p matchCompleted
	if err := e.Decode(&p); err != nil {
		return err
	}
	results := map[int]string{}
	for _, sd := range p.Sides {
		results[sd.SideNo] = sd.Result
	}
	var awards []Awards
	var users []uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		awards, users = nil, nil
		q := sqlc.New(tx)
		n, err := q.MarkProcessed(ctx, sqlc.MarkProcessedParams{MatchID: p.MatchID, Now: s.d.Clock.Now()})
		if err != nil || n == 0 {
			return err
		}
		badges, err := activeBadges(ctx, q)
		if err != nil {
			return err
		}
		for i := range p.Participants {
			me := &p.Participants[i]
			if !p.earns(me.UserID) {
				continue
			}
			if err := q.LockUser(ctx, me.UserID.String()); err != nil {
				return err
			}
			a, err := s.applyMatch(ctx, q, &p, me, results[me.SideNo], badges)
			if err != nil {
				return err
			}
			awards = append(awards, a)
			users = append(users, me.UserID)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if s.d.Pusher != nil {
		for i, a := range awards {
			if err := s.d.Pusher.Push(ctx, users[i], MsgAwards, a); err != nil && s.d.Logger != nil {
				s.d.Logger.WarnContext(ctx, "progression: awards not delivered", "user", users[i], "error", err)
			}
		}
	}
	return nil
}

// applyMatch is one participant's share of a completed match.
func (s *Service) applyMatch(ctx context.Context, q *sqlc.Queries, p *matchCompleted, me *participant, result string, badges []badge) (Awards, error) {
	left := me.Status == "left" || me.Status == "forfeited"
	won := result == "win" && !left
	a := Awards{MatchID: p.MatchID, Badges: []EarnedBadge{}, Quests: []QuestProgress{}}
	ref := p.MatchID.String()
	added, before, after, err := grant(ctx, q, me.UserID, domain.MatchXP(result, me.Correct, left), sourceMatch, ref, p.EndedAt)
	if err != nil {
		return a, err
	}
	a.XP, a.Level, a.LevelUp = added, after, after > before

	wins := 0
	if won {
		wins = 1
	}
	total, err := q.AddStats(ctx, sqlc.AddStatsParams{UserID: me.UserID, Wins: int32(wins), Correct: int32(me.Correct), Now: p.EndedAt}) //nolint:gosec // G115: small
	if err != nil {
		return a, err
	}
	facts := domain.MatchFacts{Wins: int(total), Won: won, MatchType: p.MatchType, Format: p.Format, Correct: me.Correct, Rounds: p.RoundsPlayed}
	byCat := map[uuid.UUID]int{}
	run := 0
	rounds := slices.Clone(p.Rounds)
	slices.SortFunc(rounds, func(x, y round) int { return x.RoundNo - y.RoundNo })
	for _, r := range rounds {
		if slices.Contains(r.CorrectUserIDs, me.UserID) {
			byCat[r.CategoryID]++
			run++
			facts.AnswerStreak = max(facts.AnswerStreak, run)
		} else {
			run = 0
		}
	}
	for cat, n := range byCat {
		c, err := q.AddCategoryCorrect(ctx, sqlc.AddCategoryCorrectParams{UserID: me.UserID, CategoryID: cat, Correct: int32(n)}) //nolint:gosec // G115: ≤ rounds
		if err != nil {
			return a, err
		}
		facts.CategoryCorrect = max(facts.CategoryCorrect, int(c))
	}

	st, err := loadStreak(ctx, q, me.UserID)
	if err != nil {
		return a, err
	}
	if !left {
		st = st.Play(domain.Day(p.EndedAt))
		day := st.LastDay
		if err := q.UpsertStreak(ctx, sqlc.UpsertStreakParams{
			UserID: me.UserID, CurrentDays: int32(st.Current), LongestDays: int32(st.Longest), LastPlayDay: &day, Now: p.EndedAt, //nolint:gosec // G115: days
		}); err != nil {
			return a, err
		}
	}
	facts.PlayStreak, a.Streak = st.Current, st.Current

	for i := range badges {
		b := &badges[i]
		if !b.criteria.EarnedBy(&facts) {
			continue
		}
		earned, xp, err := award(ctx, q, me.UserID, b, ref, p.EndedAt)
		if err != nil {
			return a, err
		}
		if earned {
			a.XP += xp
			a.Badges = append(a.Badges, EarnedBadge{Code: b.code, Tier: b.tier})
		}
	}
	if len(a.Badges) > 0 {
		names, err := q.BadgesBySource(ctx, sqlc.BadgesBySourceParams{Locale: "en", UserID: me.UserID, SourceRef: &ref})
		if err != nil {
			return a, err
		}
		for i := range a.Badges {
			for _, n := range names {
				if n.Code == a.Badges[i].Code {
					a.Badges[i].IconKey, a.Badges[i].Name = n.IconKey, n.Name
				}
			}
		}
	}
	if !left {
		qs, err := advanceQuests(ctx, q, me.UserID, p, me, won, s.extraQuests(ctx, me.UserID))
		if err != nil {
			return a, err
		}
		a.Quests = append(a.Quests, qs...)
	}
	prog, err := q.GetProgress(ctx, me.UserID)
	if err != nil {
		return a, err
	}
	a.XPTotal, a.Level = prog.XpTotal, int(prog.Level)
	a.LevelUp = a.Level > before
	return a, nil
}

// MatchAwards returns progression's awards of a match for a user, for a
// client that missed match.awards (null fields when not processed yet).
func (s *Service) MatchAwards(ctx context.Context, user, match uuid.UUID, locale string) (Awards, error) {
	q := sqlc.New(s.d.DB)
	ref := match.String()
	xp, err := q.MatchXP(ctx, sqlc.MatchXPParams{UserID: user, MatchRef: ref})
	if err != nil {
		return Awards{}, err
	}
	out := Awards{MatchID: match, XP: int(xp), Badges: []EarnedBadge{}, Quests: []QuestProgress{}}
	p, err := s.Progress(ctx, user)
	if err != nil {
		return Awards{}, err
	}
	out.XPTotal, out.Level, out.Streak = p.XPTotal, p.Level, p.Streak.Current
	rows, err := q.BadgesBySource(ctx, sqlc.BadgesBySourceParams{Locale: locale, UserID: user, SourceRef: &ref})
	if err != nil {
		return Awards{}, err
	}
	for _, r := range rows {
		out.Badges = append(out.Badges, EarnedBadge{Code: r.Code, Tier: int(r.Tier), IconKey: r.IconKey, Name: r.Name})
	}
	return out, nil
}
