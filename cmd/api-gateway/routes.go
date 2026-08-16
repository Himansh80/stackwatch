package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
	"github.com/stackwatch/platform/internal/middleware"
)

// buildRouter constructs the gin engine with all middleware + routes.
func buildRouter(ctx context.Context, logger *slog.Logger, pool *db.Pool, issuer *auth.Issuer) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Global middleware (panic-safe + structured + traceable)
	r.Use(middleware.RequestID())
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.CORS())
	r.Use(middleware.SecurityHeaders())

	// Health endpoints (no auth)
	r.GET("/health", func(c *gin.Context) {
		if err := pool.Health(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "degraded", "db": "down"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "ok", "version": "0.1.0-tier0"})
	})

	// Auth endpoints (no auth required)
	authH := handler.NewAuthHandler(pool, issuer, logger)
	r.POST("/api/v1/auth/login", authH.Login)
	r.POST("/api/v1/auth/signup", authH.Signup)
	r.POST("/api/v1/auth/forgot", authH.ForgotPassword)
	r.POST("/api/v1/auth/reset", authH.ResetPassword)

	// Protected endpoints (require valid JWT)
	protected := r.Group("/api/v1", handler.RequireAuth(issuer))
	protected.GET("/auth/me", authH.Me)
	protected.POST("/auth/logout", authH.Logout)
	protected.POST("/auth/change-password", authH.ChangePassword)

	// Proxmox endpoints (Tier 1)
	proxmoxH := handler.NewProxmoxHandler(pool)
	protected.POST("/proxmox/hosts", proxmoxH.CreateHost)
	protected.GET("/proxmox/hosts", proxmoxH.ListHosts)
	protected.DELETE("/proxmox/hosts/:id", proxmoxH.DeleteHost)
	protected.GET("/proxmox/hosts/:id/test", proxmoxH.TestHost)
	protected.GET("/proxmox/hosts/:id/nodes", proxmoxH.ListNodes)
	protected.GET("/proxmox/hosts/:id/vms", proxmoxH.ListVMs)
	protected.GET("/proxmox/hosts/:id/storage", proxmoxH.ListStorage)
	// Tier 1.2: VM lifecycle
	protected.GET("/proxmox/hosts/:id/nodes/:node/qemu/:vmid", proxmoxH.GetVMConfig)
	protected.PUT("/proxmox/hosts/:id/nodes/:node/qemu/:vmid/config", proxmoxH.UpdateVMConfig)
	protected.POST("/proxmox/hosts/:id/nodes/:node/qemu", proxmoxH.CreateVM)
	protected.DELETE("/proxmox/hosts/:id/nodes/:node/qemu/:vmid", proxmoxH.DeleteVM)
	protected.POST("/proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/:action", proxmoxH.VMStatusAction)
	protected.GET("/proxmox/hosts/:id/nodes/:node/tasks/:upid", proxmoxH.TaskStatus)
	// Tier 1.3: LXC lifecycle
	protected.POST("/proxmox/hosts/:id/nodes/:node/lxc", proxmoxH.CreateLXC)
	protected.GET("/proxmox/hosts/:id/nodes/:node/lxc/:vmid", proxmoxH.GetLXCConfig)
	protected.PUT("/proxmox/hosts/:id/nodes/:node/lxc/:vmid/config", proxmoxH.UpdateLXCConfig)
	protected.DELETE("/proxmox/hosts/:id/nodes/:node/lxc/:vmid", proxmoxH.DeleteLXC)
	protected.POST("/proxmox/hosts/:id/nodes/:node/lxc/:vmid/status/:action", proxmoxH.LXCStatusAction)

	return r
}
