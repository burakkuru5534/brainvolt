package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/moderation/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Case is a case in the review queue.
type Case struct {
	ID           uuid.UUID  `json:"id"`
	TargetKind   string     `json:"target_kind"`
	TargetID     uuid.UUID  `json:"target_id"`
	TargetLocale *string    `json:"target_locale,omitempty"`
	Status       string     `json:"status"`
	Priority     int32      `json:"priority"`
	ReportCount  int32      `json:"report_count"`
	SignalCount  int32      `json:"signal_count"`
	AssignedTo   *uuid.UUID `json:"assigned_to,omitempty"`
	Resolution   *string    `json:"resolution,omitempty"`
	Note         *string    `json:"resolution_note,omitempty"`
	OpenedAt     time.Time  `json:"opened_at"`
	ResolvedAt   *time.Time `json:"resolved_at,omitempty"`
}

func caseOf(c *sqlc.ModerationCase) Case {
	return Case{ID: c.ID, TargetKind: c.TargetKind, TargetID: c.TargetID, TargetLocale: c.TargetLocale, Status: c.Status,
		Priority: c.Priority, ReportCount: c.ReportCount, SignalCount: c.SignalCount, AssignedTo: c.AssignedTo,
		Resolution: c.Resolution, Note: c.ResolutionNote, OpenedAt: c.OpenedAt, ResolvedAt: c.ResolvedAt}
}

// ReportView is a report in a case.
type ReportView struct {
	ID         uuid.UUID  `json:"id"`
	ReporterID uuid.UUID  `json:"reporter_id"`
	Tier       string     `json:"reporter_tier"`
	Reason     string     `json:"reason"`
	Comment    *string    `json:"comment,omitempty"`
	MatchID    *uuid.UUID `json:"match_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// SignalView is a cheat signal in a case.
type SignalView struct {
	ID         uuid.UUID      `json:"id"`
	UserID     uuid.UUID      `json:"user_id"`
	MatchID    *uuid.UUID     `json:"match_id,omitempty"`
	Kind       string         `json:"kind"`
	Severity   int16          `json:"severity"`
	Details    map[string]any `json:"details"`
	DetectedAt time.Time      `json:"detected_at"`
}

// ActionView is an entry of the audit log.
type ActionView struct {
	ID        uuid.UUID      `json:"id"`
	ActorKind string         `json:"actor_kind"`
	ActorID   *uuid.UUID     `json:"actor_id,omitempty"`
	Action    string         `json:"action"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

// CaseDetail is GET /v1/admin/cases/{id}.
type CaseDetail struct {
	Case    Case         `json:"case"`
	Reports []ReportView `json:"reports"`
	Signals []SignalView `json:"signals"`
	History []ActionView `json:"history"`
}

// Cases is a page of the queue (open and in review, highest priority
// first) or of resolved cases.
func (s *Service) Cases(ctx context.Context, resolved bool, limit, offset int) ([]Case, error) {
	rows, err := sqlc.New(s.d.DB).Cases(ctx, sqlc.CasesParams{Resolved: resolved, Lim: int32(limit), Off: int32(offset)}) //nolint:gosec // G115: ≤ 100 / ≤ 10 000
	if err != nil {
		return nil, err
	}
	out := make([]Case, len(rows))
	for i := range rows {
		out[i] = caseOf(&rows[i])
	}
	return out, nil
}

// CaseDetail returns a case with its reports, signals and the target's
// action history.
func (s *Service) CaseDetail(ctx context.Context, id uuid.UUID) (CaseDetail, error) {
	q := sqlc.New(s.d.DB)
	c, err := q.GetCase(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return CaseDetail{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return CaseDetail{}, err
	}
	out := CaseDetail{Case: caseOf(&c), Reports: []ReportView{}, Signals: []SignalView{}, History: []ActionView{}}
	reports, err := q.CaseReports(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	for i := range reports {
		r := &reports[i]
		out.Reports = append(out.Reports, ReportView{ID: r.ID, ReporterID: r.ReporterID, Tier: r.ReporterTier, Reason: r.Reason,
			Comment: r.Comment, MatchID: r.MatchID, CreatedAt: r.CreatedAt})
	}
	signals, err := q.CaseSignals(ctx, &id)
	if err != nil {
		return CaseDetail{}, err
	}
	for _, sg := range signals {
		out.Signals = append(out.Signals, SignalView{ID: sg.ID, UserID: sg.UserID, MatchID: sg.MatchID, Kind: sg.Kind,
			Severity: sg.Severity, Details: object(sg.Details), DetectedAt: sg.DetectedAt})
	}
	actions, err := q.TargetActions(ctx, sqlc.TargetActionsParams{TargetKind: c.TargetKind, TargetID: c.TargetID})
	if err != nil {
		return CaseDetail{}, err
	}
	for i := range actions {
		a := &actions[i]
		out.History = append(out.History, ActionView{ID: a.ID, ActorKind: a.ActorKind, ActorID: a.ActorID, Action: a.Action,
			Payload: object(a.Payload), CreatedAt: a.CreatedAt})
	}
	return out, nil
}

// object decodes a JSON object column (empty on bad data).
func object(raw []byte) map[string]any {
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out) // written by this module as objects
	return out
}

// ActionInput is the body of POST /v1/admin/cases/{id}/actions.
type ActionInput struct {
	Action string `json:"action"`
	Note   string `json:"note,omitempty"`
	// Sanction is for user_sanction: kind and duration in hours (none =
	// permanent, admins only).
	Sanction *SanctionInput `json:"sanction,omitempty"`
	// SanctionID is for sanction_lift.
	SanctionID *uuid.UUID `json:"sanction_id,omitempty"`
}

// SanctionInput describes a new sanction.
type SanctionInput struct {
	Kind   string `json:"kind"`
	Hours  *int   `json:"hours,omitempty"`
	Reason string `json:"reason_code"`
}

// Sanction kinds a moderator may apply (warnings via user_warn).
var sanctionKinds = []string{"chat_mute", "ranked_ban", "matchmaking_ban", "social_ban", "suspension"}

// Actor is who acts: the staff member and whether they are an admin.
type Actor struct {
	ID    uuid.UUID
	Admin bool
}

// Act applies a moderator decision to a case (schema-moderation §5): the
// effect, its audit row and the case resolution. Admin-only: retiring a
// question and permanent sanctions. Nobody acts on a case about
// themselves.
func (s *Service) Act(ctx context.Context, actor Actor, caseID uuid.UUID, in ActionInput) (CaseDetail, error) {
	q := sqlc.New(s.d.DB)
	c, err := q.GetCase(ctx, caseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CaseDetail{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return CaseDetail{}, err
	}
	if c.TargetKind == "user" && c.TargetID == actor.ID {
		return CaseDetail{}, apierror.New(apierror.SelfReviewForbidden)
	}
	if c.Status == "resolved" && in.Action != "sanction_lift" {
		return CaseDetail{}, apierror.New(apierror.MatchStateInvalid).With("reason", "case_resolved")
	}
	question := c.TargetKind == "question"
	user := c.TargetKind == "user"
	payload := map[string]any{"note": in.Note}
	resolution := "upheld"
	var sanction *sqlc.InsertSanctionParams
	now := s.d.Clock.Now()
	switch in.Action {
	case "case_dismiss":
		resolution = "dismissed"
	case "question_send_to_review":
		if !question {
			return CaseDetail{}, invalid("action", "target_kind")
		}
		if err := s.d.Questions.PullToReview(ctx, &actor.ID, c.TargetID, "moderator"); err != nil {
			return CaseDetail{}, err
		}
	case "question_retire":
		if !question {
			return CaseDetail{}, invalid("action", "target_kind")
		}
		if !actor.Admin {
			return CaseDetail{}, apierror.New(apierror.Forbidden).With("reason", "admin_only")
		}
		if err := s.d.Questions.Retire(ctx, actor.ID, c.TargetID); err != nil {
			return CaseDetail{}, err
		}
	case "user_warn":
		if !user {
			return CaseDetail{}, invalid("action", "target_kind")
		}
		sanction = &sqlc.InsertSanctionParams{Kind: "warning", ReasonCode: reasonOr(in.Sanction, "other")}
	case "user_sanction":
		if !user || in.Sanction == nil || !slices.Contains(sanctionKinds, in.Sanction.Kind) {
			return CaseDetail{}, invalid("sanction.kind", "one_of")
		}
		sanction = &sqlc.InsertSanctionParams{Kind: in.Sanction.Kind, ReasonCode: reasonOr(in.Sanction, "other")}
		switch h := in.Sanction.Hours; {
		case h == nil && !actor.Admin:
			return CaseDetail{}, apierror.New(apierror.Forbidden).With("reason", "permanent_admin_only")
		case h != nil && (*h < 1 || *h > 24*365):
			return CaseDetail{}, invalid("sanction.hours", "range")
		case h != nil:
			end := now.Add(time.Duration(*h) * time.Hour)
			sanction.EndsAt = &end
		}
		payload["kind"], payload["hours"] = in.Sanction.Kind, in.Sanction.Hours
	case "sanction_lift":
		if in.SanctionID == nil {
			return CaseDetail{}, invalid("sanction_id", "required")
		}
		if err := s.lift(ctx, &actor.ID, *in.SanctionID, &caseID, in.Note); err != nil {
			return CaseDetail{}, err
		}
		return s.CaseDetail(ctx, caseID)
	default:
		return CaseDetail{}, invalid("action", "one_of")
	}
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if sanction != nil {
			sanction.ID, sanction.UserID, sanction.CaseID, sanction.CreatedBy, sanction.Now = ids.New(), c.TargetID, &caseID, &actor.ID, now
			if err := q.InsertSanction(ctx, *sanction); err != nil {
				return err
			}
			payload["sanction_id"] = sanction.ID
			if err := s.d.Events.PublishTx(ctx, tx, EventSanctionApplied, now, map[string]any{
				"sanction_id": sanction.ID, "user_id": c.TargetID, "kind": sanction.Kind, "ends_at": sanction.EndsAt,
			}); err != nil {
				return err
			}
		}
		if err := insertAction(ctx, q, &caseID, actorKind(actor), &actor.ID, in.Action, c.TargetKind, c.TargetID, payload, now); err != nil {
			return err
		}
		var note *string
		if in.Note != "" {
			note = &in.Note
		}
		return q.ResolveCase(ctx, sqlc.ResolveCaseParams{ID: caseID, Resolution: &resolution, Note: note, Now: &now})
	})
	if err != nil {
		return CaseDetail{}, err
	}
	if sanction != nil && sanction.Kind == "suspension" {
		if err := s.d.Users.SetStatus(ctx, c.TargetID, identity.StatusSuspended); err != nil {
			return CaseDetail{}, err
		}
	}
	return s.CaseDetail(ctx, caseID)
}

func reasonOr(in *SanctionInput, def string) string {
	if in == nil || in.Reason == "" {
		return def
	}
	return in.Reason
}

func actorKind(a Actor) string {
	if a.Admin {
		return "admin"
	}
	return "moderator"
}

// lift ends a sanction early; a suspension restores the account unless
// another suspension still holds. actor nil = system (expiry).
func (s *Service) lift(ctx context.Context, actor *uuid.UUID, id uuid.UUID, caseID *uuid.UUID, note string) error {
	now := s.d.Clock.Now()
	var restore *uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		sn, err := q.GetSanctionForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil || sn.LiftedAt != nil {
			return err
		}
		if err := q.LiftSanction(ctx, sqlc.LiftSanctionParams{ID: id, LiftedBy: actor, Now: &now}); err != nil {
			return err
		}
		kind := "system"
		if actor != nil {
			kind = "moderator"
		}
		if err := insertAction(ctx, q, caseID, kind, actor, "sanction_lift", "user", sn.UserID, map[string]any{"sanction_id": id, "note": note}, now); err != nil {
			return err
		}
		if sn.Kind == "suspension" {
			other, err := q.OtherActiveSuspension(ctx, sqlc.OtherActiveSuspensionParams{UserID: sn.UserID, ID: id, Now: &now})
			if err != nil {
				return err
			}
			if !other {
				restore = &sn.UserID
			}
		}
		return s.d.Events.PublishTx(ctx, tx, EventSanctionLifted, now, map[string]any{"sanction_id": id, "user_id": sn.UserID, "kind": sn.Kind})
	})
	if err != nil || restore == nil {
		return err
	}
	return s.d.Users.SetStatus(ctx, *restore, identity.StatusActive)
}

// ExpireSuspensions restores accounts whose timed suspension ended
// (worker job).
func (s *Service) ExpireSuspensions(ctx context.Context) error {
	now := s.d.Clock.Now()
	ended, err := sqlc.New(s.d.DB).EndedSuspensions(ctx, &now)
	if err != nil {
		return err
	}
	for i := range ended {
		sn := &ended[i]
		if err := s.lift(ctx, nil, sn.ID, sn.CaseID, "expired"); err != nil {
			return err
		}
	}
	return nil
}

// Sanction is a player's sanction as they see it.
type Sanction struct {
	ID           uuid.UUID  `json:"id"`
	Kind         string     `json:"kind"`
	ReasonCode   string     `json:"reason_code"`
	StartsAt     time.Time  `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	LiftedAt     *time.Time `json:"lifted_at"`
	Active       bool       `json:"active"`
	AppealStatus *string    `json:"appeal_status"`
}

// MySanctions lists a player's sanctions and warnings, newest first.
func (s *Service) MySanctions(ctx context.Context, user uuid.UUID) ([]Sanction, error) {
	rows, err := sqlc.New(s.d.DB).UserSanctions(ctx, user)
	if err != nil {
		return nil, err
	}
	now := s.d.Clock.Now()
	out := make([]Sanction, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, Sanction{ID: r.ID, Kind: r.Kind, ReasonCode: r.ReasonCode, StartsAt: r.StartsAt, EndsAt: r.EndsAt, LiftedAt: r.LiftedAt,
			Active: r.Kind != "warning" && r.LiftedAt == nil && (r.EndsAt == nil || r.EndsAt.After(now)), AppealStatus: r.AppealStatus})
	}
	return out, nil
}

// Appeal asks for a second look at one of the player's sanctions, once
// per sanction.
func (s *Service) Appeal(ctx context.Context, user, sanction uuid.UUID, text string) error {
	if n := len([]rune(text)); n < 10 || n > 1000 {
		return invalid("text", "length")
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		sn, err := q.GetSanctionForUpdate(ctx, sanction)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && sn.UserID != user) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		n, err := q.InsertAppeal(ctx, sqlc.InsertAppealParams{ID: ids.New(), SanctionID: sanction, UserID: user, Text: text, Now: s.d.Clock.Now()})
		if err != nil {
			return err
		}
		if n == 0 {
			return apierror.New(apierror.AppealExists)
		}
		return nil
	})
}

// AppealView is a pending appeal in the staff queue.
type AppealView struct {
	ID         uuid.UUID `json:"id"`
	SanctionID uuid.UUID `json:"sanction_id"`
	UserID     uuid.UUID `json:"user_id"`
	Kind       string    `json:"sanction_kind"`
	ReasonCode string    `json:"reason_code"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}

// PendingAppeals lists appeals waiting for a decision, oldest first.
func (s *Service) PendingAppeals(ctx context.Context) ([]AppealView, error) {
	rows, err := sqlc.New(s.d.DB).PendingAppeals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AppealView, 0, len(rows))
	for i := range rows {
		r := &rows[i]
		out = append(out, AppealView{ID: r.ID, SanctionID: r.SanctionID, UserID: r.UserID, Kind: r.SanctionKind, ReasonCode: r.ReasonCode,
			Text: r.Text, CreatedAt: r.CreatedAt})
	}
	return out, nil
}

// DecideAppeal accepts (the sanction is lifted) or rejects an appeal; it
// must be decided by someone other than the sanction's creator and the
// appellant (schema-moderation §5).
func (s *Service) DecideAppeal(ctx context.Context, actor Actor, id uuid.UUID, accept bool, note string) error {
	now := s.d.Clock.Now()
	var lift *uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		a, err := q.GetAppealForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apierror.New(apierror.NotFound)
		}
		if err != nil {
			return err
		}
		if a.UserID == actor.ID || (a.SanctionCreatedBy != nil && *a.SanctionCreatedBy == actor.ID) {
			return apierror.New(apierror.SelfReviewForbidden)
		}
		if a.Status != "pending" {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "appeal_"+a.Status)
		}
		status := "rejected"
		if accept {
			status, lift = "accepted", &a.SanctionID
		}
		var n *string
		if note != "" {
			n = &note
		}
		if err := q.DecideAppeal(ctx, sqlc.DecideAppealParams{ID: id, Status: status, DecidedBy: &actor.ID, Now: &now, Note: n}); err != nil {
			return err
		}
		return insertAction(ctx, q, nil, actorKind(actor), &actor.ID, "appeal_decide", "user", a.UserID,
			map[string]any{"appeal_id": id, "decision": status, "note": note}, now)
	})
	if err != nil || lift == nil {
		return err
	}
	return s.lift(ctx, &actor.ID, *lift, nil, "appeal_accepted")
}
