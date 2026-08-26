// Tier 7 Phase 2 — Security (D6) — Threat detection listing.
//
//	GET /api/v1/security/threats — list threats (filter ?severity=&resolved=)
//
// Read-only surface in Phase 2. Threat ingest (the pipeline that
// populates security_threats) ships in a future change. Every query
// honors tenant_id from the JWT.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// allowedThreatSeverities — defense in depth at the API edge. Spec
// defines these four values; anything else is silently dropped from
// the WHERE clause so the caller still gets a result rather than an
// empty page.
var allowedThreatSeverities = map[string]bool{
	"low": true, "medium": true, "high": true, "critical": true,
}

// securityThreatRow is the JSON shape returned for a single threat.
type securityThreatRow struct {
	ID          string  `json:"id"`
	Severity    string  `json:"severity"`
	ThreatType  string  `json:"threat_type"`
	SourceIP    *string `json:"source_ip,omitempty"`
	UserID      *string `json:"user_id,omitempty"`
	Description string  `json:"description"`
	DetectedAt  string  `json:"detected_at"`
	ResolvedAt  *string `json:"resolved_at,omitempty"`
	Metadata    string  `json:"metadata"` // raw JSON string for client-side inspection
}

// ListSecurityThreats returns threats for the caller's tenant, newest
// first. Filters:
//   - ?severity=low|medium|high|critical
//   - ?resolved=true|false  (default: both)
func ListSecurityThreats(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		severity := strings.ToLower(strings.TrimSpace(c.Query("severity")))
		resolved := strings.ToLower(strings.TrimSpace(c.Query("resolved")))
		limit := clampLimit(c.Query("limit"), 100, 200)

		args := []any{tenantID}
		q := `SELECT id::text, severity, threat_type, source_ip, user_id::text,
		             description, detected_at::text, resolved_at::text,
		             COALESCE(metadata::text, '{}')
		      FROM security_threats
		      WHERE tenant_id = $1`
		if allowedThreatSeverities[severity] {
			args = append(args, severity)
			q += " AND severity = $" + itoa(len(args))
		}
		// resolved=true  → resolved_at IS NOT NULL
		// resolved=false → resolved_at IS NULL
		// otherwise no filter
		if resolved == "true" {
			q += " AND resolved_at IS NOT NULL"
		} else if resolved == "false" {
			q += " AND resolved_at IS NULL"
		}
		args = append(args, limit)
		q += " ORDER BY detected_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []securityThreatRow{}
		for rows.Next() {
			var t securityThreatRow
			var resolvedAt *string
			if err := rows.Scan(&t.ID, &t.Severity, &t.ThreatType, &t.SourceIP,
				&t.UserID, &t.Description, &t.DetectedAt, &resolvedAt, &t.Metadata); err != nil {
				continue
			}
			t.ResolvedAt = resolvedAt
			out = append(out, t)
		}
		kernel.RespondOK(c, gin.H{"threats": out, "total": len(out)})
	}
}

// clampLimit parses the optional ?limit= param into [1, max] range.
// def is used when missing/invalid; max is the hard ceiling so a
// misbehaving client can't pull the whole table.
func clampLimit(raw string, def, max int) int {
	if raw == "" {
		return def
	}
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
		if n > max {
			return max
		}
	}
	if n <= 0 {
		return def
	}
	return n
}
