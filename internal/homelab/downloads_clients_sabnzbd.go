// SABnzbd API poller — split out of downloads_clients.go so each
// file stays under 400 LOC.
//
// SABnzbd's API is the simplest of the six — a single GET
// /api?mode=queue returns JSON with everything we need. mode=history
// (limit=50) gives us "today downloaded" — we sum the bytes for
// rows with status=Completed.
//
// The api_key is passed as ?apikey=<value>&output=json. We add
// output=json explicitly because SABnzbd defaults to its own
// pseudo-format.
package homelab

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// sabQueueResponse is the /api?mode=queue response. Only the
// fields we read are mapped.
type sabQueueResponse struct {
	Queue struct {
		Slots []struct {
			SizeLeft float64 `json:"sizeleft"` // SABnzbd reports bytes as float
		} `json:"slots"`
	} `json:"queue"`
	Speed struct {
		Speed int64 `json:"speed"` // bytes/sec download
	} `json:"speed"`
}

// sabHistoryResponse is the /api?mode=history response. Each
// slot has size + status ("Completed" | "Failed" | ...) and a
// unix timestamp.
type sabHistoryResponse struct {
	History struct {
		Slots []struct {
			Size      float64 `json:"size"`
			Status    string  `json:"status"`
			Completed int64   `json:"completed"`
		} `json:"slots"`
	} `json:"history"`
}

// pollSABnzbd fetches the queue + history from a SABnzbd server.
//
// On history-fetch failure the queue data is still returned
// (today_downloaded_bytes=0) — the dashboard renders the live
// queue even if history is unavailable.
func pollSABnzbd(ctx context.Context, baseURL string, creds Credentials) (*DownloadState, error) {
	joined, err := joinBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	c := makeClient()

	// --- queue ----------------------------------------------------
	queueURL := fmt.Sprintf("%sapi?mode=queue&output=json&apikey=%s",
		joined, url.QueryEscape(creds.APIKey))
	queueReq, qerr := http.NewRequestWithContext(ctx, http.MethodGet, queueURL, nil)
	if qerr != nil {
		return nil, fmt.Errorf("new queue request: %w", qerr)
	}
	var queueResp sabQueueResponse
	queueBody, queueErr := readJSONBody(ctx, c, queueReq, &queueResp)
	if queueErr != nil {
		return &DownloadState{RawPayload: queueBody}, fmt.Errorf("queue fetch: %w", queueErr)
	}
	state := &DownloadState{
		DownloadSpeedBps: queueResp.Speed.Speed,
		RawPayload:       queueBody,
	}
	for _, s := range queueResp.Queue.Slots {
		state.QueueSizeBytes += int64(s.SizeLeft)
	}
	state.QueueCount = len(queueResp.Queue.Slots)

	// --- history (best-effort) ------------------------------------
	histURL := fmt.Sprintf("%sapi?mode=history&output=json&limit=50&apikey=%s",
		joined, url.QueryEscape(creds.APIKey))
	histReq, herr := http.NewRequestWithContext(ctx, http.MethodGet, histURL, nil)
	if herr == nil {
		var histResp sabHistoryResponse
		if _, herr := readJSONBody(ctx, c, histReq, &histResp); herr == nil {
			for _, h := range histResp.History.Slots {
				if h.Status == "Completed" {
					state.TodayDownloadedBytes += int64(h.Size)
				}
			}
		}
	}

	return state, nil
}
