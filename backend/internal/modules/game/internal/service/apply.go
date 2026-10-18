package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain/engine"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// SidePayload is one side in game.MatchCompleted.
type SidePayload struct {
	SideNo  int               `json:"side_no"`
	Score   int               `json:"score"`
	Correct int               `json:"correct"`
	Result  domain.SideResult `json:"result"`
}

// ParticipantPayload is one participant in game.MatchCompleted.
type ParticipantPayload struct {
	UserID       uuid.UUID                `json:"user_id"`
	SideNo       int                      `json:"side_no"`
	TierAtStart  string                   `json:"tier_at_start"`
	Status       domain.ParticipantStatus `json:"status"`
	Correct      int                      `json:"correct"`
	Answered     int                      `json:"answered"`
	Contribution int                      `json:"contribution"`
	Disconnects  int                      `json:"disconnects"`
	AwayMS       int                      `json:"away_ms"`
}

// MatchCompleted is the payload of game.MatchCompleted (schema-game §4):
// final and complete, so consumers never read game tables.
type MatchCompleted struct {
	MatchID       uuid.UUID            `json:"match_id"`
	MatchType     domain.MatchType     `json:"match_type"`
	Format        domain.Format        `json:"format"`
	TeamMode      domain.TeamMode      `json:"team_answer_mode,omitempty"`
	CategoryMode  domain.CategoryMode  `json:"category_mode,omitempty"`
	Origin        domain.Origin        `json:"origin"`
	IsRated       bool                 `json:"is_rated"`
	RatingBucket  string               `json:"rating_bucket,omitempty"`
	SeasonID      *uuid.UUID           `json:"season_id,omitempty"`
	EndReason     domain.EndReason     `json:"end_reason"`
	WinnerSideNo  int                  `json:"winner_side_no,omitempty"`
	IsDraw        bool                 `json:"is_draw"`
	RoundsPlayed  int                  `json:"rounds_played"`
	StartedAt     *time.Time           `json:"started_at,omitempty"`
	EndedAt       time.Time            `json:"ended_at"`
	Sides         []SidePayload        `json:"sides"`
	Participants  []ParticipantPayload `json:"participants"`
	QuestionStats []content.AnswerStat `json:"question_stats"`
}

// MatchCancelled is the payload of game.MatchCancelled.
type MatchCancelled struct {
	MatchID        uuid.UUID           `json:"match_id"`
	Reason         domain.CancelReason `json:"reason"`
	Origin         domain.Origin       `json:"origin"`
	ParticipantIDs []uuid.UUID         `json:"participant_ids"`
	// ReadyIDs were ready when a matchmade match was cancelled before the
	// start: matchmaking re-queues them at the front (D-034).
	ReadyIDs []uuid.UUID `json:"ready_ids"`
}

// MatchStarted is the payload of game.MatchStarted.
type MatchStarted struct {
	MatchID        uuid.UUID        `json:"match_id"`
	MatchType      domain.MatchType `json:"match_type"`
	Format         domain.Format    `json:"format"`
	ParticipantIDs []uuid.UUID      `json:"participant_ids"`
}

func at(ms int64) time.Time { return time.UnixMilli(ms).UTC() }

// Apply persists one engine output in the order of match-lifecycle §5:
// a revealed round and status changes in one transaction; a terminal
// result together with its outbox event in one transaction, then the live
// locks are released. Inserts are idempotent, so a retried Apply after a
// failure is safe (round-lifecycle §5).
func (s *Service) Apply(ctx context.Context, m *engine.Match, out engine.Output) error {
	if len(out.Transitions) == 0 && out.Round == nil && out.Result == nil {
		return nil
	}
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if out.Round != nil {
			if err := insertRound(ctx, q, m.ID, out.Round); err != nil {
				return err
			}
		}
		if err := insertTransitions(ctx, q, m.ID, out.Transitions); err != nil {
			return err
		}
		for _, t := range out.Transitions {
			switch t.To {
			case domain.StatusReady, domain.StatusInProgress:
				if err := q.SetMatchStatus(ctx, sqlc.SetMatchStatusParams{ID: m.ID, Status: string(t.To), At: at(t.AtMS)}); err != nil {
					return err
				}
				if t.To == domain.StatusInProgress {
					if err := s.d.Events.PublishTx(ctx, tx, EventMatchStarted, at(t.AtMS), MatchStarted{
						MatchID: m.ID, MatchType: m.Config.MatchType, Format: m.Config.Format, ParticipantIDs: userIDs(m),
					}); err != nil {
						return err
					}
				}
			case domain.StatusWaiting, domain.StatusCompleted, domain.StatusCancelled:
				// Creation is written by Create; terminal states below.
			}
		}
		if out.Result != nil {
			return s.finishTx(ctx, tx, q, m, out.Result)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if out.Result != nil {
		seats := make([]engine.Seat, len(m.Participants))
		for i, p := range m.Participants {
			seats[i] = p.Seat
		}
		s.unlockUsers(context.WithoutCancel(ctx), m.ID, seats)
	}
	return nil
}

func userIDs(m *engine.Match) []uuid.UUID {
	out := make([]uuid.UUID, len(m.Participants))
	for i, p := range m.Participants {
		out[i] = p.UserID
	}
	return out
}

func insertRound(ctx context.Context, q *sqlc.Queries, match uuid.UUID, r *engine.RoundRecord) error {
	order := make([]int16, len(r.OptionOrder))
	for i, o := range r.OptionOrder {
		order[i] = int16(o) //nolint:gosec // G115: 1–4
	}
	no := int16(r.No) //nolint:gosec // G115: ≤ 100
	if err := q.InsertRound(ctx, sqlc.InsertRoundParams{
		MatchID: match, RoundNo: no, QuestionID: r.Question.ID, QuestionRevision: r.Question.Revision,
		DifficultyUsed: int16(r.Question.Difficulty), OptionOrder: order, //nolint:gosec // G115: 1–5
		DispatchedAt: at(r.DispatchMS), DeadlineAt: at(r.DeadlineMS), RevealedAt: at(r.RevealedMS),
	}); err != nil {
		return fmt.Errorf("round %d: %w", r.No, err)
	}
	for _, a := range r.Answers {
		var received *time.Time
		if a.ReceivedMS != nil {
			t := at(*a.ReceivedMS)
			received = &t
		}
		var resp *int32
		if a.ResponseMS != nil {
			v := int32(*a.ResponseMS) //nolint:gosec // G115: ≤ window + 250 ms
			resp = &v
		}
		if err := q.InsertAnswer(ctx, sqlc.InsertAnswerParams{
			MatchID: match, RoundNo: no, UserID: a.UserID, SideNo: int16(a.Side), OptionID: a.OptionID, //nolint:gosec // G115: 1–2
			ReceivedAt: received, ResponseMs: resp, IsCorrect: a.Correct, PointsAwarded: int32(a.Points.Total), //nolint:gosec // G115: ≤ 260
			IsSideAnswer: a.IsSideAnswer, CreatedAt: at(r.RevealedMS),
		}); err != nil {
			return fmt.Errorf("round %d answer: %w", r.No, err)
		}
	}
	for i, sd := range r.Sides {
		detail, err := json.Marshal(sd.Detail)
		if err != nil {
			return err
		}
		if err := q.InsertRoundResult(ctx, sqlc.InsertRoundResultParams{
			MatchID: match, RoundNo: no, SideNo: int16(i + 1), IsCorrect: sd.Correct, Points: int32(sd.Points), Detail: detail, //nolint:gosec // G115: small
		}); err != nil {
			return fmt.Errorf("round %d result: %w", r.No, err)
		}
	}
	return nil
}

// finishTx writes the terminal state and its event (T3/T5/T6/T7).
func (s *Service) finishTx(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, m *engine.Match, r *engine.Result) error {
	ended := at(r.EndedMS)
	if r.Status == domain.StatusCancelled {
		n, err := q.CancelMatch(ctx, sqlc.CancelMatchParams{ID: m.ID, At: &ended, CancelReason: optional(r.CancelReason)})
		if err != nil || n == 0 {
			return err // already terminal: written once (LC08)
		}
		if err := s.participantsTx(ctx, q, m.ID, r, ended); err != nil {
			return err
		}
		p := MatchCancelled{MatchID: m.ID, Reason: r.CancelReason, Origin: m.Config.Origin, ParticipantIDs: userIDs(m), ReadyIDs: []uuid.UUID{}}
		for _, pt := range m.Participants {
			if pt.Status == domain.PReady {
				p.ReadyIDs = append(p.ReadyIDs, pt.UserID)
			}
		}
		return s.d.Events.PublishTx(ctx, tx, EventMatchCancelled, ended, p)
	}

	var winner *int16
	if r.WinnerSide != 0 {
		w := int16(r.WinnerSide) //nolint:gosec // G115: 1–2
		winner = &w
	}
	n, err := q.CompleteMatch(ctx, sqlc.CompleteMatchParams{ID: m.ID, At: &ended, EndReason: optional(r.EndReason), WinnerSideNo: winner, IsDraw: r.Draw})
	if err != nil || n == 0 {
		return err
	}
	// Score invariant: a side's score is the sum of its round points.
	sums, err := q.SideScoreFromRounds(ctx, m.ID)
	if err != nil {
		return err
	}
	for _, sm := range sums {
		if int(sm.Points) != r.Sides[sm.SideNo-1].Score {
			return fmt.Errorf("game: score invariant broken for match %s side %d: rounds %d, side %d", m.ID, sm.SideNo, sm.Points, r.Sides[sm.SideNo-1].Score)
		}
	}
	payload := MatchCompleted{
		MatchID: m.ID, MatchType: m.Config.MatchType, Format: m.Config.Format, TeamMode: m.Config.TeamMode,
		CategoryMode: m.Config.CategoryMode, Origin: m.Config.Origin, IsRated: m.Derived.IsRated,
		RatingBucket: string(m.Derived.RatingBucket), EndReason: r.EndReason, WinnerSideNo: r.WinnerSide, IsDraw: r.Draw,
		RoundsPlayed: r.RoundsPlayed, EndedAt: ended,
	}
	row, err := q.GetMatch(ctx, m.ID)
	if err != nil {
		return err
	}
	payload.SeasonID, payload.StartedAt = row.SeasonID, row.StartedAt
	for i, sd := range r.Sides {
		var forfeited *time.Time
		if sd.ForfeitedMS != 0 {
			t := at(sd.ForfeitedMS)
			forfeited = &t
		}
		result := string(sd.Result)
		if err := q.SetSideResult(ctx, sqlc.SetSideResultParams{
			MatchID: m.ID, SideNo: int16(i + 1), Score: int32(sd.Score), CorrectCount: int16(sd.Correct), //nolint:gosec // G115: small
			Result: &result, ForfeitedAt: forfeited,
		}); err != nil {
			return err
		}
		payload.Sides = append(payload.Sides, SidePayload{SideNo: i + 1, Score: sd.Score, Correct: sd.Correct, Result: sd.Result})
	}
	if err := s.participantsTx(ctx, q, m.ID, r, ended); err != nil {
		return err
	}
	ps, err := q.ListParticipants(ctx, m.ID)
	if err != nil {
		return err
	}
	tier := map[uuid.UUID]string{}
	for i := range ps {
		tier[ps[i].UserID] = ps[i].AccountTierAtStart
	}
	for _, p := range r.Participants {
		payload.Participants = append(payload.Participants, ParticipantPayload{
			UserID: p.UserID, SideNo: p.Side, TierAtStart: tier[p.UserID], Status: p.Status, Correct: p.Correct,
			Answered: p.Answered, Contribution: p.Contribution, Disconnects: p.Disconnects, AwayMS: p.AwayMS,
		})
	}
	stats, err := q.RoundStats(ctx, m.ID)
	if err != nil {
		return err
	}
	payload.QuestionStats = make([]content.AnswerStat, 0, len(stats))
	for _, st := range stats {
		payload.QuestionStats = append(payload.QuestionStats, content.AnswerStat{
			QuestionID: st.QuestionID, Locale: st.Locale, Asked: st.Asked, Correct: st.Correct,
			TimedOut: st.TimedOut, SumResponseMS: st.SumResponseMs,
		})
	}
	return s.d.Events.PublishTx(ctx, tx, EventMatchCompleted, ended, payload)
}

func (s *Service) participantsTx(ctx context.Context, q *sqlc.Queries, match uuid.UUID, r *engine.Result, ended time.Time) error {
	for _, p := range r.Participants {
		if err := q.SetParticipantResult(ctx, sqlc.SetParticipantResultParams{
			MatchID: match, UserID: p.UserID, Status: string(p.Status), ScoreContribution: int32(p.Contribution), //nolint:gosec // G115: small
			CorrectCount: int16(p.Correct), AnsweredCount: int16(p.Answered), DisconnectCount: int16(p.Disconnects), //nolint:gosec // G115: small
			DisconnectedMsTotal: int32(p.AwayMS), At: &ended, //nolint:gosec // G115: < 1 h
		}); err != nil {
			return err
		}
	}
	return nil
}
