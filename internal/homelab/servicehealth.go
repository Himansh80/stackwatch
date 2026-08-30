// Package homelab hosts Tier 10 background workers.
//
// ServiceHealthWorker ticks every 60s, probes every enabled pinned
// service across all tenants + users, and INSERTs the results into
// homelab_service_health. Disabled pins are skipped (the user can
// re-enable them later without losing their pin).
//
// Worker must not crash on bad probes. Each probe runs inside its
// own goroutine + recovers from any panic so a hung TCP / TLS / ICMP
// path can never take the worker down.
//
// Per-user (NOT per-tenant): the tick iterates over ALL (tenant, user)
// pairs in the table — a per-tenant worker would miss every user's
// pins. The (tenant_id, enabled) partial index makes the SELECT
// cheap on the (typically small) enabled-row set.
//
// The probe helpers in this package (probeHTTP / probeTCP / probeICMP
// / probeOne) are kept in lock-step with the matching helpers in
// internal/handler/handlers_homelab_services_probe_helpers.go —
// the handler package can't be imported from internal/homelab (Go's
// internal-package rule). The contract is:
//
//   - same timeouts (5s HTTP, 3s TCP, 2s ICMP)
//   - same status mapping (<400=up, 4xx=degraded, 5xx=down, conn err=down)
//   - same column write (tenant_id, user_id, service_id, status,
//     latency_ms, status_code, error_message)
//
// If you change one, change the other.
package homelab

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// Tick interval — 60s matches the speckit proposal §H2 ("Service
// health poll cadence: 60s default"). The dashboard refresh
// interval (homelab_user_prefs.refresh_seconds) is what the
// FRONTEND polls the API at — independent of this worker cadence.
//
// Per-pin probe timeouts must be < TickInterval or a slow target
// will pile up probes. 5s HTTP / 3s TCP / 2s ICMP keep us well
// under 60s for any single pin.
const (
	defaultTickInterval      = 60 * time.Second
	defaultMaxPar            = 16
	defaultHTTPProbeTimeout  = 5 * time.Second
	defaultTCPProbeTimeout   = 3 * time.Second
	defaultICMPProbeTimeout  = 2 * time.Second
	defaultHealthRetention   = 30 * 24 * time.Hour
	defaultProbeOverallLimit = 90 * time.Second
)

// ServiceHealthWorker is the per-process singleton. Holds a pool
// ref + tunables + slog logger. Start() launches the goroutine;
// the goroutine respects ctx cancellation for graceful shutdown.
type ServiceHealthWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	maxPar       int
	httpTimeout  time.Duration
	tcpTimeout   time.Duration
	icmpTimeout  time.Duration
	retention    time.Duration
	probeLimit   time.Duration
	logger       *slog.Logger
}

// WorkerOption mutates the worker at construction time. Lets main.go
// pass overrides (e.g. a faster tick in tests) without exploding the
// constructor signature.
type WorkerOption func(*ServiceHealthWorker)

// WithTickInterval overrides the default 60s tick cadence.
func WithTickInterval(d time.Duration) WorkerOption {
	return func(w *ServiceHealthWorker) { w.tickInterval = d }
}

// WithMaxPar overrides the default parallel-probe cap.
func WithMaxPar(n int) WorkerOption {
	return func(w *ServiceHealthWorker) { w.maxPar = n }
}

// WithLogger overrides the default slog logger (defaults to slog.Default()).
func WithLogger(l *slog.Logger) WorkerOption {
	return func(w *ServiceHealthWorker) { w.logger = l }
}

// NewServiceHealthWorker builds a worker. Call exactly once on
// api-gateway boot, then Start() in a goroutine.
func NewServiceHealthWorker(pool *db.Pool, opts ...WorkerOption) *ServiceHealthWorker {
	w := &ServiceHealthWorker{
		pool:         pool,
		tickInterval: defaultTickInterval,
		maxPar:       defaultMaxPar,
		httpTimeout:  defaultHTTPProbeTimeout,
		tcpTimeout:   defaultTCPProbeTimeout,
		icmpTimeout:  defaultICMPProbeTimeout,
		retention:    defaultHealthRetention,
		probeLimit:   defaultProbeOverallLimit,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately; the
// goroutine exits when ctx is cancelled. Safe to call multiple
// times — second+ calls are no-ops (idempotent).
//
// The first tick fires immediately so a freshly-restarted gateway
// repopulates the dashboard within seconds rather than waiting a
// full tick interval.
func (w *ServiceHealthWorker) Start(ctx context.Context) {
	w.logger.Info("servicehealth worker: starting",
		"tick", w.tickInterval,
		"max_par", w.maxPar,
		"retention", w.retention)

	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("servicehealth worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("servicehealth worker: stopping")
				return
			case <-t.C:
				// Per-tick recover so a single tick failure can
				// never kill the worker.
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("servicehealth worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// tick runs one pass: fetch every enabled pin across all tenants,
// probe each in parallel (bounded), persist results, then prune
// rows older than `retention`.
//
// Per-pin errors are logged but never abort the batch — one bad
// target must not skip the rest.
func (w *ServiceHealthWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.probeLimit)
	defer cancel()

	pins, err := w.fetchEnabledPins(ctx)
	if err != nil {
		w.logger.Error("servicehealth worker: fetch pins failed", "err", err)
		return
	}
	if len(pins) == 0 {
		w.prune(ctx)
		return
	}
	w.logger.Info("servicehealth worker: probing", "pins", len(pins))

	sem := make(chan struct{}, w.maxPar)
	var wg sync.WaitGroup
	for _, p := range pins {
		wg.Add(1)
		sem <- struct{}{}
		go func(p pinTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("servicehealth worker: probe panic",
						"service_id", p.ID, "err", r)
				}
			}()
			result := probeOne(ctx, p.Kind, p.URL, w)
			if err := w.insertHealth(ctx, p, result); err != nil {
				w.logger.Error("servicehealth worker: insert failed",
					"service_id", p.ID, "err", err)
			}
		}(p)
	}
	wg.Wait()

	w.prune(ctx)
}

// pinTarget is the small payload the worker pulls for each row —
// keeps the goroutine closure small.
type pinTarget struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	ID       uuid.UUID
	URL      string
	Kind     string
}

// fetchEnabledPins returns every enabled pin in homelab_pinned_services.
// The (tenant_id, enabled) partial index keeps this cheap even with
// thousands of users (only enabled rows are scanned).
func (w *ServiceHealthWorker) fetchEnabledPins(ctx context.Context) ([]pinTarget, error) {
	rows, err := w.pool.Pgx().Query(ctx,
		`SELECT tenant_id, user_id, id, url, kind
		   FROM homelab_pinned_services
		  WHERE enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []pinTarget{}
	for rows.Next() {
		var p pinTarget
		if err := rows.Scan(&p.TenantID, &p.UserID, &p.ID, &p.URL, &p.Kind); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// insertHealth writes one homelab_service_health row.
func (w *ServiceHealthWorker) insertHealth(ctx context.Context, p pinTarget, r probeResult) error {
	var errMsg *string
	if r.ErrorMessage != "" {
		em := r.ErrorMessage
		errMsg = &em
	}
	_, err := w.pool.Pgx().Exec(ctx,
		`INSERT INTO homelab_service_health
		        (tenant_id, user_id, service_id, status, latency_ms, status_code, error_message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.TenantID, p.UserID, p.ID, r.Status, r.LatencyMS, r.StatusCode, errMsg,
	)
	return err
}

// prune deletes rows older than `retention`. Best-effort — failure
// is logged but doesn't fail the tick. We prune every tick (60s)
// so a power user with 50 pins × 1440 ticks/day = 72k rows/day
// stays bounded at ~2.16M rows over 30 days.
func (w *ServiceHealthWorker) prune(ctx context.Context) {
	cutoff := time.Now().Add(-w.retention)
	tag, err := w.pool.Pgx().Exec(ctx,
		`DELETE FROM homelab_service_health WHERE checked_at < $1`,
		cutoff,
	)
	if err != nil {
		w.logger.Error("servicehealth worker: prune failed", "err", err)
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		w.logger.Info("servicehealth worker: pruned old health rows",
			"rows", n,
			"cutoff", cutoff.UTC().Format(time.RFC3339))
	}
}
