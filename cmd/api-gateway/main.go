// Package main is the StackWatch API gateway entry point.
// Single binary, modular. Each route is wired in routes.go.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stackwatch/platform/internal/handler"
	"github.com/stackwatch/platform/internal/homelab"
	"github.com/stackwatch/platform/internal/synthetics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := loadConfig()
	logger.Info("starting api-gateway",
		"http_addr", cfg.HTTPAddr,
		"db_url_set", cfg.DatabaseURL != "",
		"jwt_secret_set", cfg.JWTSecret != "",
		"install_mode", cfg.InstallMode,
	)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := connectDatabase(rootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("database connected")

	issuer := newIssuer(cfg.JWTSecret)

	// Tier 7.4 — Synthetics Full background runner. Created here so its
	// lifecycle is owned by main.go alongside the HTTP server; passed
	// into buildRouter so the run-now handler can call ExecuteSync.
	synthRunner := handler.NewSyntheticsRunner(pool, logger)
	go synthRunner.Run(rootCtx)

	router := buildRouter(rootCtx, logger, pool, issuer, cfg.InstallMode, cfg.WebTerminalURL, synthRunner)

	// Start the synthetics scheduler (M7 background runner).
	sched := synthetics.NewScheduler(pool.Pgx(), 30*time.Second, logger)
	go sched.Run(rootCtx)

	// Tier 10 Phase 2 — Service Health background worker.
	// Ticks every 60s, probes every enabled pinned service in
	// homelab_pinned_services, INSERTs results into
	// homelab_service_health, prunes rows older than 30 days.
	// First tick fires immediately on Start().
	serviceHealth := homelab.NewServiceHealthWorker(pool, homelab.WithLogger(logger))
	serviceHealth.Start(rootCtx)

	// Tier 10 Phase 4 — Calendar background worker.
	// Ticks every 30min, fetches every enabled iCal URL in
	// homelab_calendars (across all tenants + users), parses with
	// github.com/arran4/golang-ical, and upserts the contained
	// VEVENTs into homelab_events. Per-calendar errors are
	// logged but never abort the batch — one bad feed must not
	// skip the rest. First tick fires immediately on Start().
	calendarWorker := homelab.NewCalendarWorker(pool, homelab.WithCalendarLogger(logger))
	calendarWorker.Start(rootCtx)

	// Tier 10 Phase 5 — Download Stats background worker.
	// Ticks every 60s, fetches every enabled download client in
	// homelab_download_clients (across all tenants + users), and
	// INSERTs a parsed-state row into homelab_download_snapshots.
	// Per-client errors are logged but never abort the batch —
	// one bad target must not skip the rest. First tick fires
	// immediately on Start().
	downloadsWorker := homelab.NewDownloadsWorker(pool, homelab.WithDownloadsLogger(logger))
	downloadsWorker.Start(rootCtx)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("http listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			cancel()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-sig:
		logger.Info("signal received", "signal", s.String())
	case <-rootCtx.Done():
		logger.Info("context cancelled")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "err", err)
	}
	logger.Info("api-gateway stopped")
}
