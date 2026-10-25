// Package transport is commerce's HTTP layer (rest.md §11): ad settings,
// rewarded-ad reservations and AdMob's SSV callback (D-103), entitlements
// and RevenueCat's webhook (D-104), Volt and the store (D-105).
package transport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/commerce/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
)

// Transport wires commerce's handlers.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	// devGrant enables POST /v1/ads/rewards/{id}/dev-grant (never in prod).
	devGrant bool
	// webhookSecret is RevenueCat's Authorization header value (D-104).
	webhookSecret string
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, devGrant bool, webhookSecret string) *Transport {
	return &Transport{svc: svc, ident: ident, devGrant: devGrant, webhookSecret: webhookSecret}
}

// RewardRequest is the body of POST /v1/ads/rewards.
type RewardRequest struct {
	Placement string `json:"placement"`
	// Ref is the match of a bonus_xp reward.
	Ref string `json:"ref,omitempty"`
}

var placements = []string{service.PlacementBonusXP, service.PlacementQuestReroll, service.PlacementStreakRepair, service.PlacementVolt}

// Register adds commerce's routes.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "commerce"; return httpserver.WithDoc(d) }
	s.Handle("GET /v1/ads/config", a(t.config), doc(httpserver.Doc{
		Auth: "any", Summary: "Ad settings for the caller: interstitial on/off and frequency, the day's rewarded-ad allowance, a repairable streak (D-103)",
		Response: service.AdsConfig{},
	}))
	s.Handle("POST /v1/ads/rewards", a(t.reserve), doc(httpserver.Doc{
		Auth: "any", Summary: "Reserve a rewarded ad (bonus_xp after a match, quest_reroll, streak_repair, volt in the store); its id goes to AdMob as custom_data",
		Request: RewardRequest{}, Response: service.Reward{}, Status: http.StatusCreated,
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.AdRewardUnavailable, apierror.AdLimitReached},
	}))
	s.Handle("GET /v1/ads/rewards/{id}", a(t.get), doc(httpserver.Doc{
		Auth: "any", Summary: "A reserved rewarded ad: pending until AdMob's verified callback, then granted",
		Response: service.Reward{}, Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("GET /v1/ads/admob/ssv", t.ssv, doc(httpserver.Doc{
		Auth: "public", Summary: "AdMob server-side verification callback (signed by AdMob; grants the reward named by custom_data)",
		Errors: []apierror.Code{apierror.Forbidden},
	}))
	s.Handle("GET /v1/me/entitlements", a(t.entitlements), doc(httpserver.Doc{
		Auth: "any", Summary: "The caller's active entitlements: plus (BrainVolt Plus, D-104) and its end",
		Response: service.Entitlements{},
	}))
	s.Handle("POST /v1/store/revenuecat/webhook", t.revenueCat, doc(httpserver.Doc{
		Auth: "public", Summary: "RevenueCat webhook (Authorization = the configured secret): subscription events → the plus entitlement, idempotent by event id",
		Errors: []apierror.Code{apierror.Unauthenticated, apierror.Unavailable},
	}))
	s.Handle("GET /v1/me/wallet", a(t.wallet), doc(httpserver.Doc{
		Auth: "any", Summary: "The caller's Volt (D-105): balance, store items, the latest movements and today's store ads left",
		Response: service.Wallet{},
	}))
	s.Handle("POST /v1/store/items/{code}/buy", a(t.buy), doc(httpserver.Doc{
		Auth: "any", Summary: "Buy a store item with Volt (streak_shield, quest_reroll); the item's effect and the charge happen together",
		Response: service.Purchase{}, Errors: []apierror.Code{apierror.NotFound, apierror.VoltInsufficient, apierror.ItemUnavailable},
	}))
	s.Handle("GET /v1/store/cosmetics", a(t.cosmetics), doc(httpserver.Doc{
		Auth: "any", Summary: "The cosmetics catalogue (D-106) with what the caller owns and wears; Plus-only items are owned while Plus lasts",
		Response: service.MyCosmetics{},
	}))
	s.Handle("POST /v1/store/cosmetics/{code}/buy", a(t.buyCosmetic), doc(httpserver.Doc{
		Auth: "any", Summary: "Buy a cosmetic with Volt, once (special avatars are then set through PATCH /v1/me/profile)",
		Response: service.CosmeticPurchase{}, Errors: []apierror.Code{apierror.NotFound, apierror.VoltInsufficient, apierror.ItemUnavailable},
	}))
	s.Handle("PUT /v1/me/cosmetics/{kind}", a(t.equip), doc(httpserver.Doc{
		Auth: "any", Summary: "Wear a cosmetic of a kind (frame, name_color, victory, match_theme); code null takes it off",
		Request: EquipRequest{}, Response: service.MyCosmetics{},
		Errors: []apierror.Code{apierror.NotFound, apierror.ValidationFailed, apierror.ItemUnavailable},
	}))
	if t.devGrant {
		s.Handle("POST /v1/ads/rewards/{id}/dev-grant", a(t.devGrantReward), doc(httpserver.Doc{
			Auth: "any", Summary: "Development only: grant a reserved reward without AdMob (test ads cannot call a local server)",
			Response: service.Reward{}, Errors: []apierror.Code{apierror.NotFound},
		}))
	}
}

func principal(r *http.Request) identity.Principal {
	p, _ := identity.PrincipalFrom(r.Context())
	return p
}

func (t *Transport) config(w http.ResponseWriter, r *http.Request) error {
	c, err := t.svc.Config(r.Context(), principal(r).UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) reserve(w http.ResponseWriter, r *http.Request) error {
	var req RewardRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	var f apierror.Fields
	if !slices.Contains(placements, req.Placement) {
		f.Add("placement", "enum", "allowed", placements)
	}
	if req.Placement == service.PlacementBonusXP {
		if _, err := uuid.Parse(req.Ref); err != nil {
			f.Add("ref", "uuid")
		}
	} else {
		req.Ref = ""
	}
	if err := f.Err(); err != nil {
		return err
	}
	out, err := t.svc.Reserve(r.Context(), principal(r).UserID, req.Placement, req.Ref)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, out)
}

func (t *Transport) get(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apierror.New(apierror.NotFound)
	}
	out, err := t.svc.Get(r.Context(), principal(r).UserID, id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) devGrantReward(w http.ResponseWriter, r *http.Request) error {
	user := principal(r).UserID
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apierror.New(apierror.NotFound)
	}
	ok, err := t.svc.Grant(r.Context(), id, user, "dev-"+id.String())
	if err != nil {
		return err
	}
	if !ok {
		return apierror.New(apierror.NotFound)
	}
	out, err := t.svc.Get(r.Context(), user, id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) entitlements(w http.ResponseWriter, r *http.Request) error {
	out, err := t.svc.MyEntitlements(r.Context(), principal(r).UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) wallet(w http.ResponseWriter, r *http.Request) error {
	out, err := t.svc.Wallet(r.Context(), principal(r).UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) buy(w http.ResponseWriter, r *http.Request) error {
	out, err := t.svc.Buy(r.Context(), principal(r).UserID, r.PathValue("code"))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

// EquipRequest is the body of PUT /v1/me/cosmetics/{kind}.
type EquipRequest struct {
	// Code is the cosmetic to wear; null or empty takes the kind off.
	Code *string `json:"code"`
}

func (t *Transport) cosmetics(w http.ResponseWriter, r *http.Request) error {
	out, err := t.svc.MyCosmetics(r.Context(), principal(r).UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) buyCosmetic(w http.ResponseWriter, r *http.Request) error {
	out, err := t.svc.BuyCosmetic(r.Context(), principal(r).UserID, r.PathValue("code"))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) equip(w http.ResponseWriter, r *http.Request) error {
	var req EquipRequest
	if err := httpserver.DecodeJSON(r, &req); err != nil {
		return err
	}
	code := ""
	if req.Code != nil {
		code = *req.Code
	}
	out, err := t.svc.Equip(r.Context(), principal(r).UserID, r.PathValue("kind"), code)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, out)
}

func (t *Transport) revenueCat(w http.ResponseWriter, r *http.Request) error {
	switch err := service.CheckWebhookAuth(t.webhookSecret, r.Header.Get("Authorization")); {
	case errors.Is(err, service.ErrWebhookOff):
		return apierror.New(apierror.Unavailable)
	case err != nil:
		return apierror.New(apierror.Unauthenticated)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return apierror.New(apierror.MalformedRequest)
	}
	if err := t.svc.HandleRevenueCat(r.Context(), body); err != nil {
		var syn *json.SyntaxError
		if errors.As(err, &syn) {
			return apierror.New(apierror.MalformedRequest)
		}
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (t *Transport) ssv(w http.ResponseWriter, r *http.Request) error {
	err := t.svc.HandleSSV(r.Context(), r.URL.RawQuery)
	if errors.Is(err, service.ErrBadSignature) {
		return apierror.New(apierror.Forbidden)
	}
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}
