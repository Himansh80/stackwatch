// Package homelab — SchedulerWorker (Tier 10 Phase 9 / H10 —
// Task Scheduler).
//
// SchedulerWorker ticks every minute (60s) and dispatches
// every enabled job whose next_run_at <= now() to a goroutine
// that calls RunSchedulerJobAndInsertRun (in scheduler_run.go).
// Per-job errors are logged but never abort the batch — one
// bad job must not skip the rest.
//
// Worker must not crash on a bad job. Each per-job dispatch
// is wrapped in a recover() so a hung HTTP connection, a
// panic inside the parser, or a DB disconnect can never
// take the worker down.
//
// Per-user (NOT per-tenant): the tick iterates over ALL
// (tenant, user) pairs in the table. The (user_id) index
// makes the SELECT cheap on the (typically small) per-user
// job set.
package homelab

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

const (
	// defaultSchedulerTickInterval matches spec §H10 — "60s".
	defaultSchedulerTickInterval = 60 * time.Second

	// defaultSchedulerMaxPar caps parallel job dispatches so
	// a user with 25 jobs that all fire on the same minute
	// doesn't drown the connection pool. 8 matches the
	// RssWorker ceiling (Phase 8).
	defaultSchedulerMaxPar = 8

	// defaultSchedulerPerTickBudget caps the whole tick so a
	// pathological job never hangs the loop forever.
	defaultSchedulerPerTickBudget = 4 * time.Minute
)

// SchedulerWorker is the per-process singleton. Start() launches
// the goroutine; the goroutine respects ctx cancellation for
// graceful shutdown.
type SchedulerWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	maxPar       int
	perTickTime  time.Duration
	logger       *slog.Logger
}

// SchedulerOption mutates the worker at construction time.
type SchedulerOption func(*SchedulerWorker)

// WithSchedulerTickInterval overrides the default 60s cadence.
func WithSchedulerTickInterval(d time.Duration) SchedulerOption {
	return func(w *SchedulerWorker) { w.tickInterval = d }
}

// WithSchedulerMaxPar overrides the parallel-dispatch cap.
func WithSchedulerMaxPar(n int) SchedulerOption {
	return func(w *SchedulerWorker) { w.maxPar = n }
}

// WithSchedulerLogger overrides the default slog logger.
func WithSchedulerLogger(l *slog.Logger) SchedulerOption {
	return func(w *SchedulerWorker) { w.logger = l }
}

// NewSchedulerWorker builds the worker. Call once on
// api-gateway boot, then Start() in a goroutine.
func NewSchedulerWorker(pool *db.Pool, opts ...SchedulerOption) *SchedulerWorker {
	w := &SchedulerWorker{
		pool:         pool,
		tickInterval: defaultSchedulerTickInterval,
		maxPar:       defaultSchedulerMaxPar,
		perTickTime:  defaultSchedulerPerTickBudget,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately;
// the goroutine exits when ctx is cancelled. The first tick
// fires immediately so a freshly-restarted gateway catches up
// on any jobs whose next_run_at already passed.
func (w *SchedulerWorker) Start(ctx context.Context) {
	w.logger.Info("scheduler worker: starting",
		"tick", w.tickInterval,
		"max_par", w.maxPar)

	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("scheduler worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("scheduler worker: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("scheduler worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// schedulerDispatch is the small payload the worker pulls for
// each row — keeps the goroutine closure small.
type schedulerDispatch struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	UserID   uuid.UUID
	Name     string
}

// tick runs one pass: fetch every enabled job whose
// next_run_at is due, dispatch each to a bounded goroutine
// pool. Per-job errors are logged but never abort the batch.
func (w *SchedulerWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.perTickTime)
	defer cancel()

	dispatchList, err := w.fetchDueJobs(ctx)
	if err != nil {
		w.logger.Error("scheduler worker: fetch due jobs failed", "err", err)
		return
	}
	if len(dispatchList) == 0 {
		return
	}
	w.logger.Info("scheduler worker: dispatching", "jobs", len(dispatchList))

	sem := make(chan struct{}, w.maxPar)
	var wg sync.WaitGroup
	for _, d := range dispatchList {
		wg.Add(1)
		sem <- struct{}{}
		go func(d schedulerDispatch) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("scheduler worker: dispatch panic",
						"job_id", d.ID, "name", d.Name, "err", r)
				}
			}()
			out, derr := RunSchedulerJobAndInsertRun(ctx, w.pool, d.ID, w.logger)
			if derr != nil {
				w.logger.Error("scheduler worker: dispatch failed",
					"job_id", d.ID, "name", d.Name, "err", derr)
				return
			}
			if out.Status == "skipped" {
				w.logger.Info("scheduler worker: skipped",
					"job_id", d.ID, "reason", out.SkippedReason)
			}
		}(d)
	}
	wg.Wait()
}

// fetchDueJobs returns the (id, tenant_id, user_id, name)
// of every enabled job whose next_run_at <= now(). Uses the
// partial index idx_homelab_scheduler_jobs_due.
func (w *SchedulerWorker) fetchDueJobs(ctx context.Context) ([]schedulerDispatch, error) {
	rows, err := w.pool.Pgx().Query(ctx,
		`SELECT id::text, tenant_id::text, user_id::text, name
		   FROM homelab_scheduler_jobs
		  WHERE enabled = TRUE
		    AND next_run_at IS NOT NULL
		    AND next_run_at <= NOW()
		  ORDER BY next_run_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("query due jobs: %w", err)
	}
	defer rows.Close()

	out := []schedulerDispatch{}
	for rows.Next() {
		var d schedulerDispatch
		var idStr, tStr, uStr string
		if err := rows.Scan(&idStr, &tStr, &uStr, &d.Name); err != nil {
			return nil, fmt.Errorf("scan due job: %w", err)
		}
		if v, perr := uuid.Parse(idStr); perr == nil {
			d.ID = v
		}
		if v, perr := uuid.Parse(tStr); perr == nil {
			d.TenantID = v
		}
		if v, perr := uuid.Parse(uStr); perr == nil {
			d.UserID = v
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
