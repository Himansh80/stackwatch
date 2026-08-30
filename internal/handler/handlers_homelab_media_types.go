// Tier 10 Phase 6 — Media Server (H6).
//
// Shared types + validation for the homelab media-server surface.
// Lives in its own file so handlers_homelab_media.go (List/Create/
// Delete + state endpoint + the shared PollMediaServerAndInsertState
// helper) can import these types without growing past the 400-LOC
// cap.
//
// Storage: homelab_media_servers + homelab_now_playing +
// homelab_recent_additions (migrations/041_homelab.sql — Phase 6
// extension).
//
// Per-user design: every query is filtered by both tenant_id and
// user_id. A user in tenant A cannot see (or modify) another
// user's media servers even if they share a tenant — matches US-8
// in the speckit proposal.
//
// Auth model: Plex uses the X-Plex-Token request header; Jellyfin
// and Emby use the X-Emby-Token request header. None of the three
// accept the token as a query parameter — the per-kind pollers in
// internal/homelab/media_clients.go set the right header on every
// request.
package handler

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// ------------------------------------------------------------------
// Validation helpers — kept in this file so handlers_homelab_media.go
// can stay under the 400-LOC cap. Each is short (≤20 LOC) and only
// depends on stdlib.
// ------------------------------------------------------------------

// validateMediaServerName enforces a sane display label. Empty /
// over-cap / leading-trailing whitespace all rejected — the
// dashboard's server chip uses the name as its label.
func validateMediaServerName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if len(name) > maxMediaServerNameLen {
		return fmt.Errorf("name must be %d characters or fewer", maxMediaServerNameLen)
	}
	return nil
}

// validateMediaServerKind rejects anything not in the
// allowedMediaServerKinds whitelist. The whitelist is the source
// of truth — see below.
func validateMediaServerKind(kind string) error {
	kind = strings.TrimSpace(strings.ToLower(kind))
	if kind == "" {
		return fmt.Errorf("kind is required")
	}
	if _, ok := allowedMediaServerKinds[kind]; !ok {
		return fmt.Errorf("kind %q is not supported (allowed: plex, jellyfin, emby)", kind)
	}
	return nil
}

// validateMediaServerBaseURL requires http(s) parseable + non-empty
// host. Same check the worker relies on, so we surface the error
// at insert time rather than at first poll.
func validateMediaServerBaseURL(raw string) error {
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

// validateMediaServerAPIKey enforces non-empty trimmed input.
// All three kinds (Plex/Jellyfin/Emby) use a token-based auth
// scheme — there's no username+password fallback (unlike
// qBittorrent on the download clients surface).
func validateMediaServerAPIKey(apiKey string) error {
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("api_key is required")
	}
	return nil
}

// countMediaServers returns the number of pinned media servers
// the caller already has. Used to enforce maxMediaServersPerUser.
func countMediaServers(ctx context.Context, pool *db.Pool, tenantID, userID uuid.UUID) (int, error) {
	var n int
	err := pool.Pgx().QueryRow(ctx,
		`SELECT COUNT(*)::int FROM homelab_media_servers
		  WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&n)
	return n, err
}

// ------------------------------------------------------------------
// mediaServerRow — what /homelab/media/servers returns.
// ------------------------------------------------------------------

// mediaServerRow is the JSON shape returned by GET
// /media/servers and POST /media/servers. Mirrors the columns of
// homelab_media_servers; api_key is NOT echoed back to the
// frontend (security + per-user UX — re-entering on edit is the
// expected flow for most homelab tools, matches the download
// clients contract).
type mediaServerRow struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	UserID         string `json:"user_id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	BaseURL        string `json:"base_url"`
	Enabled        bool   `json:"enabled"`
	LastPolledAt   string `json:"last_polled_at,omitempty"`
	LastPollStatus string `json:"last_poll_status,omitempty"`
	LastPollError  string `json:"last_poll_error,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// ------------------------------------------------------------------
// mediaServerReq — body for POST /media/servers.
// ------------------------------------------------------------------

// mediaServerReq is the body for POST /media/servers. name + kind
// + base_url + api_key are required. enabled defaults to true so
// a freshly-pinned server polls on the next worker tick (and on
// the immediate after-create fire-and-forget poll).
type mediaServerReq struct {
	Name    string `json:"name"     binding:"required"`
	Kind    string `json:"kind"     binding:"required"`
	BaseURL string `json:"base_url" binding:"required"`
	APIKey  string `json:"api_key"  binding:"required"`
	Enabled *bool  `json:"enabled"`
}

// ------------------------------------------------------------------
// nowPlayingRow + recentAdditionRow — what /media/state returns.
// ------------------------------------------------------------------

// mediaStateListReq is the parsed query string for GET
// /media/state. server_id is optional (omit = all the caller's
// servers); when present the response filters the now_playing +
// recent_additions lists to that single server.
type mediaStateListReq struct {
	ServerID string `form:"server_id"`
}

// nowPlayingRow is one row of the now_playing snapshot returned
// by GET /media/state. Mirrors the columns of homelab_now_playing
// + the server_id + server_name + server_kind so the widget can
// group sessions by server without an extra lookup.
type nowPlayingRow struct {
	ID          string `json:"id"`
	ServerID    string `json:"server_id"`
	ServerName  string `json:"server_name"`
	ServerKind  string `json:"server_kind"`
	SessionID   string `json:"session_id"`
	Title       string `json:"title"`
	UserName    string `json:"user_name,omitempty"`
	Player      string `json:"player,omitempty"`
	Transcoding bool   `json:"transcoding"`
	ProgressMs  int64  `json:"progress_ms"`
	DurationMs  int64  `json:"duration_ms"`
	PolledAt    string `json:"polled_at"`
}

// recentAdditionRow is one row of the recent_additions snapshot
// returned by GET /media/state. Mirrors the columns of
// homelab_recent_additions + server_id + server_name +
// server_kind.
type recentAdditionRow struct {
	ID         string `json:"id"`
	ServerID   string `json:"server_id"`
	ServerName string `json:"server_name"`
	ServerKind string `json:"server_kind"`
	ItemID     string `json:"item_id"`
	Title      string `json:"title"`
	ItemKind   string `json:"item_kind"`
	Year       *int   `json:"year,omitempty"`
	PosterURL  string `json:"poster_url,omitempty"`
	AddedAt    string `json:"added_at"`
	PolledAt   string `json:"polled_at"`
}

// mediaStateResp is the envelope for GET /media/state. Splits the
// now-playing list + the recent-additions list into separate
// arrays so the widget can render them as two distinct sections.
type mediaStateResp struct {
	Servers          []mediaServerRow    `json:"servers"`
	NowPlaying       []nowPlayingRow     `json:"now_playing"`
	RecentAdditions  []recentAdditionRow `json:"recent_additions"`
	ServerID         string              `json:"server_id,omitempty"`
	CountNowPlaying  int                 `json:"count_now_playing"`
	CountRecentAdded int                 `json:"count_recent_added"`
}

// ------------------------------------------------------------------
// Validation enums + caps — enforced on POST to reject bad input
// before any DB call or HTTP round-trip to the user's media server.
// ------------------------------------------------------------------

// allowedMediaServerKinds is the whitelist of supported media
// servers. Mirrors the per-kind API client functions in
// internal/homelab/media_clients.go (pollPlex, pollJellyfin,
// pollEmby). Adding a new kind here requires a matching poll*
// function in the homelab package; removing a kind orphans any
// rows pinned to it (the worker's switch will mark their
// poll_status='error').
var allowedMediaServerKinds = map[string]struct{}{
	"plex":     {},
	"jellyfin": {},
	"emby":     {},
}

// maxMediaServersPerUser is the soft cap on per-user pinned
// media servers. Matches the per-user pin count cap for download
// clients (50) and services (50) so a single user can't register
// hundreds of servers and amplify the worker's poll traffic.
const maxMediaServersPerUser = 50

// maxMediaServerNameLen is the cap on the human-readable name.
// 100 chars matches the download-clients cap and is enough for
// "Living Room Plex (192.168.1.20)" while keeping the UNIQUE
// constraint and the chip label bounded.
const maxMediaServerNameLen = 100

// mediaStateRecentAdditionsCap is the max number of recent-
// addition rows the state endpoint returns per server. 10 matches
// the upstream API's typical default ("recently added" usually
// means the last 10 items) and keeps the widget carousel render
// lightweight.
const mediaStateRecentAdditionsCap = 20

// mediaStateNowPlayingCap is the max number of active sessions
// the state endpoint returns per server. 50 is generous — a single
// home Plex server rarely has > 10 simultaneous streams — and
// keeps the response bounded for power users with multiple
// servers in a household.
const mediaStateNowPlayingCap = 50
