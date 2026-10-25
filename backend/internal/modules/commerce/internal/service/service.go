// Package service is commerce's logic (schema-platform.md §4, D-103):
// rewarded ads verified by AdMob server-side verification, and the
// entitlements other modules ask about.
package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/clock"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/events"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// EventAdRewardGranted: a rewarded ad was verified; progression gives
// its effect (backend-modules.md §3.2).
const EventAdRewardGranted = "commerce.AdRewardGranted"

// Placements of rewarded ads (D-103).
const (
	PlacementBonusXP      = "bonus_xp"
	PlacementQuestReroll  = "quest_reroll"
	PlacementStreakRepair = "streak_repair"
)

// Limits (D-031, D-103).
const (
	// RewardedPerDay is the number of rewarded ads a user may watch a day.
	RewardedPerDay = 5
	// QuestRerollsPerDay limits the reroll placement inside that.
	QuestRerollsPerDay = 2
	// StreakRepairsPerWeek: one repaired streak per rolling 7 days.
	StreakRepairsPerWeek = 1
	// InterstitialEvery: one interstitial every this many finished matches.
	InterstitialEvery = 2
	// NewUserNoInterstitial: no interstitials in a user's first days.
	NewUserNoInterstitial = 72 * time.Hour
	// rewardTTL is how long a reserved reward counts as pending.
	rewardTTL = 30 * time.Minute
)

// NoAdsCodes are the entitlements that turn interstitials off.
var NoAdsCodes = []string{"plus", "no_ads"}

// Offer is what a rewarded ad would give.
type Offer struct {
	XP         int `json:"xp,omitempty"`
	StreakDays int `json:"streak_days,omitempty"`
	Volt       int `json:"volt,omitempty"`
}

// RewardChecker says whether a placement can be earned now and what it
// gives (progression implements it; commerce → progression*).
type RewardChecker interface {
	AdOffer(ctx context.Context, user uuid.UUID, placement, ref string) (Offer, error)
}

// Users gives the account's creation time.
type Users interface {
	CreatedAt(ctx context.Context, user uuid.UUID) (time.Time, error)
}

// Deps are the service's collaborators.
type Deps struct {
	DB      *pgxpool.Pool
	Rewards RewardChecker
	// Items applies store items (progression, D-105).
	Items  ItemApplier
	Users  Users
	Events *events.Bus
	Keys   KeySource
	Clock  clock.Clock
	Logger *slog.Logger
}

// Service is commerce's logic.
type Service struct{ d Deps }

// New returns the service.
func New(d Deps) *Service {
	if d.Logger == nil {
		d.Logger = slog.New(slog.DiscardHandler)
	}
	return &Service{d: d}
}

// SetRewards wires the reward checker (internal/app, after progression).
func (s *Service) SetRewards(r RewardChecker) { s.d.Rewards = r }

var istanbul = func() *time.Location {
	l, err := time.LoadLocation("Europe/Istanbul")
	if err != nil {
		return time.FixedZone("TRT", 3*60*60)
	}
	return l
}()

// dayStart is 00:00 Europe/Istanbul of t's day (D-029).
func dayStart(t time.Time) time.Time {
	l := t.In(istanbul)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, istanbul)
}

// HasEntitlement reports whether user currently holds one of codes.
func (s *Service) HasEntitlement(ctx context.Context, user uuid.UUID, codes ...string) (bool, error) {
	return sqlc.New(s.d.DB).HasEntitlement(ctx, sqlc.HasEntitlementParams{UserID: user, Codes: codes, Now: s.d.Clock.Now()})
}

// AdsConfig is GET /v1/ads/config: what the app may show this user.
type AdsConfig struct {
	Interstitial InterstitialConfig `json:"interstitial"`
	Rewarded     RewardedConfig     `json:"rewarded"`
}

// InterstitialConfig: whether interstitials show, and how often.
type InterstitialConfig struct {
	Enabled      bool `json:"enabled"`
	EveryMatches int  `json:"every_matches"`
}

// RewardedConfig: the day's rewarded-ad allowance.
type RewardedConfig struct {
	DailyLimit       int         `json:"daily_limit"`
	UsedToday        int         `json:"used_today"`
	BonusXPLeft      int         `json:"bonus_xp_left"`
	QuestRerollsLeft int         `json:"quest_rerolls_left"`
	StreakRepair     StreakOffer `json:"streak_repair"`
}

// StreakOffer: whether a lost streak can be repaired with an ad.
type StreakOffer struct {
	Available bool `json:"available"`
	Days      int  `json:"days"`
}

func (s *Service) count(ctx context.Context, q *sqlc.Queries, user uuid.UUID, placement string, since, now time.Time) (int, error) {
	n, err := q.CountAdRewards(ctx, sqlc.CountAdRewardsParams{UserID: user, Placement: placement, Since: since, Now: now})
	return int(n), err
}

// Config returns the ad settings for user.
func (s *Service) Config(ctx context.Context, user uuid.UUID) (AdsConfig, error) {
	now := s.d.Clock.Now()
	q := sqlc.New(s.d.DB)
	out := AdsConfig{
		Interstitial: InterstitialConfig{EveryMatches: InterstitialEvery},
		Rewarded:     RewardedConfig{DailyLimit: RewardedPerDay},
	}
	noAds, err := s.HasEntitlement(ctx, user, NoAdsCodes...)
	if err != nil {
		return out, err
	}
	created, err := s.d.Users.CreatedAt(ctx, user)
	if err != nil {
		return out, err
	}
	out.Interstitial.Enabled = !noAds && now.Sub(created) >= NewUserNoInterstitial

	day := dayStart(now)
	used, err := s.count(ctx, q, user, "", day, now)
	if err != nil {
		return out, err
	}
	left := max(RewardedPerDay-used, 0)
	rerolls, err := s.count(ctx, q, user, PlacementQuestReroll, day, now)
	if err != nil {
		return out, err
	}
	out.Rewarded.UsedToday, out.Rewarded.BonusXPLeft = used, left
	out.Rewarded.QuestRerollsLeft = min(left, max(QuestRerollsPerDay-rerolls, 0))
	if left > 0 && s.d.Rewards != nil {
		repairs, err := s.count(ctx, q, user, PlacementStreakRepair, now.Add(-7*24*time.Hour), now)
		if err != nil {
			return out, err
		}
		if repairs < StreakRepairsPerWeek {
			o, err := s.d.Rewards.AdOffer(ctx, user, PlacementStreakRepair, "")
			var ae *apierror.Error
			switch {
			case err == nil:
				out.Rewarded.StreakRepair = StreakOffer{Available: true, Days: o.StreakDays}
			case !errors.As(err, &ae):
				return out, err
			}
		}
	}
	return out, nil
}

// Reward is a reserved or granted rewarded ad.
type Reward struct {
	ID        uuid.UUID `json:"id"`
	Placement string    `json:"placement"`
	Ref       string    `json:"ref,omitempty"`
	Status    string    `json:"status"`
	Offer     Offer     `json:"reward"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Reserve checks a placement and its limits and reserves a reward whose
// id the app passes to AdMob as custom_data; an open reservation of the
// same placement and match is returned again.
func (s *Service) Reserve(ctx context.Context, user uuid.UUID, placement, ref string) (Reward, error) {
	var offer Offer
	switch {
	case placement == PlacementVolt: // the store's Volt ad (D-105)
		offer = Offer{Volt: AdVolt}
	case s.d.Rewards == nil:
		return Reward{}, apierror.New(apierror.AdRewardUnavailable).With("placement", placement).With("reason", "ads_off")
	default:
		o, err := s.d.Rewards.AdOffer(ctx, user, placement, ref)
		if err != nil {
			return Reward{}, err
		}
		offer = o
	}
	now := s.d.Clock.Now()
	var out Reward
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('commerce.ads:' || $1::text))`, user.String()); err != nil {
			return err
		}
		open, err := q.OpenAdReward(ctx, sqlc.OpenAdRewardParams{UserID: user, Placement: placement, Ref: ref, Now: now})
		if err == nil {
			out = Reward{ID: open.ID, Placement: placement, Ref: ref, Status: open.Status, Offer: offer, ExpiresAt: open.ExpiresAt}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		limited := func(limit int) error {
			return apierror.New(apierror.AdLimitReached).With("placement", placement).With("limit", limit)
		}
		day := dayStart(now)
		used, err := s.count(ctx, q, user, "", day, now)
		if err != nil {
			return err
		}
		if used >= RewardedPerDay && placement != PlacementVolt {
			return limited(RewardedPerDay)
		}
		switch placement {
		case PlacementVolt:
			n, err := s.count(ctx, q, user, placement, day, now)
			if err != nil {
				return err
			}
			if n >= AdVoltPerDay {
				return limited(AdVoltPerDay)
			}
		case PlacementQuestReroll:
			n, err := s.count(ctx, q, user, placement, day, now)
			if err != nil {
				return err
			}
			if n >= QuestRerollsPerDay {
				return limited(QuestRerollsPerDay)
			}
		case PlacementStreakRepair:
			n, err := s.count(ctx, q, user, placement, now.Add(-7*24*time.Hour), now)
			if err != nil {
				return err
			}
			if n >= StreakRepairsPerWeek {
				return limited(StreakRepairsPerWeek)
			}
		}
		id := ids.New()
		exp := now.Add(rewardTTL)
		if err := q.InsertAdReward(ctx, sqlc.InsertAdRewardParams{ID: id, UserID: user, Placement: placement, Ref: ref, Now: now, ExpiresAt: exp}); err != nil {
			return err
		}
		out = Reward{ID: id, Placement: placement, Ref: ref, Status: "pending", Offer: offer, ExpiresAt: exp}
		return nil
	})
	if err != nil || out.Status != "pending" {
		return out, err
	}
	// Plus earns rewarded-ad rewards without the ad (D-104).
	plus, err := s.HasEntitlement(ctx, user, EntitlementPlus)
	if err != nil || !plus {
		return out, err
	}
	if _, err := s.Grant(ctx, out.ID, user, "plus-"+out.ID.String()); err != nil {
		return out, err
	}
	out.Status = "granted"
	return out, nil
}

// Get returns one of the caller's rewards (the app polls it after the ad).
func (s *Service) Get(ctx context.Context, user, id uuid.UUID) (Reward, error) {
	r, err := sqlc.New(s.d.DB).GetAdReward(ctx, sqlc.GetAdRewardParams{ID: id, UserID: user})
	if errors.Is(err, pgx.ErrNoRows) {
		return Reward{}, apierror.New(apierror.NotFound)
	}
	if err != nil {
		return Reward{}, err
	}
	return Reward{ID: r.ID, Placement: r.Placement, Ref: r.Ref, Status: r.Status, ExpiresAt: r.ExpiresAt}, nil
}

// AdRewardGranted is EventAdRewardGranted's payload.
type AdRewardGranted struct {
	RewardID  uuid.UUID `json:"reward_id"`
	UserID    uuid.UUID `json:"user_id"`
	Placement string    `json:"placement"`
	Ref       string    `json:"ref"`
	GrantedAt time.Time `json:"granted_at"`
}

// Grant marks a reward earned once (user must own it) and publishes
// EventAdRewardGranted in the same transaction. A second callback for the
// same reward or transaction is ignored. ok is false when the reward is
// unknown or belongs to someone else.
func (s *Service) Grant(ctx context.Context, id, user uuid.UUID, transaction string) (ok bool, err error) {
	now := s.d.Clock.Now()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		r, err := q.GetAdRewardForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if r.UserID != user {
			return nil
		}
		ok = true
		if r.Status == "granted" {
			return nil
		}
		if err := q.GrantAdReward(ctx, sqlc.GrantAdRewardParams{ID: id, TransactionID: &transaction, Now: &now}); err != nil {
			return err
		}
		if r.Placement == PlacementVolt { // commerce's own reward (D-105)
			_, _, err := credit(ctx, q, user, AdVolt, VoltAd, id.String(), "ad:"+id.String(), now)
			return err
		}
		return s.d.Events.PublishTx(ctx, tx, EventAdRewardGranted, now,
			AdRewardGranted{RewardID: id, UserID: user, Placement: r.Placement, Ref: r.Ref, GrantedAt: now})
	})
	if isUniqueViolation(err) {
		return true, nil // the same AdMob transaction again
	}
	return ok, err
}
