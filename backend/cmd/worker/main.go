// Command worker runs the BrainVolt background and scheduled jobs.
// It only wires and starts; everything else lives in internal/app.
//
// Cloud Run requires the container to bind $PORT within the startup timeout.
// We therefore start a minimal HTTP health server immediately (before the
// heavy wiring that talks to Postgres and Firebase) and only then build and
// run the rest of the worker components.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/burakkuru5534/brainvolt/backend/internal/app"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/config"
	"github.com/burakkuru5534/brainvolt/backend/internal/platform/logging"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		logging.New(os.Stderr, logging.Options{Format: logging.FormatJSON}).
			Error("invalid configuration", "error", err)
		return 1
	}
	logger := logging.New(os.Stderr, logging.Options{Format: cfg.Log.Format, Level: cfg.Log.Level})
	logger.Info("configuration loaded", "config", cfg)

	// --- Step 1: bind the port immediately so Cloud Run health probes pass ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	healthSrv := startHealthServer(ctx, addr, logger)

	// --- Step 2: build the full worker composition (Postgres, Firebase, River) ---
	a, err := app.NewWorker(ctx, cfg, logger)
	if err != nil {
		logger.Error("start-up failed", "error", err)
		shutdownHealthServer(healthSrv, logger)
		return 1
	}

	// --- Step 3: run the worker; health server keeps running alongside it ---
	if err := a.Run(ctx); err != nil {
		shutdownHealthServer(healthSrv, logger)
		return 1
	}
	shutdownHealthServer(healthSrv, logger)
	return 0
}

// startHealthServer binds addr immediately and serves /healthz + / for Cloud
// Run probes. It runs in a background goroutine; ctx cancellation stops it.
func startHealthServer(ctx context.Context, addr string, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok","role":"worker"}`))
	}
	mux.HandleFunc("/healthz", handler)
	mux.HandleFunc("/", handler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("worker health server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("worker health server error", "error", err)
		}
	}()

	// Stop when ctx is cancelled (SIGTERM / SIGINT).
	go func() {
		<-ctx.Done()
		shutdownHealthServer(srv, logger)
	}()

	return srv
}

func shutdownHealthServer(srv *http.Server, logger *slog.Logger) {
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Warn("worker health server shutdown error", "error", err)
	}
}
