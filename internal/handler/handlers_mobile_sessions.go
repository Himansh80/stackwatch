// Tier 13 Phase 2 — Mobile session routes (2 routes).
//
//	GET  /api/v1/mobile/sessions
//	POST /api/v1/mobile/test-push  (super_admin only)
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// GET /api/v1/mobile/sessions — last 30 days of activity for the calling user.
func ListMyMobileSessions(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		uid, err := uuid.Parse(claims.UserID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, COALESCE(device_name, ''), COALESCE(app_version, ''),
			        COALESCE(ip_address, ''), last_active_at, created_at
			   FROM mobile_sessions
			  WHERE user_id = $1 AND last_active_at > NOW() - INTERVAL '30 days'
			  ORDER BY last_active_at DESC
			  LIMIT 100`,
			uid,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []mobileSessionRow{}
		for rows.Next() {
			var s mobileSessionRow
			if err := rows.Scan(&s.ID, &s.DeviceName, &s.AppVersion, &s.IPAddress,
				&s.LastActiveAt, &s.CreatedAt); err != nil {
				continue
			}
			out = append(out, s)
		}
		c.JSON(http.StatusOK, gin.H{"sessions": out, "total": len(out)})
	}
}

// POST /api/v1/mobile/test-push — enqueue a push_log row for each of the
// caller's enabled devices. super_admin only (used by ops to verify
// push plumbing). Phase 1 stub: inserts rows with status='skipped'
// because the actual FCM dispatch lives in Phase 2.
func MobileTestPush(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if claims.Role != "super_admin" && claims.Role != "platform_admin" {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "forbidden",
				"test-push is super_admin or platform_admin only")
			return
		}
		var req testPushReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		uid, err := uuid.Parse(claims.UserID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		// Insert one push_log row per enabled device for this user.
		// Phase 1 stub: status='skipped' (Phase 2 will dispatch real).
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO push_log (tenant_id, device_id, alert_id, title, body, data, status, error_message, created_at)
			 SELECT $1, pd.id, NULL, $2, $3, '{"source":"test-push"}'::jsonb, 'skipped',
			        'Phase 1 stub: actual dispatch deferred to Phase 2', NOW()
			   FROM push_devices pd
			  WHERE pd.tenant_id = $1 AND pd.user_id = $4 AND pd.enabled = true`,
			claims.TenantID, req.Title, req.Body, uid,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{
			"queued":    tag.RowsAffected(),
			"note":      "Phase 1 stub: rows inserted as status='skipped'. Real FCM dispatch ships in Phase 2.",
			"queued_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
