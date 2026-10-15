package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
)

var testHTTP = config.HTTP{
	Addr: "127.0.0.1:0", ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second,
	IdleTimeout: 60 * time.Second, HandlerTimeout: 10 * time.Second, ShutdownTimeout: 5 * time.Second,
}

func newTestServer(t *testing.T, env config.Environment) (*Server, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	logger := logging.New(&logs, logging.Options{Format: logging.FormatJSON, Level: slog.LevelDebug})
	return New(testHTTP, env, logger), &logs
}

type body struct {
	Error struct {
		Code      string         `json:"code"`
		Message   string         `json:"message"`
		Details   map[string]any `json:"details"`
		RequestID string         `json:"request_id"`
	} `json:"error"`
}

func do(t *testing.T, s *Server, req *http.Request) (*httptest.ResponseRecorder, body) {
	t.Helper()
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var b body
	if rec.Code >= 400 {
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("error Content-Type = %q", ct)
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
			t.Fatalf("error body is not the envelope: %v\n%s", err, rec.Body.String())
		}
		if b.Error.RequestID == "" || b.Error.RequestID != rec.Header().Get(RequestIDHeader) {
			t.Fatalf("envelope request_id %q != header %q", b.Error.RequestID, rec.Header().Get(RequestIDHeader))
		}
	}
	return rec, b
}

func TestSuccessAndStandardHeaders(t *testing.T) {
	s, _ := newTestServer(t, config.EnvDev)
	s.Handle("GET /v1/ping", func(w http.ResponseWriter, _ *http.Request) error {
		return WriteJSON(w, http.StatusOK, map[string]string{"pong": "ok"})
	})

	rec, _ := do(t, s, httptest.NewRequest(http.MethodGet, "/v1/ping", http.NoBody))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"pong":"ok"}` {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	h := rec.Header()
	if _, err := ids.ParseV7(h.Get(RequestIDHeader)); err != nil {
		t.Errorf("generated request id %q is not a UUIDv7", h.Get(RequestIDHeader))
	}
	if h.Get("X-Content-Type-Options") != "nosniff" || h.Get("Cache-Control") != "no-store" {
		t.Errorf("security headers missing: %v", h)
	}
	if h.Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must not be sent outside prod")
	}

	prod, _ := newTestServer(t, config.EnvProd)
	rec, _ = do(t, prod, httptest.NewRequest(http.MethodGet, "/nothing", http.NoBody))
	if !strings.HasPrefix(rec.Header().Get("Strict-Transport-Security"), "max-age=") {
		t.Error("HSTS missing in prod")
	}
}

func TestClientRequestID(t *testing.T) {
	s, _ := newTestServer(t, config.EnvDev)
	client := ids.NewString()

	req := httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	req.Header.Set(RequestIDHeader, client)
	rec, b := do(t, s, req)
	if rec.Header().Get(RequestIDHeader) != client || b.Error.RequestID != client {
		t.Fatalf("valid client request id not kept: header %q body %q", rec.Header().Get(RequestIDHeader), b.Error.RequestID)
	}

	req = httptest.NewRequest(http.MethodGet, "/missing", http.NoBody)
	req.Header.Set(RequestIDHeader, "<script>")
	rec, _ = do(t, s, req)
	if got := rec.Header().Get(RequestIDHeader); got == "<script>" {
		t.Fatal("invalid client request id was echoed")
	}
}

func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	s, _ := newTestServer(t, config.EnvDev)
	s.Handle("GET /v1/me", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusNoContent)
		return nil
	})

	rec, b := do(t, s, httptest.NewRequest(http.MethodGet, "/v1/nope", http.NoBody))
	if rec.Code != http.StatusNotFound || b.Error.Code != "NOT_FOUND" {
		t.Fatalf("got %d %s", rec.Code, b.Error.Code)
	}
	if b.Error.Message != "Aradığın şey bulunamadı." || rec.Header().Get("Content-Language") != "tr" {
		t.Fatalf("default locale must be tr: %q / %q", b.Error.Message, rec.Header().Get("Content-Language"))
	}

	req := httptest.NewRequest(http.MethodDelete, "/v1/me", http.NoBody)
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	rec, b = do(t, s, req)
	if rec.Code != http.StatusMethodNotAllowed || b.Error.Code != "METHOD_NOT_ALLOWED" {
		t.Fatalf("got %d %s", rec.Code, b.Error.Code)
	}
	if allow := rec.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Fatalf("Allow = %q", allow)
	}
	if b.Error.Message != "This action is not supported at this address." {
		t.Fatalf("EN message = %q", b.Error.Message)
	}
}

func TestHandlerErrors(t *testing.T) {
	s, logs := newTestServer(t, config.EnvDev)
	s.Handle("POST /v1/friends", func(http.ResponseWriter, *http.Request) error {
		return apierror.New(apierror.AccountUpgradeRequired).With("action", "friend_request.create")
	})
	s.Handle("GET /v1/boom", func(http.ResponseWriter, *http.Request) error {
		return errors.New("pq: password authentication failed for user brainvolt")
	})
	s.Handle("GET /v1/limited", func(http.ResponseWriter, *http.Request) error {
		return apierror.New(apierror.RateLimited).WithRetryAfter(1500 * time.Millisecond)
	})
	s.Handle("GET /v1/ws-only", func(http.ResponseWriter, *http.Request) error {
		return apierror.New(apierror.AnswerLate)
	})

	rec, b := do(t, s, httptest.NewRequest(http.MethodPost, "/v1/friends", http.NoBody))
	if rec.Code != http.StatusForbidden || b.Error.Code != "ACCOUNT_UPGRADE_REQUIRED" ||
		b.Error.Message != "Bu işlem için hesabını yükseltmelisin." || b.Error.Details["action"] != "friend_request.create" {
		t.Fatalf("got %d %+v", rec.Code, b.Error)
	}

	rec, b = do(t, s, httptest.NewRequest(http.MethodGet, "/v1/boom", http.NoBody))
	if rec.Code != http.StatusInternalServerError || b.Error.Code != "INTERNAL" {
		t.Fatalf("got %d %+v", rec.Code, b.Error)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Fatal("internal cause leaked to the client")
	}
	if !strings.Contains(logs.String(), "password authentication failed") {
		t.Fatal("internal cause not logged")
	}

	rec, b = do(t, s, httptest.NewRequest(http.MethodGet, "/v1/limited", http.NoBody))
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "2" || b.Error.Details["retry_after_s"] != float64(2) {
		t.Fatalf("got %d Retry-After=%q details=%v", rec.Code, rec.Header().Get("Retry-After"), b.Error.Details)
	}

	rec, b = do(t, s, httptest.NewRequest(http.MethodGet, "/v1/ws-only", http.NoBody))
	if rec.Code != http.StatusInternalServerError || b.Error.Code != "INTERNAL" {
		t.Fatalf("WebSocket-only code over REST: got %d %s", rec.Code, b.Error.Code)
	}
}

func TestPanicRecovery(t *testing.T) {
	s, logs := newTestServer(t, config.EnvDev)
	s.Handle("GET /v1/panic", func(http.ResponseWriter, *http.Request) error { panic("nil map") })
	s.Handle("GET /v1/late-panic", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusOK)
		panic("after write")
	})

	rec, b := do(t, s, httptest.NewRequest(http.MethodGet, "/v1/panic", http.NoBody))
	if rec.Code != http.StatusInternalServerError || b.Error.Code != "INTERNAL" {
		t.Fatalf("got %d %+v", rec.Code, b.Error)
	}
	if !strings.Contains(logs.String(), "handler panicked") || !strings.Contains(logs.String(), "nil map") {
		t.Fatalf("panic not logged:\n%s", logs.String())
	}

	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/late-panic", http.NoBody))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("response after a late panic was rewritten: %d %q", rec.Code, rec.Body.String())
	}
}

func TestAbortHandlerPanicPropagates(t *testing.T) {
	s, _ := newTestServer(t, config.EnvDev)
	s.Handle("GET /v1/abort", func(http.ResponseWriter, *http.Request) error { panic(http.ErrAbortHandler) })
	defer func() {
		if v, _ := recover().(error); !errors.Is(v, http.ErrAbortHandler) {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()
	s.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/abort", http.NoBody))
	t.Fatal("ErrAbortHandler was swallowed")
}

type nameReq struct {
	Name string `json:"name"`
}

func TestBodyDecodingAndLimits(t *testing.T) {
	s, _ := newTestServer(t, config.EnvDev)
	echo := func(w http.ResponseWriter, r *http.Request) error {
		var in nameReq
		if err := DecodeJSON(r, &in); err != nil {
			return err
		}
		return WriteJSON(w, http.StatusOK, in)
	}
	s.Handle("POST /v1/echo", echo)
	s.Handle("POST /v1/big", echo, WithBodyLimit(256<<10))

	big := `{"name":"` + strings.Repeat("a", 100<<10) + `"}`
	tests := []struct {
		name, path, body string
		chunked          bool // hide Content-Length so the limit is hit while reading
		status           int
		code             string
	}{
		{"ok", "/v1/echo", `{"name":"ali"}`, false, 200, ""},
		{"unknown fields ignored", "/v1/echo", `{"name":"ali","new_field":1}`, false, 200, ""},
		{"malformed", "/v1/echo", `{"name":`, false, 400, "MALFORMED_REQUEST"},
		{"empty", "/v1/echo", ``, false, 400, "MALFORMED_REQUEST"},
		{"trailing data", "/v1/echo", `{"name":"a"} {"name":"b"}`, false, 400, "MALFORMED_REQUEST"},
		{"over limit by Content-Length", "/v1/echo", big, false, 413, "PAYLOAD_TOO_LARGE"},
		{"over limit while reading", "/v1/echo", big, true, 413, "PAYLOAD_TOO_LARGE"},
		{"route with larger limit", "/v1/big", big, false, 200, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r io.Reader = strings.NewReader(tc.body)
			if tc.chunked {
				r = io.MultiReader(r) // unknown length
			}
			req := httptest.NewRequest(http.MethodPost, tc.path, r)
			if tc.chunked {
				req.ContentLength = -1
			}
			rec, b := do(t, s, req)
			if rec.Code != tc.status || b.Error.Code != tc.code {
				t.Fatalf("got %d %q, want %d %q", rec.Code, b.Error.Code, tc.status, tc.code)
			}
		})
	}
}

func TestHandlerDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, _ := newTestServer(t, config.EnvDev)
		s.Handle("GET /v1/slow", func(_ http.ResponseWriter, r *http.Request) error {
			<-r.Context().Done() // e.g. a query cancelled by the deadline
			return r.Context().Err()
		})
		start := time.Now()
		rec, b := do(t, s, httptest.NewRequest(http.MethodGet, "/v1/slow", http.NoBody))
		if took := time.Since(start); took != testHTTP.HandlerTimeout {
			t.Fatalf("handler ran %v, want %v", took, testHTTP.HandlerTimeout)
		}
		if rec.Code != http.StatusServiceUnavailable || b.Error.Code != "UNAVAILABLE" || rec.Header().Get("Retry-After") != "1" {
			t.Fatalf("got %d %s Retry-After=%q", rec.Code, b.Error.Code, rec.Header().Get("Retry-After"))
		}
	})
}

func TestAccessLog(t *testing.T) {
	s, logs := newTestServer(t, config.EnvDev)
	s.Handle("GET /v1/users/{id}", func(w http.ResponseWriter, _ *http.Request) error {
		w.WriteHeader(http.StatusNoContent)
		return nil
	})
	req := httptest.NewRequest(http.MethodGet, "/v1/users/42", http.NoBody)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var line map[string]any
	for l := range strings.Lines(logs.String()) {
		var m map[string]any
		if json.Unmarshal([]byte(l), &m) == nil && m["msg"] == "request" {
			line = m
		}
	}
	if line == nil {
		t.Fatalf("no access log line:\n%s", logs.String())
	}
	want := map[string]any{
		"method": "GET", "path": "/v1/users/42", "route": "GET /v1/users/{id}",
		"status": float64(204), "request_id": rec.Header().Get(RequestIDHeader),
	}
	for k, v := range want {
		if line[k] != v {
			t.Errorf("access log %s = %v, want %v", k, line[k], v)
		}
	}
}

func TestServeAndGracefulShutdown(t *testing.T) {
	s, _ := newTestServer(t, config.EnvDev)
	inFlight := make(chan struct{})
	release := make(chan struct{})
	s.Handle("GET /v1/slow", func(w http.ResponseWriter, _ *http.Request) error {
		close(inFlight)
		<-release
		return WriteJSON(w, http.StatusOK, map[string]bool{"done": true})
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	served := make(chan error, 1)
	go func() { served <- s.Serve(ctx, ln) }()

	url := "http://" + ln.Addr().String() + "/v1/slow"
	got := make(chan int, 1)
	go func() {
		req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			got <- -1
			return
		}
		resp.Body.Close()
		got <- resp.StatusCode
	}()

	<-inFlight
	cancel() // shutdown starts while the request is still running
	time.Sleep(50 * time.Millisecond)
	close(release)

	if code := <-got; code != http.StatusOK {
		t.Fatalf("in-flight request got %d, want 200 (graceful shutdown)", code)
	}
	if err := <-served; err != nil {
		t.Fatalf("Serve = %v, want nil", err)
	}
	if _, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second); err == nil {
		t.Fatal("listener still accepting after shutdown")
	}
}

func TestRunFailsOnBadAddress(t *testing.T) {
	cfg := testHTTP
	cfg.Addr = "256.0.0.1:1"
	s := New(cfg, config.EnvDev, slog.New(slog.DiscardHandler))
	if err := s.Run(t.Context()); err == nil {
		t.Fatal("Run on an invalid address succeeded")
	}
}
