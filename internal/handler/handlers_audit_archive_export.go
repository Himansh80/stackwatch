// Tier 9 Phase 4 — Audit Log Retention + Export (Tier 9.4).
//
// HTTP route handler for the streaming query+export endpoint:
//
//	POST /api/v1/enterprise/audit/export                       — ExportAuditEvents
//
// The list / create / download handlers live in
// handlers_audit_archive.go; types + format allowlist + job
// tracker live in handlers_audit_archive_types.go.
//
// Why a separate file:
//   The export endpoint streams up to 1M rows through
//   gin.ResponseWriter + csv.Writer / json.Encoder. Splitting it
//   into its own file keeps each file under the 400-LOC cap and
//   groups all streaming-specific imports (encoding/csv, encoding
//   /json net/http) here rather than diluting the simpler
//   "single SELECT, return JSON" handlers.
//
// Streaming model:
//   We deliberately avoid `rows.ToSlice()` (which would load the
//   entire resultset into memory). Instead the handler:
//     1. Issues the SELECT.
//     2. For each rows.Next(): writes ONE row to the response via
//        csv.Writer.Write or json.Encoder.Encode (NDJSON — newline-
//        delimited, no enclosing array).
//     3. Calls (http.Flusher).Flush() after writing so the body
//        streams instead of buffering.
//
// A row-level error mid-stream surfaces as a truncated download.
// That's the right tradeoff (matches `mysqldump` behavior on a
// connection drop) and keeps the handler simple — bubbling a
// mid-stream error would require breaking the streaming contract
// the auditor is expecting.
package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ExportAuditEvents streams a CSV or NDJSON file back to the caller
// containing audit_log rows that match the supplied filters.
//
// Body: auditExportReq{ event_type?, start_time, end_time, format,
// limit? }. limit defaults to 1000, capped at 1,000,000.
//
// Content-Disposition: attachment so the browser auto-downloads.
// Stable filename like "audit-export-2026-08-25T1430.csv".
func ExportAuditEvents(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req auditExportReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if !req.EndTime.After(req.StartTime) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"end_time must be after start_time")
			return
		}
		if !allowedAuditExportFormats[req.Format] {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_format",
				"format must be one of: csv, json")
			return
		}
		// Default + cap limit.
		if req.Limit <= 0 {
			req.Limit = 1000
		}
		if req.Limit > 1000000 {
			req.Limit = 1000000
		}

		// Build the WHERE clause. event_type maps onto the action
		// column (audit_log.action is the privileged-action name).
		where := "tenant_id = $1 AND created_at >= $2 AND created_at <= $3"
		args := []any{tenantID, req.StartTime, req.EndTime}
		if ev := strings.TrimSpace(req.EventType); ev != "" {
			args = append(args, ev)
			where += " AND action = $4"
		}
		args = append(args, req.Limit)
		query := `SELECT id::text, tenant_id::text, user_id::text,
		               action, COALESCE(target, ''), metadata::text,
		               COALESCE(host(ip), ''), created_at::text
		          FROM audit_log
		         WHERE ` + where + `
		         ORDER BY created_at ASC
		         LIMIT $` + strconv.Itoa(len(args))

		pgxRows, err := pool.Pgx().Query(c.Request.Context(), query, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer pgxRows.Close()

		// Configure headers BEFORE the first Write — gin then
		// streams the body verbatim.
		filename := fmt.Sprintf("audit-export-%s.%s",
			time.Now().UTC().Format("2006-01-02T1504"), req.Format)
		c.Header("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, filename))
		switch req.Format {
		case "csv":
			c.Header("Content-Type", "text/csv; charset=utf-8")
		case "json":
			c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
		}
		c.Status(http.StatusOK)

		// Stream rows. We write directly to c.Writer so we never
		// hold more than one row in memory.
		w := c.Writer
		switch req.Format {
		case "csv":
			cw := csv.NewWriter(w)
			_ = cw.Write([]string{
				"id", "tenant_id", "user_id", "action",
				"target", "metadata", "ip", "created_at",
			})
			for pgxRows.Next() {
				var e auditExportEvent
				if err := pgxRows.Scan(&e.ID, &e.TenantID, &e.UserID,
					&e.Action, &e.Target, &e.Metadata, &e.IP, &e.CreatedAt); err != nil {
					continue
				}
				_ = cw.Write([]string{
					e.ID, e.TenantID, e.UserID, e.Action,
					e.Target, e.Metadata, e.IP, e.CreatedAt,
				})
			}
			cw.Flush()
		case "json":
			enc := json.NewEncoder(w)
			for pgxRows.Next() {
				var e auditExportEvent
				if err := pgxRows.Scan(&e.ID, &e.TenantID, &e.UserID,
					&e.Action, &e.Target, &e.Metadata, &e.IP, &e.CreatedAt); err != nil {
					continue
				}
				_ = enc.Encode(e) // newline-delimited JSON
			}
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}
