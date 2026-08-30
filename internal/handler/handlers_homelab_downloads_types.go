// Tier 10 Phase 5 — Download Stats (H5).
//
// Shared types for the homelab download-clients + snapshots surface.
// Lives in its own file so handlers_homelab_downloads.go (3 CRUD
// routes) can import these types without growing past the 400-LOC
// cap, and so handlers_homelab_downloads_snapshots.go (GET
// snapshots + PollClientAndInsertSnapshot helper) shares them.
//
// Storage: homelab_download_clients + homelab_download_snapshots
// (migrations/041_homelab.sql — Phase 5 extension).
//
// Per-user design: every query is filtered by both tenant_id and
// user_id. A user in tenant A cannot see (or modify) another
// user's download clients even if they share a tenant — matches
// US-8 in the speckit proposal.
package handler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// ------------------------------------------------------------------
// Validation helpers — kept in this file so handlers_homelab_downloads.go
// can stay under the 400-LOC cap. Each is short (≤25 LOC) and
// only depends on stdlib + db.
// ------------------------------------------------------------------

// validateDownloadClientName enforces a sane display label. Empty
// / over-cap / leading-trailing whitespace all rejected — the
// dashboard's client chip uses the name as its label.
func validateDownloadClientName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > maxDownloadNameLen {
		return fmt.Errorf("name must be %d characters or fewer", maxDownloadNameLen)
	}
	return nil
}

// validateDownloadKind rejects anything not in the
// allowedDownloadClientKinds whitelist. The whitelist is the
// source of truth — see below.
func validateDownloadKind(kind string) error {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "" {
		return fmt.Errorf("kind is required")
	}
	if _, ok := allowedDownloadClientKinds[kind]; !ok {
		return fmt.Errorf("kind %q is not supported (allowed: sonarr, radarr, qbittorrent, sabnzbd, lidarr, readarr)", kind)
	}
	return nil
}

// validateDownloadBaseURL requires http(s) parseable + non-empty
// host. Same check the worker relies on, so we surface the error
// at insert time rather than at first poll.
func validateDownloadBaseURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("base_url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("base_url is not parseable: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("base_url must be http:// or https://")
	}
	if u.Host == "" {
		return fmt.Errorf("base_url is missing host")
	}
	return nil
}

// validateDownloadCredentials enforces "the kind has SOME way to
// authenticate": *arr + SABnzbd require api_key; qBittorrent
// accepts api_key OR (username + password). Rejects the empty
// case so we don't insert a row the worker can never poll.
func validateDownloadCredentials(kind, apiKey, username, password string) error {
	switch kind {
	case "qbittorrent":
		hasAPI := strings.TrimSpace(apiKey) != ""
		hasUser := strings.TrimSpace(username) != "" && strings.TrimSpace(password) != ""
		if !hasAPI && !hasUser {
			return fmt.Errorf("qbittorrent clients require api_key OR (username + password)")
		}
	default:
		if strings.TrimSpace(apiKey) == "" {
			return fmt.Errorf("%s clients require api_key", kind)
		}
	}
	return nil
}

// countDownloadClients returns the number of pinned clients the
// caller already has. Used to enforce maxDownloadClientsPerUser.
func countDownloadClients(ctx context.Context, pool *db.Pool, tenantID, userID uuid.UUID) (int, error) {
	var n int
	err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::int FROM homelab_download_clients
		  WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&n)
	return n, err
}

// ------------------------------------------------------------------
// downloadClientRow — what /homelab/downloads/clients returns.
// ------------------------------------------------------------------

// downloadClientRow is the JSON shape returned by GET
// /downloads/clients and POST /downloads/clients. Mirrors the
// columns of homelab_download_clients plus a latest_snapshot
// block that the handler LEFT JOINs so the dashboard can render
// the per-client stat row in one round-trip.
//
// api_key / username / password are NOT echoed back to the
// frontend (security + per-user UX — re-entering on edit is the
// expected flow for most homelab tools). A future "show secret"
// toggle could surface them; out of scope for Phase 5.
type downloadClientRow struct {
	ID             string                 `json:"id"`
	TenantID       string                 `json:"tenant_id"`
	UserID         string                 `json:"user_id"`
	Name           string                 `json:"name"`
	Kind           string                 `json:"kind"`
	BaseURL        string                 `json:"base_url"`
	Enabled        bool                   `json:"enabled"`
	LastPolledAt   string                 `json:"last_polled_at,omitempty"`
	LastPollStatus string                 `json:"last_poll_status,omitempty"`
	LastPollError  string                 `json:"last_poll_error,omitempty"`
	LatestSnapshot *downloadSnapshotBlock `json:"latest_snapshot,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	UpdatedAt      string                 `json:"updated_at"`
}

// downloadSnapshotBlock is the per-client stat block the
// dashboard renders next to each client name. Embeds the
// downloadSnapshotRow + the client_id so the frontend can match
// snapshots to clients without an extra lookup.
type downloadSnapshotBlock struct {
	ClientID                 string `json:"client_id"`
	QueueCount               int    `json:"queue_count"`
	QueueSizeBytes           int64  `json:"queue_size_bytes"`
	DownloadSpeedBytesPerSec int64  `json:"download_speed_bytes_per_sec"`
	UploadSpeedBytesPerSec   int64  `json:"upload_speed_bytes_per_sec"`
	TodayDownloadedBytes     int64  `json:"today_downloaded_bytes"`
	TodayUploadedBytes       int64  `json:"today_uploaded_bytes"`
	PolledAt                 string `json:"polled_at"`
}

// ------------------------------------------------------------------
// downloadClientReq — body for POST /downloads/clients.
// ------------------------------------------------------------------

// downloadClientReq is the body for POST /downloads/clients. name
// + kind + base_url are required; api_key is required for *arr +
// SABnzbd (qBittorrent can fall back to username+password);
// username + password are qBittorrent-only.
//
// enabled defaults to true so a freshly-pinned client polls on
// the next worker tick (and on the immediate after-create fire-
// and-forget poll). A user who wants to dry-run a config can
// pass enabled=false.
type downloadClientReq struct {
	Name     string `json:"name"      binding:"required"`
	Kind     string `json:"kind"      binding:"required"`
	BaseURL  string `json:"base_url"  binding:"required"`
	APIKey   string `json:"api_key"`
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  *bool  `json:"enabled"`
}

// ------------------------------------------------------------------
// downloadSnapshotRow — what /downloads/snapshots returns.
// ------------------------------------------------------------------

// downloadSnapshotRow is the JSON shape returned by GET
// /downloads/snapshots. client_id is the homelab_download_clients
// row id — the frontend uses it to match snapshots to clients in
// the per-client stat row.
type downloadSnapshotRow struct {
	ID                       string    `json:"id"`
	TenantID                 string    `json:"tenant_id"`
	UserID                   string    `json:"user_id"`
	ClientID                 string    `json:"client_id"`
	QueueCount               int       `json:"queue_count"`
	QueueSizeBytes           int64     `json:"queue_size_bytes"`
	DownloadSpeedBytesPerSec int64     `json:"download_speed_bytes_per_sec"`
	UploadSpeedBytesPerSec   int64     `json:"upload_speed_bytes_per_sec"`
	TodayDownloadedBytes     int64     `json:"today_downloaded_bytes"`
	TodayUploadedBytes       int64     `json:"today_uploaded_bytes"`
	PolledAt                 time.Time `json:"polled_at"`
}

// ------------------------------------------------------------------
// downloadSnapshotsListReq — parsed query string for
// GET /downloads/snapshots.
// ------------------------------------------------------------------

// downloadSnapshotsListReq is the parsed query string for GET
// /downloads/snapshots. client_id is optional (omit = all the
// caller's clients); limit caps the response (default 100, max
// 1000) so the dashboard's chart endpoint can't blow the response
// size.
type downloadSnapshotsListReq struct {
	ClientID string `form:"client_id"`
	Limit    int    `form:"limit"`
}

// ------------------------------------------------------------------
// Validation enums — enforced on POST to reject bad input before
// any DB call or HTTP round-trip to the user's download client.
// ------------------------------------------------------------------

// allowedDownloadClientKinds is the whitelist of supported kinds.
// Mirrors the per-kind API client functions in
// internal/homelab/downloads_clients.go (pollSonarr, pollRadarr,
// pollQBittorrent, pollSABnzbd, pollLidarr, pollReadarr). Adding
// a new kind here requires a matching poll* function in the
// homelab package; removing a kind orphans any rows pinned to it
// (the worker's switch will mark their poll_status='error').
var allowedDownloadClientKinds = map[string]struct{}{
	"sonarr":      {},
	"radarr":      {},
	"qbittorrent": {},
	"sabnzbd":     {},
	"lidarr":      {},
	"readarr":     {},
}

// downloadSnapshotsDefaultLimit is the default cap on GET
// /downloads/snapshots. The dashboard's chart endpoint renders up
// to 100 points; 100 is a generous ceiling that keeps the response
// under 100 KB.
const downloadSnapshotsDefaultLimit = 100

// downloadSnapshotsLimitMax caps the user-supplied ?limit= so a
// malicious caller can't request millions of rows. 1000 covers a
// full day of 60s ticks for one client.
const downloadSnapshotsLimitMax = 1000

// maxDownloadClientsPerUser is the soft cap on per-user pinned
// download clients. Mirrors the per-user pin count cap (50) for
// services so a single user can't register hundreds of clients
// and amplify the worker's poll traffic.
const maxDownloadClientsPerUser = 50

// maxDownloadNameLen is the cap on the human-readable name. 100
// chars is enough for "Home Sonarr (192.168.1.50)" while keeping
// the UNIQUE constraint and the chip label bounded.
const maxDownloadNameLen = 100

// _ keeps time referenced so the imports list doesn't get pruned
// by goimports in case future revisions want to use the time
// package directly (e.g. for RFC3339 helpers in snapshot rows).
// Cheap insurance vs. a future import churn.
var _ = time.RFC3339
