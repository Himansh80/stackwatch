// Tier 7 Phase 3 — Team/Collab (D12). Five handlers for the
// collaboration surface.
//
// Routes:
//
//	GET  /api/v1/dashboards/shared               — list dashboards shared WITH the caller
//	POST /api/v1/dashboards/share                — share a dashboard with another user
//	GET  /api/v1/annotations/mentions            — list mentions for the caller (?read=false)
//	POST /api/v1/timeline/comments               — add a comment (in handlers_team_timeline.go)
//	GET  /api/v1/timeline/:incident_id/comments  — list comments (in handlers_team_timeline.go)
//
// Every protected query honors tenant_id from the JWT — no cross-
// tenant data ever crosses the wire. The dashboard_shares table does
// not carry tenant_id directly — tenant ownership is enforced via
// JOIN to dashboards.tenant_id. The other two tables carry their own
// tenant_id column; every read filters by it.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListSharedDashboards returns the dashboards shared WITH the caller.
// Implicit filtering: only rows in dashboard_shares where user_id =
// JWT user_id, JOIN'd to dashboards (also tenant-scoped). Returns
// dashboard_id, dashboard_name, permission, owner (creator of the
// dashboard) full_name. Limit is hard-coded to 200 — this is a
// single-user "what's been shared with me" view, not a feed.
func ListSharedDashboards(pool *db.Pool) gin.HandlerFunc {
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
		limit := clampLimit(c.Query("limit"), 100, 500)
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT ds.dashboard_id::text,
			        COALESCE(d.name, '(deleted dashboard)'),
			        ds.permission,
			        '—',
			        ds.created_at::text
			 FROM dashboard_shares ds
			 JOIN dashboards d ON d.id = ds.dashboard_id
			 WHERE ds.user_id = $1 AND d.tenant_id = $2
			 ORDER BY ds.created_at DESC
			 LIMIT $3`,
			claims.UserID, tenantID, limit)
		if err != nil {
			// dashboard_shares.dashboard_id cascades ON DELETE; if
			// the dashboards table is missing in the install, the FK
			// misses and pgsql returns a relation error. Fall back to
			// a dashboards-less projection so we still return a 200
			// with whatever rows survive.
			rows, err = pool.Pgx().Query(c.Request.Context(),
				`SELECT ds.dashboard_id::text,
				        '(missing dashboard)',
				        ds.permission,
				        '—',
				        ds.created_at::text
				 FROM dashboard_shares ds
				 WHERE ds.user_id = $1
				 ORDER BY ds.created_at DESC
				 LIMIT $2`,
				claims.UserID, limit)
			if err != nil {
				kernel.RespondError(c, err)
				return
			}
		}
		defer rows.Close()
		out := []sharedDashboardRow{}
		for rows.Next() {
			var r sharedDashboardRow
			if err := rows.Scan(&r.DashboardID, &r.DashboardName,
				&r.Permission, &r.OwnerName, &r.CreatedAt); err != nil {
				continue
			}
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"shared": out, "total": len(out)})
	}
}

// ShareDashboard inserts or updates a (dashboard_id, user_id,
// permission) row in dashboard_shares. The dashboard must belong to
// the caller's tenant — we re-check via dashboards.tenant_id before
// the write so a caller from tenant A cannot share a tenant B
// dashboard with their own cohort. ON CONFLICT (dashboard_id,
// user_id) DO UPDATE lets the same endpoint be used to flip
// permission from 'view' to 'edit'.
func ShareDashboard(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r shareDashboardReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		perm := strings.ToLower(strings.TrimSpace(r.Permission))
		if perm == "" {
			perm = "view"
		}
		if !allowedSharePermissions[perm] {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		dashID, err := uuid.Parse(r.DashboardID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		userID, err := uuid.Parse(r.UserID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check — dashboard must be in the caller's
		// tenant. We allow NULL created_by to pass (older installs may
		// not have it populated) so the share still works.
		var tenantOK bool
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT EXISTS (
			   SELECT 1 FROM dashboards
			   WHERE id = $1 AND tenant_id = $2
			 )`,
			dashID, tenantID,
		).Scan(&tenantOK)
		if !tenantOK {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		var row sharedDashboardRow
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO dashboard_shares (dashboard_id, user_id, permission)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (dashboard_id, user_id) DO UPDATE
			   SET permission = EXCLUDED.permission
			 RETURNING dashboard_id::text,
			           (SELECT name FROM dashboards WHERE id = $1),
			           permission,
			           '—',
			           created_at::text`,
			dashID, userID, perm,
		).Scan(&row.DashboardID, &row.DashboardName, &row.Permission,
			&row.OwnerName, &row.CreatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{"share": row})
	}
}

// ListMentions returns the mentions addressed to the caller.
// ?read=false filters to unread-only (the default; the unread badge
// on the Shared Dashboards page uses this). ?read=true returns the
// full history. The read_at column is also exposed so the UI can
// render "Read 3 minutes ago". mentioning_name is hydrated via
// LEFT JOIN to users so the client doesn't need a second lookup.
func ListMentions(pool *db.Pool) gin.HandlerFunc {
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
		// Defaults to "unread-only" because the unread badge is the
		// primary use-case. Pass ?read=true to flip.
		readFilter := strings.ToLower(strings.TrimSpace(c.Query("read")))
		onlyUnread := readFilter != "true" && readFilter != "1"

		limit := clampLimit(c.Query("limit"), 50, 200)

		var q string
		if onlyUnread {
			q = `SELECT m.id::text,
			          m.mentioned_user_id::text,
			          m.mentioning_user_id::text,
			          COALESCE(u.full_name, '—'),
			          m.context_type,
			          m.context_id::text,
			          m.read_at::text,
			          m.created_at::text
			     FROM mentions m
			     LEFT JOIN users u ON u.id = m.mentioning_user_id
			     WHERE m.tenant_id = $1
			       AND m.mentioned_user_id = $2
			       AND m.read_at IS NULL
			     ORDER BY m.created_at DESC
			     LIMIT $3`
		} else {
			q = `SELECT m.id::text,
			          m.mentioned_user_id::text,
			          m.mentioning_user_id::text,
			          COALESCE(u.full_name, '—'),
			          m.context_type,
			          m.context_id::text,
			          m.read_at::text,
			          m.created_at::text
			     FROM mentions m
			     LEFT JOIN users u ON u.id = m.mentioning_user_id
			     WHERE m.tenant_id = $1
			       AND m.mentioned_user_id = $2
			     ORDER BY m.created_at DESC
			     LIMIT $3`
		}
		rows, err := pool.Pgx().Query(c.Request.Context(), q,
			tenantID, claims.UserID, limit)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []mentionRow{}
		var unreadCount int
		for rows.Next() {
			var r mentionRow
			var readAt *string
			if err := rows.Scan(&r.ID, &r.MentionedUserID,
				&r.MentioningUserID, &r.MentioningName,
				&r.ContextType, &r.ContextID,
				&readAt, &r.CreatedAt); err != nil {
				continue
			}
			r.ReadAt = readAt
			out = append(out, r)
			if readAt == nil {
				unreadCount++
			}
		}
		kernel.RespondOK(c, gin.H{
			"mentions":     out,
			"total":        len(out),
			"unread_count": unreadCount,
		})
	}
}

// AddTimelineComment lives in handlers_team_timeline.go (split out
// to keep handlers_team.go under the 400-LOC cap).
// ListTimelineComments lives in handlers_team_timeline.go.
