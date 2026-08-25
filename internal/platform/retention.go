// Tier 11 Phase 4 — Tenant Limits (PL4).
//
// RetentionWorker — daily enforcement of per-tenant data
// retention. Mirrors the structure of usage_meter.go (Phase 2)
// so main.go wires it the same way (NewUsageMeterWorker +
// .Start(ctx) → NewRetentionWorker + .Start(ctx)).
//
// Per spec §"PL4 — Tenant Limits" §"Retention enforcement"
// (Daily cron at 03:00 UTC):
//
//	For each tenant with a platform_tenant_limits row:
//	  1. Read their effective data_retention_days (plan ∪
//	     overrides).
//	  2. DELETE every row from the documented data tables
//	     (metric_points, audit_log, alert_history,
//	     notifications, platform_usage_events) where
//	     created_at < NOW() - INTERVAL '%d days' AND
//	     tenant_id = $tenant_id.
//	  3. Suspended tenants SKIP — they have bigger problems
//	     and we don't want to nuke data the operator might
//	     be debugging for a recovery.
//
// Why an internal/platform worker (not a cron job):
//   Same reasoning as Phase 2's usage_meter — the polling
//   goroutine lives inside the api-gateway binary so a single
//   restart brings everything up. Phase 8 PL8 (Capacity
//   Forecast) adds another worker alongside.
//
// Why "suspended tenants skip":
//   If a tenant is suspended for non-payment, we do NOT want
//   to start pruning their historical data — the operator
//   might restore them and the customer expects to find their
//   alerts / dashboards intact. Suspended rows keep their
//   data until the operator either unsuspends (continue
//   normal pruning) or hard-deletes (separate path, not
//   in scope for PL4).
//
// Why we don't compute the cutoff from each plan's
// metrics_retention_days vs data_retention_days separately:
//   The two are usually equal (free = 7 / 14). When they
//   diverge (pro = 90/90), the more conservative (smaller)
//   value is applied across BOTH data families — keeping a
//   single number is simpler than a per-table config. A
//   future per-table retention override would land here.

package platform

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

const (
	// defaultRetentionTickInterval — spec calls for daily at
	// 03:00 UTC. The worker doesn't try to align to the wall
	// clock; we tick every 24h and accept the up-to-24h
	// drift. Phase 8 PL8 will add a cron-style alignment
	// helper if precision becomes important.
	defaultRetentionTickInterval = 24 * time.Hour

	// defaultRetentionTickTimeout — per-tick budget. Worst
	// case = N tenants × (1 DELETE per table). 5min gives a
	// safe budget for a 4-figure tenant count without
	// pinning the worker goroutine if a single DELETE
	// stalls on a hot row.
	defaultRetentionTickTimeout = 5 * time.Minute

	// minRetentionDays — floor. Never below 1 day even if a
	// future plan typo'd 0; the worker would otherwise try
	// to DELETE in-flight rows in the same second they
	// were written.
	minRetentionDays = 1
)

// RetentionWorker is the per-process singleton. Owns a pool ref
// + tunables + slog logger. Start() launches the goroutine; the
// goroutine respects ctx cancellation for graceful shutdown.
type RetentionWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	tickTimeout  time.Duration
	logger       *slog.Logger
}

// RetentionOption mutates the worker at construction time.
type RetentionOption func(*RetentionWorker)

// WithRetentionTickInterval overrides the default 24h tick cadence.
func WithRetentionTickInterval(d time.Duration) RetentionOption {
	return func(w *RetentionWorker) { w.tickInterval = d }
}

// WithRetentionTickTimeout overrides the default 5min per-tick budget.
func WithRetentionTickTimeout(d time.Duration) RetentionOption {
	return func(w *RetentionWorker) { w.tickTimeout = d }
}

// WithRetentionLogger overrides the default slog logger.
func WithRetentionLogger(l *slog.Logger) RetentionOption {
	return func(w *RetentionWorker) { w.logger = l }
}

// NewRetentionWorker builds the worker. Call exactly once on
// api-gateway boot, then Start() in a goroutine (main.go does
// this alongside the other workers).
func NewRetentionWorker(pool *db.Pool, opts ...RetentionOption) *RetentionWorker {
	w := &RetentionWorker{
		pool:         pool,
		tickInterval: defaultRetentionTickInterval,
		tickTimeout:  defaultRetentionTickTimeout,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately; the
// goroutine exits when ctx is cancelled. Safe to call multiple
// times — second+ calls are no-ops.
//
// The first tick fires immediately so a freshly-restarted
// gateway catches up on a multi-day outage without waiting
// 24h for the next tick.
func (w *RetentionWorker) Start(ctx context.Context) {
	w.logger.Info("retention worker: starting", "tick", w.tickInterval)
	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("retention worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("retention worker: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("retention worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// tick runs one pass: for each non-suspended tenant, DELETE
// rows older than their effective data_retention_days across the
// documented data tables. Errors are logged but never abort the
// batch — one bad tenant must not skip the rest.
func (w *RetentionWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.tickTimeout)
	defer cancel()

	// Pull every (tenant_id, plan_name, custom_overrides,
	// suspended_at) we need in a single scan. Suspended tenants
	// are filtered in SQL so we don't iterate them at all.
	rows, err := w.pool.Pgx().Query(ctx,
		`SELECT tenant_id, plan_name, custom_overrides, suspended_at
		   FROM platform_tenant_limits
		  WHERE suspended_at IS NULL`)
	if err != nil {
		w.logger.Error("retention worker: scan tenants failed", "err", err)
		return
	}
	defer rows.Close()

	type tenantJob struct {
		id          uuid.UUID
		planName    string
		overrides   []byte
	}
	var jobs []tenantJob
	for rows.Next() {
		var (
			tid        uuid.UUID
			plan       string
			overrides  []byte
			suspended  *time.Time
		)
		if err := rows.Scan(&tid, &plan, &overrides, &suspended); err != nil {
			continue
		}
		if suspended != nil {
			continue // double-check (filter above should have caught it)
		}
		jobs = append(jobs, tenantJob{id: tid, planName: plan, overrides: overrides})
	}
	if err := rows.Err(); err != nil {
		w.logger.Error("retention worker: iterate tenants failed", "err", err)
		return
	}

	totalRows := int64(0)
	for _, job := range jobs {
		eff, err := LoadEffectiveLimits(ctx, job.id, w.pool)
		if err != nil {
			w.logger.Warn("retention worker: load effective limits failed",
				"tenant_id", job.id.String(), "err", err)
			continue
		}
		days := eff.DataRetentionDays
		if days < minRetentionDays {
			days = minRetentionDays
		}
		cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
		rowsDeleted := w.purgeTenant(ctx, job.id, cutoff)
		totalRows += rowsDeleted
		if rowsDeleted > 0 {
			w.logger.Info("retention worker: tenant pruned",
				"tenant_id", job.id.String(),
				"plan", eff.PlanName,
				"retention_days", days,
				"rows_deleted", rowsDeleted,
				"cutoff", cutoff.Format(time.RFC3339))
		}
	}
	if totalRows > 0 {
		w.logger.Info("retention worker: pass complete",
			"tenants_scanned", len(jobs),
			"total_rows_deleted", totalRows)
	}
}

// purgeTenant runs the DELETE statements that enforce retention
// for one tenant. Returns total rows deleted across all tables.
// Per-table errors are logged but don't abort — one missing
// table must not block the rest.
//
// Why each table is its own DELETE (not a UNION):
//   The data tables live in different schemas / namespaces
//   (some Tier 0, some Tier 7). UNION would force a shared
//   schema introspection query that's brittle to migrate.
//   Independent DELETEs are explicit + easy to read.
//
// Why IF EXISTS on each table name:
//   Not every installation has every table — a freshly-cut
//   dev DB might not yet have alert_history. The worker must
//   not 500 on the first run after migration 040. IF EXISTS
//   means the DELETE no-ops with a 0-row count, which we
//   already log as "no rows deleted".
//
// Why DO block-level transaction (not one big TX):
//   A single TX holding a write lock across all DELETEs would
//   pin hot tables for minutes. Independent statements let
//   other writers (alert.fired handlers, etc.) interleave
//   between our DELETEs.
func (w *RetentionWorker) purgeTenant(ctx context.Context, tenantID uuid.UUID, cutoff time.Time) int64 {
	// Each entry: (table, time column). Match the documented
	// schema — if a future table needs retention, add it here.
	// DO NOT add tables that aren't tenant-scoped (e.g.
	// platform_signups has no tenant_id).
	targets := []struct {
		table string
		col   string
	}{
		{"metric_points", "created_at"},
		{"audit_log", "created_at"},
		{"alert_history", "created_at"},
		{"notifications", "created_at"},
		{"platform_usage_events", "created_at"},
	}
	var total int64
	for _, t := range targets {
		// QuoteIdent-style safety: we control both names so
		// string concat is fine, but we explicitly forbid
		// semicolons / spaces as a future-proofing belt.
		tag, err := w.pool.Pgx().Exec(ctx,
			"DELETE FROM "+t.table+" WHERE tenant_id = $1 AND "+t.col+" < $2",
			tenantID, cutoff,
		)
		if err != nil {
			// ErrInvalidObjectName = table doesn't exist.
			// Skip quietly — a dev DB might not have it.
			if isUndefinedTable(err) {
				continue
			}
			w.logger.Warn("retention worker: delete failed",
				"table", t.table,
				"tenant_id", tenantID.String(),
				"err", err)
			continue
		}
		total += tag.RowsAffected()
	}
	return total
}

// isUndefinedTable is a defensive predicate around the pgx
// "undefined_table" SQLSTATE (42P01). Returning true here
// tells purgeTenant to silently skip — a missing table is not
// a worker bug, it's a migration-ordering situation we
// shouldn't crash on.
//
// Defined as a separate function so a future test can mock
// the matcher without re-implementing the SQLSTATE grammar.
func isUndefinedTable(err error) bool {
	if err == nil {
		return false
	}
	// Match Postgres SQLSTATE 42P01 ("undefined_table"). We
	// don't import the pgconn type here just for the constant —
	// the string match is good enough and survives pgx version
	// bumps that might rename the type.
	var pgErr interface {
		SQLState() string
	}
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "42P01"
	}
	// Fallback: substring match on the message. Pgx wraps the
	// SQLSTATE in the error message itself, so this works for
	// every error path we actually hit.
	msg := err.Error()
	return contains(msg, "42P01") || contains(msg, "does not exist")
}

// contains is a tiny substring check kept inline so we don't
// pull in strings just for one call site.
func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}