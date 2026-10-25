package service

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/store/sqlc"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/postgres"
)

// Plus (D-104).
const (
	// EntitlementPlus is BrainVolt Plus, the RevenueCat entitlement of the
	// same identifier.
	EntitlementPlus = "plus"
	// PlusProductPrefix: Play subscription brainvolt_plus (base plans
	// monthly / yearly; RevenueCat names them brainvolt_plus:monthly …).
	PlusProductPrefix = "brainvolt_plus"
	storeRevenueCat   = "revenuecat"
)

// ErrBadWebhookAuth: the webhook's Authorization header is wrong.
var ErrBadWebhookAuth = errors.New("commerce: bad webhook authorization")

// ErrWebhookOff: no webhook secret is configured.
var ErrWebhookOff = errors.New("commerce: webhook not configured")

// CheckWebhookAuth compares the Authorization header with the configured
// secret (RevenueCat sends the value set in its dashboard; "Bearer x" and
// "x" are both accepted).
func CheckWebhookAuth(secret, header string) error {
	if secret == "" {
		return ErrWebhookOff
	}
	got := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(header), "Bearer "))
	if subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
		return ErrBadWebhookAuth
	}
	return nil
}

// rcEvent is the part of a RevenueCat webhook event BrainVolt uses
// (docs: revenuecat.com/docs/integrations/webhooks/event-types-and-fields).
type rcEvent struct {
	ID               string   `json:"id"`
	Type             string   `json:"type"`
	AppUserID        string   `json:"app_user_id"`
	OriginalAppUser  string   `json:"original_app_user_id"`
	Aliases          []string `json:"aliases"`
	ProductID        string   `json:"product_id"`
	EntitlementIDs   []string `json:"entitlement_ids"`
	ExpirationAtMs   *int64   `json:"expiration_at_ms"`
	PurchasedAtMs    *int64   `json:"purchased_at_ms"`
	EventTimestampMs int64    `json:"event_timestamp_ms"`
	OriginalTxID     string   `json:"original_transaction_id"`
	TransactionID    string   `json:"transaction_id"`
	CancelReason     string   `json:"cancel_reason"`
	Environment      string   `json:"environment"`
	TransferredFrom  []string `json:"transferred_from"`
	TransferredTo    []string `json:"transferred_to"`
}

// user resolves the BrainVolt user: the app logs in to RevenueCat with
// our user id, so it is the app user id or one of its aliases.
func (e *rcEvent) user() (uuid.UUID, bool) {
	for _, c := range append([]string{e.AppUserID, e.OriginalAppUser}, e.Aliases...) {
		if id, err := uuid.Parse(c); err == nil && id != uuid.Nil {
			return id, true
		}
	}
	return uuid.Nil, false
}

func usersOf(list []string) []uuid.UUID {
	var out []uuid.UUID
	for _, c := range list {
		if id, err := uuid.Parse(c); err == nil && id != uuid.Nil && !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

func (e *rcEvent) plus() bool {
	return slices.Contains(e.EntitlementIDs, EntitlementPlus) || strings.HasPrefix(e.ProductID, PlusProductPrefix)
}

func ms(v *int64) *time.Time {
	if v == nil {
		return nil
	}
	t := time.UnixMilli(*v).UTC()
	return &t
}

// HandleRevenueCat records a webhook event once and applies it to the
// Plus entitlement: purchases, renewals and un-cancellations extend it to
// the event's expiration; an expiration ends it there; a refund
// (cancellation by customer support) ends it now; a transfer moves it.
// A cancellation keeps access until the period ends. Other products (Volt
// packs, a later phase) are only recorded.
func (s *Service) HandleRevenueCat(ctx context.Context, body []byte) error {
	var wrap struct {
		Event rcEvent `json:"event"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return err
	}
	e := wrap.Event
	if e.ID == "" || e.Type == "" {
		return errors.New("commerce: webhook without event id or type")
	}
	now := s.d.Clock.Now()
	user, known := e.user()
	var uid *uuid.UUID
	if known {
		uid = &user
	}
	return postgres.InTx(ctx, s.d.DB, func(tx pgx.Tx) error {
		q := sqlc.New(tx)
		n, err := q.InsertStoreEvent(ctx, sqlc.InsertStoreEventParams{
			ID: ids.New(), Store: storeRevenueCat, EventID: e.ID, Kind: e.Type, UserID: uid, Payload: body, Now: now,
		})
		if err != nil || n == 0 {
			return err // a redelivery
		}
		if err := s.applyRevenueCat(ctx, q, &e, user, known, now); err != nil {
			return err
		}
		return q.MarkStoreEventProcessed(ctx, sqlc.MarkStoreEventProcessedParams{Store: storeRevenueCat, EventID: e.ID, Now: &now})
	})
}

func (s *Service) applyRevenueCat(ctx context.Context, q *sqlc.Queries, e *rcEvent, user uuid.UUID, known bool, now time.Time) error {
	if e.Type == "TRANSFER" {
		return transferPlus(ctx, q, usersOf(e.TransferredFrom), usersOf(e.TransferredTo), now)
	}
	if !known || !e.plus() {
		if !known {
			s.d.Logger.WarnContext(ctx, "commerce: store event for an unknown user", "event", e.ID, "type", e.Type)
		}
		return nil
	}
	source := e.OriginalTxID
	if source == "" {
		source = e.ProductID
	}
	exp := ms(e.ExpirationAtMs)
	start := now
	if p := ms(e.PurchasedAtMs); p != nil && p.Before(now) {
		start = *p
	}
	switch e.Type {
	case "INITIAL_PURCHASE", "RENEWAL", "PRODUCT_CHANGE", "UNCANCELLATION", "SUBSCRIPTION_EXTENDED",
		"TEMPORARY_ENTITLEMENT_GRANT", "BILLING_ISSUE", "SUBSCRIPTION_PAUSED":
		if exp == nil || !exp.After(now) {
			return nil
		}
		return q.ExtendEntitlement(ctx, sqlc.ExtendEntitlementParams{
			UserID: user, Code: EntitlementPlus, SourceKind: "subscription", SourceID: source, StartsAt: start, EndsAt: exp,
		})
	case "CANCELLATION":
		if e.CancelReason != "CUSTOMER_SUPPORT" {
			return nil // access lasts until the period ends
		}
		return q.EndEntitlement(ctx, sqlc.EndEntitlementParams{UserID: user, Code: EntitlementPlus, SourceKind: "subscription", SourceID: source, EndsAt: &now})
	case "EXPIRATION":
		end := now
		if exp != nil && exp.Before(now) {
			end = *exp
		}
		return q.EndEntitlement(ctx, sqlc.EndEntitlementParams{UserID: user, Code: EntitlementPlus, SourceKind: "subscription", SourceID: source, EndsAt: &end})
	}
	return nil
}

// transferPlus moves running subscription entitlements from one set of
// users to another (a purchase restored on a different account).
func transferPlus(ctx context.Context, q *sqlc.Queries, from, to []uuid.UUID, now time.Time) error {
	for _, f := range from {
		rows, err := q.SubscriptionEntitlements(ctx, f)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if r.EndsAt != nil && !r.EndsAt.After(now) {
				continue
			}
			for _, t := range to {
				if err := q.ExtendEntitlement(ctx, sqlc.ExtendEntitlementParams{
					UserID: t, Code: r.Code, SourceKind: r.SourceKind, SourceID: r.SourceID, StartsAt: now, EndsAt: r.EndsAt,
				}); err != nil {
					return err
				}
			}
			if err := q.EndEntitlement(ctx, sqlc.EndEntitlementParams{UserID: f, Code: r.Code, SourceKind: r.SourceKind, SourceID: r.SourceID, EndsAt: &now}); err != nil {
				return err
			}
		}
	}
	return nil
}

// Entitlement is one active entitlement of GET /v1/me/entitlements.
type Entitlement struct {
	Code string `json:"code"`
	// EndsAt is null for an entitlement without end.
	EndsAt *time.Time `json:"ends_at"`
}

// Entitlements is GET /v1/me/entitlements.
type Entitlements struct {
	Plus         bool          `json:"plus"`
	PlusUntil    *time.Time    `json:"plus_until"`
	Entitlements []Entitlement `json:"entitlements"`
}

// MyEntitlements returns user's active entitlements.
func (s *Service) MyEntitlements(ctx context.Context, user uuid.UUID) (Entitlements, error) {
	rows, err := sqlc.New(s.d.DB).ActiveEntitlements(ctx, sqlc.ActiveEntitlementsParams{UserID: user, Now: s.d.Clock.Now()})
	if err != nil {
		return Entitlements{}, err
	}
	out := Entitlements{Entitlements: []Entitlement{}}
	for _, r := range rows {
		ent := Entitlement{Code: r.Code}
		if !r.Forever {
			t := r.EndsAt
			ent.EndsAt = &t
		}
		out.Entitlements = append(out.Entitlements, ent)
		if r.Code == EntitlementPlus {
			out.Plus, out.PlusUntil = true, ent.EndsAt
		}
	}
	return out, nil
}
