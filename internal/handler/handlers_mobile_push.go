// Tier 13 Phase 2 — Push routes (4 routes).
//
//	GET    /api/v1/mobile/push/devices
//	POST   /api/v1/mobile/push/register        (UPSERT on (tenant,platform,fcm_token))
//	DELETE /api/v1/mobile/push/devices/:id
//	GET    /api/v1/mobile/push/log
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// GET /api/v1/mobile/push/devices
// Returns devices registered to the calling user (tenant-scoped).
func ListMyPushDevices(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, platform, app_version, device_name, last_active_at, enabled
			   FROM push_devices
			  WHERE tenant_id = $1 AND user_id = $2
			  ORDER BY last_active_at DESC NULLS LAST`,
			claims.TenantID, claims.UserID)
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

// POST /api/v1/mobile/push/register — UPSERT
func RegisterPushDevice(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req registerDeviceReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		uid, err := uuid.Parse(claims.UserID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var id string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO push_devices (tenant_id, user_id, platform, fcm_token, app_version, device_name, last_active_at)
			 VALUES ($1, $2, $3, $4, $5, $6, NOW())
			 ON CONFLICT (tenant_id, platform, fcm_token) DO UPDATE
			   SET app_version = EXCLUDED.app_version,
			       device_name = EXCLUDED.device_name,
			       last_active_at = NOW(),
			       enabled = true
			 RETURNING id::text`,
			claims.TenantID, uid, req.Platform, req.FCMToken, req.AppVersion, req.DeviceName,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "platform": req.Platform})
	}
}

// DELETE /api/v1/mobile/push/devices/:id — soft-disable
func UnregisterPushDevice(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE push_devices SET enabled = false WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, claims.TenantID, claims.UserID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		c.JSON(http.StatusOK, gin.H{"unregistered": true})
	}
}

// GET /api/v1/mobile/push/log
func ListPushLog(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := tenantClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT pl.id::text, pl.device_id::text, COALESCE(pl.alert_id::text, ''),
			        pl.title, pl.body, pl.status, COALESCE(pl.fcm_message_id, ''),
			        COALESCE(pl.apns_id, ''), pl.sent_at, COALESCE(pl.error_message, ''),
			        pl.created_at
			   FROM push_log pl
			   JOIN push_devices pd ON pd.id = pl.device_id
			  WHERE pl.tenant_id = $1 AND pd.user_id = $2
			  ORDER BY pl.created_at DESC
			  LIMIT 50`,
			claims.TenantID, claims.UserID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []pushLogRow{}
		for rows.Next() {
			var l pushLogRow
			var sa *time.Time
			if err := rows.Scan(&l.ID, &l.DeviceID, &l.AlertID, &l.Title, &l.Body,
				&l.Status, &l.FCMMessageID, &l.APNsID, &sa, &l.ErrorMessage, &l.CreatedAt); err != nil {
				continue
			}
			l.SentAt = sa
			out = append(out, l)
		}
		c.JSON(http.StatusOK, gin.H{"log": out, "total": len(out)})
	}
}

// --- helpers ---

// tenantClaimsFromContext returns (claims, ok) for any JWT-bearer auth.
func tenantClaimsFromContext(c *gin.Context) (*authClaimsLite, bool) {
	v, ok := c.Get("auth_claims")
	if !ok {
		return nil, false
	}
	c2, ok := v.(*authClaimsLite)
	return c2, ok
}

// authClaimsLite is the minimal claims subset needed by Phase 2 handlers.
// Avoids importing the full Claims struct (which has 20+ fields).
type authClaimsLite struct {
	TenantID string
	UserID   string
	Role     string
}
