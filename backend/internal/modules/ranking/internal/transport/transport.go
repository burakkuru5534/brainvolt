// Package transport exposes ranking's REST endpoints (rest.md §9).
package transport

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/ranking/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
)

// Transport serves ranking's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module) *Transport {
	return &Transport{svc: svc, ident: ident}
}

// Leagues is GET /v1/leagues.
type Leagues struct {
	Leagues []service.League `json:"leagues"`
}

// Register adds ranking's routes; every board is readable by any
// signed-in account.
func (t *Transport) Register(s *httpserver.Server) {
	a := t.ident.Auth
	doc := func(d httpserver.Doc) httpserver.RouteOption {
		d.Tag = "ranking"
		d.Auth = "any"
		return httpserver.WithDoc(d)
	}
	s.Handle("GET /v1/leaderboards", a(t.leaderboard), doc(httpserver.Doc{
		Summary:  "A leaderboard page: period × entity × scope × category (D-029); `me` is the caller's row on user boards",
		Response: service.Board{}, Errors: []apierror.Code{apierror.ValidationFailed, apierror.NotFound},
		Query: []httpserver.QueryParam{
			{Name: "period", Type: "string", Enum: service.BoardPeriods, Description: "default weekly; season = SP board"},
			{Name: "entity", Type: "string", Enum: service.BoardEntities, Description: "default user"},
			{Name: "scope", Type: "string", Enum: service.BoardScopes, Description: "default global; friends only for users"},
			{Name: "country", Type: "string", Description: "ISO country for scope=country (default the caller's)"},
			{Name: "category_id", Type: "string", Format: "uuid", Description: "category board (not with season)"},
			{Name: "season_id", Type: "string", Format: "uuid", Description: "season board of a past season"},
			{Name: "limit", Type: "integer", Description: "1–100, default 50"},
			{Name: "cursor", Type: "string", Description: "next_cursor of the previous page"},
		},
	}))
	s.Handle("GET /v1/leaderboards/daily-challenge/{date}", a(t.dailyBoard), doc(httpserver.Doc{
		Summary:  "A day's Daily Challenge board (correct ↓, total time ↑, completion ↑; D-040); `me` is the caller's place",
		Response: service.DailyBoard{}, Errors: []apierror.Code{apierror.ValidationFailed},
		Query: []httpserver.QueryParam{
			{Name: "scope", Type: "string", Enum: []string{"global", "country", "friends"}, Description: "default global"},
			{Name: "limit", Type: "integer", Description: "1–100, default 50"},
			{Name: "cursor", Type: "string", Description: "next_cursor of the previous page"},
		},
	}))
	s.Handle("GET /v1/seasons/current", a(t.currentSeason), doc(httpserver.Doc{
		Summary: "The active season", Response: service.Season{},
	}))
	s.Handle("GET /v1/seasons/{id}", a(t.season), doc(httpserver.Doc{
		Summary: "A season", Response: service.Season{}, Errors: []apierror.Code{apierror.NotFound},
	}))
	s.Handle("GET /v1/standings/{entity_kind}/{entity_id}", a(t.standing), doc(httpserver.Doc{
		Summary:  "League, division, SP and record of a user, team or couple (current season or season_id)",
		Response: service.Standing{}, Errors: []apierror.Code{apierror.NotFound},
		Query: []httpserver.QueryParam{{Name: "season_id", Type: "string", Format: "uuid"}},
	}))
	s.Handle("GET /v1/leagues", a(t.leagues), doc(httpserver.Doc{
		Summary: "League table with division floors in the caller's language", Response: Leagues{},
	}))
}

func optionalID(r *http.Request, name string) (*uuid.UUID, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return nil, nil //nolint:nilnil // absent
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return nil, apierror.Validation(apierror.FieldError{Field: name, Rule: "format"})
	}
	return &id, nil
}

func or(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func (t *Transport) leaderboard(w http.ResponseWriter, r *http.Request) error {
	qs := r.URL.Query()
	p, _ := identity.PrincipalFrom(r.Context())
	q := service.BoardQuery{
		Viewer: p.UserID, Period: or(qs.Get("period"), "weekly"), Entity: or(qs.Get("entity"), "user"),
		Scope: or(qs.Get("scope"), "global"), Country: strings.ToUpper(qs.Get("country")), Limit: 50,
	}
	var err error
	if q.CategoryID, err = optionalID(r, "category_id"); err != nil {
		return err
	}
	if q.SeasonID, err = optionalID(r, "season_id"); err != nil {
		return err
	}
	if v := qs.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return apierror.Validation(apierror.FieldError{Field: "limit", Rule: "range"})
		}
		q.Limit = n
	}
	if v := qs.Get("cursor"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 10_000 {
			return apierror.Validation(apierror.FieldError{Field: "cursor", Rule: "format"})
		}
		q.Offset = n
	}
	b, err := t.svc.Leaderboard(r.Context(), q)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, b)
}

func (t *Transport) dailyBoard(w http.ResponseWriter, r *http.Request) error {
	day, err := time.Parse(time.DateOnly, r.PathValue("date"))
	if err != nil {
		return apierror.Validation(apierror.FieldError{Field: "date", Rule: "format"})
	}
	qs := r.URL.Query()
	limit, offset := 50, 0
	if v := qs.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return apierror.Validation(apierror.FieldError{Field: "limit", Rule: "range"})
		}
		limit = n
	}
	if v := qs.Get("cursor"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 10_000 {
			return apierror.Validation(apierror.FieldError{Field: "cursor", Rule: "format"})
		}
		offset = n
	}
	p, _ := identity.PrincipalFrom(r.Context())
	b, err := t.svc.DailyBoard(r.Context(), p.UserID, day, or(qs.Get("scope"), "global"), limit, offset)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, b)
}

func (t *Transport) currentSeason(w http.ResponseWriter, r *http.Request) error {
	se, err := t.svc.CurrentSeason(r.Context())
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, se)
}

func (t *Transport) season(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apierror.New(apierror.NotFound)
	}
	se, err := t.svc.SeasonByID(r.Context(), id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, se)
}

func (t *Transport) standing(w http.ResponseWriter, r *http.Request) error {
	id, err := uuid.Parse(r.PathValue("entity_id"))
	if err != nil {
		return apierror.New(apierror.NotFound)
	}
	season, err := optionalID(r, "season_id")
	if err != nil {
		return err
	}
	st, err := t.svc.Standing(r.Context(), r.PathValue("entity_kind"), id, season)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, st)
}

func (t *Transport) leagues(w http.ResponseWriter, r *http.Request) error {
	ls, err := t.svc.Leagues(r.Context(), string(i18n.FromContext(r.Context())))
	if err != nil {
		return err
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Add("Vary", "Accept-Language")
	return httpserver.WriteJSON(w, http.StatusOK, Leagues{Leagues: ls})
}
