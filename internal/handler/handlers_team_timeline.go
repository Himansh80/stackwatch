// Tier 7 Phase 3 — Team/Collab (D12). Timeline comments endpoints.
//
// Routes:
//
//	POST /api/v1/timeline/comments              — add a comment to an incident timeline
//	GET  /api/v1/timeline/:incident_id/comments — list comments for an incident
//
// Split out of handlers_team.go to keep that file under the 400-LOC
// cap. Comments live in the `timeline_comments` table (see
// migrations/038_team.sql) — tenant_id filter is the primary ACL.
package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// AddTimelineComment inserts a comment on an incident's timeline.
// incident_id is optional — comments with no incident are stored as
// audit-trail rows. The author_id is the caller's JWT user_id. The
// composite (tenant_id, incident_id, created_at DESC) index makes
// the per-incident list query near-free.
func AddTimelineComment(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r timelineCommentReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		body := strings.TrimSpace(r.Body)
		if body == "" {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var (
			incidentID *uuid.UUID
			row        timelineCommentRow
			userName   string
		)
		if r.IncidentID != "" {
			id, err := uuid.Parse(r.IncidentID)
			if err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			// Tenant ownership check — comment must reference an
			// incident in the caller's tenant. We return 404 (not
			// 403) on miss so a tenant B caller can't enumerate
			// which incident ids exist in tenant A.
			var tenantOK bool
			_ = pool.Pgx().QueryRow(c.Request.Context(),
				`SELECT EXISTS (
				   SELECT 1 FROM incidents
				   WHERE id = $1 AND tenant_id = $2
				 )`,
				id, tenantID,
			).Scan(&tenantOK)
			if !tenantOK {
				kernel.RespondError(c, kernel.ErrNotFound)
				return
			}
			incidentID = &id
		}
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO timeline_comments (tenant_id, incident_id, user_id, body)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id::text,
			           incident_id::text,
			           user_id::text,
			           (SELECT COALESCE(full_name, '—') FROM users WHERE id = $3),
			           body,
			           created_at::text`,
			tenantID, incidentID, claims.UserID, body,
		).Scan(&row.ID, &row.IncidentID, &row.UserID, &userName,
			&row.Body, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		row.UserName = userName
		kernel.RespondCreated(c, gin.H{"comment": row})
	}
}

// ListTimelineComments returns comments for an incident, oldest
// first (?asc=false flips to newest-first). Cap is 200 rows; the
// timeline UI never pages further. 404 if the incident doesn't
// exist in the caller's tenant — same 404-as-no-403 rule as
// AddTimelineComment so we don't leak existence.
func ListTimelineComments(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		incidentID, err := uuid.Parse(c.Param("incident_id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check.
		var tenantOK bool
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT EXISTS (
			   SELECT 1 FROM incidents
			   WHERE id = $1 AND tenant_id = $2
			 )`,
			incidentID, tenantID,
		).Scan(&tenantOK)
		if !tenantOK {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		ascending := strings.ToLower(strings.TrimSpace(c.Query("asc"))) != "false"
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
				limit = n
			}
		}
		order := "ASC"
		if !ascending {
			order = "DESC"
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT tc.id::text,
			        tc.incident_id::text,
			        tc.user_id::text,
			        COALESCE(u.full_name, '—'),
			        tc.body,
			        tc.created_at::text
			 FROM timeline_comments tc
			 LEFT JOIN users u ON u.id = tc.user_id
			 WHERE tc.tenant_id = $1 AND tc.incident_id = $2
			 ORDER BY tc.created_at `+order+`
			 LIMIT $3`,
			tenantID, incidentID, limit)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []timelineCommentRow{}
		for rows.Next() {
			var r timelineCommentRow
			if err := rows.Scan(&r.ID, &r.IncidentID, &r.UserID,
				&r.UserName, &r.Body, &r.CreatedAt); err != nil {
				continue
			}
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{
			"comments": out,
			"total":    len(out),
		})
	}
}
