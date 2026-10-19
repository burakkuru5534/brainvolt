package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/content"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/domain"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/profile"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
)

// Cards returns players' public cards (profile).
type Cards interface {
	Cards(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.Card, error)
}

// MatchView is the result screen of GET /v1/matches/{id} (rest.md §4).
// Rounds are listed once the match is over.
type MatchView struct {
	ID           uuid.UUID         `json:"id"`
	MatchType    domain.MatchType  `json:"match_type"`
	Format       domain.Format     `json:"format"`
	CategoryMode *string           `json:"category_mode"`
	Origin       domain.Origin     `json:"origin"`
	Status       domain.Status     `json:"status"`
	IsRated      bool              `json:"is_rated"`
	CreatedAt    time.Time         `json:"created_at"`
	StartedAt    *time.Time        `json:"started_at"`
	EndedAt      *time.Time        `json:"ended_at"`
	EndReason    *string           `json:"end_reason"`
	CancelReason *string           `json:"cancel_reason"`
	WinnerSideNo *int16            `json:"winner_side_no"`
	IsDraw       bool              `json:"is_draw"`
	Sides        []SideView        `json:"sides"`
	Participants []ParticipantView `json:"participants"`
	Rounds       []RoundView       `json:"rounds"`
	// Revenge is the caller's revenge state once the match is over.
	Revenge *RevengeView `json:"revenge,omitempty"`
}

// RevengeView says whether the caller may ask for revenge (losing side,
// 24 h, chain < 3, D-045/D-046) and the series score of the revenge chain
// from the caller's view (first match included).
type RevengeView struct {
	Available bool `json:"available"`
	Wins      int  `json:"wins"`
	Losses    int  `json:"losses"`
	Chain     int  `json:"chain"`
}

// SideView is one side's total.
type SideView struct {
	SideNo  int16   `json:"side_no"`
	Score   int32   `json:"score"`
	Correct int16   `json:"correct_count"`
	Result  *string `json:"result"`
}

// ParticipantView is one participant with their public card.
type ParticipantView struct {
	UserID       uuid.UUID `json:"user_id"`
	SideNo       int16     `json:"side_no"`
	SeatNo       int16     `json:"seat_no"`
	Status       string    `json:"status"`
	Contribution int32     `json:"score_contribution"`
	Correct      int16     `json:"correct_count"`
	Answered     int16     `json:"answered_count"`
	DisplayName  string    `json:"display_name"`
	AvatarID     string    `json:"avatar_id"`
	CountryCode  *string   `json:"country_code"`
}

// RoundView is one played round in the caller's language.
type RoundView struct {
	RoundNo         int16             `json:"round_no"`
	QuestionID      uuid.UUID         `json:"question_id"`
	Prompt          string            `json:"prompt"`
	Explanation     string            `json:"explanation,omitempty"`
	Category        string            `json:"category"`
	Options         []OptionView      `json:"options"`
	CorrectOptionID uuid.UUID         `json:"correct_option_id"`
	Answers         []AnswerView      `json:"answers"`
	Sides           []RoundResultView `json:"sides"`
}

// OptionView is an option as it was shown.
type OptionView struct {
	ID   uuid.UUID `json:"id"`
	Text string    `json:"text"`
}

// AnswerView is one participant's answer (option nil = timeout).
type AnswerView struct {
	UserID     uuid.UUID  `json:"user_id"`
	OptionID   *uuid.UUID `json:"option_id"`
	ResponseMS *int32     `json:"response_ms"`
	Correct    bool       `json:"correct"`
	Points     int32      `json:"points"`
}

// RoundResultView is one side's round result.
type RoundResultView struct {
	SideNo  int16 `json:"side_no"`
	Correct bool  `json:"correct"`
	Points  int32 `json:"points"`
}

// Match returns the result screen data for a participant.
func (s *Service) Match(ctx context.Context, caller, id uuid.UUID, locale string) (MatchView, error) {
	q := sqlc.New(s.d.DB)
	m, err := q.GetMatch(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return MatchView{}, apierror.New(apierror.MatchNotFound)
	}
	if err != nil {
		return MatchView{}, err
	}
	ps, err := q.ListParticipants(ctx, id)
	if err != nil {
		return MatchView{}, err
	}
	users := make([]uuid.UUID, len(ps))
	in := false
	for i := range ps {
		users[i] = ps[i].UserID
		in = in || ps[i].UserID == caller
	}
	if !in {
		return MatchView{}, apierror.New(apierror.NotAParticipant)
	}
	cards, err := s.d.Cards.Cards(ctx, users)
	if err != nil {
		return MatchView{}, err
	}
	v := MatchView{
		ID: m.ID, MatchType: domain.MatchType(m.MatchType), Format: domain.Format(m.Format), CategoryMode: m.CategoryMode,
		Origin: domain.Origin(m.Origin), Status: domain.Status(m.Status), IsRated: m.IsRated, CreatedAt: m.CreatedAt,
		StartedAt: m.StartedAt, EndedAt: m.EndedAt, EndReason: m.EndReason, CancelReason: m.CancelReason,
		WinnerSideNo: m.WinnerSideNo, IsDraw: m.IsDraw, Sides: []SideView{}, Participants: []ParticipantView{}, Rounds: []RoundView{},
	}
	for i := range ps {
		p, c := ps[i], cards[ps[i].UserID]
		v.Participants = append(v.Participants, ParticipantView{
			UserID: p.UserID, SideNo: p.SideNo, SeatNo: p.SeatNo, Status: p.Status, Contribution: p.ScoreContribution,
			Correct: p.CorrectCount, Answered: p.AnsweredCount, DisplayName: c.DisplayName, AvatarID: c.AvatarID, CountryCode: c.CountryCode,
		})
	}
	sides, err := q.ListSides(ctx, id)
	if err != nil {
		return MatchView{}, err
	}
	for _, sd := range sides {
		v.Sides = append(v.Sides, SideView{SideNo: sd.SideNo, Score: sd.Score, Correct: sd.CorrectCount, Result: sd.Result})
	}
	if !domain.Status(m.Status).Terminal() {
		return v, nil
	}
	if v.Rounds, err = s.rounds(ctx, q, id, locale); err != nil {
		return MatchView{}, err
	}
	if m.Status == string(domain.StatusCompleted) {
		if v.Revenge, err = s.revengeView(ctx, q, &m, caller); err != nil {
			return MatchView{}, err
		}
	}
	return v, nil
}

// revengeView walks the revenge chain back to its first match.
func (s *Service) revengeView(ctx context.Context, q *sqlc.Queries, m *sqlc.GameMatch, caller uuid.UUID) (*RevengeView, error) {
	out := &RevengeView{}
	lost := false
	for cur, first := m, true; cur != nil; first = false {
		ps, err := q.ListParticipants(ctx, cur.ID)
		if err != nil {
			return nil, err
		}
		side := int16(0)
		for i := range ps {
			if ps[i].UserID == caller {
				side = ps[i].SideNo
			}
		}
		if cur.WinnerSideNo != nil && side != 0 {
			if *cur.WinnerSideNo == side {
				out.Wins++
			} else {
				out.Losses++
				lost = lost || first
			}
		}
		if cur.Origin != string(domain.Revenge) || cur.ParentMatchID == nil {
			break
		}
		out.Chain++
		parent, err := q.GetMatch(ctx, *cur.ParentMatchID)
		if err != nil {
			return nil, err
		}
		cur = &parent
	}
	out.Available = lost && m.EndedAt != nil && s.d.Clock.Now().Sub(*m.EndedAt) < RevengeWindow && out.Chain < RevengeChain
	return out, nil
}

func (s *Service) rounds(ctx context.Context, q *sqlc.Queries, id uuid.UUID, locale string) ([]RoundView, error) {
	rounds, err := q.ListRounds(ctx, id)
	if err != nil {
		return nil, err
	}
	answers, err := q.ListAnswers(ctx, id)
	if err != nil {
		return nil, err
	}
	results, err := q.ListRoundResults(ctx, id)
	if err != nil {
		return nil, err
	}
	qids := make([]uuid.UUID, len(rounds))
	for i, r := range rounds {
		qids[i] = r.QuestionID
	}
	texts, err := s.d.Questions.Texts(ctx, qids, []string{locale, "tr", "en"})
	if err != nil {
		return nil, err
	}
	out := make([]RoundView, 0, len(rounds))
	for _, r := range rounds {
		t := pickText(texts[r.QuestionID], locale)
		rv := RoundView{
			RoundNo: r.RoundNo, QuestionID: r.QuestionID, Prompt: t.Prompt, Explanation: t.Explanation, Category: t.Category,
			CorrectOptionID: t.Correct, Options: []OptionView{}, Answers: []AnswerView{}, Sides: []RoundResultView{},
		}
		for _, pos := range r.OptionOrder {
			if int(pos) >= 1 && int(pos) <= len(t.Order) {
				id := t.Order[pos-1]
				rv.Options = append(rv.Options, OptionView{ID: id, Text: t.Options[id]})
			}
		}
		for _, a := range answers {
			if a.RoundNo == r.RoundNo {
				rv.Answers = append(rv.Answers, AnswerView{UserID: a.UserID, OptionID: a.OptionID, ResponseMS: a.ResponseMs, Correct: a.IsCorrect, Points: a.PointsAwarded})
			}
		}
		for _, rr := range results {
			if rr.RoundNo == r.RoundNo {
				rv.Sides = append(rv.Sides, RoundResultView{SideNo: rr.SideNo, Correct: rr.IsCorrect, Points: rr.Points})
			}
		}
		out = append(out, rv)
	}
	return out, nil
}

// pickText prefers the caller's language and falls back to any.
func pickText(byLocale map[string]content.QuestionText, locale string) content.QuestionText {
	if t, ok := byLocale[locale]; ok {
		return t
	}
	for _, l := range []string{"tr", "en"} {
		if t, ok := byLocale[l]; ok {
			return t
		}
	}
	return content.QuestionText{}
}

// Current is the caller's live match for GET /v1/matches/current.
type Current struct {
	Match *CurrentMatch `json:"match"`
}

// CurrentMatch names the live match and its stream to resume.
type CurrentMatch struct {
	ID     uuid.UUID     `json:"id"`
	Status domain.Status `json:"status"`
	Stream string        `json:"stream"`
}

// Current returns the caller's live match, if any.
func (s *Service) Current(ctx context.Context, user uuid.UUID) (Current, error) {
	v, err := s.d.Redis.Get(ctx, liveKey(user)).Result()
	if err != nil {
		return Current{}, nil //nolint:nilerr // no live lock (or Redis down): no live match to resume
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return Current{}, nil //nolint:nilerr // corrupted lock value: nothing to resume
	}
	m, err := sqlc.New(s.d.DB).GetMatch(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && domain.Status(m.Status).Terminal()) {
		return Current{}, nil
	}
	if err != nil {
		return Current{}, err
	}
	return Current{Match: &CurrentMatch{ID: id, Status: domain.Status(m.Status), Stream: "match:" + id.String()}}, nil
}

// Summary is what other modules read about a match: its state and seating
// (lobby: rematch, rooms after a match).
type Summary struct {
	ID        uuid.UUID
	Status    domain.Status
	Origin    domain.Origin
	OriginRef *uuid.UUID
	EndedAt   *time.Time
	// WinnerSide is 0 for a draw or an unfinished match.
	WinnerSide int
	// Couples are the sides' couple profiles (Couple Contest).
	Couples      [2]*uuid.UUID
	Participants []Seated
}

// Seated is one participant of a Summary.
type Seated struct {
	UserID uuid.UUID
	Side   int
	Seat   int
	Locale string
}

// Summary returns a match's state and participants.
func (s *Service) Summary(ctx context.Context, id uuid.UUID) (Summary, error) {
	q := sqlc.New(s.d.DB)
	m, err := q.GetMatch(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, apierror.New(apierror.MatchNotFound)
	}
	if err != nil {
		return Summary{}, err
	}
	ps, err := q.ListParticipants(ctx, id)
	if err != nil {
		return Summary{}, err
	}
	out := Summary{ID: id, Status: domain.Status(m.Status), Origin: domain.Origin(m.Origin), OriginRef: m.OriginRef, EndedAt: m.EndedAt}
	if m.WinnerSideNo != nil {
		out.WinnerSide = int(*m.WinnerSideNo)
	}
	sides, err := q.SideEntities(ctx, id)
	if err != nil {
		return Summary{}, err
	}
	for _, sd := range sides {
		out.Couples[sd.SideNo-1] = sd.CoupleID
	}
	for i := range ps {
		p := &ps[i]
		out.Participants = append(out.Participants, Seated{UserID: p.UserID, Side: int(p.SideNo), Seat: int(p.SeatNo), Locale: p.Locale})
	}
	return out, nil
}

// CategoryExists reports an active top-level category (content).
func (s *Service) CategoryExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.d.Questions.CategoryExists(ctx, id)
}
