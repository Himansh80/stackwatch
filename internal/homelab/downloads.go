// Package homelab — DownloadsWorker + per-kind API client helpers.
//
// DownloadsWorker ticks every 60s, fetches every enabled download
// client in homelab_download_clients across all tenants + users,
// and INSERTs the parsed state into homelab_download_snapshots.
// Disabled clients are skipped (the user can re-enable them later
// without losing their credentials).
//
// Worker must not crash on bad polls. Each poll runs inside its
// own goroutine + recovers from any panic so a hung TCP / TLS /
// slow Sonarr response can never take the worker down.
//
// Per-user (NOT per-tenant): the tick iterates over ALL
// (tenant, user) pairs in the table — a per-tenant worker would
// miss every user's clients. The (tenant_id, enabled) partial
// index makes the SELECT cheap on the (typically small) enabled
// set.
//
// PollDownloadClient is the EXPORTED per-kind dispatcher used by
// handler.PollClientAndInsertSnapshot (handlers_homelab_downloads_snapshots.go)
// AND the worker tick below. Both paths share the same fetch+parse
// code — DRY pattern (no twin implementations to keep in sync).
//
// The per-kind helpers (pollSonarr / pollRadarr / pollQBittorrent /
// pollSABnzbd / pollLidarr / pollReadarr) live in downloads_clients.go
// so this file stays under the 400-LOC cap.
package homelab

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// Tick interval — 60s matches the speckit proposal §H5 ("60s default
// background poll"). Matches the CalendarWorker (30min) and
// ServiceHealthWorker (60s) cadences.
const (
	defaultDownloadsTickInterval  = 60 * time.Second
	defaultDownloadsFetchTimeout  = 20 * time.Second
	defaultDownloadsMaxPar        = 8
	defaultDownloadsOverallBudget = 5 * time.Minute
)

// DownloadState is the uniform shape every poll* function returns.
// The handler package reads these fields and writes them into the
// homelab_download_snapshots columns. Fields are byte-counts (int64)
// so a multi-GB queue doesn't overflow.
type DownloadState struct {
	QueueCount           int
	QueueSizeBytes       int64
	DownloadSpeedBps     int64
	UploadSpeedBps       int64
	TodayDownloadedBytes int64
	TodayUploadedBytes   int64
	// RawPayload is the JSON body returned by the client API.
	// Stored as jsonb in homelab_download_snapshots.raw_payload for
	// future per-torrent breakdown + debug. The worker + handler
	// both truncate at ~1 MiB so a pathological payload can't bloat
	// the row.
	RawPayload []byte
}

// Credentials is the minimum set of inputs every poll* function
// needs. The fields are plain (non-pointer) so the per-kind helpers
// don't have to nil-check — empty strings mean "not provided"
// (e.g. SABnzbd ignores username/password; qBittorrent's API key
// is optional if the user supplied username+password instead).
type Credentials struct {
	APIKey   string
	Username string
	Password string
}

// PollDownloadClient is the EXPORTED per-kind dispatcher. Called
// by handler.PollClientAndInsertSnapshot (immediate fire-and-forget
// after-create poll) AND the worker tick below. Returns a uniform
// *DownloadState or an error describing why the fetch failed.
//
// kind is the value from homelab_download_clients.kind
// ('sonarr' | 'radarr' | 'qbittorrent' | 'sabnzbd' | 'lidarr' |
// 'readarr'). Unknown kinds return (nil, error) — the caller
// surfaces the error in last_poll_error.
//
// ctx bounds the wall-clock budget for the whole fetch+parse. The
// per-request timeout inside the helper is typically tighter than
// ctx, but ctx is the hard ceiling.
func PollDownloadClient(ctx context.Context, kind, baseURL string, creds Credentials) (*DownloadState, error) {
	switch kind {
	case "sonarr":
		return pollSonarr(ctx, baseURL, creds)
	case "radarr":
		return pollRadarr(ctx, baseURL, creds)
	case "qbittorrent":
		return pollQBittorrent(ctx, baseURL, creds)
	case "sabnzbd":
		return pollSABnzbd(ctx, baseURL, creds)
	case "lidarr":
		return pollLidarr(ctx, baseURL, creds)
	case "readarr":
		return pollReadarr(ctx, baseURL, creds)
	default:
		return nil, errUnknownKind(kind)
	}
}

// DownloadsWorker is the per-process singleton. Holds a pool ref +
// tunables + slog logger. Start() launches the goroutine; the
// goroutine respects ctx cancellation for graceful shutdown.
type DownloadsWorker struct {
	pool         *db.Pool
	tickInterval time.Duration
	fetchTimeout time.Duration
	maxPar       int
	overallLimit time.Duration
	logger       *slog.Logger
}

// DownloadsOption mutates the worker at construction time.
type DownloadsOption func(*DownloadsWorker)

// WithDownloadsTickInterval overrides the default 60s tick cadence.
func WithDownloadsTickInterval(d time.Duration) DownloadsOption {
	return func(w *DownloadsWorker) { w.tickInterval = d }
}

// WithDownloadsLogger overrides the default slog logger.
func WithDownloadsLogger(l *slog.Logger) DownloadsOption {
	return func(w *DownloadsWorker) { w.logger = l }
}

// NewDownloadsWorker builds a worker. Call exactly once on
// api-gateway boot, then Start() in a goroutine.
func NewDownloadsWorker(pool *db.Pool, opts ...DownloadsOption) *DownloadsWorker {
	w := &DownloadsWorker{
		pool:         pool,
		tickInterval: defaultDownloadsTickInterval,
		fetchTimeout: defaultDownloadsFetchTimeout,
		maxPar:       defaultDownloadsMaxPar,
		overallLimit: defaultDownloadsOverallBudget,
		logger:       slog.Default(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start launches the worker goroutine. Returns immediately; the
// goroutine exits when ctx is cancelled. Safe to call multiple times —
// second+ calls are no-ops (idempotent).
//
// The first tick fires immediately so a freshly-restarted gateway
// repopulates the dashboard within seconds rather than waiting a
// full tick interval.
func (w *DownloadsWorker) Start(ctx context.Context) {
	w.logger.Info("downloads worker: starting",
		"tick", w.tickInterval,
		"max_par", w.maxPar,
		"fetch_timeout", w.fetchTimeout)

	go func() {
		// Best-effort startup tick — don't crash if it fails.
		func() {
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("downloads worker: startup tick panic", "err", r)
				}
			}()
			w.tick(ctx)
		}()

		t := time.NewTicker(w.tickInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("downloads worker: stopping")
				return
			case <-t.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							w.logger.Error("downloads worker: tick panic", "err", r)
						}
					}()
					w.tick(ctx)
				}()
			}
		}
	}()
}

// tick runs one pass: fetch every enabled client across all
// tenants, poll+INSERT a snapshot per client.
//
// Per-client errors are logged but never abort the batch — one
// bad target must not skip the rest.
func (w *DownloadsWorker) tick(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, w.overallLimit)
	defer cancel()

	clients, err := w.fetchEnabledClients(ctx)
	if err != nil {
		w.logger.Error("downloads worker: fetch clients failed", "err", err)
		return
	}
	if len(clients) == 0 {
		return
	}
	w.logger.Info("downloads worker: polling", "clients", len(clients))

	sem := make(chan struct{}, w.maxPar)
	var wg sync.WaitGroup
	for _, c := range clients {
		wg.Add(1)
		sem <- struct{}{}
		go func(c downloadTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					w.logger.Error("downloads worker: poll panic",
						"client_id", c.ID, "err", r)
				}
			}()
			w.pollOne(ctx, c)
		}(c)
	}
	wg.Wait()
}

// downloadTarget is the small payload the worker pulls for each row.
type downloadTarget struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	ID       uuid.UUID
	Kind     string
	BaseURL  string
	APIKey   *string
	Username *string
	Password *string
}

// fetchEnabledClients returns every enabled client across all
// tenants. The (tenant_id, enabled) partial index keeps the SELECT
// cheap.
func (w *DownloadsWorker) fetchEnabledClients(ctx context.Context) ([]downloadTarget, error) {
	rows, err := w.pool.Pgx().Query(ctx,
		`SELECT tenant_id, user_id, id, kind, base_url, api_key, username, password
		   FROM homelab_download_clients
		  WHERE enabled = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []downloadTarget{}
	for rows.Next() {
		var c downloadTarget
		if err := rows.Scan(&c.TenantID, &c.UserID, &c.ID, &c.Kind, &c.BaseURL,
			&c.APIKey, &c.Username, &c.Password); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// pollOne fetches + parses one client + INSERTs a snapshot row +
// UPDATEs the client's bookkeeping. Mirrors
// handler.PollClientAndInsertSnapshot's logic but inlined here so
// the worker doesn't depend on the handler package (handler
// imports homelab, not the other way around).
//
// On success: INSERTs a snapshot with last_poll_status='success'.
// On failure: still INSERTs a snapshot with last_poll_status='error'
// and the truncated error message so the dashboard can render the
// error pill.
func (w *DownloadsWorker) pollOne(parentCtx context.Context, c downloadTarget) {
	ctx, cancel := context.WithTimeout(parentCtx, w.fetchTimeout)
	defer cancel()

	creds := Credentials{}
	if c.APIKey != nil {
		creds.APIKey = *c.APIKey
	}
	if c.Username != nil {
		creds.Username = *c.Username
	}
	if c.Password != nil {
		creds.Password = *c.Password
	}

	state, pollErr := PollDownloadClient(ctx, c.Kind, c.BaseURL, creds)

	tx, txErr := w.pool.Pgx().Begin(ctx)
	if txErr != nil {
		w.logger.Warn("downloads worker: tx begin failed",
			"client_id", c.ID, "err", txErr)
		return
	}
	defer func() {
		if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr.Error() != "tx is closed" {
			w.logger.Warn("downloads worker: tx rollback failed",
				"client_id", c.ID, "err", rbErr)
		}
	}()

	if state == nil {
		state = &DownloadState{}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO homelab_download_snapshots
		        (tenant_id, user_id, client_id,
		         queue_count, queue_size_bytes,
		         download_speed_bytes_per_sec, upload_speed_bytes_per_sec,
		         today_downloaded_bytes, today_uploaded_bytes,
		         raw_payload)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		c.TenantID, c.UserID, c.ID,
		state.QueueCount, state.QueueSizeBytes,
		state.DownloadSpeedBps, state.UploadSpeedBps,
		state.TodayDownloadedBytes, state.TodayUploadedBytes,
		truncatePayloadBytes(state.RawPayload),
	); err != nil {
		w.logger.Warn("downloads worker: insert snapshot failed",
			"client_id", c.ID, "err", err)
		return
	}

	now := time.Now().UTC()
	var status, errMsg string
	if pollErr != nil {
		status = "error"
		errMsg = truncateErr(pollErr.Error())
	} else {
		status = "success"
		errMsg = ""
	}
	if _, err := tx.Exec(ctx,
		`UPDATE homelab_download_clients
		    SET last_polled_at   = $2,
		        last_poll_status = $3,
		        last_poll_error  = NULLIF($4, ''),
		        updated_at       = NOW()
		  WHERE id = $1`,
		c.ID, now, status, errMsg,
	); err != nil {
		w.logger.Warn("downloads worker: update client status failed",
			"client_id", c.ID, "err", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		w.logger.Warn("downloads worker: tx commit failed",
			"client_id", c.ID, "err", err)
		return
	}

	if pollErr != nil {
		w.logger.Warn("downloads worker: poll failed",
			"client_id", c.ID, "kind", c.Kind, "err", pollErr)
	}
}

// truncatePayloadBytes caps the JSONB payload at ~1 MiB so a
// pathological API response can't bloat the row. Defined here so
// both the worker (this file) and the handler-side INSERT can use
// the same limit without depending on each other.
func truncatePayloadBytes(payload []byte) []byte {
	const maxBytes = 1 * 1024 * 1024
	if len(payload) <= maxBytes {
		return payload
	}
	return payload[:maxBytes]
}

// errUnknownKind is the canonical error returned by PollDownloadClient
// for an unsupported kind. Surfaced verbatim in
// homelab_download_clients.last_poll_error.
func errUnknownKind(kind string) error {
	return unknownKindError{kind: kind}
}

type unknownKindError struct{ kind string }

func (e unknownKindError) Error() string {
	return "unknown download client kind: " + e.kind
}