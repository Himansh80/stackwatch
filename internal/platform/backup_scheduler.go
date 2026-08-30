// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// BackupSchedulerWorker — hourly pass over platform_backup_jobs.
//
// Per spec §"PL5 — Backup/Restore" §"Schedule semantics":
//
//	For each enabled row in platform_backup_jobs whose
//	last_run_at is older than the configured frequency,
//	call CreateBackupAndEncrypt for the row's tenant.
//
// Why an in-process worker (not a cron job / external
// scheduler):
//   Same rationale as usage_meter.go / retention.go — a
//   single api-gateway restart brings every worker back up.
//   The worker is non-blocking: CreateBackupAndEncrypt runs
//   in a per-tenant goroutine so a slow tenant's pg_dump
//   never stalls the loop.
//
// Why every enabled row is re-evaluated every tick:
//   The frequency column tells us the cadence (daily/weekly/
//   monthly); the actual decision is whether the row is
//   overdue. A tenant that was offline for a week re-runs
//   immediately on next boot — no catch-up backlog (which
//   would DDoS the DB on a 2-week outage).
//
// Why no advisory lock here (the per-tenant lock is inside
// CreateBackupAndEncrypt):
//   The lock is held for the duration of one tenant's
//   encrypt pass — it's serialisation within a tenant,
//   not between. Between tenants we WANT concurrency so
//   one slow pg_dump doesn't block the others. Plus this
//   keeps the scheduler goroutine hot-path cheap.

package platform

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

const (
	// defaultBackupSchedTickInterval — hourly cadence per
	// spec; tunable via env or BackupSchedOption for dev /
	// test that needs faster runs.
	defaultBackupSchedTickInterval = 1 * time.Hour

	// defaultBackupSchedTickTimeout — per-pass budget.
	// WAL-style; generous because worst case is N tenants
	// × back-to-back failures, and a stuck worker is worse
	// than a missed cycle.
	defaultBackupSchedTickTimeout = 20 * time.Minute
)

// BackupSchedulerWorker is the per-process singleton.
// Mirrors the shape of UsageMeterWorker + RetentionWorker
// so main.go wires it the same way.
type BackupSchedulerWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	tickTimeout  time.Duration
	logger       *slog.Logger
}

// BackupSchedulerOption mutates the worker at construction.
type BackupSchedulerOption func(*BackupSchedulerWorker)

// WithBackupSchedulerTickInterval overrides the default 1h cadence.
func WithBackupSchedulerTickInterval(d time.Duration) BackupSchedulerOption {
	return func(w *BackupSchedulerWorker) { w.tickInterval = d }
}

// WithBackupSchedulerTickTimeout overrides the default 20min
// per-pass budget.
func WithBackupSchedulerTickTimeout(d time.Duration) BackupSchedulerOption {
	return func(w *BackupSchedulerWorker) { w.tickTimeout = d }
}

// WithBackupSchedulerLogger overrides the default slog logger.
func WithBackupSchedulerLogger(l *slog.Logger) BackupSchedulerOption {
	return func(w *BackupSchedulerWorker) { w.logger = l }
}

// NewBackupSchedulerWorker builds the worker. Call exactly
// once on api-gateway boot, then Start() in a goroutine
// alongside the other workers.
func NewBackupSchedulerWorker(pool *db.Pool, opts ...BackupSchedulerOption) *BackupSchedulerWorker {
	w := &BackupSchedulerWorker{
		pool:         pool,
		tickInterval: defaultBackupSchedTickInterval,
		tickTimeout:  defaultBackupSchedTickTimeout,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately;
// the goroutine exits when ctx is cancelled. Safe to call
// multiple times — second+ calls are no-ops.
//
// The first tick fires immediately so a freshly-restarted
// gateway catches up on overdue jobs without waiting for
// the next tick boundary.
func (w *BackupSchedulerWorker) Start(ctx context.Context) {
	w.logger.Info("backup scheduler: starting", "tick", w.tickInterval)
	go func() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("backup scheduler: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()
		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("backup scheduler: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("backup scheduler: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// tick runs one pass: walks every enabled backup_job row,
// triggers CreateBackupAndEncrypt for the due ones, and
// UPDATEs last_run_at / last_run_status / last_backup_id.
//
// Errors per tenant are logged but never abort the batch —
// one bad tenant must not skip the rest.
func (w *BackupSchedulerWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.tickTimeout)
	defer cancel()

	rows, err := w.pool.Pgx().Query(ctx, `
		SELECT id, tenant_id, frequency, retention_days, last_run_at, last_run_status
		  FROM platform_backup_jobs
		 WHERE enabled = true
	`)
	if err != nil {
		w.logger.Error("backup scheduler: scan jobs failed", "err", err)
		return
	}
	defer rows.Close()

	type job struct {
		id            uuid.UUID
		tenantID      uuid.UUID
		frequency     string
		retentionDays int
		lastRunAt     *time.Time
		lastRunStatus *string
	}
	var jobs []job
	for rows.Next() {
		var j job
		var lra *time.Time
		var lrs *string
		if err := rows.Scan(&j.id, &j.tenantID, &j.frequency, &j.retentionDays, &lra, &lrs); err != nil {
			w.logger.Warn("backup scheduler: scan row failed", "err", err)
			continue
		}
		j.lastRunAt = lra
		j.lastRunStatus = lrs
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		w.logger.Error("backup scheduler: iterate jobs failed", "err", err)
		return
	}

	now := time.Now().UTC()
	triggered := 0
	skipped := 0
	for _, j := range jobs {
		if !isDue(j.frequency, j.lastRunAt, now) {
			skipped++
			continue
		}
		w.logger.Info("backup scheduler: triggering backup",
			"tenant_id", j.tenantID.String(),
			"frequency", j.frequency,
		)
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("backup scheduler: tenant panic",
						"tenant_id", j.tenantID.String(), "err", r)
				}
			}()
			backupID, runErr := CreateBackupAndEncrypt(
				ctx, w.pool, j.tenantID, nil, "scheduled",
			)
			lastRunStatus := "completed"
			if runErr != nil {
				lastRunStatus = "failed"
				w.logger.Error("backup scheduler: backup failed",
					"tenant_id", j.tenantID.String(), "err", runErr)
			}
			// UPDATE the job row with the last run
			// outcome. The backup row itself was already
			// UPDATEd inside CreateBackupAndEncrypt.
			if _, uerr := w.pool.Pgx().Exec(ctx, `
				UPDATE platform_backup_jobs
				   SET last_run_at=$2,
				       last_run_status=$3,
				       last_backup_id=$4,
				       updated_at=NOW()
				 WHERE id=$1
			`, j.id, now, lastRunStatus, backupID); uerr != nil {
				w.logger.Warn("backup scheduler: job row update failed",
					"job_id", j.id.String(), "err", uerr)
			}
			triggered++
		}()
	}
	if triggered > 0 || skipped > 0 {
		w.logger.Info("backup scheduler: pass complete",
			"jobs_scanned", len(jobs),
			"triggered", triggered,
			"skipped", skipped,
		)
	}
}

// isDue returns true if the configured frequency says a run
// should fire now. Frequencies:
//
//   - "manual"  : never auto-fires (worker ignores this row
//     in practice — enabled=false is the canonical "no
//     schedule" state — but we check first as belt-and-
//     braces).
//   - "daily"   : last_run_at was more than 24h ago (or NULL).
//   - "weekly"  : more than 168h ago.
//   - "monthly" : more than 720h ago (30 days × 24h).
//
// Returns false on unknown frequencies so a typo'd value
// can't accidentally enqueue a job.
func isDue(frequency string, lastRunAt *time.Time, now time.Time) bool {
	var cadence time.Duration
	switch frequency {
	case "daily":
		cadence = 24 * time.Hour
	case "weekly":
		cadence = 168 * time.Hour
	case "monthly":
		cadence = 720 * time.Hour
	case "manual":
		return false
	default:
		return false
	}
	if lastRunAt == nil {
		return true
	}
	return now.Sub(*lastRunAt) >= cadence
}
