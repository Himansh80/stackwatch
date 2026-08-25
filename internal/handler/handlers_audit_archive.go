// Tier 9 Phase 4 — Audit Log Retention + Export (Tier 9.4).
//
// HTTP route handlers for the list / create / download archive
// endpoints. The streaming export endpoint lives in
// handlers_audit_archive_export.go; types + allowlist + sync.Map
// live in handlers_audit_archive_types.go.
//
//	GET    /api/v1/enterprise/audit/archives                   — ListAuditArchives
//	POST   /api/v1/enterprise/audit/archive                    — CreateAuditArchive
//	GET    /api/v1/enterprise/audit/archives/:id/download      — DownloadAuditArchive
//
// All three routes honor tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. The archive table itself is defined
// in migrations/040_enterprise.sql (table: audit_log_archive).
//
// Background job model (POST /audit/archive):
//   1. Validate period_end > period_start.
//   2. Generate batch_id (RFC3339-based prefix) + INSERT a row with
//      status='running', event_count=0, size_bytes=0.
//   3. Acquire a slot in auditArchiveSem (buffered, size=4) to cap
//      concurrent compression goroutines. Drop into the semaphore
//      (`send`) BEFORE `go runArchiveJob(...)` so a burst of 10
//      parallel POSTs only spawns 4 goroutines + 6 queued submits.
//   4. Return 201 + the row id IMMEDIATELY. The goroutine does the
//      work asynchronously and UPDATEs the row to status='completed'
//      or 'failed'.
//   5. Release the semaphore slot when the goroutine returns.
package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/audit/archives
// ------------------------------------------------------------------

// ListAuditArchives returns every archive row for the caller's
// tenant, sorted newest-first. Honors the optional ?since=ISO query
// (filter for archives created since that timestamp) and ?limit=N
// (cap result count; default 50, max 200).
//
// Query params:
//
//	since  optional RFC 3339 timestamp — only archives whose
//	       created_at > since are returned.
//	limit  optional 1..200; default 50.
//
// Sorted by created_at DESC so the freshest archive is first.
func ListAuditArchives(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		q := `SELECT id::text, tenant_id::text, batch_id, event_count,
		             period_start::text, period_end::text,
		             size_bytes, status, created_at::text
		      FROM audit_log_archive
		      WHERE tenant_id = $1`
		args := []any{tenantID}
		if sinceRaw := strings.TrimSpace(c.Query("since")); sinceRaw != "" {
			since, err := time.Parse(time.RFC3339, sinceRaw)
			if err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"since query param must be RFC 3339 / ISO 8601")
				return
			}
			args = append(args, since)
			q += ` AND created_at > $2`
		}
		limit := 50
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			if parsed, perr := strconv.Atoi(v); perr == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}
		args = append(args, limit)
		q += ` ORDER BY created_at DESC LIMIT $` + strconv.Itoa(len(args))

		pgxRows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer pgxRows.Close()
		rows := []auditArchiveRow{}
		for pgxRows.Next() {
			var r auditArchiveRow
			if err := pgxRows.Scan(&r.ID, &r.TenantID, &r.BatchID, &r.EventCount,
				&r.PeriodStart, &r.PeriodEnd, &r.SizeBytes, &r.Status, &r.CreatedAt); err != nil {
				continue
			}
			rows = append(rows, r)
		}
		kernel.RespondOK(c, gin.H{"archives": rows, "total": len(rows)})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/enterprise/audit/archive
// ------------------------------------------------------------------

// CreateAuditArchive inserts a 'running' archive row + spawns the
// background compression job, then returns 201 + the row id. The
// goroutine fills in compressed_payload + size_bytes + status.
//
// Body: auditArchiveReq{ period_start, period_end }. Both required.
// Returns 400 if period_end <= period_start; 409 on the (extremely
// unlikely) batch_id collision.
func CreateAuditArchive(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req auditArchiveReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if !req.PeriodEnd.After(req.PeriodStart) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"period_end must be after period_start")
			return
		}

		// batch_id is operator-facing (UI displays it) so we make
		// it human-readable: "BATCH-2026-08-25T1430-7f2a" using
		// the dispatch minute + a uuid prefix. UNIQUE constraint
		// ensures we don't collide on simultaneous submissions.
		batchID := fmt.Sprintf("BATCH-%s-%s",
			req.PeriodStart.UTC().Format("2006-01-02T1504"),
			uuid.New().String()[:4])

		var (
			rowID     string
			createdAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO audit_log_archive
			   (tenant_id, batch_id, event_count, period_start, period_end, status)
			 VALUES ($1, $2, 0, $3, $4, 'running')
			 RETURNING id::text, created_at::text`,
			tenantID, batchID, req.PeriodStart, req.PeriodEnd,
		).Scan(&rowID, &createdAt)
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "batch_collision",
					"an archive with this period is already in flight")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Register in sync.Map for the future queue-depth endpoint.
		job := &auditArchiveJob{
			RowID:        rowID,
			TenantID:     tenantID.String(),
			BatchID:      batchID,
			DispatchedAt: time.Now(),
		}
		auditArchiveJobs.Store(rowID, job)

		// Acquire a semaphore slot BEFORE launching — bursty POSTs
		// backpressure here, returning 201 only after all preceding
		// jobs have freed a slot. The goroutine below runs
		// asynchronously; the slot release happens in its defer.
		auditArchiveSem <- struct{}{}
		go runArchiveJob(pool, job, req.PeriodStart, req.PeriodEnd)

		kernel.RespondCreated(c, auditArchiveRow{
			ID:          rowID,
			TenantID:    tenantID.String(),
			BatchID:     batchID,
			EventCount:  0,
			PeriodStart: req.PeriodStart.UTC().Format(time.RFC3339),
			PeriodEnd:   req.PeriodEnd.UTC().Format(time.RFC3339),
			SizeBytes:   0,
			Status:      "running",
			CreatedAt:   createdAt,
		})
	}
}

// runArchiveJob is the background compression worker. It (1) fetches
// every audit_log row for the (tenant, period), (2) JSON-marshals
// the resulting []auditExportEvent, (3) gzip-compresses, and (4)
// UPDATEs the row to status='completed' + size_bytes + event_count.
//
// On error it UPDATEs status='failed'. Either way the semaphore
// slot is released (defer) and the in-flight entry is removed from
// auditArchiveJobs (defer) so subsequent requests don't see a stale
// job record.
func runArchiveJob(pool *db.Pool, job *auditArchiveJob, periodStart, periodEnd time.Time) {
	defer func() {
		// Release the semaphore slot even on panic.
		<-auditArchiveSem
		auditArchiveJobs.Delete(job.RowID)
	}()

	ctxTimeout, cancel := context.WithTimeout(context.Background(), auditArchiveJobTimeout)
	defer cancel()

	tenantUUID, err := uuid.Parse(job.TenantID)
	if err != nil {
		markArchiveFailed(pool, job.RowID, err)
		return
	}

	// (1) Fetch every audit_log row for the period. Scan into a
	// local slice — this in-memory list is what we JSON-marshal in
	// step (2) before gzip. For VERY large archives (>1M rows)
	// operators should chunk to multiple smaller archives.
	pgxRows, err := pool.Pgx().Query(ctxTimeout,
		`SELECT id::text, tenant_id::text, user_id::text,
		        action, COALESCE(target, ''), metadata::text,
		        COALESCE(host(ip), ''), created_at::text
		   FROM audit_log
		  WHERE tenant_id = $1
		    AND created_at >= $2 AND created_at <= $3
		  ORDER BY created_at ASC`,
		tenantUUID, periodStart, periodEnd)
	if err != nil {
		markArchiveFailed(pool, job.RowID, err)
		return
	}
	defer pgxRows.Close()
	events := []auditExportEvent{}
	for pgxRows.Next() {
		var e auditExportEvent
		if err := pgxRows.Scan(&e.ID, &e.TenantID, &e.UserID,
			&e.Action, &e.Target, &e.Metadata, &e.IP, &e.CreatedAt); err != nil {
			continue
		}
		events = append(events, e)
	}

	// (2) JSON-marshal the deterministic array.
	payload, err := json.Marshal(events)
	if err != nil {
		markArchiveFailed(pool, job.RowID, err)
		return
	}

	// (3) Gzip-compress. We use a bytes.Buffer + stdlib gzip so no
	// new dependencies land in go.mod.
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(payload); err != nil {
		markArchiveFailed(pool, job.RowID, err)
		return
	}
	if err := gz.Close(); err != nil {
		markArchiveFailed(pool, job.RowID, err)
		return
	}

	// (4) UPDATE row → 'completed' with the gzip bytes.
	_, err = pool.Pgx().Exec(ctxTimeout,
		`UPDATE audit_log_archive
		    SET event_count = $1,
		        compressed_payload = $2,
		        size_bytes = $3,
		        status = 'completed'
		  WHERE id = $4 AND tenant_id = $5`,
		len(events), compressed.Bytes(), int64(compressed.Len()),
		job.RowID, tenantUUID)
	if err != nil {
		markArchiveFailed(pool, job.RowID, err)
		return
	}
}

// markArchiveFailed flips the row to status='failed' so the UI
// surfaces the error rather than spinning forever on 'running'.
// Used by runArchiveJob on every error path. Best-effort —
// background goroutine can't propagate failures, so the row just
// stays at 'failed' until a future Tier 9.x janitor sweeps.
func markArchiveFailed(pool *db.Pool, rowID string, cause error) {
	if cause != nil {
		_ = cause.Error()
	}
	_, _ = pool.Pgx().Exec(context.Background(),
		`UPDATE audit_log_archive SET status = 'failed' WHERE id = $1`, rowID)
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/audit/archives/:id/download
// ------------------------------------------------------------------

// DownloadAuditArchive streams the gzip blob for a single archive
// row back to the caller with Content-Disposition: attachment so
// the browser auto-downloads. Returns 404 if the row doesn't exist
// OR belongs to a different tenant (we don't leak existence);
// 409 if status != 'completed'.
func DownloadAuditArchive(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rowID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var (
			batchID string
			status  string
			payload []byte
		)
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT batch_id, status, compressed_payload
			   FROM audit_log_archive
			  WHERE tenant_id = $1 AND id = $2`,
			tenantID, rowID,
		).Scan(&batchID, &status, &payload)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		if status != "completed" {
			kernel.RespondErrorWithCode(c, http.StatusConflict, "not_ready",
				fmt.Sprintf("archive is %q — only completed archives can be downloaded", status))
			return
		}
		c.Header("Content-Type", "application/gzip")
		c.Header("Content-Disposition",
			fmt.Sprintf(`attachment; filename="audit-archive-%s.json.gz"`, batchID))
		c.Header("Content-Length", strconv.Itoa(len(payload)))
		c.Data(http.StatusOK, "application/gzip", payload)
	}
}
