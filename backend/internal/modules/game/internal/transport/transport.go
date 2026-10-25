// Package transport exposes game's REST endpoints (rest.md §4).
package transport

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/burakkuru5534/brainvolt/backend/internal/modules/game/internal/service"
	"github.com/burakkuru5534/brainvolt/backend/internal/modules/identity"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/i18n"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/idempotency"
)

// Transport serves game's routes.
type Transport struct {
	svc   *service.Service
	ident *identity.Module
	idem  *idempotency.Store
}

// New returns the transport.
func New(svc *service.Service, ident *identity.Module, idem *idempotency.Store) *Transport {
	return &Transport{svc: svc, ident: ident, idem: idem}
}

// DailyStart is the body of POST /v1/daily/runs.
type DailyStart struct {
	Kind string `json:"kind"`
}

// Register adds game's routes.
func (t *Transport) Register(s *httpserver.Server) {
	s.Handle("GET /v1/matches/current", t.ident.Auth(t.current), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "any",
		Summary:  "The caller's live match, if any, with the WebSocket stream to resume",
		Response: service.Current{},
	}))
	s.Handle("GET /v1/matches/{id}", t.ident.Auth(t.match), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "any",
		Summary:  "Result screen data for a participant: sides, participants and, once over, every round with answers",
		Response: service.MatchView{},
		Errors:   []apierror.Code{apierror.MatchNotFound, apierror.NotAParticipant},
	}))
	s.Handle("GET /v1/daily/today", t.ident.Auth(t.dailyToday), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "any",
		Summary:  "Today's Daily Challenge (Europe/Istanbul day): set available, the caller's official run, practice allowed",
		Response: service.DailyToday{},
	}))
	s.Handle("POST /v1/daily/runs", t.ident.Auth(t.idem.Wrap(idempotency.Required, t.dailyStart)), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "any", Idempotency: true, Status: http.StatusCreated,
		Summary: "Start a daily run (official once a day, practice after it); play it with daily.join / daily.answer",
		Request: DailyStart{}, Response: service.DailyRunView{},
		Errors: []apierror.Code{apierror.ValidationFailed, apierror.DailyAttemptUsed, apierror.DailyAttemptInProgress, apierror.DailyNotAvailable,
			apierror.IdempotencyKeyReused, apierror.IdempotencyInProgress},
	}))
	s.Handle("GET /v1/daily/answers", t.ident.Auth(t.dailyAnswers), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "any",
		Summary: "A past day's Daily Challenge answers (?day=YYYY-MM-DD, default yesterday, up to 7 days back) with the caller's own; " +
			"today's are never served (D-100)",
		Response: service.DailyAnswers{}, Errors: []apierror.Code{apierror.ValidationFailed, apierror.DailyAnswersNotOpen, apierror.NotFound},
	}))
	s.Handle("GET /v1/daily/runs/{id}", t.ident.Auth(t.dailyRun), httpserver.WithDoc(httpserver.Doc{
		Tag: "game", Auth: "owner", Summary: "A daily run with its answers once completed",
		Response: service.DailyRunView{}, Errors: []apierror.Code{apierror.NotFound},
	}))
}

func (t *Transport) dailyToday(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	v, err := t.svc.DailyToday(r.Context(), p.UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, v)
}

func (t *Transport) dailyStart(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	var in DailyStart
	if err := httpserver.DecodeJSON(r, &in); err != nil {
		return err
	}
	v, err := t.svc.StartDaily(r.Context(), p.UserID, in.Kind, string(i18n.FromContext(r.Context())))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusCreated, v)
}

func (t *Transport) dailyAnswers(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	var day *time.Time
	if v := r.URL.Query().Get("day"); v != "" {
		d, err := time.Parse(time.DateOnly, v)
		if err != nil {
			return apierror.New(apierror.ValidationFailed).With("field", "day")
		}
		day = &d
	}
	v, err := t.svc.DailyAnswers(r.Context(), p.UserID, day, string(i18n.FromContext(r.Context())))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, v)
}

func (t *Transport) dailyRun(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apierror.New(apierror.NotFound)
	}
	v, err := t.svc.DailyRun(r.Context(), p.UserID, id)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, v)
}

func (t *Transport) current(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	c, err := t.svc.Current(r.Context(), p.UserID)
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, c)
}

func (t *Transport) match(w http.ResponseWriter, r *http.Request) error {
	p, _ := identity.PrincipalFrom(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return apierror.New(apierror.MatchNotFound)
	}
	v, err := t.svc.Match(r.Context(), p.UserID, id, string(i18n.FromContext(r.Context())))
	if err != nil {
		return err
	}
	return httpserver.WriteJSON(w, http.StatusOK, v)
}
