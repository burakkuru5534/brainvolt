package health

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
)

var httpCfg = config.HTTP{
	ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute,
	HandlerTimeout: 10 * time.Second, ShutdownTimeout: time.Second,
}

func get(t *testing.T, checks ...Check) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var logs bytes.Buffer
	logger := logging.New(&logs, logging.Options{Format: logging.FormatJSON, Level: slog.LevelDebug})
	s := httpserver.New(httpCfg, config.EnvDev, logger)
	Register(s, logger, checks...)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", http.NoBody))
	return rec, logs.String()
}

func ok(context.Context) error { return nil }

func TestHealthzAlwaysOK(t *testing.T) {
	s := httpserver.New(httpCfg, config.EnvDev, slog.New(slog.DiscardHandler))
	Register(s, slog.New(slog.DiscardHandler), Check{Name: "postgres", Fn: func(context.Context) error { return errors.New("down") }})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("/healthz = %d %s; liveness must not depend on dependencies", rec.Code, rec.Body.String())
	}
}

func TestReadyWhenAllChecksPass(t *testing.T) {
	rec, _ := get(t, Check{"postgres", ok}, Check{"redis", ok})
	want := `{"checks":{"postgres":"ok","redis":"ok"},"status":"ready"}`
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
		t.Fatalf("/readyz = %d %s", rec.Code, rec.Body.String())
	}
}

func TestNotReadyWhenACheckFails(t *testing.T) {
	rec, logs := get(t,
		Check{"postgres", ok},
		Check{"redis", func(context.Context) error { return errors.New("dial tcp: connection refused (pw=hunter2)") }},
	)
	if rec.Code != http.StatusServiceUnavailable || rec.Header().Get("Retry-After") != "5" {
		t.Fatalf("/readyz = %d Retry-After=%q", rec.Code, rec.Header().Get("Retry-After"))
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Details struct {
				Checks map[string]string `json:"checks"`
			} `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "UNAVAILABLE" || body.Error.Details.Checks["redis"] != "failed" || body.Error.Details.Checks["postgres"] != "ok" {
		t.Fatalf("body = %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatal("check error leaked to the response")
	}
	if !strings.Contains(logs, "readiness check failed") || !strings.Contains(logs, "connection refused") {
		t.Fatalf("failure not logged:\n%s", logs)
	}
}

func TestHangingCheckTimesOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		rec, _ := get(t, Check{"postgres", func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}})
		if took := time.Since(start); took != checkTimeout {
			t.Fatalf("readyz took %v, want %v", took, checkTimeout)
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("/readyz = %d", rec.Code)
		}
	})
}
