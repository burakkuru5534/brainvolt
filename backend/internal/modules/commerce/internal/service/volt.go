package service

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Volt ledger reasons (D-105).
const (
	VoltQuest       = "quest"
	VoltDaily       = "daily_challenge"
	VoltLevelUp     = "level_up"
	VoltTournament  = "tournament"
	VoltSeason      = "season"
	VoltAd          = "ad"
	VoltPlusMonthly = "plus_monthly"
	VoltPlusRefund  = "plus_refund"
	VoltItem        = "item"
)

// Volt amounts and limits (D-105).
const (
	// PlacementVolt is the store's rewarded ad: AdVolt Volt, AdVoltPerDay a
	// day, outside the other placements' daily limit.
	PlacementVolt = "volt"
	AdVolt        = 5
	AdVoltPerDay  = 3
	// PlusMonthlyVolt is Plus's Volt per calendar month (not in a trial).
	PlusMonthlyVolt = 400
	historySize     = 30
)

// Store items bought with Volt (D-105); progression applies them.
const (
	ItemStreakShield = "streak_shield"
	ItemQuestReroll  = "quest_reroll"
)

// StoreItem is a store item and its price in Volt.
type StoreItem struct {
	Code  string `json:"code"`
	Price int    `json:"price"`
}

// Items is the store's catalogue.
var Items = []StoreItem{{Code: ItemStreakShield, Price: 100}, {Code: ItemQuestReroll, Price: 30}}

// ItemApplier gives a bought item its effect inside the purchase's
// transaction, or refuses it with ITEM_UNAVAILABLE (progression implements
// it; commerce → progression*).
type ItemApplier interface {
	ApplyItemTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, item, ref string, now time.Time) error
}

// SetItems wires the item applier (internal/app, after progression).
func (s *Service) SetItems(a ItemApplier) { s.d.Items = a }

// credit writes one ledger movement once per (user, key) and moves the
// balance; added is false for a repeated key.
func credit(ctx context.Context, q *sqlc.Queries, user uuid.UUID, amount int, reason, ref, key string, now time.Time) (added bool, balance int64, err error) {
	bal, err := q.LockWallet(ctx, sqlc.LockWalletParams{UserID: user, Now: now})
	if err != nil || amount == 0 {
		return false, bal, err
	}
	next := bal + int64(amount)
	n, err := q.InsertVolt(ctx, sqlc.InsertVoltParams{
		ID: ids.New(), UserID: user, Amount: int32(amount), Reason: reason, Ref: ref, Key: key, BalanceAfter: next, Now: now, //nolint:gosec // G115: small amounts
	})
	if err != nil || n == 0 {
		return false, bal, err
	}
	return true, next, q.SetBalance(ctx, sqlc.SetBalanceParams{UserID: user, Balance: next, Now: now})
}

// CreditTx adds amount Volt (negative takes it back) to user inside tx,
// once per key (progression's rewards, D-105).
func CreditTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, amount int, reason, ref, key string, now time.Time) error {
	_, _, err := credit(ctx, sqlc.New(tx), user, amount, reason, ref, key, now)
	return err
}

// VoltEntry is one ledger movement.
type VoltEntry struct {
	Amount    int       `json:"amount"`
	Reason    string    `json:"reason"`
	Ref       string    `json:"ref"`
	CreatedAt time.Time `json:"created_at"`
}

// Wallet is GET /v1/me/wallet.
type Wallet struct {
	Balance int64       `json:"balance"`
	Items   []StoreItem `json:"items"`
	History []VoltEntry `json:"history"`
	// AdVoltLeft is how many store ads can still be watched today.
	AdVoltLeft int `json:"ad_volt_left"`
	AdVolt     int `json:"ad_volt"`
	// Packs are the Volt packs for sale (D-107); prices come from the
	// store. A once-only pack already bought is left out.
	Packs []VoltPack `json:"packs"`
}

func month(t time.Time) string { return t.In(istanbul).Format("2006-01") }

// plusMonthly gives a paying Plus subscriber the month's Volt once.
func (s *Service) plusMonthly(ctx context.Context, user uuid.UUID, now time.Time) error {
	paid, err := sqlc.New(s.d.DB).PaidEntitlement(ctx, sqlc.PaidEntitlementParams{UserID: user, Code: EntitlementPlus, Now: now})
	if err != nil || !paid {
		return err
	}
	m := month(now)
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		_, _, err := credit(ctx, sqlc.New(tx), user, PlusMonthlyVolt, VoltPlusMonthly, m, "plus:"+m, now)
		return err
	})
}

// Wallet returns user's balance, the store's items and the latest
// movements, after giving Plus's Volt of the month.
func (s *Service) Wallet(ctx context.Context, user uuid.UUID) (Wallet, error) {
	now := s.d.Clock.Now()
	if err := s.plusMonthly(ctx, user, now); err != nil {
		return Wallet{}, err
	}
	q := sqlc.New(s.d.DB)
	out := Wallet{Items: Items, History: []VoltEntry{}, AdVolt: AdVolt}
	bal, err := q.GetBalance(ctx, user)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out.Balance = bal
	rows, err := q.VoltHistory(ctx, sqlc.VoltHistoryParams{UserID: user, Lim: historySize})
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		out.History = append(out.History, VoltEntry{Amount: int(r.Amount), Reason: r.Reason, Ref: r.Ref, CreatedAt: r.CreatedAt})
	}
	used, err := s.count(ctx, q, user, PlacementVolt, dayStart(now), now)
	if err != nil {
		return out, err
	}
	out.AdVoltLeft = max(AdVoltPerDay-used, 0)
	out.Packs = []VoltPack{}
	for _, p := range VoltPacks {
		if p.Once && p.Cosmetic != "" {
			owned, err := q.OwnsCosmetic(ctx, sqlc.OwnsCosmeticParams{UserID: user, Code: p.Cosmetic})
			if err != nil {
				return out, err
			}
			if owned {
				continue
			}
		}
		out.Packs = append(out.Packs, p)
	}
	return out, nil
}

// Purchase is the answer to a store purchase.
type Purchase struct {
	Item    string `json:"item"`
	Price   int    `json:"price"`
	Balance int64  `json:"balance"`
}

// Buy spends an item's price and gives its effect in one transaction:
// progression applies the item first (and may refuse it), then the
// balance is checked and charged; a refusal or a short balance undoes
// both.
func (s *Service) Buy(ctx context.Context, user uuid.UUID, code string) (Purchase, error) {
	i := slices.IndexFunc(Items, func(it StoreItem) bool { return it.Code == code })
	if i < 0 {
		return Purchase{}, apierror.New(apierror.NotFound)
	}
	item := Items[i]
	if s.d.Items == nil {
		return Purchase{}, apierror.New(apierror.ItemUnavailable).With("item", code).With("reason", "store_off")
	}
	now := s.d.Clock.Now()
	if err := s.plusMonthly(ctx, user, now); err != nil {
		return Purchase{}, err
	}
	out := Purchase{Item: code, Price: item.Price}
	err := postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		ref := ids.NewString()
		// progression locks the user first; the wallet is locked after it,
		// in the same order as progression's rewards.
		if err := s.d.Items.ApplyItemTx(ctx, tx, user, code, ref, now); err != nil {
			return err
		}
		q := sqlc.New(tx)
		bal, err := q.LockWallet(ctx, sqlc.LockWalletParams{UserID: user, Now: now})
		if err != nil {
			return err
		}
		if bal < int64(item.Price) {
			return apierror.New(apierror.VoltInsufficient).With("price", item.Price).With("balance", bal)
		}
		_, out.Balance, err = credit(ctx, q, user, -item.Price, VoltItem, code, "item:"+ref, now)
		return err
	})
	return out, err
}

// refundPlusVolt takes back the monthly Volt a refunded Plus purchase gave
// since start; the balance may go below zero, which blocks spending.
func refundPlusVolt(ctx context.Context, q *sqlc.Queries, user uuid.UUID, start, now time.Time) error {
	rows, err := q.VoltSince(ctx, sqlc.VoltSinceParams{UserID: user, Reason: VoltPlusMonthly, Since: start})
	if err != nil {
		return err
	}
	for _, r := range rows {
		if _, _, err := credit(ctx, q, user, -int(r.Amount), VoltPlusRefund, r.Key, "refund:"+r.Key, now); err != nil {
			return err
		}
	}
	return nil
}

// DebitTx spends amount Volt of user inside tx, once per key; a short
// balance refuses with VOLT_INSUFFICIENT (the Season Pass, D-108).
func DebitTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, amount int, reason, ref, key string, now time.Time) error {
	q := sqlc.New(tx)
	bal, err := q.LockWallet(ctx, sqlc.LockWalletParams{UserID: user, Now: now})
	if err != nil {
		return err
	}
	if bal < int64(amount) {
		return apierror.New(apierror.VoltInsufficient).With("price", amount).With("balance", bal)
	}
	_, _, err = credit(ctx, q, user, -amount, reason, ref, key, now)
	return err
}

// GrantCosmeticTx adds a cosmetic to user's inventory inside tx (a Season
// Pass reward, D-108); an owned one is left as it is.
func GrantCosmeticTx(ctx context.Context, tx pgx.Tx, user uuid.UUID, code string, now time.Time) error {
	_, err := sqlc.New(tx).InsertCosmetic(ctx, sqlc.InsertCosmeticParams{UserID: user, Code: code, Source: "reward", Now: now})
	return err
}
