// Package service holds moderation's use cases: player reports grouped
// into cases, the moderator queue and actions, sanctions with their
// effects and appeals (schema-moderation.md, D-030).
package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/moderation/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Users is what moderation needs from identity.
type Users interface {
	User(ctx context.Context, id uuid.UUID) (identity.User, error)
	SetStatus(ctx context.Context, user uuid.UUID, status identity.Status) error
}

// Questions is what moderation needs from content.
type Questions interface {
	PullToReview(ctx context.Context, actor *uuid.UUID, id uuid.UUID, reason string) error
	Retire(ctx context.Context, actor, id uuid.UUID) error
}

// Deps are the service's collaborators.
type Deps struct {
	DB        *pgxpool.Pool
	Users     Users
	Questions Questions
	Events    *events.Bus
	Clock     clock.Clock
	// ServerVoids counts a user's server-cancelled matches (game).
	ServerVoids func(ctx context.Context, user uuid.UUID, since time.Time) (int, error)
}

// Service implements moderation.
type Service struct {
	d Deps
}

// New builds the service.
func New(d Deps) *Service { return &Service{d: d} }

// Report rules (D-030).
const (
	ReportsPerDay        = 20
	wrongAnswerThreshold = 3
)

// Events (schema-moderation §6).
const (
	EventReportFiled     = "moderation.ReportFiled"
	EventSanctionApplied = "moderation.SanctionApplied"
	EventSanctionLifted  = "moderation.SanctionLifted"
)

// Target kinds and the reasons each accepts.
var (
	questionReasons = []string{"wrong_answer", "ambiguous", "outdated", "typo_or_translation", "offensive_content", "other"}
	userReasons     = []string{"cheating", "offensive_name", "harassment", "spam", "impersonation", "other"}
	reasonsOf       = map[string][]string{
		"question": questionReasons, "question_translation": questionReasons, "creator_quiz": questionReasons,
		"user": userReasons, "team": userReasons, "couple": userReasons, "chat_message": userReasons,
	}
)

// ReportInput is the body of POST /v1/reports.
type ReportInput struct {
	TargetKind   string     `json:"target_kind"`
	TargetID     uuid.UUID  `json:"target_id"`
	TargetLocale *string    `json:"target_locale,omitempty"`
	MatchID      *uuid.UUID `json:"match_id,omitempty"`
	Reason       string     `json:"reason"`
	Comment      *string    `json:"comment,omitempty"`
}

// Filed is the answer to a report.
type Filed struct {
	ID     uuid.UUID `json:"id"`
	CaseID uuid.UUID `json:"case_id"`
}

// tierWeight is a reporter's weight in case priority: restricted
// accounts count less (D-030).
var tierWeight = map[identity.Tier]int32{identity.TierFull: 3, identity.TierUnverified: 2, identity.TierGuest: 1}

func invalid(field, rule string) error {
	return apierror.Validation(apierror.FieldError{Field: field, Rule: rule})
}

// Report files a player's report: it joins the target's open case (or
// opens one), once per reporter and target, at most 20 a day. The third
// wrong_answer report pulls a question back to review (D-030).
func (s *Service) Report(ctx context.Context, reporter uuid.UUID, in ReportInput) (Filed, error) {
	reasons, ok := reasonsOf[in.TargetKind]
	switch {
	case !ok:
		return Filed{}, invalid("target_kind", "one_of")
	case in.TargetID == uuid.Nil:
		return Filed{}, invalid("target_id", "required")
	case !slices.Contains(reasons, in.Reason):
		return Filed{}, invalid("reason", "one_of")
	case in.TargetKind == "question_translation" && (in.TargetLocale == nil || (*in.TargetLocale != "tr" && *in.TargetLocale != "en")):
		return Filed{}, invalid("target_locale", "required")
	case in.Comment != nil && len([]rune(*in.Comment)) > 500:
		return Filed{}, invalid("comment", "max_length")
	case in.TargetKind == "user" && in.TargetID == reporter:
		return Filed{}, invalid("target_id", "not_self")
	}
	if in.TargetKind != "question_translation" {
		in.TargetLocale = nil
	}
	u, err := s.d.Users.User(ctx, reporter)
	if err != nil {
		return Filed{}, err
	}
	now := s.d.Clock.Now()
	out := Filed{ID: ids.New()}
	pull := false
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		n, err := q.ReportsSince(ctx, sqlc.ReportsSinceParams{ReporterID: reporter, Since: now.Add(-24 * time.Hour)})
		if err != nil {
			return err
		}
		if n >= ReportsPerDay {
			return apierror.New(apierror.ReportLimitReached).WithRetryAfter(24 * time.Hour)
		}
		c, err := q.OpenCaseForUpdate(ctx, sqlc.OpenCaseForUpdateParams{TargetKind: in.TargetKind, TargetID: in.TargetID, TargetLocale: in.TargetLocale})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c = sqlc.ModerationCase{ID: ids.New()}
			if err := q.InsertCase(ctx, sqlc.InsertCaseParams{ID: c.ID, TargetKind: in.TargetKind, TargetID: in.TargetID, TargetLocale: in.TargetLocale, Now: now}); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		out.CaseID = c.ID
		added, err := q.InsertReport(ctx, sqlc.InsertReportParams{
			ID: out.ID, CaseID: c.ID, ReporterID: reporter, ReporterTier: string(u.Tier), TargetKind: in.TargetKind, TargetID: in.TargetID,
			TargetLocale: in.TargetLocale, MatchID: in.MatchID, Reason: in.Reason, Comment: in.Comment, Now: now,
		})
		if err != nil {
			return err
		}
		if added == 0 {
			return apierror.New(apierror.AlreadyReported)
		}
		if err := q.BumpCase(ctx, sqlc.BumpCaseParams{ID: c.ID, Weight: tierWeight[u.Tier]}); err != nil {
			return err
		}
		if in.TargetKind == "question" && in.Reason == "wrong_answer" {
			wrong, err := q.CaseReasonCount(ctx, sqlc.CaseReasonCountParams{CaseID: c.ID, Reason: "wrong_answer"})
			if err != nil {
				return err
			}
			if wrong == wrongAnswerThreshold {
				pull = true
				if err := insertAction(ctx, q, &c.ID, "system", nil, "question_send_to_review", in.TargetKind, in.TargetID,
					map[string]any{"reason": "wrong_answer_reports", "reports": wrong}, now); err != nil {
					return err
				}
			}
		}
		return s.d.Events.PublishTx(ctx, tx, EventReportFiled, now, map[string]any{"report_id": out.ID, "case_id": c.ID, "target_kind": in.TargetKind})
	})
	if err != nil {
		return Filed{}, err
	}
	if pull {
		if err := s.d.Questions.PullToReview(ctx, nil, in.TargetID, "wrong_answer_reports"); err != nil {
			return Filed{}, err
		}
	}
	return out, nil
}

func insertAction(ctx context.Context, q *sqlc.Queries, caseID *uuid.UUID, actorKind string, actor *uuid.UUID, action, kind string, target uuid.UUID, payload map[string]any, now time.Time) error {
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return q.InsertAction(ctx, sqlc.InsertActionParams{
		ID: ids.New(), CaseID: caseID, ActorKind: actorKind, ActorID: actor, Action: action, TargetKind: kind, TargetID: target,
		Payload: body, Now: now,
	})
}

// Blocking reports the kind of an active sanction that keeps a user out
// of a match of this type and origin (game's SanctionChecker): suspension
// always, matchmaking_ban for matchmade matches, ranked_ban for Ranked.
func (s *Service) Blocking(ctx context.Context, user uuid.UUID, t game.MatchType, origin game.Origin) (string, error) {
	active, err := sqlc.New(s.d.DB).ActiveSanctions(ctx, sqlc.ActiveSanctionsParams{UserID: user, Now: s.d.Clock.Now()})
	if err != nil {
		return "", err
	}
	for i := range active {
		a := &active[i]
		switch a.Kind {
		case "suspension":
			return a.Kind, nil
		case "matchmaking_ban":
			if origin == game.OriginMatchmaking {
				return a.Kind, nil
			}
		case "ranked_ban":
			if t == "ranked" {
				return a.Kind, nil
			}
		}
	}
	return "", nil
}
