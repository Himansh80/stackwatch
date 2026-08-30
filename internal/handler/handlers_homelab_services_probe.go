// Tier 10 Phase 2 — Service Status (H2). Probe + history endpoints:
//
//	POST /api/v1/homelab/services/:id/probe  — ProbeHomelabService
//	GET  /api/v1/homelab/services/:id/history — GetHomelabServiceHistory
//	POST /api/v1/homelab/services/probe-all  — ProbeAllHomelabServices
//
// The pure probe helpers (probeHTTP / probeTCP / probeICMP / probeOne)
// live in handlers_homelab_services_probe_helpers.go so both files
// stay under 400 LOC. The internal/homelab worker keeps its own
// copy (Go's internal-package rule) — they MUST stay in sync by
// convention; see the helpers file for the sync contract.
//
// All three handlers honor tenant_id + user_id from the JWT —
// never probe or read another user's pins.
package handler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// probeAllMaxPar caps parallel probes per /probe-all request. Eight
// concurrent probes per user keeps the worker responsive even when
// a single user pins the cap of 50 services.
const probeAllMaxPar = 8

// insertProbeResult writes one homelab_service_health row. Uses the
// request context for cancellation so a caller giving up mid-probe
// aborts the INSERT too.
func insertProbeResult(ctx context.Context, pool *db.Pool, tenantID, userID, serviceID uuid.UUID, r serviceProbeResult) error {
	var errMsg *string
	if r.ErrorMessage != "" {
		em := r.ErrorMessage
		errMsg = &em
	}
	_, err := pool.Pgx().Exec(ctx,
		`INSERT INTO homelab_service_health
		        (tenant_id, user_id, service_id, status, latency_ms, status_code, error_message)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID, userID, serviceID, r.Status, r.LatencyMS, r.StatusCode, errMsg,
	)
	return err
}

// nullableString converts an empty string to nil so the JSON
// response omits the field rather than emitting "error": "".
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/services/:id/probe
// ------------------------------------------------------------------

// ProbeHomelabService runs a single one-shot probe and stores the
// result. Synchronous — the response includes the just-recorded
// health row so the UI can update the pill without an extra GET.
func ProbeHomelabService(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		var (
			name, urlStr, kind string
			enabled            bool
		)
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT name, url, kind, enabled FROM homelab_pinned_services
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		).Scan(&name, &urlStr, &kind, &enabled)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				kernel.RespondError(c, kernel.ErrNotFound)
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if !enabled {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "disabled",
				"service is disabled; enable it via PATCH before probing")
			return
		}

		result := probeOne(c.Request.Context(), kind, urlStr)
		if err := insertProbeResult(c.Request.Context(), pool, tenantID, userID, id, result); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"service_id":  id.String(),
			"name":        name,
			"status":      result.Status,
			"latency_ms":  result.LatencyMS,
			"status_code": result.StatusCode,
			"error":       nullableString(result.ErrorMessage),
		})
	}
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/services/:id/history
// ------------------------------------------------------------------

// GetHomelabServiceHistory returns the most recent probe results
// for one service, newest first. Limited to historyRowLimit rows
// (the timeline widget doesn't need more).
func GetHomelabServiceHistory(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		// Confirm the pin belongs to the caller before reading
		// its history. Cheap on (id) PK + (tenant_id, user_id) index.
		var owned bool
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM homelab_pinned_services
			                WHERE id = $1 AND tenant_id = $2 AND user_id = $3)`,
			id, tenantID, userID,
		).Scan(&owned); err != nil {
			kernel.RespondError(c, err)
			return
		}
		if !owned {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, service_id::text, status, latency_ms, status_code,
			        error_message, checked_at
			   FROM homelab_service_health
			  WHERE service_id = $1 AND tenant_id = $2 AND user_id = $3
			  ORDER BY checked_at DESC
			  LIMIT $4`,
			id, tenantID, userID, historyRowLimit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []serviceHealthRow{}
		for rows.Next() {
			var (
				rowID, sid, status string
				latency, code      *int
				errMsg             *string
				checkedAt          time.Time
			)
			if err := rows.Scan(&rowID, &sid, &status, &latency, &code, &errMsg, &checkedAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			out = append(out, serviceHealthRow{
				ID:           rowID,
				ServiceID:    sid,
				Status:       status,
				LatencyMS:    latency,
				StatusCode:   code,
				ErrorMessage: errMsg,
				CheckedAt:    checkedAt,
			})
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"service_id": id.String(),
			"history":    out,
			"count":      len(out),
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/services/probe-all
// ------------------------------------------------------------------

// probeAllTarget is the small payload each background probe goroutine
// receives. Defined here (rather than anonymous in ProbeAll...) so
// the runProbeAll goroutine gets a stable type the compiler can
// reason about.
type probeAllTarget struct {
	ID   uuid.UUID
	URL  string
	Kind string
}

// ProbeAllHomelabServices probes every enabled pin for the caller
// in parallel and returns 202 with the probed count. The actual
// probes run in a fire-and-forget goroutine so the HTTP response
// isn't held up by the slowest service.
//
// Why fire-and-forget: a user with 50 pins at 5s HTTP timeout
// each (worst case) = 250s of wall time. The UI doesn't need the
// results synchronously — it polls /aggregate a few seconds later
// and the pills flip. Returning 202 immediately is the right UX.
func ProbeAllHomelabServices(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, url, kind
			   FROM homelab_pinned_services
			  WHERE tenant_id = $1 AND user_id = $2 AND enabled = true`,
			tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		targets := []probeAllTarget{}
		for rows.Next() {
			var idStr, urlStr, kind string
			if err := rows.Scan(&idStr, &urlStr, &kind); err != nil {
				kernel.RespondError(c, err)
				return
			}
			id, _ := uuid.Parse(idStr)
			targets = append(targets, probeAllTarget{ID: id, URL: urlStr, Kind: kind})
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Fire-and-forget. We detach from the request context so
		// the probes survive the HTTP response — the caller might
		// navigate away mid-batch and we don't want to abort the
		// probe halfway through.
		if len(targets) > 0 {
			go runProbeAll(pool, tenantID, userID, targets)
		}

		c.JSON(http.StatusAccepted, gin.H{
			"probed": len(targets),
			"status": "queued",
			"note":   "results land in homelab_service_health; poll GET /aggregate or /services for fresh pills",
		})
	}
}

// runProbeAll is the background goroutine that probes every
// supplied target in parallel (capped at probeAllMaxPar). Each
// probe error is logged but does not abort the batch — one bad
// host should not skip the rest of the user's pins.
//
// We use a fresh background context (NOT the request context)
// so the probes survive the HTTP response — and a 60s overall
// deadline so a hung target can't keep the goroutine forever.
func runProbeAll(pool *db.Pool, tenantID, userID uuid.UUID, targets []probeAllTarget) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sem := make(chan struct{}, probeAllMaxPar)
	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(t probeAllTarget) {
			defer wg.Done()
			defer func() { <-sem }()
			result := probeOne(ctx, t.Kind, t.URL)
			if err := insertProbeResult(ctx, pool, tenantID, userID, t.ID, result); err != nil {
				log.Printf("homelab probe-all insert failed for service %s: %v", t.ID, err)
			}
		}(t)
	}
	wg.Wait()
}
