// Tier 11 Phase 6 — Multi-Region / HA (PL6).
//
// HTTP probe helpers shared by CreateRegion (fire-and-forget
// goroutine) + ProbeRegionsHealth (synchronous inline loop).
//
// The HTTP probe itself is intentionally minimal:
//   - 2s per-region timeout (regionProbeTimeout) so a slow
//     region can't stall the dashboard.
//   - 200 + < 500ms latency → "up"
//   - 200 + >= 500ms latency → "degraded"
//   - non-200 / error → "down"
//   - never probed → "unknown" (returned only by the DB row
//     itself — the probe helper always returns one of the
//     other three).
//
// Status is written back to platform_regions.last_health_*
// columns so a subsequent ListRegions call surfaces the
// probe result inline.

package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// probeOneRegion GETs `endpoint` with a 2s timeout and
// writes the result back to platform_regions.last_health_*.
// Returns the same status / latency / error so the inline
// /regions/health path can echo them back without a
// re-fetch.
//
// Status mapping:
//   - 200 + < 500ms latency   → "up"
//   - 200 + >= 500ms latency  → "degraded"
//   - non-200 / error         → "down"
//   - missing / never probed  → "unknown"
func probeOneRegion(pool *db.Pool, regionID uuid.UUID, endpoint string) (status string, latencyMS int, errStr string) {
	ctx, cancel := context.WithTimeout(context.Background(), regionProbeTimeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		writeProbeResult(pool, regionID, "down", 0, err.Error())
		return "down", 0, err.Error()
	}
	// Don't follow redirects — a redirect to a login
	// page would falsely look like "200 up".
	client := &http.Client{
		Timeout: regionProbeTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	latency := int(time.Since(start) / time.Millisecond)
	if err != nil {
		writeProbeResult(pool, regionID, "down", latency, err.Error())
		return "down", latency, err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg := "non-200 status: " + resp.Status
		writeProbeResult(pool, regionID, "down", latency, msg)
		return "down", latency, msg
	}
	status = "up"
	if latency >= 500 {
		status = "degraded"
	}
	writeProbeResult(pool, regionID, status, latency, "")
	return status, latency, ""
}

// writeProbeResult writes the probe result back to
// platform_regions. Errors are logged but never returned —
// the caller has already sent its response and there's no
// one to surface a write failure to.
func writeProbeResult(pool *db.Pool, regionID uuid.UUID, status string, latencyMS int, errStr string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var errPtr *string
	if errStr != "" {
		s := errStr
		errPtr = &s
	}
	_, err := pool.Pgx().Exec(ctx, `
		UPDATE platform_regions
		   SET last_health_at = now(),
		       last_health_status = $2,
		       last_health_latency_ms = $3,
		       last_health_error = $4,
		       updated_at = now()
		 WHERE id = $1
	`, regionID, status, latencyMS, errPtr)
	if err != nil {
		slog.Default().Error("region probe write failed",
			"region_id", regionID.String(), "err", err)
	}
}
