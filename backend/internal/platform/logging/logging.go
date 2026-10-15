// Package logging builds the process logger (log/slog, ADR-0002) and carries
// request, user and match ids through context.Context so that every log line
// written with a *Context method includes them (docs/api/conventions.md §5).
package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Format selects the slog handler.
type Format string

const (
	// FormatJSON is one JSON object per line, for deployed environments.
	FormatJSON Format = "json"
	// FormatText is logfmt-style key=value output, for local development.
	FormatText Format = "text"
)

// UnmarshalText lets env parsing (platform/config) read a Format.
func (f *Format) UnmarshalText(b []byte) error {
	switch v := Format(strings.ToLower(strings.TrimSpace(string(b)))); v {
	case FormatJSON, FormatText:
		*f = v
		return nil
	default:
		return fmt.Errorf("unknown log format %q (want json or text)", string(b))
	}
}

// Options configures New.
type Options struct {
	Format Format
	Level  slog.Level
}

// New returns a logger writing to w. Records written with a context add the
// ids stored by WithRequestID, WithUserID and WithMatchID.
func New(w io.Writer, opts Options) *slog.Logger {
	hopts := &slog.HandlerOptions{Level: opts.Level}
	var h slog.Handler
	if opts.Format == FormatText {
		h = slog.NewTextHandler(w, hopts)
	} else {
		h = slog.NewJSONHandler(w, hopts)
	}
	return slog.New(contextHandler{h})
}

type ctxKey int

const (
	requestIDKey ctxKey = iota
	userIDKey
	matchIDKey
)

// Log attribute names; they match the JSON field names used by the API.
const (
	RequestIDAttr = "request_id"
	UserIDAttr    = "user_id"
	MatchIDAttr   = "match_id"
)

// WithRequestID returns a copy of ctx carrying the request id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// WithUserID returns a copy of ctx carrying the authenticated user id.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// WithMatchID returns a copy of ctx carrying the match id.
func WithMatchID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, matchIDKey, id)
}

// RequestID returns the request id stored in ctx, or "".
func RequestID(ctx context.Context) string { return value(ctx, requestIDKey) }

// UserID returns the user id stored in ctx, or "".
func UserID(ctx context.Context) string { return value(ctx, userIDKey) }

// MatchID returns the match id stored in ctx, or "".
func MatchID(ctx context.Context) string { return value(ctx, matchIDKey) }

func value(ctx context.Context, key ctxKey) string {
	if ctx == nil {
		return ""
	}
	s, _ := ctx.Value(key).(string)
	return s
}

// contextHandler adds the context ids to every record.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String(RequestIDAttr, id))
	}
	if id := UserID(ctx); id != "" {
		r.AddAttrs(slog.String(UserIDAttr, id))
	}
	if id := MatchID(ctx); id != "" {
		r.AddAttrs(slog.String(MatchIDAttr, id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}
