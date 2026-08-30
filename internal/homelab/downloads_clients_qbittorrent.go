// qBittorrent Web API v2 poller — split out of
// downloads_clients.go so each file stays under 400 LOC.
//
// qBittorrent is the most complex of the six clients — the Web
// API requires a cookie-based login before any other call. Flow:
//
//  1. POST /api/v2/auth/login (form-encoded username=...&password=...)
//     → sets SID cookie
//  2. GET /api/v2/transfer/info   → current speeds (up/down bytes/sec)
//  3. GET /api/v2/torrents/info   → queue details
//
// The SID cookie is scoped to the single poll — no cross-poll
// leakage.
//
// queue_size_bytes = sum of torrent.size - torrent.completed
// (bytes remaining across the active queue), matching what most
// homelab dashboards display.
package homelab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// qbittTransferInfo is the /api/v2/transfer/info response shape.
type qbittTransferInfo struct {
	UpInfoSpeed int64 `json:"up_info_speed"` // bytes/sec
	DlInfoSpeed int64 `json:"dl_info_speed"` // bytes/sec
}

// qbittTorrentInfo is one item in /api/v2/torrents/info.
type qbittTorrentInfo struct {
	Size      int64 `json:"size"`      // total bytes
	Completed int64 `json:"completed"` // bytes downloaded
}

// qbittLoginOK is the prefix qBittorrent returns from a successful
// login. Older versions include a leading newline ("\nOk.").
const qbittLoginOK = "Ok."

// pollQBittorrent fetches speeds + queue from a qBittorrent v2
// server. Implements the cookie-login flow described in the
// package comment above.
func pollQBittorrent(ctx context.Context, baseURL string, creds Credentials) (*DownloadState, error) {
	joined, err := joinBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	c := makeClient()

	// --- 1. login --------------------------------------------------
	loginForm := url.Values{}
	loginForm.Set("username", creds.Username)
	loginForm.Set("password", creds.Password)
	loginReq, lerr := http.NewRequestWithContext(ctx, http.MethodPost,
		joined+"api/v2/auth/login", strings.NewReader(loginForm.Encode()))
	if lerr != nil {
		return nil, fmt.Errorf("new login request: %w", lerr)
	}
	loginReq.Header.Set("User-Agent", "StackWatch-Homelab-DownloadsWorker/1.0")
	loginReq.Header.Set("Accept", "*/*")
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("Referer", joined)
	loginResp, lerr := c.Do(loginReq)
	if lerr != nil {
		return nil, fmt.Errorf("login: %w", lerr)
	}
	loginBody, _ := io.ReadAll(io.LimitReader(loginResp.Body, 200))
	loginResp.Body.Close()
	if loginResp.StatusCode == http.StatusUnauthorized || loginResp.StatusCode == http.StatusForbidden {
		return nil, errAuthRequired
	}
	if loginResp.StatusCode >= 400 {
		return nil, fmt.Errorf("login http %d: %s", loginResp.StatusCode, strings.TrimSpace(string(loginBody)))
	}
	if !strings.HasPrefix(strings.TrimSpace(string(loginBody)), qbittLoginOK) {
		return nil, fmt.Errorf("login failed: %q", strings.TrimSpace(string(loginBody)))
	}
	// Capture the SID cookie for the subsequent requests.
	var sid string
	for _, ck := range loginResp.Cookies() {
		if ck.Name == "SID" {
			sid = ck.Value
			break
		}
	}
	if sid == "" {
		return nil, errors.New("login succeeded but no SID cookie returned")
	}

	// --- 2. transfer info (current speeds) -------------------------
	transReq, terr := http.NewRequestWithContext(ctx, http.MethodGet,
		joined+"api/v2/transfer/info", nil)
	if terr != nil {
		return nil, fmt.Errorf("new transfer request: %w", terr)
	}
	transReq.Header.Set("User-Agent", "StackWatch-Homelab-DownloadsWorker/1.0")
	transReq.Header.Set("Accept", "application/json")
	transReq.AddCookie(&http.Cookie{Name: "SID", Value: sid})
	transReq.Header.Set("Referer", joined)

	var transInfo qbittTransferInfo
	transBody, terr := readJSONBody(ctx, c, transReq, &transInfo)
	if terr != nil {
		return &DownloadState{RawPayload: transBody}, fmt.Errorf("transfer info: %w", terr)
	}

	state := &DownloadState{
		DownloadSpeedBps: transInfo.DlInfoSpeed,
		UploadSpeedBps:   transInfo.UpInfoSpeed,
		RawPayload:       transBody,
	}

	// --- 3. torrents info (queue) ----------------------------------
	torrentsReq, qerr := http.NewRequestWithContext(ctx, http.MethodGet,
		joined+"api/v2/torrents/info", nil)
	if qerr != nil {
		return state, fmt.Errorf("new torrents request: %w", qerr)
	}
	torrentsReq.Header.Set("User-Agent", "StackWatch-Homelab-DownloadsWorker/1.0")
	torrentsReq.Header.Set("Accept", "application/json")
	torrentsReq.AddCookie(&http.Cookie{Name: "SID", Value: sid})
	torrentsReq.Header.Set("Referer", joined)

	var torrents []qbittTorrentInfo
	torrentsBody, terr := readJSONBody(ctx, c, torrentsReq, &torrents)
	if terr != nil {
		// Queue fetch failed — return what we have (speeds) plus
		// the raw body so the dashboard can show partial data.
		state.RawPayload = torrentsBody
		return state, fmt.Errorf("torrents info: %w", terr)
	}
	state.QueueCount = len(torrents)
	for _, t := range torrents {
		remaining := t.Size - t.Completed
		if remaining < 0 {
			remaining = 0
		}
		state.QueueSizeBytes += remaining
	}
	// Transfer-info body is the more useful raw payload for
	// debugging — keep it as-is.
	return state, nil
}
