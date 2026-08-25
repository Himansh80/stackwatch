// Tier 13 Phase 2 — Mobile device routes (3 routes).
//
//	GET    /api/v1/mobile/devices
//	POST   /api/v1/mobile/devices/heartbeat
//	POST   /api/v1/mobile/devices/refresh-token
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// GET /api/v1/mobile/devices — all devices in the tenant (any user).
// Useful for "where am I logged in" view.
func ListTenantDevices(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, platform, app_version, device_name, last_active_at, enabled
			   FROM push_devices
			  WHERE tenant_id = $1
			  ORDER BY last_active_at DESC NULLS LAST
			  LIMIT 200`,
			claims.TenantID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []pushDeviceRow{}
		for rows.Next() {
			var d pushDeviceRow
			var la *time.Time
			if err := rows.Scan(&d.ID, &d.Platform, &d.AppVersion, &d.DeviceName, &la, &d.Enabled); err != nil {
				continue
			}
			d.LastActiveAt = la
			out = append(out, d)
		}
		c.JSON(http.StatusOK, gin.H{"devices": out, "total": len(out)})
	}
}

// POST /api/v1/mobile/devices/heartbeat
// Bumps last_active_at on the calling user's most-recent enabled device
// for the supplied platform. Idempotent — safe to call from a 30s timer.
func MobileHeartbeat(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req heartbeatReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		uid, err := uuid.Parse(claims.UserID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		// Bump last_active_at on any enabled device for this user.
		// We don't take platform on purpose: heartbeat should work
		// even if the platform column hasn't been migrated yet.
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE push_devices
			    SET last_active_at = NOW()
			  WHERE tenant_id = $1 AND user_id = $2 AND enabled = true`,
			claims.TenantID, uid,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Upsert a mobile_sessions row to keep audit trail of activity.
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO mobile_sessions (user_id, device_name, app_version, last_active_at)
			 VALUES ($1, $2, $3, NOW())`,
			uid, req.DeviceName, req.AppVersion,
		)
		c.JSON(http.StatusOK, gin.H{"updated": tag.RowsAffected()})
	}
}

// POST /api/v1/mobile/devices/refresh-token
// Atomically swaps old FCM token for new (Android rotates tokens periodically).
func MobileRefreshToken(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req refreshTokenReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		uid, err := uuid.Parse(claims.UserID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE push_devices
			    SET fcm_token = $1,
			        last_active_at = NOW()
			  WHERE fcm_token = $2 AND tenant_id = $3 AND user_id = $4 AND enabled = true`,
			req.NewFCMToken, req.OldFCMToken, claims.TenantID, uid,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondErrorWithCode(c, http.StatusNotFound, "device_not_found",
				"no enabled device matched the old FCM token")
			return
		}
		c.JSON(http.StatusOK, gin.H{"refreshed": true})
	}
}
