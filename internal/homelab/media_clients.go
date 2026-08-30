// Per-kind API client pollers for the MediaWorker + the
// handler.PollMediaServerAndInsertState wrapper.
//
// These MUST stay in lock-step with the kind whitelist in
// internal/handler/handlers_homelab_media_types.go
// (allowedMediaServerKinds) and the dispatcher in
// internal/homelab/media.go (PollMediaServer). Adding a new kind
// requires:
//  1. add the kind string to allowedMediaServerKinds
//  2. add a poll* function here
//  3. add a case to PollMediaServer's switch
//  4. add the kind to the frontend's <select> in
//     MediaWidget.tsx
//
// Each poll* function returns a uniform *MediaState (or an
// error). They are NOT required to fetch every possible field —
// missing fields fall back to zero values, which is the
// documented behavior for servers whose API doesn't expose the
// field (e.g. Plex's /status/sessions returns null for sessions
// when nothing is active).
//
// All pollers:
//   - use ctx-bounded http.Client (defaultMediaTimeout)
//   - parse JSON with the stdlib decoder
//   - set the kind-appropriate auth header
//     (X-Plex-Token for Plex; X-Emby-Token for Jellyfin/Emby)
//   - keep error messages under 200 chars (via mediaTruncateErr)
//   - never panic — malformed responses return an error, not a
//     nil deref
//
// Auth header notes:
//   - Plex uses X-Plex-Token (also accepts ?X-Plex-Token= as a
//     query param, but the header is the documented / preferred
//     way per support.plex.tv).
//   - Jellyfin and Emby both use X-Emby-Token (case-sensitive
//     in older Jellyfin versions; Emby is forgiving). Neither
//     accepts the token as a query parameter.
package homelab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// defaultMediaTimeout is the per-HTTP-request timeout the per-kind
// pollers enforce. Tighter than the worker-level fetchTimeout so
// a slow target doesn't pin a goroutine for the full 20s window.
const defaultMediaTimeout = 10 * time.Second

// maxMediaResponseBody caps the per-poll HTTP response body at
// ~4 MiB so a pathological Jellyfin /Sessions response with
// thousands of items can't exhaust the worker's memory.
const maxMediaResponseBody = 4 * 1024 * 1024

// errMediaAuthRequired is the canonical error returned by
// pollers when the API responds 401/403. Surfaced verbatim so
// the dashboard can show "auth failed" without a generic "poll
// failed" message.
var errMediaAuthRequired = errors.New("auth failed (401/403) — check api_key")

// ------------------------------------------------------------------
// Shared types — uniform shape every poll* function returns.
// ------------------------------------------------------------------

// MediaState is the uniform shape every poll* function returns.
// The handler package reads these fields and UPSERTs them into
// homelab_now_playing + homelab_recent_additions.
//
// NowPlaying mirrors homelab_now_playing's per-session columns.
// RecentAdditions mirrors homelab_recent_additions's per-item
// columns. Both lists are empty when the upstream server has no
// active sessions / no recent items (the common case for a
// household Plex server).
type MediaState struct {
	NowPlaying      []MediaSession
	RecentAdditions []MediaItem
}

// MediaSession is one active session from /Sessions (or Plex's
// /status/sessions). SessionID is the upstream server's session
// id (Plex hex; Jellyfin/Emby UUID string). Title is the item
// title. UserName is the viewer's display name (may be empty if
// the server doesn't expose it). Player is the client app name.
// Transcoding is true when the server is transcoding the stream.
// ProgressMs / DurationMs are playback position + total duration
// in milliseconds (Plex uses ms natively; Jellyfin/Emby use
// ticks of 100ns — we convert to ms at the per-kind boundary).
type MediaSession struct {
	SessionID   string
	Title       string
	UserName    string
	Player      string
	Transcoding bool
	ProgressMs  int64
	DurationMs  int64
}

// MediaItem is one recently-added item from /library/recentlyAdded
// (Plex) or /Library/Media/Recent (Jellyfin/Emby). ItemID is the
// upstream rating key / item id. Title is the item title. Kind
// is 'movie' | 'show' | 'episode' (Plex uses lower-case type
// strings; we normalize to the same vocabulary). Year is the
// release year (nullable for shows/episodes). PosterURL is the
// thumbnail URL returned by the upstream API. AddedAt is the
// upstream library's addedAt timestamp.
type MediaItem struct {
	ItemID    string
	Title     string
	Kind      string
	Year      *int
	PosterURL string
	AddedAt   time.Time
}

// ------------------------------------------------------------------
// PollMediaServer — the per-kind dispatcher.
// ------------------------------------------------------------------

// PollMediaServer is the EXPORTED per-kind dispatcher. Called by
// handler.PollMediaServerAndInsertState (immediate fire-and-forget
// after-create poll) AND the MediaWorker tick below. Returns a
// uniform *MediaState or an error describing why the fetch
// failed.
//
// kind is the value from homelab_media_servers.kind ('plex' |
// 'jellyfin' | 'emby'). Unknown kinds return (nil, error) — the
// caller surfaces the error in last_poll_error.
//
// apiKey is the server's auth token (Plex X-Plex-Token; Jellyfin
// /Emby X-Emby-Token). The per-kind helper sets the right header.
//
// ctx bounds the wall-clock budget for the whole fetch+parse.
// The per-request timeout inside the helper is typically tighter
// than ctx, but ctx is the hard ceiling.
func PollMediaServer(ctx context.Context, kind, baseURL, apiKey string) (*MediaState, error) {
	switch kind {
	case "plex":
		return pollPlex(ctx, baseURL, apiKey)
	case "jellyfin":
		return pollJellyfin(ctx, baseURL, apiKey)
	case "emby":
		return pollEmby(ctx, baseURL, apiKey)
	default:
		return nil, errUnknownMediaKind(kind)
	}
}

// errUnknownMediaKind is the canonical error returned by
// PollMediaServer for an unsupported kind. Surfaced verbatim in
// homelab_media_servers.last_poll_error.
func errUnknownMediaKind(kind string) error {
	return unknownMediaKindError{kind: kind}
}

type unknownMediaKindError struct{ kind string }

func (e unknownMediaKindError) Error() string {
	return "unknown media server kind: " + e.kind
}

// ------------------------------------------------------------------
// Shared helpers.
// ------------------------------------------------------------------

// makeMediaClient returns an http.Client with the per-poll
// timeout baked in. Same options on every kind so the per-kind
// code stays focused on URL + auth + parsing.
func makeMediaClient() *http.Client {
	return &http.Client{
		Timeout: defaultMediaTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// readMediaJSONBody does the supplied request, reads up to
// maxMediaResponseBody bytes, decodes JSON into out. Returns the
// raw bytes AND the decoded value. Any error from net/http /
// io / json encoding is wrapped with the URL + status code for
// log readability.
func readMediaJSONBody(ctx context.Context, c *http.Client, req *http.Request, out interface{}) ([]byte, error) {
	req.Header.Set("User-Agent", "StackWatch-Homelab-MediaWorker/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errMediaAuthRequired
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaResponseBody))
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return body, fmt.Errorf("decode: %w", err)
		}
	}
	return body, nil
}

// joinMediaBaseURL returns rawURL with a trailing slash so callers
// can join paths without double-slash bugs.
func joinMediaBaseURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse base_url: %w", err)
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u.String(), nil
}
