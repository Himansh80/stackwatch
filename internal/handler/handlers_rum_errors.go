// Tier 7 — RUM Full (D4) — Error group query helpers.
//
//	GET /api/v1/rum/error-groups           — list deduped error groups
//	                                        (sorted by occurrence_count DESC)
//
// pgxRows is a tiny alias so we can share the typed rows handle across
// the error-group query without depending on the underlying pgx type
// at every call site. For our usage we only need Next/Scan/Close + Err.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListRUMErrorGroups returns the deduped error list for the tenant,
// sorted by occurrence_count DESC. Optional ?service=X filter.
func ListRUMErrorGroups(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := rumLimit(c, 100)
		service := strings.TrimSpace(c.Query("service"))
		ctx := c.Request.Context()
		var (
			rows pgxRows
			err  error
		)
		if service == "" {
			rows, err = pool.Pgx().Query(ctx,
				`SELECT id::text, fingerprint, message,
				        COALESCE(stack_trace, ''), COALESCE(service, ''),
				        COALESCE(source, ''), occurrence_count,
				        first_seen_at::text, last_seen_at::text
				 FROM rum_error_groups
				 WHERE tenant_id = $1
				 ORDER BY occurrence_count DESC, last_seen_at DESC
				 LIMIT $2`,
				tenantID, limit,
			)
		} else {
			rows, err = pool.Pgx().Query(ctx,
				`SELECT id::text, fingerprint, message,
				        COALESCE(stack_trace, ''), COALESCE(service, ''),
				        COALESCE(source, ''), occurrence_count,
				        first_seen_at::text, last_seen_at::text
				 FROM rum_error_groups
				 WHERE tenant_id = $1 AND service = $2
				 ORDER BY occurrence_count DESC, last_seen_at DESC
				 LIMIT $3`,
				tenantID, service, limit,
			)
		}
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var id, fp, msg, stack, svc, src, firstSeen, lastSeen string
			var count int
			if err := rows.Scan(&id, &fp, &msg, &stack, &svc, &src, &count, &firstSeen, &lastSeen); err != nil {
				continue
			}
			out = append(out, gin.H{
				"id":               id,
				"fingerprint":      fp,
				"message":          msg,
				"stack_trace":      stack,
				"service":          svc,
				"source":           src,
				"occurrence_count": count,
				"first_seen_at":    firstSeen,
				"last_seen_at":     lastSeen,
			})
		}
		kernel.RespondOK(c, gin.H{"groups": out, "total": len(out)})
	}
}

// pgxRows is the minimal interface the query helpers depend on. Keeps
// every handler signature free of pgx-specific type names.
type pgxRows interface {
	Close()
	Next() bool
	Scan(...any) error
	Err() error
}
