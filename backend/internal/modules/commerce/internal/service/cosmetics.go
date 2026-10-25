package service

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Cosmetic kinds (D-106). Frames and name colours are shown to everyone,
// victory animations and match themes only to their owner; special
// avatars are worn through the profile's avatar_id.
const (
	KindFrame      = "frame"
	KindAvatar     = "avatar"
	KindNameColor  = "name_color"
	KindVictory    = "victory"
	KindMatchTheme = "match_theme"

	// VoltCosmetic is the ledger reason of a cosmetic purchase.
	VoltCosmetic = "cosmetic"
	// AvatarPrefix turns an avatar cosmetic into a profile avatar id
	// (avatar_dragon → dragon).
	AvatarPrefix = "avatar_"
)

// Cosmetic is a catalogue entry; PlusOnly items have no price and are worn
// while Plus lasts.
type Cosmetic struct {
	Code     string `json:"code"`
	Kind     string `json:"kind"`
	Price    int    `json:"price"`
	PlusOnly bool   `json:"plus_only"`
}

// Cosmetics is the catalogue (D-106), in display order.
var Cosmetics = []Cosmetic{
	{Code: "frame_bronze", Kind: KindFrame, Price: 200},
	{Code: "frame_neon", Kind: KindFrame, Price: 300},
	{Code: "frame_flame", Kind: KindFrame, Price: 400},
	{Code: "frame_crown", Kind: KindFrame, Price: 600},
	{Code: "frame_plus", Kind: KindFrame, PlusOnly: true},

	{Code: "avatar_octopus", Kind: KindAvatar, Price: 150},
	{Code: "avatar_eagle", Kind: KindAvatar, Price: 150},
	{Code: "avatar_robot", Kind: KindAvatar, Price: 200},
	{Code: "avatar_alien", Kind: KindAvatar, Price: 200},
	{Code: "avatar_dragon", Kind: KindAvatar, Price: 250},
	{Code: "avatar_unicorn", Kind: KindAvatar, Price: 250},

	{Code: "name_gold", Kind: KindNameColor, Price: 300},
	{Code: "name_neon", Kind: KindNameColor, Price: 300},
	{Code: "name_pink", Kind: KindNameColor, Price: 300},
	{Code: "name_ice", Kind: KindNameColor, Price: 300},
	{Code: "name_rainbow", Kind: KindNameColor, PlusOnly: true},

	{Code: "victory_confetti", Kind: KindVictory, Price: 250},
	{Code: "victory_fireworks", Kind: KindVictory, Price: 250},
	{Code: "victory_lightning", Kind: KindVictory, Price: 250},

	{Code: "theme_midnight", Kind: KindMatchTheme, Price: 400},
	{Code: "theme_sunset", Kind: KindMatchTheme, Price: 400},
	{Code: "theme_emerald", Kind: KindMatchTheme, Price: 400},
}

// cosmetic finds a catalogue entry.
func cosmetic(code string) (Cosmetic, bool) {
	i := slices.IndexFunc(Cosmetics, func(c Cosmetic) bool { return c.Code == code })
	if i < 0 {
		return Cosmetic{}, false
	}
	return Cosmetics[i], true
}

// equipKinds are the kinds worn through commerce (avatars go through the
// profile).
var equipKinds = []string{KindFrame, KindNameColor, KindVictory, KindMatchTheme}

// CosmeticItem is a catalogue entry for the caller.
type CosmeticItem struct {
	Cosmetic
	Owned    bool `json:"owned"`
	Equipped bool `json:"equipped"`
}

// MyCosmetics is GET /v1/store/cosmetics.
type MyCosmetics struct {
	Items []CosmeticItem `json:"items"`
	// Equipped is the worn code per kind (frame, name_color, victory,
	// match_theme); Plus-only ones only while Plus lasts.
	Equipped map[string]string `json:"equipped"`
	Plus     bool              `json:"plus"`
}

func unavailable(code, reason string) error {
	return apierror.New(apierror.ItemUnavailable).With("item", code).With("reason", reason)
}

// MyCosmetics returns the catalogue with what user owns and wears.
func (s *Service) MyCosmetics(ctx context.Context, user uuid.UUID) (MyCosmetics, error) {
	q := sqlc.New(s.d.DB)
	plus, err := s.HasEntitlement(ctx, user, EntitlementPlus)
	if err != nil {
		return MyCosmetics{}, err
	}
	owned, err := q.OwnedCosmetics(ctx, user)
	if err != nil {
		return MyCosmetics{}, err
	}
	worn, err := s.equipped(ctx, []uuid.UUID{user})
	if err != nil {
		return MyCosmetics{}, err
	}
	out := MyCosmetics{Items: make([]CosmeticItem, 0, len(Cosmetics)), Equipped: worn[user], Plus: plus}
	if out.Equipped == nil {
		out.Equipped = map[string]string{}
	}
	for _, c := range Cosmetics {
		it := CosmeticItem{Cosmetic: c, Owned: slices.Contains(owned, c.Code) || (c.PlusOnly && plus)}
		it.Equipped = out.Equipped[c.Kind] == c.Code
		out.Items = append(out.Items, it)
	}
	return out, nil
}

// CosmeticPurchase is the answer to a cosmetic purchase.
type CosmeticPurchase struct {
	Code    string `json:"code"`
	Price   int    `json:"price"`
	Balance int64  `json:"balance"`
}

// BuyCosmetic spends a cosmetic's price and adds it to the inventory, once.
func (s *Service) BuyCosmetic(ctx context.Context, user uuid.UUID, code string) (CosmeticPurchase, error) {
	c, ok := cosmetic(code)
	if !ok {
		return CosmeticPurchase{}, apierror.New(apierror.NotFound)
	}
	if c.PlusOnly {
		return CosmeticPurchase{}, unavailable(code, "plus_only")
	}
	now := s.d.Clock.Now()
	if err := s.plusMonthly(ctx, user, now); err != nil {
		return CosmeticPurchase{}, err
	}
	out := CosmeticPurchase{Code: code, Price: c.Price}
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		bal, err := q.LockWallet(ctx, sqlc.LockWalletParams{UserID: user, Now: now})
		if err != nil {
			return err
		}
		n, err := q.InsertCosmetic(ctx, sqlc.InsertCosmeticParams{UserID: user, Code: code, Source: "volt", Now: now})
		if err != nil {
			return err
		}
		if n == 0 {
			return unavailable(code, "already_owned")
		}
		if bal < int64(c.Price) {
			return apierror.New(apierror.VoltInsufficient).With("price", c.Price).With("balance", bal)
		}
		_, out.Balance, err = credit(ctx, q, user, -c.Price, VoltCosmetic, code, "cosmetic:"+code, now)
		return err
	})
	return out, err
}

// Equip wears code in kind ("" takes it off). The code must be of that
// kind and owned, or Plus-only with Plus.
func (s *Service) Equip(ctx context.Context, user uuid.UUID, kind, code string) (MyCosmetics, error) {
	if !slices.Contains(equipKinds, kind) {
		return MyCosmetics{}, apierror.New(apierror.NotFound)
	}
	q := sqlc.New(s.d.DB)
	now := s.d.Clock.Now()
	if code == "" {
		if err := q.UnequipCosmetic(ctx, sqlc.UnequipCosmeticParams{UserID: user, Kind: kind}); err != nil {
			return MyCosmetics{}, err
		}
		return s.MyCosmetics(ctx, user)
	}
	c, ok := cosmetic(code)
	if !ok || c.Kind != kind {
		return MyCosmetics{}, apierror.Validation(apierror.FieldError{Field: "code", Rule: "enum"})
	}
	if err := s.canWear(ctx, q, user, c); err != nil {
		return MyCosmetics{}, err
	}
	if err := q.EquipCosmetic(ctx, sqlc.EquipCosmeticParams{UserID: user, Kind: kind, Code: code, Now: now}); err != nil {
		return MyCosmetics{}, err
	}
	return s.MyCosmetics(ctx, user)
}

func (s *Service) canWear(ctx context.Context, q *sqlc.Queries, user uuid.UUID, c Cosmetic) error {
	if c.PlusOnly {
		plus, err := s.HasEntitlement(ctx, user, EntitlementPlus)
		if err != nil {
			return err
		}
		if !plus {
			return unavailable(c.Code, "plus_only")
		}
		return nil
	}
	owns, err := q.OwnsCosmetic(ctx, sqlc.OwnsCosmeticParams{UserID: user, Code: c.Code})
	if err != nil {
		return err
	}
	if !owns {
		return unavailable(c.Code, "not_owned")
	}
	return nil
}

// OwnsAvatar reports whether user may wear a special avatar (profile asks
// before setting avatar_id; D-106).
func (s *Service) OwnsAvatar(ctx context.Context, user uuid.UUID, avatarID string) (bool, error) {
	return sqlc.New(s.d.DB).OwnsCosmetic(ctx, sqlc.OwnsCosmeticParams{UserID: user, Code: AvatarPrefix + avatarID})
}

// IsSpecialAvatar reports whether avatarID is a cosmetic avatar.
func IsSpecialAvatar(avatarID string) bool {
	c, ok := cosmetic(AvatarPrefix + avatarID)
	return ok && c.Kind == KindAvatar
}

// equipped returns what users wear, per kind, without Plus-only items of
// users whose Plus has ended.
func (s *Service) equipped(ctx context.Context, users []uuid.UUID) (map[uuid.UUID]map[string]string, error) {
	q := sqlc.New(s.d.DB)
	rows, err := q.EquippedOf(ctx, users)
	if err != nil || len(rows) == 0 {
		return map[uuid.UUID]map[string]string{}, err
	}
	var plusNeeded []uuid.UUID
	for _, r := range rows {
		if c, ok := cosmetic(r.Code); ok && c.PlusOnly {
			plusNeeded = append(plusNeeded, r.UserID)
		}
	}
	var plus []uuid.UUID
	if len(plusNeeded) > 0 {
		if plus, err = q.UsersWithEntitlement(ctx, sqlc.UsersWithEntitlementParams{UserIds: plusNeeded, Code: EntitlementPlus, Now: s.d.Clock.Now()}); err != nil {
			return nil, err
		}
	}
	out := map[uuid.UUID]map[string]string{}
	for _, r := range rows {
		c, ok := cosmetic(r.Code)
		if !ok || (c.PlusOnly && !slices.Contains(plus, r.UserID)) {
			continue
		}
		if out[r.UserID] == nil {
			out[r.UserID] = map[string]string{}
		}
		out[r.UserID][r.Kind] = r.Code
	}
	return out, nil
}

// Worn is what others see of a player's cosmetics.
type Worn struct {
	Frame     string
	NameColor string
}

// WornBy returns the public cosmetics (frame, name colour) of users.
func (s *Service) WornBy(ctx context.Context, users []uuid.UUID) (map[uuid.UUID]Worn, error) {
	eq, err := s.equipped(ctx, users)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]Worn, len(eq))
	for u, m := range eq {
		if w := (Worn{Frame: m[KindFrame], NameColor: m[KindNameColor]}); w != (Worn{}) {
			out[u] = w
		}
	}
	return out, nil
}
