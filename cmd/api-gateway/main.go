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

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/handler"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/homelab"
	"github.com/stackwatch/platform/internal/platform"
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

	// Tier 11 Phase 7 — Rate Limiting (PL7) — in-memory
	// token-bucket limiter + per-tenant plan cache. The
	// limiter holds one bucket per tenant_id and refills
	// at the per-plan rate (free=10/min, starter=60/min,
	// pro=100/min, enterprise=1000/min — see
	// platform.PlanDefaults). The plan cache is a
	// sync.Map keyed by tenant_id; on cache miss it
	// reads tenants.plan once and caches forever (until
	// process restart). Both are threaded into
	// buildRouter so the middleware + handlers share the
	// same instances.
	rateLimiter := platform.NewLimiter()
	ratePlans := platform.NewPlanCache(func(tid uuid.UUID) string {
		var plan string
		// One query per tenant per process lifetime.
		// On error (tenant not found, etc.) fall back
		// to "free" — the safest default that doesn't
		// grant a paid tier accidentally.
		if err := pool.Pgx().QueryRow(rootCtx,
			`SELECT COALESCE(plan, 'free') FROM tenants WHERE id = $1`,
			tid,
		).Scan(&plan); err != nil {
			return platform.DefaultPlan
		}
		if plan == "" {
			return platform.DefaultPlan
		}
		return plan
	})

	// Tier 7.4 — Synthetics Full background runner. Created here so its
	// lifecycle is owned by main.go alongside the HTTP server; passed
	// into buildRouter so the run-now handler can call ExecuteSync.
	synthRunner := handler.NewSyntheticsRunner(pool, logger)
	go synthRunner.Run(rootCtx)

	router := buildRouter(rootCtx, logger, pool, issuer, cfg.InstallMode, cfg.WebTerminalURL, synthRunner, rateLimiter, ratePlans)

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

	// Tier 10 Phase 6 — Media Server background worker.
	// Ticks every 60s, fetches every enabled media server in
	// homelab_media_servers (across all tenants + users), and
	// UPSERTs the parsed-state into homelab_now_playing +
	// homelab_recent_additions. Per-server errors are logged but
	// never abort the batch — one bad target must not skip the
	// rest. First tick fires immediately on Start().
	mediaWorker := homelab.NewMediaWorker(pool, homelab.WithMediaLogger(logger))
	mediaWorker.Start(rootCtx)

	// Tier 10 Phase 8 — RSS / Activity Feed background worker.
	// Ticks every 5min, fetches every enabled feed URL in
	// homelab_rss_feeds (across all tenants + users), parses
	// each one with github.com/mmcdole/gofeed, and UPSERTs the
	// contained <item>s into homelab_rss_items. Per-feed
	// errors are logged but never abort the batch — one bad
	// feed must not skip the rest. First tick fires
	// immediately on Start().
	rssWorker := homelab.NewRssWorker(pool, homelab.WithRssLogger(logger))
	rssWorker.Start(rootCtx)

	// Tier 10 Phase 9 — Task Scheduler background worker
	// (H10 — FINAL feature phase of Tier 10).
	// SECURITY-CRITICAL: this worker dispatches user-defined
	// HTTP probes. The guardrails (URL allowlist, header
	// blacklist, body cap, rate limit, audit log) are
	// enforced inside homelab.RunSchedulerJobAndInsertRun +
	// handlers_homelab_scheduler_validation.go — not here.
	// Ticks every 60s, fetches every enabled job in
	// homelab_scheduler_jobs whose next_run_at <= NOW(),
	// and runs them through the same export used by the
	// POST /jobs/:id/run handler (DRY). Per-job errors are
	// logged but never abort the batch. First tick fires
	// immediately on Start().
	schedulerWorker := homelab.NewSchedulerWorker(pool, homelab.WithSchedulerLogger(logger))
	schedulerWorker.Start(rootCtx)

	// Tier 11 Phase 2 — Usage Metering background worker (PL2).
	// Ticks every hour, UPSERTs the most-recent completed
	// (tenant_id, event_kind) bucket into platform_usage_aggregates,
	// then prunes platform_usage_events rows older than 365 days.
	// First tick fires immediately on Start().
	usageMeter := platform.NewUsageMeterWorker(pool, platform.WithUsageLogger(logger))
	usageMeter.Start(rootCtx)

	// Tier 11 Phase 4 — Seed built-in plan definitions (PL4).
	// Runs once on boot BEFORE the workers / routes need the
	// catalog. Idempotent INSERT ... ON CONFLICT (name) DO
	// NOTHING so re-running on a populated DB is a no-op —
	// manual edits to the catalog (e.g. a Black Friday price
	// bump) survive gateway restarts.
	auth.SeedBuiltinPlans(rootCtx, pool, logger)

	// Tier 11 Phase 4 — Retention background worker (PL4).
	// Ticks every 24h, walks every non-suspended tenant in
	// platform_tenant_limits, reads their effective
	// data_retention_days, and DELETEs rows older than the
	// cutoff from the documented data tables (metric_points,
	// audit_log, alert_history, notifications,
	// platform_usage_events). Per-tenant errors are logged
	// but never abort the batch. Suspended tenants SKIP —
	// they have bigger problems and the operator might
	// restore them. First tick fires immediately on Start().
	retentionWorker := platform.NewRetentionWorker(pool, platform.WithRetentionLogger(logger))
	retentionWorker.Start(rootCtx)

	// Tier 11 Phase 5 — Backup Scheduler background worker
	// (PL5). SECURITY-CRITICAL: this worker triggers
	// CreateBackupAndEncrypt which holds the per-tenant
	// AES-256-GCM encryption key in memory for the duration
	// of the encrypt pass. The key is derived inside the
	// helper via HKDF-SHA256(masterKey, info=tenant_id) so
	// a stolen master key does NOT auto-compromise every
	// tenant's backups.
	//
	// Ticks every 1h, walks every ENABLED row in
	// platform_backup_jobs whose frequency says it's due
	// (daily/weekly/monthly cadence), and creates a
	// 'scheduled' kind backup per tenant. Per-tenant errors
	// are logged but never abort the batch. First tick fires
	// immediately on Start().
	backupSched := platform.NewBackupSchedulerWorker(pool, platform.WithBackupSchedulerLogger(logger))
	backupSched.Start(rootCtx)

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
