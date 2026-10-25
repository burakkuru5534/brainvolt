package service

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/progression/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Season Pass (D-108): 40 tiers, one per 500 XP earned inside the season.
const (
	PassTiers    = 40
	PassTierXP   = 500
	PassPrice    = 900
	passShieldVo = 30 // Volt instead of a shield when two are held
	trackFree    = "free"
	trackPremium = "premium"

	reasonPassBuy    = "season_pass"
	reasonPassReward = "pass_reward"
)

// SeasonInfo is the current ranked season (ranking).
type SeasonInfo struct {
	ID       uuid.UUID
	Number   int
	StartsAt time.Time
	EndsAt   time.Time
	Names    map[string]string
}

// SeasonSource gives the current season (ranking, adapted in the module).
type SeasonSource interface {
	CurrentSeason(ctx context.Context) (SeasonInfo, error)
}

// PassReward is one tier's reward on a track.
type PassReward struct {
	Volt     int    `json:"volt,omitempty"`
	Shield   bool   `json:"shield,omitempty"`
	Cosmetic string `json:"cosmetic,omitempty"`
}

// Season 1, "Storm Season" (owner approved, D-108).
var (
	passFree = map[int]PassReward{
		5: {Volt: 50}, 10: {Shield: true}, 15: {Volt: 75}, 25: {Volt: 100},
		30: {Shield: true}, 35: {Volt: 125}, 40: {Volt: 150},
	}
	passPremium = func() map[int]PassReward {
		m := map[int]PassReward{
			1:  {Cosmetic: "frame_s1_storm"},
			10: {Cosmetic: "avatar_stormcloud"},
			20: {Cosmetic: "name_s1_electric"},
			30: {Cosmetic: "theme_s1_storm"},
			40: {Cosmetic: "victory_s1_thunder", Volt: 150},
		}
		for _, t := range []int{5, 15, 25, 35} {
			m[t] = PassReward{Shield: true}
		}
		for t := 2; t < PassTiers; t += 2 {
			if _, ok := m[t]; !ok {
				m[t] = PassReward{Volt: 60}
			}
		}
		return m
	}()
)

// PassTier is one tier with both tracks.
type PassTier struct {
	Tier           int         `json:"tier"`
	Free           *PassReward `json:"free"`
	Premium        *PassReward `json:"premium"`
	FreeClaimed    bool        `json:"free_claimed"`
	PremiumClaimed bool        `json:"premium_claimed"`
}

// PassSeason is the season of the pass.
type PassSeason struct {
	ID     uuid.UUID `json:"id"`
	Number int       `json:"number"`
	Name   string    `json:"name"`
	EndsAt time.Time `json:"ends_at"`
}

// SeasonPass is GET /v1/me/season-pass.
type SeasonPass struct {
	Season PassSeason `json:"season"`
	XP     int64      `json:"xp"`
	// Tier is the highest tier reached (0–40); TierXP the XP inside the
	// next one.
	Tier      int        `json:"tier"`
	TierXP    int64      `json:"tier_xp"`
	TierSize  int        `json:"tier_size"`
	Premium   bool       `json:"premium"`
	Price     int        `json:"price"`
	Claimable int        `json:"claimable"`
	Tiers     []PassTier `json:"tiers"`
}

// PassClaimed is what a claim gave.
type PassClaimed struct {
	Pass      SeasonPass `json:"pass"`
	Volt      int        `json:"volt"`
	Shields   int        `json:"shields"`
	Cosmetics []string   `json:"cosmetics"`
}

func (s *Service) season(ctx context.Context) (SeasonInfo, error) {
	if s.d.Season == nil {
		return SeasonInfo{}, apierror.New(apierror.NotFound)
	}
	return s.d.Season.CurrentSeason(ctx)
}

type claimKey struct {
	track string
	tier  int
}

// pass builds the caller's pass inside q.
func (s *Service) pass(ctx context.Context, q *sqlc.Queries, user uuid.UUID, se SeasonInfo, locale string) (SeasonPass, map[claimKey]bool, error) {
	xp, err := q.XPBetween(ctx, sqlc.XPBetweenParams{UserID: user, Since: se.StartsAt, Until: se.EndsAt})
	if err != nil {
		return SeasonPass{}, nil, err
	}
	premium, err := q.PassPremium(ctx, sqlc.PassPremiumParams{UserID: user, SeasonID: se.ID})
	if err != nil {
		return SeasonPass{}, nil, err
	}
	rows, err := q.PassClaims(ctx, sqlc.PassClaimsParams{UserID: user, SeasonID: se.ID})
	if err != nil {
		return SeasonPass{}, nil, err
	}
	claimed := map[claimKey]bool{}
	for _, r := range rows {
		claimed[claimKey{r.Track, int(r.Tier)}] = true
	}
	name := se.Names[locale]
	if name == "" {
		name = se.Names["en"]
	}
	tier := min(int(xp/PassTierXP), PassTiers)
	out := SeasonPass{
		Season: PassSeason{ID: se.ID, Number: se.Number, Name: name, EndsAt: se.EndsAt},
		XP:     xp, Tier: tier, TierSize: PassTierXP, Premium: premium, Price: PassPrice, Tiers: make([]PassTier, 0, PassTiers),
	}
	if tier < PassTiers {
		out.TierXP = xp - int64(tier*PassTierXP)
	}
	for t := 1; t <= PassTiers; t++ {
		pt := PassTier{Tier: t, FreeClaimed: claimed[claimKey{trackFree, t}], PremiumClaimed: claimed[claimKey{trackPremium, t}]}
		if r, ok := passFree[t]; ok {
			pt.Free = &r
			if t <= tier && !pt.FreeClaimed {
				out.Claimable++
			}
		}
		if r, ok := passPremium[t]; ok {
			pt.Premium = &r
			if premium && t <= tier && !pt.PremiumClaimed {
				out.Claimable++
			}
		}
		out.Tiers = append(out.Tiers, pt)
	}
	return out, claimed, nil
}

// SeasonPass returns the caller's pass for the current season.
func (s *Service) SeasonPass(ctx context.Context, user uuid.UUID, locale string) (SeasonPass, error) {
	se, err := s.season(ctx)
	if err != nil {
		return SeasonPass{}, err
	}
	p, _, err := s.pass(ctx, sqlc.New(s.d.DB), user, se, locale)
	return p, err
}

// BuyPremium opens the premium track for PassPrice Volt, once a season.
func (s *Service) BuyPremium(ctx context.Context, user uuid.UUID, locale string) (SeasonPass, error) {
	se, err := s.season(ctx)
	if err != nil {
		return SeasonPass{}, err
	}
	if s.d.Wallet == nil {
		return SeasonPass{}, apierror.New(apierror.ItemUnavailable).With("item", "season_pass").With("reason", "store_off")
	}
	now := s.d.Clock.Now()
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, user.String()); err != nil {
			return err
		}
		n, err := q.InsertPassPremium(ctx, sqlc.InsertPassPremiumParams{UserID: user, SeasonID: se.ID, Now: now})
		if err != nil {
			return err
		}
		if n == 0 {
			return apierror.New(apierror.ItemUnavailable).With("item", "season_pass").With("reason", "already_owned")
		}
		id := se.ID.String()
		return s.d.Wallet.DebitTx(ctx, tx, user, PassPrice, reasonPassBuy, id, "pass:"+id, now)
	})
	if err != nil {
		return SeasonPass{}, err
	}
	return s.SeasonPass(ctx, user, locale)
}

// ClaimPass gives every reached, unclaimed reward of the caller's tracks.
func (s *Service) ClaimPass(ctx context.Context, user uuid.UUID, locale string) (PassClaimed, error) {
	se, err := s.season(ctx)
	if err != nil {
		return PassClaimed{}, err
	}
	now := s.d.Clock.Now()
	out := PassClaimed{Cosmetics: []string{}}
	err = postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		out = PassClaimed{Cosmetics: []string{}}
		q := sqlc.New(tx)
		if err := q.LockUser(ctx, user.String()); err != nil {
			return err
		}
		p, _, err := s.pass(ctx, q, user, se, locale)
		if err != nil {
			return err
		}
		for _, pt := range p.Tiers {
			if pt.Tier > p.Tier {
				break
			}
			for _, c := range []struct {
				track   string
				reward  *PassReward
				claimed bool
			}{{trackFree, pt.Free, pt.FreeClaimed}, {trackPremium, pt.Premium, pt.PremiumClaimed}} {
				if c.reward == nil || c.claimed || (c.track == trackPremium && !p.Premium) {
					continue
				}
				if err := s.givePass(ctx, tx, q, user, se.ID, c.track, pt.Tier, *c.reward, now, &out); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return PassClaimed{}, err
	}
	out.Pass, err = s.SeasonPass(ctx, user, locale)
	return out, err
}

func (s *Service) givePass(ctx context.Context, tx pgx.Tx, q *sqlc.Queries, user, season uuid.UUID, track string, tier int, r PassReward, now time.Time, out *PassClaimed) error {
	n, err := q.InsertPassClaim(ctx, sqlc.InsertPassClaimParams{UserID: user, SeasonID: season, Track: track, Tier: int32(tier), Now: now}) //nolint:gosec // G115: ≤ 40
	if err != nil || n == 0 {
		return err
	}
	key := "passr:" + season.String() + ":" + track + ":" + strconv.Itoa(tier)
	volt := r.Volt
	if r.Shield {
		_, err := q.AddShield(ctx, sqlc.AddShieldParams{UserID: user, Now: now})
		switch {
		case errors.Is(err, pgx.ErrNoRows): // two held: Volt instead
			volt += passShieldVo
		case err != nil:
			return err
		default:
			out.Shields++
		}
	}
	if volt > 0 && s.d.Wallet != nil {
		if err := s.d.Wallet.CreditTx(ctx, tx, user, volt, reasonPassReward, strconv.Itoa(tier), key, now); err != nil {
			return err
		}
		out.Volt += volt
	}
	if r.Cosmetic != "" && s.d.Wallet != nil {
		if err := s.d.Wallet.GrantCosmeticTx(ctx, tx, user, r.Cosmetic, now); err != nil {
			return err
		}
		if !slices.Contains(out.Cosmetics, r.Cosmetic) {
			out.Cosmetics = append(out.Cosmetics, r.Cosmetic)
		}
	}
	return nil
}
