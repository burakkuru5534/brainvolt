package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/tournament/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Tournament kinds (D-109).
const (
	KindOfficial = "official"
	KindFriends  = "friends"
)

// Friend tournament rules (D-109).
const (
	// FriendFee is the set-up fee in Volt; Plus sets up for free.
	FriendFee = 300
	// FriendMaxOpen is how many unfinished friend tournaments one owner may have.
	FriendMaxOpen  = 2
	friendMinStart = 15 * time.Minute
	friendMaxStart = 7 * 24 * time.Hour
	reasonFee      = "friend_tournament"
	reasonRefund   = "friend_tournament_refund"
)

// FriendSizes are the places a friend tournament can have.
var FriendSizes = []int{4, 8, 16}

// FriendMatchTypes are the match types a friend tournament can use.
var FriendMatchTypes = []game.MatchType{"quick_duel", "sudden_death", "marathon"}

// Friends answers friendship questions (social, wired in internal/app).
type Friends interface {
	AreFriends(ctx context.Context, a, b uuid.UUID) (bool, error)
	FriendIDs(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error)
}

// Wallet spends and refunds Volt inside tournament's transactions
// (commerce.VoltWriter).
type Wallet interface {
	CreditTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, amount int, reason, ref, key string, now time.Time) error
	DebitTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, amount int, reason, ref, key string, now time.Time) error
}

// PlusChecker reports entitlements (commerce).
type PlusChecker interface {
	HasEntitlement(ctx context.Context, user uuid.UUID, codes ...string) (bool, error)
}

// owners are the users whose friend tournaments user may see: user and
// user's friends.
func (s *Service) owners(ctx context.Context, user uuid.UUID) ([]uuid.UUID, error) {
	out := []uuid.UUID{user}
	if s.d.Friends == nil {
		return out, nil
	}
	friends, err := s.d.Friends.FriendIDs(ctx, user)
	return append(out, friends...), err
}

// canSee reports whether user may see (and join) owner's friend tournament.
func (s *Service) canSee(ctx context.Context, user, owner uuid.UUID) (bool, error) {
	if user == owner {
		return true, nil
	}
	if s.d.Friends == nil {
		return false, nil
	}
	return s.d.Friends.AreFriends(ctx, user, owner)
}

// FriendTournamentInput is the body of POST /v1/tournaments/friends.
type FriendTournamentInput struct {
	Name        string         `json:"name"`
	MatchType   game.MatchType `json:"match_type"`
	MaxEntrants int            `json:"max_entrants"`
	StartsAt    time.Time      `json:"starts_at"`
}

// CreateFriends sets up a friend tournament: 1v1, single elimination,
// open for registration at once, unrated and without rewards. It costs
// FriendFee Volt unless the owner has Plus.
func (s *Service) CreateFriends(ctx context.Context, owner uuid.UUID, full bool, in FriendTournamentInput) (Tournament, error) {
	if !full {
		return Tournament{}, apierror.New(apierror.AccountUpgradeRequired).With("feature", "friend_tournament")
	}
	bad := func(field, rule string) error {
		return apierror.Validation(apierror.FieldError{Field: field, Rule: rule})
	}
	name := strings.TrimSpace(in.Name)
	now := s.d.Clock.Now()
	switch n := len([]rune(name)); {
	case n < 3 || n > 40:
		return Tournament{}, bad("name", "length")
	case !slices.Contains(FriendMatchTypes, in.MatchType):
		return Tournament{}, bad("match_type", "enum")
	case !slices.Contains(FriendSizes, in.MaxEntrants):
		return Tournament{}, bad("max_entrants", "enum")
	case in.StartsAt.Before(now.Add(friendMinStart)) || in.StartsAt.After(now.Add(friendMaxStart)):
		return Tournament{}, bad("starts_at", "range")
	}
	cfg := game.Config{MatchType: in.MatchType, Format: "1v1", Origin: game.OriginTournament}
	if vs := game.ValidateConfig(cfg); len(vs) > 0 {
		return Tournament{}, apierror.New(apierror.ModeNotAllowed).With("rule_id", vs[0].RuleID).With("violations", vs)
	}
	cfgJSON, err := json.Marshal(cfg)
	if err != nil {
		return Tournament{}, err
	}
	rewards, err := json.Marshal(TournamentRewards{Places: []TournamentPlaceReward{}})
	if err != nil {
		return Tournament{}, err
	}
	fee := FriendFee
	if s.d.Plus != nil {
		plus, err := s.d.Plus.HasEntitlement(ctx, owner, "plus")
		if err != nil {
			return Tournament{}, err
		}
		if plus {
			fee = 0
		}
	}
	if fee > 0 && s.d.Wallet == nil {
		return Tournament{}, apierror.New(apierror.ItemUnavailable).With("item", "friend_tournament").With("reason", "store_off")
	}
	id := ids.New()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('tournament.friends:' || $1::text))`, owner.String()); err != nil {
			return err
		}
		open, err := q.OpenFriendTournaments(ctx, owner)
		if err != nil {
			return err
		}
		if open >= FriendMaxOpen {
			return apierror.New(apierror.ItemUnavailable).With("item", "friend_tournament").With("reason", "too_many_open")
		}
		if err := q.InsertTournament(ctx, sqlc.InsertTournamentParams{
			ID: id, Status: StatusRegistration, EntityKind: "user", MatchConfig: cfgJSON, MinEntrants: 2,
			MaxEntrants: int32(in.MaxEntrants), RegistrationOpensAt: now.Add(-time.Second), RegistrationClosesAt: in.StartsAt, //nolint:gosec // 4–16
			StartsAt: in.StartsAt, RoundReadyWindowS: 300, Rewards: rewards, CreatedBy: owner, Now: now, Kind: KindFriends, Fee: int32(fee), //nolint:gosec // small
		}); err != nil {
			return err
		}
		if err := q.InsertTranslation(ctx, sqlc.InsertTranslationParams{TournamentID: id, Locale: "tr", Name: name}); err != nil {
			return err
		}
		if fee > 0 {
			return s.d.Wallet.DebitTx(ctx, tx, owner, fee, reasonFee, id.String(), "ft:"+id.String(), now)
		}
		return nil
	})
	if err != nil {
		return Tournament{}, err
	}
	d, err := s.Get(ctx, owner, id, "tr", false)
	return d.Tournament, err
}

// refundFee gives a cancelled friend tournament's fee back to its owner,
// once.
func (s *Service) refundFee(ctx context.Context, tx pgx.Tx, t *sqlc.TournamentTournament, now time.Time) error {
	if t.Kind != KindFriends || t.Fee <= 0 || s.d.Wallet == nil {
		return nil
	}
	id := t.ID.String()
	return s.d.Wallet.CreditTx(ctx, tx, t.CreatedBy, int(t.Fee), reasonRefund, id, "ft-refund:"+id, now)
}

// CancelFriends lets the owner cancel a friend tournament before it starts;
// the fee comes back.
func (s *Service) CancelFriends(ctx context.Context, owner, id uuid.UUID) error {
	var users []uuid.UUID
	now := s.d.Clock.Now()
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		t, err := q.LockTournament(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (t.Kind != KindFriends || t.CreatedBy != owner)) {
			return notFound()
		}
		if err != nil {
			return err
		}
		if t.Status != StatusRegistration {
			return apierror.New(apierror.MatchStateInvalid).With("reason", "tournament_"+t.Status)
		}
		if users, err = lineups(ctx, q, id); err != nil {
			return err
		}
		if err := q.SetTournamentStatus(ctx, sqlc.SetTournamentStatusParams{ID: id, Status: StatusCancelled, Now: now}); err != nil {
			return err
		}
		return s.refundFee(ctx, tx, &t, now)
	})
	if err == nil {
		s.pushUpdated(ctx, id, users)
	}
	return err
}

// FriendTrophies counts the friend tournaments user won.
func (s *Service) FriendTrophies(ctx context.Context, user uuid.UUID) (int, error) {
	n, err := sqlc.New(s.d.DB).FriendTrophies(ctx, user)
	return int(n), err
}
