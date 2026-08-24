// Tier 7 Phase 2 — Security (D6) — SIEM event stream.
//
//	GET /api/v1/security/siem — paginated event feed (filter ?severity=&time_range=&limit=)
//
// Phase 2 ships a read-only feed; ingestion lives in a future change
// (the auth path can already write to siem_events via a separate
// helper). Every query honors tenant_id from the JWT.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// siemEventRow is the JSON shape returned for a single SIEM event.
type siemEventRow struct {
	ID         string `json:"id"`
	EventType  string `json:"event_type"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Source     string `json:"source"`
	OccurredAt string `json:"occurred_at"`
	Metadata   string `json:"metadata"` // raw JSON string
}

// siemTimeRange parses the optional ?time_range= param. Mirrors
// handlers_rum_query.go's rumTimeRange vocabulary so the UI can pass
// the same strings across surfaces.
func siemTimeRange(c *gin.Context) string {
	switch strings.ToLower(strings.TrimSpace(c.Query("time_range"))) {
	case "1h", "1hour":
		return "1 hour"
	case "7d", "7days", "week":
		return "7 days"
	case "30d", "30days", "month":
		return "30 days"
	case "24h", "24hours", "1d", "day", "":
		return "24 hours"
	default:
		return "24 hours"
	}
}

// ListSecuritySIEM returns the SIEM event feed for the caller's tenant.
// Default window is 24h to keep the response size manageable; clients
// can expand with ?time_range=30d. Severity filter accepts the same
// four values as the threats endpoint.
func ListSecuritySIEM(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		severity := strings.ToLower(strings.TrimSpace(c.Query("severity")))
		interval := siemTimeRange(c)
		limit := clampLimit(c.Query("limit"), 50, 500)

		args := []any{tenantID, interval}
		q := `SELECT id::text, event_type, severity, message,
		             COALESCE(source, ''), occurred_at::text,
		             COALESCE(metadata::text, '{}')
		      FROM siem_events
		      WHERE tenant_id = $1 AND occurred_at > NOW() - $2::interval`
		if allowedThreatSeverities[severity] {
			args = append(args, severity)
			q += " AND severity = $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY occurred_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []siemEventRow{}
		for rows.Next() {
			var e siemEventRow
			if err := rows.Scan(&e.ID, &e.EventType, &e.Severity, &e.Message,
				&e.Source, &e.OccurredAt, &e.Metadata); err != nil {
				continue
			}
			out = append(out, e)
		}
		kernel.RespondOK(c, gin.H{"events": out, "total": len(out)})
	}
}
