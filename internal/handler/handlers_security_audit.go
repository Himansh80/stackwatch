// Tier 7 Phase 2 — Security (D6) — Audit trail listing.
//
//	GET /api/v1/security/audit-trails — list audit events (alias to existing audit_log)
//
// Phase 2 ships a read-only surface over the legacy audit_log table
// (Tier 0). Future writes (a dedicated audit ingest) can target the
// same table — the security endpoint is just a tenant-scoped view.
//
// Filters:
//   - ?event_type=<action prefix or exact match>
//   - ?start_time=<RFC3339>          (default: 30 days ago)
//   - ?limit=<1..500>                (default: 50)
package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// auditEventRow is the JSON shape returned for a single audit event.
type auditEventRow struct {
	ID        int64   `json:"id"`
	UserID    *string `json:"user_id,omitempty"`
	Action    string  `json:"action"`
	Target    *string `json:"target,omitempty"`
	IP        *string `json:"ip,omitempty"`
	Metadata  string  `json:"metadata"` // raw JSON
	CreatedAt string  `json:"created_at"`
}

// ListSecurityAuditTrails returns audit events for the caller's tenant.
// The legacy audit_log uses BIGSERIAL ids; we expose them as int64 so
// pagination cursors can stay opaque.
func ListSecurityAuditTrails(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		eventType := strings.TrimSpace(c.Query("event_type"))
		startTime := strings.TrimSpace(c.Query("start_time"))
		limit := clampLimit(c.Query("limit"), 50, 500)

		// Parse the optional start_time; default to 30 days ago.
		// Falls back to the default on any parse error so a malformed
		// query string returns a usable response rather than 400.
		since := time.Now().UTC().Add(-30 * 24 * time.Hour)
		if startTime != "" {
			if t, err := time.Parse(time.RFC3339, startTime); err == nil {
				since = t.UTC()
			}
		}

		args := []any{tenantID, since}
		q := `SELECT id, user_id::text, action, target,
		             host(ip)::text AS ip_text,
		             COALESCE(metadata::text, '{}'), created_at::text
		      FROM audit_log
		      WHERE tenant_id = $1 AND created_at >= $2`
		if eventType != "" {
			args = append(args, "%"+eventType+"%")
			q += " AND action ILIKE $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY created_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []auditEventRow{}
		for rows.Next() {
			var e auditEventRow
			var ipText *string
			if err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.Target,
				&ipText, &e.Metadata, &e.CreatedAt); err != nil {
				continue
			}
			e.IP = ipText
			out = append(out, e)
		}
		kernel.RespondOK(c, gin.H{"events": out, "total": len(out)})
	}
}
