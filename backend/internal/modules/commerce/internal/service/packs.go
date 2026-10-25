package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/store/sqlc"
)

// Volt ledger reasons of packs (D-107).
const (
	VoltPurchase       = "purchase"
	VoltPurchaseRefund = "purchase_refund"
)

// VoltPack is a Google Play one-time product that gives Volt (D-107).
type VoltPack struct {
	ProductID string `json:"product_id"`
	Volt      int    `json:"volt"`
	// Bonus is the part of Volt given on top (shown as "+50 gift").
	Bonus int `json:"bonus,omitempty"`
	// Cosmetic comes with the pack (the starter frame).
	Cosmetic string `json:"cosmetic,omitempty"`
	// Once: the starter pack is sold once per account.
	Once bool `json:"once,omitempty"`
}

// VoltPacks are the packs in display order (prices come from the store).
var VoltPacks = []VoltPack{
	{ProductID: "volt_starter", Volt: 250, Cosmetic: "frame_starter", Once: true},
	{ProductID: "volt_100", Volt: 100},
	{ProductID: "volt_350", Volt: 350, Bonus: 50},
	{ProductID: "volt_800", Volt: 800},
	{ProductID: "volt_2000", Volt: 2000},
}

// voltPack finds the pack of a store product id ("volt_100" or a
// "volt_100:…" variant).
func voltPack(product string) (VoltPack, bool) {
	id, _, _ := strings.Cut(product, ":")
	for _, p := range VoltPacks {
		if p.ProductID == id {
			return p, true
		}
	}
	return VoltPack{}, false
}

// applyPack gives (or, on a refund, takes back) a Volt pack bought through
// RevenueCat, once per store transaction. handled is false for events that
// are not about a pack.
func applyPack(ctx context.Context, q *sqlc.Queries, e *rcEvent, user uuid.UUID, now time.Time) (handled bool, err error) {
	pack, ok := voltPack(e.ProductID)
	if !ok {
		return false, nil
	}
	tx := e.TransactionID
	if tx == "" {
		tx = e.OriginalTxID
	}
	if tx == "" {
		tx = e.ID
	}
	key := "iap:" + tx
	switch {
	case e.Type == "NON_RENEWING_PURCHASE" || e.Type == "INITIAL_PURCHASE":
		if _, _, err := credit(ctx, q, user, pack.Volt, VoltPurchase, pack.ProductID, key, now); err != nil {
			return true, err
		}
		if pack.Cosmetic != "" {
			if _, err := q.InsertCosmetic(ctx, sqlc.InsertCosmeticParams{UserID: user, Code: pack.Cosmetic, Source: "purchase", Now: now}); err != nil {
				return true, err
			}
		}
	case e.Type == "CANCELLATION" && e.CancelReason == "CUSTOMER_SUPPORT":
		paid, err := q.VoltKeyExists(ctx, sqlc.VoltKeyExistsParams{UserID: user, Key: key})
		if err != nil || !paid {
			return true, err
		}
		if _, _, err := credit(ctx, q, user, -pack.Volt, VoltPurchaseRefund, pack.ProductID, "iap-refund:"+tx, now); err != nil {
			return true, err
		}
		if pack.Cosmetic != "" {
			if err := q.UnequipCode(ctx, sqlc.UnequipCodeParams{UserID: user, Code: pack.Cosmetic}); err != nil {
				return true, err
			}
			if err := q.RemoveCosmetic(ctx, sqlc.RemoveCosmeticParams{UserID: user, Code: pack.Cosmetic}); err != nil {
				return true, err
			}
		}
	}
	return true, nil
}
