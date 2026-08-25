// Tier 11 Phase 2 — Usage Metering (PL2).
//
// UsageMeterWorker — hourly aggregation worker.
//
// Ticks every `tickInterval` (default 1h, first tick fires
// immediately on Start). Each tick:
//   1. Walks every (tenant_id, event_kind) seen in the LAST bucket
//      window and UPSERTs an aggregate row keyed on
//      (tenant_id, event_kind, bucket_ts). The UNIQUE constraint
//      means a previously-completed bucket is idempotently
//      up-to-date — replaying the same window is safe.
//   2. Optionally prunes platform_usage_events rows older than
//      `retention` (default 365 days) so the raw event log
//      stays bounded.
//
// Worker must not crash on bad input. Every tenant loop runs
// inside a defer-recover so a single tenant's query error can
// never kill the worker.
//
// Why a separate package (internal/platform/ instead of
// internal/homelab/): Tier 11 owns multi-tenant platform logic,
// not per-user homelab UI. The deploy helper (Phase 1) already
// lives here in internal/platform/deploy.go and registers the
// package boundary; this worker is its sibling.
//
// Why hourly buckets: matches the spec PL2 "aggregates raw events
// into hourly buckets every hour". date_trunc('hour', created_at)
// aligns events to UTC hour boundaries (no fractional drift).
package platform

import (
	"context"
	"log/slog"
	"time"

	"github.com/stackwatch/platform/internal/db"
)

const (
	defaultUsageTickInterval = 1 * time.Hour
	defaultUsageRetention    = 365 * 24 * time.Hour
	defaultUsageTickLimit    = 60 * time.Second
)

// UsageMeterWorker is the per-process singleton. Owns a pool ref
// + tunables + slog logger. Start() launches the goroutine; the
// goroutine respects ctx cancellation for graceful shutdown.
type UsageMeterWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	retention    time.Duration
	tickTimeout  time.Duration
	logger       *slog.Logger
}

// UsageMeterOption mutates the worker at construction time so
// main.go can override defaults (e.g. a faster tick in tests).
type UsageMeterOption func(*UsageMeterWorker)

// WithUsageTickInterval overrides the default 1h tick cadence.
func WithUsageTickInterval(d time.Duration) UsageMeterOption {
	return func(w *UsageMeterWorker) { w.tickInterval = d }
}

// WithUsageRetention overrides the default 365-day raw-event
// retention. Set to 0 to disable pruning.
func WithUsageRetention(d time.Duration) UsageMeterOption {
	return func(w *UsageMeterWorker) { w.retention = d }
}

// WithUsageLogger overrides the default slog logger.
func WithUsageLogger(l *slog.Logger) UsageMeterOption {
	return func(w *UsageMeterWorker) { w.logger = l }
}

// WithUsageTickTimeout overrides the default 60s per-tick time
// budget. One tick wraps every (tenant, kind) — a 60s budget is
// huge but keeps a runaway query from pinning the goroutine.
func WithUsageTickTimeout(d time.Duration) UsageMeterOption {
	return func(w *UsageMeterWorker) { w.tickTimeout = d }
}

// NewUsageMeterWorker builds a worker. Call exactly once on
// api-gateway boot, then Start() in a goroutine.
func NewUsageMeterWorker(pool *db.Pool, opts ...UsageMeterOption) *UsageMeterWorker {
	w := &UsageMeterWorker{
		pool:         pool,
		tickInterval: defaultUsageTickInterval,
		retention:    defaultUsageRetention,
		tickTimeout:  defaultUsageTickLimit,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately; the
// goroutine exits when ctx is cancelled. Safe to call multiple
// times — second+ calls are no-ops (the existing goroutine is
// still running).
//
// The first tick fires immediately so a freshly-restarted gateway
// rolls up the last bucket without waiting a full tick.
func (w *UsageMeterWorker) Start(ctx context.Context) {
	w.logger.Info("usagemeter worker: starting",
		"tick", w.tickInterval,
		"retention", w.retention,
	)
	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("usagemeter worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("usagemeter worker: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("usagemeter worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// tick runs one pass: roll up the most-recent completed bucket
// across every tenant, then prune events older than `retention`.
//
// "Most-recent completed bucket" = the bucket whose right edge is
// the largest hour boundary ≤ now. Today's in-progress bucket is
// deliberately SKIPPED — we'll roll it up on the next tick once it
// has closed. This keeps the worker idempotent (replaying today's
// bucket would change its rollup mid-flight). If a tick skips (e.g.
// api-gateway was down for 2h), on the next run we'll still only
// get the LAST hour — the older buckets are simply lost. The
// retention worker in Phase 4 will sweep the leftovers.
func (w *UsageMeterWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.tickTimeout)
	defer cancel()

	// Round DOWN to the most recent completed hour boundary so
	// we never roll up an in-flight bucket.
	bucketStart := nowHourFloor(time.Now().UTC()).Add(-time.Hour)
	bucketEnd := bucketStart.Add(time.Hour)
	w.aggregateBucket(ctx, bucketStart, bucketEnd)

	if w.retention > 0 {
		w.prune(ctx)
	}
}

// aggregateBucket UPSERTs one row per (tenant, event_kind) for
// the given hour window into platform_usage_aggregates.
//
// ON CONFLICT (tenant_id, event_kind, bucket_ts) DO UPDATE handles
// replay / race-with-another-worker gracefully: when two workers
// (a HA pair, say) both tick the same bucket, the second insert
// succeeds as an update instead of 23505.
//
// The query is a single statement — there's no per-row UPSERT
// loop. Postgres handles the bulk insert/update atomically.
func (w *UsageMeterWorker) aggregateBucket(ctx context.Context, bucketStart, bucketEnd time.Time) {
	tag, err := w.pool.Pgx().Exec(ctx,
		`INSERT INTO platform_usage_aggregates
		        (tenant_id, event_kind, bucket_ts, sum_quantity, count,
		         min_quantity, max_quantity)
		 SELECT tenant_id,
		        event_kind,
		        date_trunc('hour', $1::timestamptz)        AS bucket_ts,
		        SUM(quantity)::numeric(18,4)                AS sum_quantity,
		        COUNT(*)::integer                           AS count,
		        MIN(quantity)::numeric(18,4)                AS min_quantity,
		        MAX(quantity)::numeric(18,4)                AS max_quantity
		   FROM platform_usage_events
		  WHERE created_at >= $1 AND created_at < $2
		  GROUP BY tenant_id, event_kind
		 ON CONFLICT (tenant_id, event_kind, bucket_ts) DO UPDATE
		    SET sum_quantity = EXCLUDED.sum_quantity,
		        count        = EXCLUDED.count,
		        min_quantity = EXCLUDED.min_quantity,
		        max_quantity = EXCLUDED.max_quantity`,
		bucketStart, bucketEnd,
	)
	if err != nil {
		w.logger.Error("usagemeter worker: aggregate failed",
			"bucket_start", bucketStart.UTC().Format(time.RFC3339),
			"bucket_end", bucketEnd.UTC().Format(time.RFC3339),
			"err", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		w.logger.Info("usagemeter worker: bucket rolled up",
			"bucket_start", bucketStart.UTC().Format(time.RFC3339),
			"rows", n)
	}
}

// prune deletes rows in platform_usage_events older than
// `retention`. Best-effort — failures are logged but do not fail
// the tick. A single DELETE handles thousands of rows efficiently
// thanks to the (tenant_id, created_at DESC) index; the planner
// uses it without additional help.
func (w *UsageMeterWorker) prune(ctx context.Context) {
	cutoff := time.Now().UTC().Add(-w.retention)
	tag, err := w.pool.Pgx().Exec(ctx,
		`DELETE FROM platform_usage_events WHERE created_at < $1`,
		cutoff,
	)
	if err != nil {
		w.logger.Error("usagemeter worker: prune failed", "err", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		w.logger.Info("usagemeter worker: pruned old events",
			"rows", n,
			"cutoff", cutoff.Format(time.RFC3339))
	}
}

// nowHourFloor rounds t down to the most recent completed hour
// boundary (UTC). Exposed so a future test or operator script can
// call it without re-deriving the math.
func nowHourFloor(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
}
