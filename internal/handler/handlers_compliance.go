// Tier 9 Phase 5 — Compliance Reports (Tier 9.5).
//
// HTTP route handlers for the four protected report endpoints. The
// background generator goroutine (runReportJob + markReportFailed)
// lives in handlers_compliance_worker.go; the three schedule
// endpoints live in handlers_compliance_schedules.go; types +
// allowlists + sync.Map live in handlers_compliance_types.go.
//
//	GET    /api/v1/enterprise/compliance/reports              — ListComplianceReports
//	POST   /api/v1/enterprise/compliance/reports              — CreateComplianceReport
//	GET    /api/v1/enterprise/compliance/reports/:id           — GetComplianceReport
//	GET    /api/v1/enterprise/compliance/reports/:id/download  — DownloadComplianceReport
//
// All four routes honor tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. The report tables are defined in
// migrations/040_enterprise.sql (tables: compliance_reports,
// compliance_schedules).
//
// Background generator model (POST /compliance/reports):
//   1. Validate framework (allowlist) and period_end > period_start.
//   2. INSERT a row with status='pending' (the worker flips to
//      'running' immediately on entry).
//   3. Acquire a slot in complianceReportSem (buffered, size=4) to
//      cap concurrent generators. The send happens BEFORE
//      `go runReportJob(...)` so a burst of 10 parallel POSTs only
//      spawns 4 goroutines + 6 queued submits.
//   4. Return 201 + the row id IMMEDIATELY. The worker flips
//      status to 'running', assembles a synthetic evidence document
//      from real DB counts (audit log rows / anomaly events / active
//      users / etc.), writes it to the filesystem at
//      /opt/stackwatch/reports/{tenant_id}/{report_id}.txt, and
//      finally UPDATEs the row to 'completed' (with artifact_path
//      + completed_at) or 'failed' (with error_message).
//   5. Release the semaphore slot when the goroutine returns.
package handler

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/compliance/reports
// ------------------------------------------------------------------

// ListComplianceReports returns every compliance_reports row for the
// caller's tenant, sorted newest-first. Honors the optional
// ?framework=X (allowlist), ?status=Y filter and ?limit=N cap.
//
// Query params:
//
//	framework  optional — one of allowedComplianceFrameworks.
//	status     optional — 'pending'|'running'|'completed'|'failed'.
//	limit      optional 1..200; default 50.
//
// Sorted by created_at DESC so the freshest report is first.
func ListComplianceReports(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		q := `SELECT id::text, tenant_id::text, framework,
		             period_start::text, period_end::text,
		             status, COALESCE(artifact_path, ''),
		             COALESCE(scheduled_id::text, ''),
		             COALESCE(requested_by::text, ''),
		             created_at::text,
		             COALESCE(completed_at::text, ''),
		             COALESCE(error_message, '')
		      FROM compliance_reports
		      WHERE tenant_id = $1`
		args := []any{tenantID}

		if v := strings.TrimSpace(c.Query("framework")); v != "" {
			if !allowedComplianceFrameworks[v] {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest,
					"bad_request", "framework not in allowlist")
				return
			}
			args = append(args, v)
			q += " AND framework = $" + strconv.Itoa(len(args))
		}
		if v := strings.TrimSpace(c.Query("status")); v != "" {
			args = append(args, v)
			q += " AND status = $" + strconv.Itoa(len(args))
		}

		limit := 50
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			if parsed, perr := strconv.Atoi(v); perr == nil && parsed > 0 && parsed <= 200 {
				limit = parsed
			}
		}
		args = append(args, limit)
		q += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args))

		pgxRows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer pgxRows.Close()
		rows := []complianceReportRow{}
		for pgxRows.Next() {
			var r complianceReportRow
			if err := pgxRows.Scan(&r.ID, &r.TenantID, &r.Framework,
				&r.PeriodStart, &r.PeriodEnd, &r.Status, &r.ArtifactPath,
				&r.ScheduledID, &r.RequestedBy, &r.CreatedAt,
				&r.CompletedAt, &r.ErrorMessage); err != nil {
				continue
			}
			rows = append(rows, r)
		}
		kernel.RespondOK(c, gin.H{"reports": rows, "total": len(rows)})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/enterprise/compliance/reports
// ------------------------------------------------------------------

// CreateComplianceReport validates the request, INSERTs a
// status='pending' row, acquires a semaphore slot, and spawns the
// background goroutine that assembles the artifact. Returns 201 +
// the row id IMMEDIATELY so the caller can poll /reports/:id for
// status while the goroutine works.
//
// Body: complianceReportReq{ framework, period_start, period_end }.
// Returns 400 on invalid framework or end <= start.
func CreateComplianceReport(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req complianceReportReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if !allowedComplianceFrameworks[req.Framework] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "framework not in allowlist")
			return
		}
		if !req.PeriodEnd.After(req.PeriodStart) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "period_end must be after period_start")
			return
		}

		var (
			rowID     string
			createdAt string
		)
		// requested_by is the JWT subject (user id) — read from the
		// auth claims stashed by RequireAuth. We don't fail the
		// request if it's missing (system-generated reports from
		// the future cron sweep won't have one) — the column is
		// nullable.
		var requestedBy *uuid.UUID
		if v, present := c.Get("auth.user_id"); present {
			if s, ok2 := v.(uuid.UUID); ok2 {
				requestedBy = &s
			}
		}

		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO compliance_reports
			   (tenant_id, framework, period_start, period_end,
			    status, requested_by)
			 VALUES ($1, $2, $3, $4, 'pending', $5)
			 RETURNING id::text, created_at::text`,
			tenantID, req.Framework, req.PeriodStart, req.PeriodEnd,
			requestedBy,
		).Scan(&rowID, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Register in sync.Map for the future queue-depth endpoint.
		job := &complianceReportJob{
			RowID:        rowID,
			TenantID:     tenantID.String(),
			Framework:    req.Framework,
			DispatchedAt: time.Now(),
		}
		complianceReportJobs.Store(rowID, job)

		// Acquire a semaphore slot BEFORE launching — bursty POSTs
		// backpressure here. The goroutine below runs asynchronously;
		// the slot release happens in its defer.
		complianceReportSem <- struct{}{}
		go runReportJob(pool, job, req.Framework,
			req.PeriodStart, req.PeriodEnd)

		kernel.RespondCreated(c, complianceReportRow{
			ID:          rowID,
			TenantID:    tenantID.String(),
			Framework:   req.Framework,
			PeriodStart: req.PeriodStart.UTC().Format(time.RFC3339),
			PeriodEnd:   req.PeriodEnd.UTC().Format(time.RFC3339),
			Status:      "pending",
			CreatedAt:   createdAt,
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/compliance/reports/:id
// ------------------------------------------------------------------

// GetComplianceReport returns a single report row (status +
// artifact_path + completed_at + error_message). The /download
// endpoint is separate so the caller can poll status cheaply
// without streaming the artifact.
//
// Returns 404 if the row doesn't exist OR belongs to a different
// tenant (we don't leak existence).
func GetComplianceReport(pool *db.Pool) gin.HandlerFunc {
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
		var r complianceReportRow
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, tenant_id::text, framework,
			        period_start::text, period_end::text,
			        status, COALESCE(artifact_path, ''),
			        COALESCE(scheduled_id::text, ''),
			        COALESCE(requested_by::text, ''),
			        created_at::text,
			        COALESCE(completed_at::text, ''),
			        COALESCE(error_message, '')
			   FROM compliance_reports
			  WHERE tenant_id = $1 AND id = $2`,
			tenantID, rowID,
		).Scan(&r.ID, &r.TenantID, &r.Framework, &r.PeriodStart,
			&r.PeriodEnd, &r.Status, &r.ArtifactPath, &r.ScheduledID,
			&r.RequestedBy, &r.CreatedAt, &r.CompletedAt, &r.ErrorMessage)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, r)
	}
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/compliance/reports/:id/download
// ------------------------------------------------------------------

// DownloadComplianceReport streams the artifact file for a single
// completed report back to the caller with Content-Disposition:
// attachment so the browser auto-downloads.
//
// Returns 404 if the row doesn't exist OR belongs to a different
// tenant; 409 if status != 'completed'; 500 if the artifact file
// is missing on disk (shouldn't happen but defensive).
func DownloadComplianceReport(pool *db.Pool) gin.HandlerFunc {
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
			status       string
			artifactPath string
			framework    string
		)
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT status, COALESCE(artifact_path, ''), framework
			   FROM compliance_reports
			  WHERE tenant_id = $1 AND id = $2`,
			tenantID, rowID,
		).Scan(&status, &artifactPath, &framework)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		if status != "completed" {
			kernel.RespondErrorWithCode(c, http.StatusConflict,
				"not_ready",
				fmt.Sprintf("report is %q — only completed reports can be downloaded", status))
			return
		}
		if artifactPath == "" {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError,
				"missing_artifact",
				"report row is completed but artifact_path is empty")
			return
		}

		payload, err := os.ReadFile(artifactPath)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError,
				"read_failed",
				fmt.Sprintf("artifact file is unreadable: %v", err))
			return
		}
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.Header("Content-Disposition",
			fmt.Sprintf(`attachment; filename="compliance-%s-%s.txt"`,
				framework, rowID.String()[:8]))
		c.Header("Content-Length", strconv.Itoa(len(payload)))
		c.Data(http.StatusOK, "text/plain; charset=utf-8", payload)
	}
}
