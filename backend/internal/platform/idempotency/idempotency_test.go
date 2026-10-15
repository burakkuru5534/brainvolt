package idempotency

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/apierror"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/httpserver"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/redis"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

var discard = slog.New(slog.DiscardHandler)

var httpCfg = config.HTTP{ReadTimeout: time.Second, WriteTimeout: 3 * time.Second, HandlerTimeout: 2 * time.Second, ShutdownTimeout: time.Second}

// fixture is a server whose POST /v1/rooms creates a "room" and counts runs.
type fixture struct {
	srv   *httpserver.Server
	rdb   *redis.Client
	runs  atomic.Int32
	user  string
	gate  chan struct{} // when non-nil, the handler waits on it
	fail  atomic.Bool   // when set, the handler returns a 500
	limit atomic.Bool   // when set, the handler returns a 403
}

func newFixture(t *testing.T, mode Mode) *fixture {
	t.Helper()
	rdb, err := redis.Connect(t.Context(), testenv.RedisURL(t), "brainvolt-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	f := &fixture{rdb: rdb, user: ids.NewString()}
	store := New(rdb, discard)
	f.srv = httpserver.New(httpCfg, config.EnvDev, discard)
	// Stand-in for the authentication middleware (roadmap 1.6).
	f.srv.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if u := r.Header.Get("X-Test-User"); u != "" {
				r = r.WithContext(logging.WithUserID(r.Context(), u))
			}
			next.ServeHTTP(w, r)
		})
	})
	f.srv.Handle("POST /v1/rooms", store.Wrap(mode, func(w http.ResponseWriter, r *http.Request) error {
		n := f.runs.Add(1)
		if f.gate != nil {
			<-f.gate
		}
		if f.fail.Load() {
			return errors.New("db down")
		}
		if f.limit.Load() {
			return apierror.New(apierror.TeamLimitReached)
		}
		var in struct {
			Name string `json:"name"`
		}
		if err := httpserver.DecodeJSON(r, &in); err != nil {
			return err
		}
		w.Header().Set("Location", "/v1/rooms/"+in.Name)
		return httpserver.WriteJSON(w, http.StatusCreated, map[string]any{"name": in.Name, "run": n})
	}))
	return f
}

func (f *fixture) post(t *testing.T, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/rooms", strings.NewReader(body))
	req.Header.Set("X-Test-User", f.user)
	if key != "" {
		req.Header.Set(Header, key)
	}
	rec := httptest.NewRecorder()
	f.srv.Handler().ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var b struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &b)
	return b.Error.Code
}

func TestReplayReturnsStoredResponse(t *testing.T) {
	f := newFixture(t, Required)
	key := ids.NewString()

	first := f.post(t, key, `{"name":"quiz-night"}`)
	if first.Code != http.StatusCreated || first.Header().Get(ReplayedHeader) != "" {
		t.Fatalf("first = %d %v", first.Code, first.Header())
	}
	second := f.post(t, key, `{"name":"quiz-night"}`)
	if second.Code != http.StatusCreated || second.Header().Get(ReplayedHeader) != "true" {
		t.Fatalf("replay = %d replayed=%q", second.Code, second.Header().Get(ReplayedHeader))
	}
	if second.Body.String() != first.Body.String() || second.Header().Get("Location") != "/v1/rooms/quiz-night" {
		t.Fatalf("replay body/headers differ:\n%s\n%s", first.Body.String(), second.Body.String())
	}
	if n := f.runs.Load(); n != 1 {
		t.Fatalf("handler ran %d times, want 1", n)
	}
	ttl, err := f.rdb.TTL(t.Context(), redis.Key("idem", f.user, key)).Result()
	if err != nil || ttl < 23*time.Hour || ttl > Retention {
		t.Fatalf("record TTL = %v, %v; want ~24h", ttl, err)
	}

	// The same key from another user is a different record.
	f.user = ids.NewString()
	if rec := f.post(t, key, `{"name":"quiz-night"}`); rec.Header().Get(ReplayedHeader) != "" || f.runs.Load() != 2 {
		t.Fatal("keys must be scoped per user")
	}
}

func TestKeyReusedWithDifferentBody(t *testing.T) {
	f := newFixture(t, Required)
	key := ids.NewString()
	f.post(t, key, `{"name":"a"}`)
	rec := f.post(t, key, `{"name":"b"}`)
	if rec.Code != http.StatusConflict || errorCode(t, rec) != "IDEMPOTENCY_KEY_REUSED" {
		t.Fatalf("got %d %s", rec.Code, errorCode(t, rec))
	}
}

func TestHeaderValidation(t *testing.T) {
	f := newFixture(t, Required)
	for key, rule := range map[string]string{"": "required", "not-a-uuid": "uuid"} {
		rec := f.post(t, key, `{"name":"a"}`)
		if rec.Code != http.StatusBadRequest || errorCode(t, rec) != "VALIDATION_FAILED" || !strings.Contains(rec.Body.String(), `"rule":"`+rule+`"`) {
			t.Errorf("key %q: %d %s", key, rec.Code, rec.Body.String())
		}
	}
	if f.runs.Load() != 0 {
		t.Fatal("handler ran for an invalid key")
	}

	opt := newFixture(t, Optional)
	if rec := opt.post(t, "", `{"name":"a"}`); rec.Code != http.StatusCreated {
		t.Fatalf("optional without key = %d", rec.Code)
	}
}

func TestClientErrorsAreStoredServerErrorsAreNot(t *testing.T) {
	f := newFixture(t, Required)

	key := ids.NewString()
	f.limit.Store(true)
	if rec := f.post(t, key, `{}`); rec.Code != http.StatusConflict || errorCode(t, rec) != "TEAM_LIMIT_REACHED" {
		t.Fatalf("4xx = %d", rec.Code)
	}
	f.limit.Store(false)
	if rec := f.post(t, key, `{}`); rec.Header().Get(ReplayedHeader) != "true" || errorCode(t, rec) != "TEAM_LIMIT_REACHED" {
		t.Fatalf("4xx must be replayed, got %d %s", rec.Code, rec.Body.String())
	}

	key = ids.NewString()
	f.fail.Store(true)
	if rec := f.post(t, key, `{"name":"x"}`); rec.Code != http.StatusInternalServerError {
		t.Fatalf("5xx = %d", rec.Code)
	}
	f.fail.Store(false)
	rec := f.post(t, key, `{"name":"x"}`)
	if rec.Code != http.StatusCreated || rec.Header().Get(ReplayedHeader) != "" {
		t.Fatalf("retry after 5xx must run again, got %d replayed=%q", rec.Code, rec.Header().Get(ReplayedHeader))
	}
}

func TestConcurrentDuplicateIsInProgress(t *testing.T) {
	f := newFixture(t, Required)
	f.gate = make(chan struct{})
	key := ids.NewString()

	var wg sync.WaitGroup
	var first *httptest.ResponseRecorder
	wg.Go(func() { first = f.post(t, key, `{"name":"a"}`) })
	for f.runs.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	rec := f.post(t, key, `{"name":"a"}`)
	close(f.gate)
	wg.Wait()

	if rec.Code != http.StatusConflict || errorCode(t, rec) != "IDEMPOTENT_REQUEST_IN_PROGRESS" || rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("duplicate in flight = %d %s", rec.Code, rec.Body.String())
	}
	if first.Code != http.StatusCreated || f.runs.Load() != 1 {
		t.Fatalf("first = %d, runs = %d", first.Code, f.runs.Load())
	}
}

func TestFailsClosedWithoutRedisOrUser(t *testing.T) {
	f := newFixture(t, Required)
	_ = f.rdb.Close()
	if rec := f.post(t, ids.NewString(), `{}`); rec.Code != http.StatusServiceUnavailable || f.runs.Load() != 0 {
		t.Fatalf("redis down = %d (runs %d), want 503 without running", rec.Code, f.runs.Load())
	}

	g := newFixture(t, Required)
	req := httptest.NewRequest(http.MethodPost, "/v1/rooms", strings.NewReader(`{}`))
	req.Header.Set(Header, ids.NewString())
	rec := httptest.NewRecorder()
	g.srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError || g.runs.Load() != 0 {
		t.Fatalf("unauthenticated route = %d (runs %d), want 500 wiring error", rec.Code, g.runs.Load())
	}
}
