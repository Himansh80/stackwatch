// Per-kind API client pollers for the DownloadsWorker + the
// handler.PollClientAndInsertSnapshot wrapper.
//
// These MUST stay in lock-step with the kind whitelist in
// internal/handler/handlers_homelab_downloads_types.go
// (allowedDownloadClientKinds) and the dispatcher in
// internal/homelab/downloads.go (PollDownloadClient). Adding a
// new kind requires:
//  1. add the kind string to allowedDownloadClientKinds
//  2. add a poll* function here (or in downloads_clients_arr.go
//     / downloads_clients_qbittorrent.go /
//     downloads_clients_sabnzbd.go if you split further)
//  3. add a case to PollDownloadClient's switch
//  4. add the kind to the frontend's <select> in
//     DownloadStatsWidget.tsx
//
// Each poll* function returns a uniform *DownloadState (or an
// error). They are NOT required to fetch every possible field —
// missing fields fall back to zero, which is the documented
// behavior for clients whose API doesn't expose the field
// (e.g. SABnzbd doesn't expose a torrent-style upload speed).
//
// All pollers:
//   - use ctx-bounded http.Client (defaultPerKindTimeout)
//   - parse JSON with the stdlib decoder
//   - keep error messages under 200 chars (via truncateErr)
//   - never panic — malformed responses return an error, not a
//     nil deref
//
// The shared `truncateErr` lives in servicehealth_probe.go (the
// same package) so this file stays focused on shared helpers +
// the *arr fetchers (Sonarr/Radarr/Lidarr/Readarr). qBittorrent
// lives in downloads_clients_qbittorrent.go and SABnzbd lives in
// downloads_clients_sabnzbd.go to keep each file under 400 LOC.
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

// defaultPerKindTimeout is the per-HTTP-request timeout the
// per-kind pollers enforce. Tighter than the worker-level
// fetchTimeout so a slow target doesn't pin a goroutine for the
// full 20s window.
const defaultPerKindTimeout = 10 * time.Second

// maxResponseBody caps the per-poll HTTP response body at ~4 MiB
// so a pathological Sonarr /api/v3/queue with thousands of items
// can't exhaust the worker's memory.
const maxResponseBody = 4 * 1024 * 1024

// errAuthRequired is the canonical error returned by pollers
// when the API responds 401/403. Surfaced verbatim so the
// dashboard can show "auth failed" without a generic "poll
// failed" message.
var errAuthRequired = errors.New("auth failed (401/403) — check api_key")

// makeClient returns an http.Client with the per-poll timeout
// baked in. Same options on every kind so the per-kind code stays
// focused on URL + auth + parsing.
func makeClient() *http.Client {
	return &http.Client{
		Timeout: defaultPerKindTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// readJSONBody does the supplied request, reads up to
// maxResponseBody bytes, decodes JSON into out. Returns the
// raw bytes (for the raw_payload column) AND the decoded value.
// Any error from net/http / io / json encoding is wrapped with
// the URL + status code for log readability.
func readJSONBody(ctx context.Context, c *http.Client, req *http.Request, out interface{}) ([]byte, error) {
	req.Header.Set("User-Agent", "StackWatch-Homelab-DownloadsWorker/1.0")
	req.Header.Set("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errAuthRequired
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
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

// joinBaseURL returns rawURL with a trailing slash so callers
// can join paths without double-slash bugs.
func joinBaseURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse base_url: %w", err)
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	return u.String(), nil
}

// ------------------------------------------------------------------
// *arr shared — Sonarr / Radarr / Lidarr / Readarr
// ------------------------------------------------------------------
//
// All four expose the same v3 (Sonarr/Radarr) or v1 (Lidarr/Readarr)
// API surface: GET /api/v3/queue (or v1) returns the active
// download queue. Each queue record carries size + sizeleft, so
// queue_size_bytes = sum(size - sizeleft) (bytes already
// downloaded are no longer "remaining").
//
// "Today" totals aren't a built-in stat on any of these apps —
// we derive a rough number by summing the size of completed
// downloads in the last 24h via /api/v3/history (Sonarr/Radarr)
// or /api/v1/history (Lidarr/Readarr). If the history endpoint
// errors or times out we still return the queue data with
// today_downloaded_bytes=0 — the dashboard renders the live
// queue stats even if history is unavailable.

// arrQueueRecord is the subset of a *arr queue item we read.
// "size" is the total size of the download; "sizeleft" is bytes
// remaining. Subtracting gives bytes downloaded so far.
type arrQueueRecord struct {
	Size     int64 `json:"size"`
	SizeLeft int64 `json:"sizeleft"`
}

// arrQueueResponse is the wrapper Sonarr/Radarr/Lidarr/Readarr
// return. The "records" field holds the per-item queue list.
type arrQueueResponse struct {
	Records []arrQueueRecord `json:"records"`
}

// arrHistoryRecord is one history row from /api/v3/history
// (Sonarr/Radarr) or /api/v1/history (Lidarr/Readarr). We sum
// the size field for rows where eventType=downloadFolderImported.
type arrHistoryRecord struct {
	Size int64 `json:"size"`
}

// pollArr is the shared fetch for all four *arr apps. The only
// differences between them are the api version segment (v3 for
// Sonarr/Radarr, v1 for Lidarr/Readarr) and whether the user
// supplies an api_key.
func pollArr(ctx context.Context, baseURL, apiKey, apiVersion string) (*DownloadState, error) {
	joined, err := joinBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	c := makeClient()

	queueReq, qerr := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%sapi/%s/queue?pageSize=1000", joined, apiVersion), nil)
	if qerr != nil {
		return nil, fmt.Errorf("new queue request: %w", qerr)
	}
	if apiKey != "" {
		queueReq.Header.Set("X-Api-Key", apiKey)
	}
	var queueResp arrQueueResponse
	queueBody, queueErr := readJSONBody(ctx, c, queueReq, &queueResp)
	if queueErr != nil {
		// History fallback is best-effort — return what we have
		// from the queue (which failed) plus a wrapped error so
		// the dashboard knows the poll failed. We still set
		// raw_payload to whatever bytes we got (might be empty).
		return &DownloadState{
			RawPayload: queueBody,
		}, fmt.Errorf("queue fetch: %w", queueErr)
	}

	state := &DownloadState{
		RawPayload: queueBody,
	}
	for _, r := range queueResp.Records {
		remaining := r.SizeLeft
		if remaining < 0 {
			remaining = 0
		}
		state.QueueSizeBytes += remaining
	}
	state.QueueCount = len(queueResp.Records)

	// History: best-effort today_downloaded_bytes. Failure here
	// doesn't fail the whole poll — the queue data is the more
	// important number.
	histReq, herr := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%sapi/%s/history?pageSize=500&eventType=downloadFolderImported", joined, apiVersion), nil)
	if herr == nil {
		if apiKey != "" {
			histReq.Header.Set("X-Api-Key", apiKey)
		}
		var histRows []arrHistoryRecord
		if _, herr := readJSONBody(ctx, c, histReq, &histRows); herr == nil {
			for _, h := range histRows {
				state.TodayDownloadedBytes += h.Size
			}
		}
	}

	return state, nil
}

// pollSonarr fetches the queue + history from a Sonarr v3 server.
func pollSonarr(ctx context.Context, baseURL string, creds Credentials) (*DownloadState, error) {
	return pollArr(ctx, baseURL, creds.APIKey, "v3")
}

// pollRadarr fetches the queue + history from a Radarr v3 server.
func pollRadarr(ctx context.Context, baseURL string, creds Credentials) (*DownloadState, error) {
	return pollArr(ctx, baseURL, creds.APIKey, "v3")
}

// pollLidarr fetches the queue from a Lidarr v1 server.
func pollLidarr(ctx context.Context, baseURL string, creds Credentials) (*DownloadState, error) {
	return pollArr(ctx, baseURL, creds.APIKey, "v1")
}

// pollReadarr fetches the queue from a Readarr v1 server.
func pollReadarr(ctx context.Context, baseURL string, creds Credentials) (*DownloadState, error) {
	return pollArr(ctx, baseURL, creds.APIKey, "v1")
}
