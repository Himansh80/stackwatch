package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountMobileRoutes registers Tier 13 — Mobile (React Native + Push) routes.
// Speckit change 011-tier13-mobile.
//
// Phase 0 + Phase 2 complete. 9 protected routes:
//   - GET    /mobile/push/devices
//   - POST   /mobile/push/register          (UPSERT)
//   - DELETE /mobile/push/devices/:id      (soft-disable)
//   - GET    /mobile/push/log
//   - GET    /mobile/devices               (tenant-wide view)
//   - POST   /mobile/devices/heartbeat
//   - POST   /mobile/devices/refresh-token
//   - GET    /mobile/sessions
//   - POST   /mobile/test-push             (super_admin only)
//
// Phase 3+ will add public deep-link routes (e.g. /mobile/confirm/:token).
func mountMobileRoutes(public, protected, admin *gin.RouterGroup, pool *db.Pool) {
	// Push
	protected.GET("/mobile/push/devices", handler.ListMyPushDevices(pool))
	protected.POST("/mobile/push/register", handler.RegisterPushDevice(pool))
	protected.DELETE("/mobile/push/devices/:id", handler.UnregisterPushDevice(pool))
	protected.GET("/mobile/push/log", handler.ListPushLog(pool))

	// Devices
	protected.GET("/mobile/devices", handler.ListTenantDevices(pool))
	protected.POST("/mobile/devices/heartbeat", handler.MobileHeartbeat(pool))
	protected.POST("/mobile/devices/refresh-token", handler.MobileRefreshToken(pool))

	// Sessions
	protected.GET("/mobile/sessions", handler.ListMyMobileSessions(pool))
	protected.POST("/mobile/test-push", handler.MobileTestPush(pool))
}
