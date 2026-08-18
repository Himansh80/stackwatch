// Package synthetics — scheduler.
//
// RunScheduler is a goroutine-friendly ticker that wakes every
// `interval` (typically 30s), scans for checks where
// `now - last_run_at >= interval_sec`, and runs them in parallel
// (bounded by maxConcurrency).
//
// State is in the DB: synthetics_checks.last_run_at and last_status
// are updated after each run, so a restart picks up where it left off.
package synthetics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

// PoolExecutor is the slice of the DB pool we need. We accept
// *pgxpool.Pool directly.
type PoolExecutor interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgxpoolRows, error)
	Exec(ctx context.Context, sql string, args ...interface{}) (pgxpoolCommandTag, error)
}

// pgxpoolRows is the subset of pgxpool.Rows that we use.
type pgxpoolRows interface {
	Close()
	Next() bool
	Scan(...interface{}) error
	Err() error
}

// pgxpoolCommandTag is the subset of pgxpool.CommandTag that we use.
type pgxpoolCommandTag interface{}

// adapt wraps a *pgxpool.Pool as a PoolExecutor.
func adapt(p *pgxpool.Pool) PoolExecutor { return &pgxPoolAdapter{p: p} }

// pgxPoolAdapter wraps *pgxpool.Pool to expose the methods we need.
type pgxPoolAdapter struct{ p *pgxpool.Pool }

func (a *pgxPoolAdapter) Query(ctx context.Context, sql string, args ...interface{}) (pgxpoolRows, error) {
	return a.p.Query(ctx, sql, args...)
}
func (a *pgxPoolAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (pgxpoolCommandTag, error) {
	return a.p.Exec(ctx, sql, args...)
}

// Scheduler polls the DB and runs due checks.
type Scheduler struct {
	pool            PoolExecutor
	interval        time.Duration
	maxConcurrency  int
	logger          *slog.Logger
}

// NewScheduler returns a Scheduler. interval=30s and maxConcurrency=10
// are sane defaults.
func NewScheduler(pool *pgxpool.Pool, interval time.Duration, logger *slog.Logger) *Scheduler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Scheduler{
		pool:           adapt(pool),
		interval:       interval,
		maxConcurrency: 10,
		logger:         logger,
	}
}

// Run blocks until ctx is cancelled. Intended to be called as a
// goroutine: go scheduler.Run(ctx).
func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()
	s.logger.Info("synthetics scheduler started", "interval", s.interval)
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("synthetics scheduler stopping")
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

// tick runs one pass: fetch due checks, run them in parallel, persist.
func (s *Scheduler) tick(ctx context.Context) {
	checks, err := s.fetchDue(ctx)
	if err != nil {
		s.logger.Error("fetch due checks", "err", err)
		return
	}
	if len(checks) == 0 {
		return
	}
	s.logger.Info("running synthetics", "n_due", len(checks))

	// Bounded concurrency via semaphore channel
	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup
	for _, c := range checks {
		wg.Add(1)
		sem <- struct{}{}
		go func(c Check) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOne(ctx, c)
		}(c)
	}
	wg.Wait()
}

// fetchDue returns enabled checks where last_run_at is older than
// the check's interval_sec (or never run).
func (s *Scheduler) fetchDue(ctx context.Context) ([]Check, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, tenant_id, name, kind, target, timeout_ms
		FROM synthetics_checks
		WHERE enabled = true
		  AND (last_run_at IS NULL
		       OR last_run_at < NOW() - (interval_sec || ' seconds')::interval)
		ORDER BY last_run_at NULLS FIRST
		LIMIT 100`, // safety cap
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Check{}
	for rows.Next() {
		var id, tid uuid.UUID
		var name, target, kind string
		var timeoutMs int
		if err := rows.Scan(&id, &tid, &name, &kind, &target, &timeoutMs); err != nil {
			continue
		}
		out = append(out, Check{
			ID:        id.String(),
			TenantID:  tid.String(),
			Name:      name,
			Kind:      Kind(kind),
			Target:    target,
			TimeoutMs: timeoutMs,
			Enabled:   true,
		})
	}
	return out, nil
}

// runOne executes a check and persists the result + updates last_run_at.
func (s *Scheduler) runOne(ctx context.Context, c Check) {
	result := Run(ctx, c)

	// Persist to synthetics_results
	checkID, _ := uuid.Parse(c.ID)
	tenantID, _ := uuid.Parse(c.TenantID)
	_, err := s.pool.Exec(ctx,
		`INSERT INTO synthetics_results (check_id, tenant_id, status, response_ms, status_code, error)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		checkID, tenantID, result.Status, result.ResponseMs, result.StatusCode, result.Error)
	if err != nil {
		s.logger.Error("insert synthetics_result", "check", c.Name, "err", err)
	}

	// Update check's last_run_at + last_status
	_, err = s.pool.Exec(ctx,
		`UPDATE synthetics_checks
		 SET last_run_at = NOW(), last_status = $1, last_ms = $2
		 WHERE id = $3`,
		result.Status, result.ResponseMs, checkID)
	if err != nil {
		s.logger.Error("update synthetics_check", "check", c.Name, "err", err)
	}

	s.logger.Info("synthetics check done",
		"check", c.Name, "kind", string(c.Kind),
		"status", result.Status, "ms", result.ResponseMs,
	)
}
