// Package httpserver is the REST server: net/http ServeMux routing
// (ADR-0002), the standard middleware chain, the error envelope
// (docs/api/conventions.md §6) and graceful shutdown.
//
// Every request passes, outermost first: security headers → request id →
// access log + panic recovery → locale → route (pattern, body limit,
// handler deadline, error rendering). Unmatched requests get the
// NOT_FOUND / METHOD_NOT_ALLOWED envelope.
package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
)

// DefaultBodyLimit is the REST request body limit (docs/api/limits.md §2).
const DefaultBodyLimit int64 = 64 << 10

// maxHeaderBytes bounds request headers; a Firebase ID token is ~1 KB.
const maxHeaderBytes = 32 << 10

// Server is the HTTP server component of cmd/api.
type Server struct {
	cfg     config.HTTP
	logger  *slog.Logger
	mux     *http.ServeMux
	handler http.Handler
	// dispatch is the part of the chain that Use extends: it runs after
	// request id, logging and locale, so its errors are logged and localized.
	dispatch http.Handler
	uses     []func(http.Handler) http.Handler
}

// New builds a server. HSTS is sent only in prod (deployed over HTTPS).
func New(cfg config.HTTP, env config.Environment, logger *slog.Logger) *Server {
	s := &Server{cfg: cfg, logger: logger, mux: http.NewServeMux()}
	s.dispatch = notFound(s.mux, logger)
	h := locale(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { s.dispatch.ServeHTTP(w, r) }))
	h = accessLog(logger)(h)
	h = requestID(h)
	s.handler = securityHeaders(env == config.EnvProd)(h)
	return s
}

// Use adds a middleware that runs for every request (matched or not) after
// the built-in ones; the first Use call runs first. Call it before serving.
func (s *Server) Use(mw func(http.Handler) http.Handler) {
	s.uses = append(s.uses, mw)
	h := notFound(s.mux, s.logger)
	for i := len(s.uses) - 1; i >= 0; i-- {
		h = s.uses[i](h)
	}
	s.dispatch = h
}

// Logger returns the server's logger, for middleware that renders errors.
func (s *Server) Logger() *slog.Logger { return s.logger }

// RouteOption adjusts one route.
type RouteOption func(*routeOptions)

type routeOptions struct {
	bodyLimit int64
}

// WithBodyLimit overrides DefaultBodyLimit for one route
// (e.g. 256 KB for creator quiz saves, limits.md §2).
func WithBodyLimit(n int64) RouteOption {
	return func(o *routeOptions) { o.bodyLimit = n }
}

// Handle registers h for a ServeMux pattern such as "GET /v1/me".
func (s *Server) Handle(pattern string, h HandlerFunc, opts ...RouteOption) {
	o := routeOptions{bodyLimit: DefaultBodyLimit}
	for _, opt := range opts {
		opt(&o)
	}
	s.mux.Handle(pattern, route(pattern, h, o.bodyLimit, s.cfg.HandlerTimeout, s.logger))
}

// Handler returns the complete handler chain (used by tests and Run).
func (s *Server) Handler() http.Handler { return s.handler }

// Name implements app.Component.
func (s *Server) Name() string { return "http" }

// Run listens on cfg.Addr and serves until ctx is cancelled, then shuts
// down gracefully: in-flight requests get cfg.ShutdownTimeout to finish.
func (s *Server) Run(ctx context.Context) error {
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		return err
	}
	return s.Serve(ctx, ln)
}

// Serve is Run on an existing listener.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler:           s.handler,
		ReadHeaderTimeout: s.cfg.ReadTimeout,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(s.logger.Handler(), slog.LevelWarn),
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	s.logger.InfoContext(ctx, "http server listening", "addr", ln.Addr().String())

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.ShutdownTimeout)
	defer cancel()
	start := time.Now()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		s.logger.WarnContext(ctx, "graceful shutdown timed out; closing connections", "error", err)
		_ = srv.Close()
	}
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	s.logger.InfoContext(ctx, "http server stopped", "took", time.Since(start))
	return nil
}
