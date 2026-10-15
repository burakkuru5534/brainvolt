package redis

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/ids"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/testenv"
)

func TestConnectErrorsHideURL(t *testing.T) {
	_, err := Connect(t.Context(), "redis://:secret@127.0.0.1:1/0", "test")
	if err == nil || !strings.Contains(err.Error(), "cannot reach 127.0.0.1:1") {
		t.Fatalf("unreachable: err = %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("error leaks password: %v", err)
	}
	if _, err := Connect(t.Context(), "http://:secret@host", "test"); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("invalid URL: err = %v", err)
	}
}

func TestKey(t *testing.T) {
	if got := Key("game", "user", "u1", "live"); got != "game:user:u1:live" {
		t.Fatalf("Key = %q", got)
	}
}

func TestConnectAndServerConfig(t *testing.T) {
	c, err := Connect(t.Context(), testenv.RedisURL(t), "brainvolt-test")
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	ctx := t.Context()

	if err := Ping(ctx, c); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if name, err := c.ClientGetName(ctx).Result(); err != nil || name != "brainvolt-test" {
		t.Fatalf("client name = %q, %v", name, err)
	}

	// The Compose service must run with the settings of redis-keys.md §1.
	cfg, err := c.ConfigGet(ctx, "*").Result()
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"maxmemory-policy": "noeviction", "appendonly": "yes", "appendfsync": "everysec"} {
		if cfg[k] != want {
			t.Errorf("CONFIG %s = %q, want %q", k, cfg[k], want)
		}
	}
	info, err := c.Info(ctx, "server").Result()
	if err != nil || !strings.Contains(info, "redis_version:8.10.") {
		t.Errorf("server is not Redis 8.10 (ADR-0001): %v", err)
	}

	key := Key("test", ids.NewString())
	t.Cleanup(func() { c.Del(ctx, key) })
	if err := c.Set(ctx, key, "v", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Get(ctx, key).Result(); err != nil || got != "v" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	if _, err := c.Get(ctx, key+":missing").Result(); !errors.Is(err, ErrNil) {
		t.Fatalf("missing key err = %v, want redis.ErrNil", err)
	}
}
