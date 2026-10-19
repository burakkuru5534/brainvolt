package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/moderation/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Signal kinds moderation records (schema-moderation §4, D-084).
const (
	SignalSuperhuman     = "superhuman_response_time"
	SignalRejectBurst    = "rejected_submission_burst"
	SignalProtocol       = "protocol_violation"
	SignalRepeatedVoids  = "repeated_server_voids"
	fastCorrectThreshold = 2
	rejectsThreshold     = 5
	voidsThreshold       = 3
	voidsWindow          = 7 * 24 * time.Hour
)

// Signal is one automatic detection.
type Signal struct {
	User     uuid.UUID
	Match    *uuid.UUID
	Kind     string
	Severity int
	Details  map[string]any
}

// RecordSignal stores a signal on the user's open case (opening one):
// once per user, kind and match, or once an hour per user and kind
// without a match. The case's priority grows by the severity; signals
// never sanction by themselves (schema-moderation §4).
func (s *Service) RecordSignal(ctx context.Context, sg Signal) error {
	body, err := json.Marshal(sg.Details)
	if err != nil {
		return err
	}
	now := s.d.Clock.Now()
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, sg.User.String()); err != nil {
			return err
		}
		dup, err := q.SignalExists(ctx, sqlc.SignalExistsParams{UserID: sg.User, Kind: sg.Kind, MatchID: sg.Match, Since: now.Add(-time.Hour)})
		if err != nil || dup {
			return err
		}
		c, err := q.OpenCaseForUpdate(ctx, sqlc.OpenCaseForUpdateParams{TargetKind: "user", TargetID: sg.User})
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			c = sqlc.ModerationCase{ID: ids.New()}
			if err := q.InsertCase(ctx, sqlc.InsertCaseParams{ID: c.ID, TargetKind: "user", TargetID: sg.User, Now: now}); err != nil {
				return err
			}
		case err != nil:
			return err
		}
		if err := q.InsertSignal(ctx, sqlc.InsertSignalParams{
			ID: ids.New(), UserID: sg.User, MatchID: sg.Match, Kind: sg.Kind, Severity: int16(sg.Severity), //nolint:gosec // G115: 1–5
			Details: body, CaseID: &c.ID, Now: now,
		}); err != nil {
			return err
		}
		return q.BumpCaseSignal(ctx, sqlc.BumpCaseSignalParams{ID: c.ID, Weight: int32(sg.Severity)}) //nolint:gosec // G115: 1–5
	})
}

// completed is the part of game.MatchCompleted the signals read.
type completed struct {
	MatchID      uuid.UUID `json:"match_id"`
	Participants []struct {
		UserID      uuid.UUID `json:"user_id"`
		Answered    int       `json:"answered"`
		Rejects     int       `json:"rejects"`
		FastCorrect int       `json:"fast_correct"`
		FastestMS   int       `json:"fastest_ms"`
	} `json:"participants"`
}

// OnMatchCompleted turns a completed match's measures into signals:
// two or more correct answers under the reading-time floor, five or more
// refused submissions (D-084).
func (s *Service) OnMatchCompleted(ctx context.Context, e events.Envelope) error {
	var p completed
	if err := e.Decode(&p); err != nil {
		return err
	}
	match := p.MatchID
	for i := range p.Participants {
		me := &p.Participants[i]
		if me.FastCorrect >= fastCorrectThreshold {
			if err := s.RecordSignal(ctx, Signal{User: me.UserID, Match: &match, Kind: SignalSuperhuman, Severity: min(5, me.FastCorrect),
				Details: map[string]any{"fast_correct": me.FastCorrect, "answered": me.Answered, "fastest_ms": me.FastestMS,
					"floor": "300 ms + 15 ms per prompt character", "threshold": fastCorrectThreshold}}); err != nil {
				return err
			}
		}
		if me.Rejects >= rejectsThreshold {
			if err := s.RecordSignal(ctx, Signal{User: me.UserID, Match: &match, Kind: SignalRejectBurst, Severity: min(3, me.Rejects/rejectsThreshold),
				Details: map[string]any{"rejects": me.Rejects, "threshold": rejectsThreshold}}); err != nil {
				return err
			}
		}
	}
	return nil
}

// cancelled is the part of game.MatchCancelled the signals read.
type cancelled struct {
	MatchID        uuid.UUID   `json:"match_id"`
	Reason         string      `json:"reason"`
	ParticipantIDs []uuid.UUID `json:"participant_ids"`
}

// OnMatchCancelled raises a signal for a player whose matches the server
// cancelled three times or more in seven days (crash or desync abuse).
func (s *Service) OnMatchCancelled(ctx context.Context, e events.Envelope) error {
	var p cancelled
	if err := e.Decode(&p); err != nil {
		return err
	}
	if (p.Reason != "server_error" && p.Reason != "integrity_violation") || s.d.ServerVoids == nil {
		return nil
	}
	since := s.d.Clock.Now().Add(-voidsWindow)
	for _, u := range p.ParticipantIDs {
		n, err := s.d.ServerVoids(ctx, u, since)
		if err != nil {
			return err
		}
		if n < voidsThreshold {
			continue
		}
		match := p.MatchID
		if err := s.RecordSignal(ctx, Signal{User: u, Match: &match, Kind: SignalRepeatedVoids, Severity: min(3, n/voidsThreshold),
			Details: map[string]any{"voids_7d": n, "threshold": voidsThreshold, "reason": p.Reason}}); err != nil {
			return err
		}
	}
	return nil
}
