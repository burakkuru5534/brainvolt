package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// TournamentRegistration is the body of POST /v1/tournaments/{id}/entries: nothing
// for a user or couple tournament; the team and its lineup for a team one.
type TournamentRegistration struct {
	TeamID *uuid.UUID  `json:"team_id,omitempty"`
	Lineup []uuid.UUID `json:"lineup,omitempty"`
}

// Register enters the caller (user), their couple or their team (captain,
// with a lineup of current members) while registration is open.
func (s *Service) Register(ctx context.Context, user uuid.UUID, full bool, id uuid.UUID, in TournamentRegistration) (TournamentEntry, error) {
	if !full {
		return TournamentEntry{}, apierror.New(apierror.AccountUpgradeRequired).With("feature", "tournament")
	}
	var entryID uuid.UUID
	var users []uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		t, err := q.LockTournament(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && t.Status == StatusDraft) {
			return notFound()
		}
		if err != nil {
			return err
		}
		if t.Kind == KindFriends { // the owner and the owner's friends (D-109)
			if ok, err := s.canSee(ctx, user, t.CreatedBy); err != nil || !ok {
				if err == nil {
					err = notFound()
				}
				return err
			}
		}
		now := s.d.Clock.Now()
		if t.Status != StatusRegistration || now.Before(t.RegistrationOpensAt) || !now.Before(t.RegistrationClosesAt) {
			return apierror.New(apierror.TournamentRegistrationClosed)
		}
		var cfg game.Config
		if err := json.Unmarshal(t.MatchConfig, &cfg); err != nil {
			return err
		}
		entity, lineup, err := s.lineup(ctx, user, t.EntityKind, cfg.Format, in)
		if err != nil {
			return err
		}
		taken, err := q.LineupTaken(ctx, sqlc.LineupTakenParams{TournamentID: id, Lineup: lineup})
		if err != nil {
			return err
		}
		if taken {
			return apierror.New(apierror.TournamentRequirementsNotMet).With("reason", "already_registered")
		}
		n, err := q.CountEntries(ctx, id)
		if err != nil {
			return err
		}
		if n >= t.MaxEntrants {
			return apierror.New(apierror.TournamentFull)
		}
		users = lineup
		old, err := q.EntityEntry(ctx, sqlc.EntityEntryParams{TournamentID: id, EntityKind: t.EntityKind, EntityID: entity})
		switch {
		case err == nil:
			entryID = old.ID
			return q.ReviveEntry(ctx, sqlc.ReviveEntryParams{ID: old.ID, Lineup: lineup, Now: now})
		case errors.Is(err, pgx.ErrNoRows):
			entryID = ids.New()
			return q.InsertEntry(ctx, sqlc.InsertEntryParams{ID: entryID, TournamentID: id, EntityKind: t.EntityKind, EntityID: entity, Lineup: lineup, Now: now})
		default:
			return err
		}
	})
	if err != nil {
		return TournamentEntry{}, err
	}
	s.pushUpdated(ctx, id, users)
	return s.entry(ctx, user, id, entryID)
}

// lineup is who plays for the caller's entry.
func (s *Service) lineup(ctx context.Context, user uuid.UUID, kind string, f game.Format, in TournamentRegistration) (uuid.UUID, []uuid.UUID, error) {
	switch kind {
	case "user":
		return user, []uuid.UUID{user}, nil
	case "couple":
		c, ok, err := s.d.Teams.ActiveCoupleOf(ctx, user)
		if err != nil {
			return uuid.Nil, nil, err
		}
		if !ok {
			return uuid.Nil, nil, apierror.New(apierror.TournamentRequirementsNotMet).With("reason", "no_couple")
		}
		return c.ID, c.Users[:], nil
	}
	if in.TeamID == nil {
		return uuid.Nil, nil, apierror.Validation(apierror.FieldError{Field: "team_id", Rule: "required"})
	}
	t, err := s.d.Teams.TeamInfo(ctx, *in.TeamID)
	if err != nil || !t.Active {
		return uuid.Nil, nil, notFound()
	}
	if t.Captain != user {
		return uuid.Nil, nil, apierror.New(apierror.NotCaptain)
	}
	size, _ := sideSize(kind, f)
	lineup := slices.Clone(in.Lineup)
	slices.SortFunc(lineup, func(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) })
	if len(slices.Compact(slices.Clone(lineup))) != size || len(lineup) != size {
		return uuid.Nil, nil, apierror.Validation(apierror.FieldError{Field: "lineup", Rule: "size"})
	}
	for _, u := range lineup {
		if !slices.Contains(t.Members, u) {
			return uuid.Nil, nil, apierror.Validation(apierror.FieldError{Field: "lineup", Rule: "members"})
		}
	}
	return t.ID, lineup, nil
}

func (s *Service) entry(ctx context.Context, user, tournament, id uuid.UUID) (TournamentEntry, error) {
	d, err := s.Get(ctx, user, tournament, "tr", false)
	if err != nil {
		return TournamentEntry{}, err
	}
	for i := range d.Entries {
		if d.Entries[i].ID == id {
			return d.Entries[i], nil
		}
	}
	return TournamentEntry{}, notFound()
}

// liveEntry locks the tournament and returns the caller's live entry.
func liveEntry(ctx context.Context, q *sqlc.Queries, user, id uuid.UUID) (sqlc.TournamentTournament, sqlc.TournamentEntry, error) {
	t, err := q.LockTournament(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && t.Status == StatusDraft) {
		return t, sqlc.TournamentEntry{}, notFound()
	}
	if err != nil {
		return t, sqlc.TournamentEntry{}, err
	}
	e, err := q.EntryOf(ctx, sqlc.EntryOfParams{TournamentID: id, UserID: user})
	if errors.Is(err, pgx.ErrNoRows) {
		return t, e, notFound()
	}
	return t, e, err
}

// Withdraw takes the caller's entry out before the start (for a team, the
// captain; for a couple, either partner).
func (s *Service) Withdraw(ctx context.Context, user, id uuid.UUID) error {
	var users []uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		t, e, err := liveEntry(ctx, q, user, id)
		if err != nil {
			return err
		}
		if t.Status != StatusRegistration {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "tournament_"+t.Status)
		}
		if e.EntityKind == "team" {
			info, err := s.d.Teams.TeamInfo(ctx, e.EntityID)
			if err != nil {
				return err
			}
			if info.Captain != user {
				return apierror.New(apierror.NotCaptain)
			}
		}
		users = e.Lineup
		return q.SetEntryStatus(ctx, sqlc.SetEntryStatusParams{ID: e.ID, Status: "withdrawn"})
	})
	if err == nil {
		s.pushUpdated(ctx, id, users)
	}
	return err
}

// CheckIn confirms the caller's entry in the hour before the start; only
// checked-in entries are seeded.
func (s *Service) CheckIn(ctx context.Context, user, id uuid.UUID) (TournamentEntry, error) {
	var entryID uuid.UUID
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		t, e, err := liveEntry(ctx, q, user, id)
		if err != nil {
			return err
		}
		now := s.d.Clock.Now()
		if t.Status != StatusRegistration || now.Before(t.StartsAt.Add(-CheckInOpens)) || !now.Before(t.StartsAt) {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "check_in_closed")
		}
		entryID = e.ID
		return q.SetEntryStatus(ctx, sqlc.SetEntryStatusParams{ID: e.ID, Status: "checked_in"})
	})
	if err != nil {
		return TournamentEntry{}, err
	}
	return s.entry(ctx, user, id, entryID)
}
