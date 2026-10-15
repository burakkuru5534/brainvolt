// Package redis opens the go-redis client (ADR-0002) used for live match
// state, queues, presence, caches and rate limits (docs/domain/redis-keys.md).
package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// connectTimeout bounds the start-up ping.
const connectTimeout = 5 * time.Second

// Client is the go-redis client type, re-exported so modules depend on this
// package rather than on the driver's import path.
type Client = goredis.Client

// ErrNil is returned by reads of a missing key.
var ErrNil = goredis.Nil

// Connect creates the client and verifies Redis is reachable. The returned
// error never contains the URL (it may hold a password).
func Connect(ctx context.Context, url, clientName string) (*Client, error) {
	opts, err := goredis.ParseURL(url)
	if err != nil {
		return nil, errors.New("redis: invalid connection URL")
	}
	opts.ClientName = clientName
	client := goredis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: cannot reach %s: %w", opts.Addr, err)
	}
	return client, nil
}

// Ping reports whether Redis answers (used by /readyz).
func Ping(ctx context.Context, c *Client) error {
	return c.Ping(ctx).Err()
}

// Key joins key parts with ":" (redis-keys.md §1: every key starts with its
// module prefix, e.g. Key("game", "user", id, "live")).
func Key(parts ...string) string {
	return strings.Join(parts, ":")
}
