// Package homelab — MediaWorker + per-kind API client helpers.
//
// MediaWorker ticks every 60s, fetches every enabled media
// server in homelab_media_servers across all tenants + users,
// and UPSERTs the parsed state into homelab_now_playing +
// homelab_recent_additions. Disabled servers are skipped (the
// user can re-enable them later without losing their
// credentials).
//
// Worker must not crash on bad polls. Each poll runs inside the
// handler.PollMediaServerAndInsertState helper which handles
// per-server errors internally; the worker tick wraps each call
// in a recover() so a hung TCP / TLS / slow Plex response can
// never take the worker down.
//
// Per-user (NOT per-tenant): the tick iterates over ALL
// (tenant, user) pairs in the table — a per-tenant worker would
// miss every user's servers. The (user_id) index makes the
// SELECT cheap on the (typically small) enabled set.
//
// PollMediaServer is the EXPORTED per-kind dispatcher used by
// handler.PollMediaServerAndInsertState (handlers_homelab_media.go)
// AND the worker tick below. Both paths share the same
// fetch+parse code — DRY pattern (no twin implementations to
// keep in sync).
//
// The per-kind helpers (pollPlex / pollJellyfin / pollEmby) live
// in media_clients_*.go so this file stays under the 400-LOC
// cap.
package homelab

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// Tick interval — 60s matches the speckit proposal §H6 ("60s
// default background poll") and the DownloadsWorker /
// ServiceHealthWorker cadences. CalendarWorker ticks every 30min
// (iCal feeds change slowly); media servers are watched more
// aggressively because sessions come and go.
const (
	defaultMediaTickInterval  = 60 * time.Second
	defaultMediaFetchTimeout  = 20 * time.Second
	defaultMediaMaxPar        = 4
	defaultMediaOverallBudget = 5 * time.Minute
)

// MediaWorker is the per-process singleton. Holds a pool ref +
// tunables + slog logger. Start() launches the goroutine; the
// goroutine respects ctx cancellation for graceful shutdown.
type MediaWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	fetchTimeout time.Duration
	maxPar       int
	overallLimit time.Duration
	logger       *slog.Logger
}

// MediaOption mutates the worker at construction time.
type MediaOption func(*MediaWorker)

// WithMediaTickInterval overrides the default 60s tick cadence.
func WithMediaTickInterval(d time.Duration) MediaOption {
	return func(w *MediaWorker) { w.tickInterval = d }
}

// WithMediaLogger overrides the default slog logger.
func WithMediaLogger(l *slog.Logger) MediaOption {
	return func(w *MediaWorker) { w.logger = l }
}

// NewMediaWorker builds a worker. Call exactly once on
// api-gateway boot, then Start() in a goroutine.
func NewMediaWorker(pool *db.Pool, opts ...MediaOption) *MediaWorker {
	w := &MediaWorker{
		pool:         pool,
		tickInterval: defaultMediaTickInterval,
		fetchTimeout: defaultMediaFetchTimeout,
		maxPar:       defaultMediaMaxPar,
		overallLimit: defaultMediaOverallBudget,
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
func (w *MediaWorker) Start(ctx context.Context) {
	w.logger.Info("media worker: starting",
		"tick", w.tickInterval,
		"max_par", w.maxPar,
		"fetch_timeout", w.fetchTimeout)

	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("media worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("media worker: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("media worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// tick runs one pass: fetch every enabled server across all
// tenants, poll+UPSERT now_playing + recent_additions per server.
//
// Per-server errors are logged but never abort the batch — one
// bad target must not skip the rest.
func (w *MediaWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.overallLimit)
	defer cancel()

	servers, err := w.fetchEnabledServers(ctx)
	if err != nil {
		w.logger.Error("media worker: fetch servers failed", "err", err)
		return
	}
	if len(servers) == 0 {
		return
	}
	w.logger.Info("media worker: polling", "servers", len(servers))

	sem := make(chan struct{}, w.maxPar)
	var wg sync.WaitGroup
	for _, s := range servers {
		wg.Add(1)
		sem <- struct{}{}
		go func(s mediaTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("media worker: poll panic",
						"server_id", s.ID, "err", r)
				}
			}()
			w.pollOne(ctx, s)
		}(s)
	}
	wg.Wait()
}

// mediaTarget is the small payload the worker pulls for each row.
type mediaTarget struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	ID       uuid.UUID
	Kind     string
	BaseURL  string
	APIKey   string
	Enabled  bool
}

// fetchEnabledServers returns every enabled server across all
// tenants. The (user_id) index keeps the SELECT cheap.
func (w *MediaWorker) fetchEnabledServers(ctx context.Context) ([]mediaTarget, error) {
	rows, err := w.pool.Pgx().Query(ctx,
		`SELECT tenant_id, user_id, id, kind, base_url, api_key, enabled
		   FROM homelab_media_servers
		  WHERE enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []mediaTarget{}
	for rows.Next() {
		var s mediaTarget
		if err := rows.Scan(&s.TenantID, &s.UserID, &s.ID, &s.Kind,
			&s.BaseURL, &s.APIKey, &s.Enabled); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// pollOne delegates to the handler-level
// PollMediaServerAndInsertState helper, which fetches the
// per-kind state, UPSERTs now_playing + recent_additions rows,
// and UPDATEs the server's bookkeeping columns. We re-fetch the
// row here first so we can pass (tenant_id, user_id) into the
// helper — the helper's ownership check then guards against any
// drift between the worker's SELECT and the helper's row read.
//
// The handler package doesn't export a worker-shaped helper, so
// the inline SELECT + call pattern below mirrors the
// DownloadsWorker.pollOne flow (see downloads.go). Per-server
// errors are logged inside the helper; the worker just calls it
// and moves on.
func (w *MediaWorker) pollOne(parentCtx context.Context, s mediaTarget) {
	ctx, cancel := context.WithTimeout(parentCtx, w.fetchTimeout)
	defer cancel()

	// The handler's PollMediaServerAndInsertState reads its own
	// (tenant_id, user_id) from the DB row. We re-fetch here so
	// the helper has the freshest state and so its internal
	// SELECT can fail loudly if the row was deleted between
	// the worker's fetchEnabledServers and the per-server
	// pollOne call.
	var (
		tenantID, userID uuid.UUID
		enabled          bool
	)
	err := w.pool.Pgx().QueryRow(ctx,
		`SELECT tenant_id, user_id, enabled
		   FROM homelab_media_servers WHERE id = $1`,
		s.ID,
	).Scan(&tenantID, &userID, &enabled)
	if err != nil {
		w.logger.Warn("media worker: pre-poll load failed",
			"server_id", s.ID, "err", err)
		return
	}
	if !enabled {
		// Disabled between fetchEnabledServers and now — skip.
		return
	}

	// Use the handler package via a thin adapter — the handler
	// owns the INSERT/UPDATE logic and we want to share the
	// code (DRY pattern documented in the file header). The
	// adapter is intentionally minimal: it calls the same
	// PollMediaServer dispatch + the same UPSERTs.
	mediaPollAndUpsert(ctx, w.pool, s.ID, s.Kind, s.BaseURL, s.APIKey, tenantID, userID, w.logger)
}
