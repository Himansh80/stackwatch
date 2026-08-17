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
	// Tier 1.4: Storage management
	protected.GET("/proxmox/hosts/:id/storage", proxmoxH.ListStorageEntries)
	protected.POST("/proxmox/hosts/:id/storage", proxmoxH.CreateStorage)
	protected.DELETE("/proxmox/hosts/:id/storage/:name", proxmoxH.DeleteStorage)
	protected.GET("/proxmox/hosts/:id/nodes/:node/storage/:storage/content", proxmoxH.ListStorageContent)
	protected.DELETE("/proxmox/hosts/:id/nodes/:node/storage/:storage/content", proxmoxH.DeleteStorageContent)
	// Tier 1.5: Networking
	protected.GET("/proxmox/hosts/:id/nodes/:node/network", proxmoxH.ListNetwork)
	protected.POST("/proxmox/hosts/:id/nodes/:node/network", proxmoxH.CreateNetwork)
	protected.PUT("/proxmox/hosts/:id/nodes/:node/network/:iface", proxmoxH.UpdateNetwork)
	protected.DELETE("/proxmox/hosts/:id/nodes/:node/network/:iface", proxmoxH.DeleteNetwork)
	// Tier 1.6: Firewall
	protected.GET("/proxmox/hosts/:id/nodes/:node/firewall/rules", proxmoxH.ListFirewallRules)
	protected.POST("/proxmox/hosts/:id/nodes/:node/firewall/rules", proxmoxH.CreateFirewallRule)
	protected.PUT("/proxmox/hosts/:id/nodes/:node/firewall/rules/:pos", proxmoxH.UpdateFirewallRule)
	protected.DELETE("/proxmox/hosts/:id/nodes/:node/firewall/rules/:pos", proxmoxH.DeleteFirewallRule)
	protected.GET("/proxmox/hosts/:id/firewall/ipsets", proxmoxH.ListIPsets)
	protected.POST("/proxmox/hosts/:id/firewall/ipsets", proxmoxH.CreateIPset)
	protected.DELETE("/proxmox/hosts/:id/firewall/ipsets/:name", proxmoxH.DeleteIPset)
	// Tier 1.7: Disks
	protected.GET("/proxmox/hosts/:id/nodes/:node/disks/list", proxmoxH.ListDisks)
	protected.GET("/proxmox/hosts/:id/nodes/:node/disks/zfs", proxmoxH.ListZFSPools)
	protected.POST("/proxmox/hosts/:id/nodes/:node/disks/zfs", proxmoxH.CreateZFSPool)
	protected.DELETE("/proxmox/hosts/:id/nodes/:node/disks/zfs/:name", proxmoxH.DestroyZFSPool)
	// Tier 1.8: Users & tokens
	protected.GET("/proxmox/hosts/:id/access/users", proxmoxH.ListUsers)
	protected.POST("/proxmox/hosts/:id/access/users", proxmoxH.CreateUser)
	protected.GET("/proxmox/hosts/:id/access/users/:userid", proxmoxH.GetUser)
	protected.PUT("/proxmox/hosts/:id/access/users/:userid", proxmoxH.UpdateUser)
	protected.DELETE("/proxmox/hosts/:id/access/users/:userid", proxmoxH.DeleteUser)
	protected.GET("/proxmox/hosts/:id/access/users/:userid/token", proxmoxH.ListAPITokens)
	protected.POST("/proxmox/hosts/:id/access/users/:userid/token", proxmoxH.CreateAPIToken)
	protected.DELETE("/proxmox/hosts/:id/access/users/:userid/token/:tokenid", proxmoxH.DeleteAPIToken)
	// Tier 1.9: Tasks
	protected.GET("/proxmox/hosts/:id/cluster/tasks", proxmoxH.ListClusterTasks)
	protected.GET("/proxmox/hosts/:id/nodes/:node/tasks", proxmoxH.ListNodeTasks)
	protected.GET("/proxmox/hosts/:id/nodes/:node/tasks/:upid/status", proxmoxH.GetTaskStatus)
	protected.GET("/proxmox/hosts/:id/nodes/:node/tasks/:upid/log", proxmoxH.GetTaskLog)
	protected.DELETE("/proxmox/hosts/:id/nodes/:node/tasks/:upid", proxmoxH.StopTask)
	// Tier 1.10: Storage content (ISOs, backups, templates) + ISCSI
	protected.GET("/proxmox/hosts/:id/nodes/:node/storage/:storage/download/*volume", proxmoxH.DownloadStorageFile)
	protected.GET("/proxmox/hosts/:id/nodes/:node/storage/:storage/upload", proxmoxH.GetStorageUploadURL)
	protected.GET("/proxmox/hosts/:id/nodes/:node/iscsi", proxmoxH.ListISCSI)
	// Tier 1.11: Pools + Cluster resources
	protected.GET("/proxmox/hosts/:id/pools", proxmoxH.ListPools)
	protected.GET("/proxmox/hosts/:id/pools/:poolid", proxmoxH.GetPool)
	protected.POST("/proxmox/hosts/:id/pools", proxmoxH.CreatePool)
	protected.PUT("/proxmox/hosts/:id/pools/:poolid", proxmoxH.UpdatePool)
	protected.DELETE("/proxmox/hosts/:id/pools/:poolid", proxmoxH.DeletePool)
	protected.GET("/proxmox/hosts/:id/cluster/resources", proxmoxH.ListClusterResources)
	protected.GET("/proxmox/hosts/:id/cluster/status", proxmoxH.GetClusterStatus)
	protected.GET("/proxmox/hosts/:id/cluster/info", proxmoxH.GetClusterInfo)
	// Tier 1.12: Backup
	protected.GET("/proxmox/hosts/:id/cluster/backup", proxmoxH.ListBackupJobs)
	protected.GET("/proxmox/hosts/:id/cluster/backup/:jobid", proxmoxH.GetBackupJob)
	protected.POST("/proxmox/hosts/:id/cluster/backup", proxmoxH.CreateBackupJob)
	protected.PUT("/proxmox/hosts/:id/cluster/backup/:jobid", proxmoxH.UpdateBackupJob)
	protected.DELETE("/proxmox/hosts/:id/cluster/backup/:jobid", proxmoxH.DeleteBackupJob)
	protected.POST("/proxmox/hosts/:id/nodes/:node/vzdump", proxmoxH.BackupNow)
	// Tier 1.13: Certificates + ACME (FINAL)
	protected.GET("/proxmox/hosts/:id/nodes/:node/certificates", proxmoxH.ListCertificates)
	protected.GET("/proxmox/hosts/:id/cluster/acme/account", proxmoxH.ListACMEAccounts)
	protected.GET("/proxmox/hosts/:id/cluster/acme/account/:name", proxmoxH.GetACMEAccount)
	protected.POST("/proxmox/hosts/:id/cluster/acme/account", proxmoxH.CreateACMEAccount)
	protected.DELETE("/proxmox/hosts/:id/cluster/acme/account/:name", proxmoxH.DeleteACMEAccount)
	protected.GET("/proxmox/hosts/:id/cluster/acme/plugins", proxmoxH.ListACMEPlugins)
	protected.POST("/proxmox/hosts/:id/cluster/acme/plugins", proxmoxH.CreateACMEPlugin)
	protected.DELETE("/proxmox/hosts/:id/cluster/acme/plugins/:name", proxmoxH.DeleteACMEPlugin)
	protected.GET("/proxmox/hosts/:id/cluster/acme/challenge-schema", proxmoxH.ListACMEChallengeSchema)
	protected.GET("/proxmox/hosts/:id/cluster/acme/directories", proxmoxH.ListACMEDirectories)
	protected.GET("/proxmox/hosts/:id/cluster/acme/info", proxmoxH.GetACMEInfo)

	return r
}
