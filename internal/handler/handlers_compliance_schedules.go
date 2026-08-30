// Tier 9 Phase 5 — Compliance Reports (Tier 9.5).
//
// HTTP route handlers for the three protected schedule endpoints.
// The four report endpoints live in handlers_compliance.go; the
// background generator lives in handlers_compliance_worker.go;
// types + allowlists + sync.Map live in handlers_compliance_types.go.
//
//	GET    /api/v1/enterprise/compliance/schedules             — ListComplianceSchedules
//	POST   /api/v1/enterprise/compliance/schedules             — CreateComplianceSchedule
//	DELETE /api/v1/enterprise/compliance/schedules/:id         — DeleteComplianceSchedule
//
// All three routes honor tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. The compliance_schedules table is
// defined in migrations/040_enterprise.sql.
//
// Phase 5 ships the CONFIG surface only — POST creates a row,
// GET lists them, DELETE removes them. The actual cron sweep
// (read WHERE enabled = true AND next_run_at <= now(), generate a
// report, dispatch to recipients) is a future Tier 9.x janitor.
// For now operators can configure schedules ahead of the sweep
// landing and the table will be picked up unchanged.
package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/compliance/schedules
// ------------------------------------------------------------------

// ListComplianceSchedules returns every compliance_schedules row
// for the caller's tenant, sorted by next_run_at ASC (next-due
// first) — that's the natural display order for an operator
// triaging "what's running when".
func ListComplianceSchedules(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		pgxRows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, framework, frequency,
			        recipients, enabled, next_run_at::text,
			        created_at::text
			   FROM compliance_schedules
			  WHERE tenant_id = $1
			  ORDER BY next_run_at ASC`,
			tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer pgxRows.Close()

		rows := []complianceScheduleRow{}
		for pgxRows.Next() {
			var (
				r          complianceScheduleRow
				enabled    bool
				nextRunRaw string
				createdRaw string
			)
			if err := pgxRows.Scan(&r.ID, &r.TenantID, &r.Framework,
				&r.Frequency, &r.Recipients, &enabled, &nextRunRaw,
				&createdRaw); err != nil {
				continue
			}
			r.Enabled = enabled
			r.CreatedAt = createdRaw
			// next_run_at comes back as RFC 3339 text — surface
			// it as-is so the UI can parse it with new Date(...).
			// We deliberately don't store time.Time on the row
			// struct because pgx would scan an instant the caller
			// would have to re-format; the text path keeps the
			// JSON shape flat and identical to the DB shape.
			r.NextRunAt = parseTimeRFC3339(nextRunRaw)
			rows = append(rows, r)
		}
		kernel.RespondOK(c, gin.H{"schedules": rows, "total": len(rows)})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/enterprise/compliance/schedules
// ------------------------------------------------------------------

// CreateComplianceSchedule validates the request and INSERTs a
// new row. Returns 400 on invalid framework, frequency, or empty
// recipients list. Recipients is the email list the future cron
// sweep will mail the generated report to — Phase 5 just persists
// it.
//
//	Body: complianceScheduleReq{
//	  framework, frequency, recipients (optional), enabled (optional,
//	  default true), next_run_at (RFC 3339).
//	}
func CreateComplianceSchedule(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req complianceScheduleReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if !allowedComplianceFrameworks[req.Framework] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "framework not in allowlist")
			return
		}
		if !allowedComplianceFrequencies[req.Frequency] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "frequency must be monthly|quarterly|yearly")
			return
		}
		if req.NextRunAt.IsZero() {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "next_run_at is required")
			return
		}
		// Recipients is optional in the body but if omitted we
		// still want a non-nil empty array to land in the DB so
		// future cron code can iterate it without a nil check.
		if req.Recipients == nil {
			req.Recipients = []string{}
		}

		var (
			rowID     string
			createdAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO compliance_schedules
			   (tenant_id, framework, frequency, recipients,
			    enabled, next_run_at)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, created_at::text`,
			tenantID, req.Framework, req.Frequency, req.Recipients,
			req.Enabled, req.NextRunAt,
		).Scan(&rowID, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, complianceScheduleRow{
			ID:         rowID,
			TenantID:   tenantID.String(),
			Framework:  req.Framework,
			Frequency:  req.Frequency,
			Recipients: req.Recipients,
			Enabled:    req.Enabled,
			NextRunAt:  req.NextRunAt,
			CreatedAt:  createdAt,
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: DELETE /api/v1/enterprise/compliance/schedules/:id
// ------------------------------------------------------------------

// DeleteComplianceSchedule hard-deletes a schedule row. The future
// cron sweep reads WHERE enabled = true so a disabled schedule is
// already a no-op — DELETE is reserved for "I no longer want this
// schedule at all" and is irreversible. The handler returns 204 on
// success and 404 on missing/wrong-tenant id (we don't leak
// existence).
func DeleteComplianceSchedule(pool *db.Pool) gin.HandlerFunc {
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
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM compliance_schedules
			  WHERE tenant_id = $1 AND id = $2`,
			tenantID, rowID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ------------------------------------------------------------------
// Internal helpers
// ------------------------------------------------------------------

// parseTimeRFC3339 returns a time.Time parsed from an RFC 3339
// string — used by the list endpoint which reads next_run_at as
// text from pgx to keep the JSON shape flat. Falls back to the
// zero time on parse error so a corrupt row doesn't 500 the
// whole list.
func parseTimeRFC3339(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
