// Package transport is commerce's HTTP layer (rest.md §11, D-103): the
// ad settings, rewarded-ad reservations and AdMob's SSV callback.
package transport

import (
	"errors"
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
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, devGrant bool) *Transport {
	return &Transport{svc: svc, ident: ident, devGrant: devGrant}
}

// RewardRequest is the body of POST /v1/ads/rewards.
type RewardRequest struct {
	Placement string `json:"placement"`
	// Ref is the match of a bonus_xp reward.
	Ref string `json:"ref,omitempty"`
}

var placements = []string{service.PlacementBonusXP, service.PlacementQuestReroll, service.PlacementStreakRepair}

// Register adds commerce's routes.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	doc := func(d httpserver.Doc) httpserver.RouteOption { d.Tag = "commerce"; return httpserver.WithDoc(d) }
	s.Handle("GET /v1/ads/config", a(t.config), doc(httpserver.Doc{
		Auth: "any", Summary: "Ad settings for the caller: interstitial on/off and frequency, the day's rewarded-ad allowance, a repairable streak (D-103)",
		Response: service.AdsConfig{},
	}))
	s.Handle("POST /v1/ads/rewards", a(t.reserve), doc(httpserver.Doc{
		Auth: "any", Summary: "Reserve a rewarded ad (bonus_xp after a match, quest_reroll, streak_repair); its id goes to AdMob as custom_data",
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
