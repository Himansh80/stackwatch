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
func buildRouter(ctx context.Context, logger *slog.Logger, pool *db.Pool, issuer *auth.Issuer, installMode, webTerminalURL string) *gin.Engine {
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
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "degraded",
				"db":      "down",
				"mode":    installMode,
				"version": "0.1.0-tier0.5",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"db":      "ok",
			"mode":    installMode,
			"version": "0.1.0-tier0.5",
		})
	})

	// Setup wizard endpoints (Tier 0.5) — public, no auth
	setupH := handler.NewSetupHandler(pool, installMode, issuer)
	r.GET("/api/v1/setup/status", setupH.GetSetupStatus)
	r.POST("/api/v1/setup/initialize", setupH.InitializeSetup)
	r.POST("/api/v1/setup/regenerate-cert", setupH.RegenerateCert)

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
	// /auth/refresh is intentionally UNPROTECTED — handler reads token
	// from request body, not the Authorization header. Putting it under
	// the protected group means RequireAuth aborts before Refresh runs.
	r.POST("/api/v1/auth/refresh", authH.Refresh)

	// Tier 0 tenant management (scoped to caller's JWT tenant).
	protected.GET("/tenants", handler.ListTenants(pool))
	protected.GET("/tenants/me", handler.GetTenantMe(pool))
	protected.GET("/tenants/:id", handler.GetTenant(pool))
	protected.PATCH("/tenants/:id", handler.UpdateTenant(pool))

	// Tier 0 user management.
	protected.GET("/users", handler.ListTier0Users(pool))
	protected.GET("/users/me", handler.GetTier0UserMe(pool))
	protected.GET("/users/:id", handler.GetTier0User(pool))
	protected.POST("/users", handler.CreateTier0User(pool))
	protected.PATCH("/users/:id", handler.UpdateTier0User(pool))
	protected.DELETE("/users/:id", handler.DeleteTier0User(pool))

	// Tier 0 API key management.
	protected.GET("/api-keys", handler.ListAPIKeys(pool))
	protected.POST("/api-keys", handler.CreateAPIKey(pool))
	protected.POST("/api-keys/:id/revoke", handler.RevokeAPIKey(pool))

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

	// Tier 3.1: Terminal (Termius replacement)
	termH := handler.NewTerminalHandler(pool)
	// SSH keys
	protected.GET("/terminal/keys", termH.ListSSHKeys)
	protected.GET("/terminal/keys/:id", termH.GetSSHKey)
	protected.POST("/terminal/keys", termH.CreateSSHKey)
	protected.DELETE("/terminal/keys/:id", termH.DeleteSSHKey)
	// Connections
	protected.GET("/terminal/connections", termH.ListConnections)
	protected.GET("/terminal/connections/groups", termH.ListConnectionGroups)
	protected.GET("/terminal/connections/:id", termH.GetConnection)
	protected.POST("/terminal/connections", termH.CreateConnection)
	protected.PUT("/terminal/connections/:id", termH.UpdateConnection)
	protected.DELETE("/terminal/connections/:id", termH.DeleteConnection)
	protected.POST("/terminal/connections/:id/test", termH.TestConnection)
	// Connection history
	protected.GET("/terminal/history", termH.ListConnectionHistory)
	// Tier 3.4: Verification mode per connection
	protected.GET("/terminal/connections/:id/verification", termH.GetConnectionVerificationMode)
	protected.PUT("/terminal/connections/:id/verification", termH.UpdateConnectionVerificationMode)
	// Tier 3.6: Auth method management
	protected.GET("/terminal/connections/:id/auth", termH.GetConnectionAuth)
	protected.PUT("/terminal/connections/:id/auth", termH.UpdateConnectionAuth)
	// Tier 3.3: SFTP file browser
	protected.GET("/terminal/connections/:id/fs", termH.ListFS)
	protected.GET("/terminal/connections/:id/fs/read", termH.ReadFS)
	protected.POST("/terminal/connections/:id/fs/write", termH.WriteFS)
	protected.POST("/terminal/connections/:id/fs/mkdir", termH.MkdirFS)
	protected.DELETE("/terminal/connections/:id/fs", termH.DeleteFS)
	protected.POST("/terminal/connections/:id/fs/rename", termH.RenameFS)
	protected.GET("/terminal/connections/:id/fs/stat", termH.StatFS)
	// Tier 3.2: Credentials vault + known_hosts
	protected.GET("/terminal/credentials", termH.ListCredentials)
	protected.GET("/terminal/credentials/:id", termH.GetCredential)
	protected.POST("/terminal/credentials", termH.CreateCredential)
	protected.PUT("/terminal/credentials/:id", termH.UpdateCredential)
	protected.DELETE("/terminal/credentials/:id", termH.DeleteCredential)
	protected.POST("/terminal/credentials/:id/reveal", termH.RevealCredential)
	// known_hosts
	protected.GET("/terminal/known-hosts", termH.ListKnownHosts)
	protected.GET("/terminal/known-hosts/:id", termH.GetKnownHost)
	protected.POST("/terminal/known-hosts/trust", termH.TrustHost)
	protected.DELETE("/terminal/known-hosts/:id", termH.DeleteKnownHost)

	// Tier 4: Server Admin (Cockpit parity) — SSH-based via web-terminal sidecar
	adminH := handler.NewAdminHandler(webTerminalURL)
	protected.GET("/admin/services", adminH.ListServices)
	protected.POST("/admin/services/:connection_id/:action", adminH.ServiceAction)
	protected.GET("/admin/storage", adminH.ListStorage)
	protected.GET("/admin/network", adminH.ListNetwork)
	protected.GET("/admin/processes", adminH.ListProcesses)
	protected.GET("/admin/updates", adminH.ListUpdates)
	protected.GET("/admin/logs", adminH.ListLogs)
	protected.GET("/admin/users", adminH.ListUsers)
	protected.GET("/admin/timers", adminH.ListTimers)
	protected.GET("/admin/performance", adminH.ListPerformance)

	// Tier 5: Container Management (Portainer + Watchtower parity)
	// — SSH-based via web-terminal sidecar, runs `docker` CLI commands
	containerH := handler.NewContainerHandler(adminH)
	// C1: List + Get
	protected.GET("/containers", containerH.ListContainers)
	protected.GET("/containers/:id", containerH.GetContainer)
	// C2: Actions
	protected.POST("/containers/:id/:action", containerH.ContainerAction)
	// C3/C4: Logs
	protected.GET("/containers/:id/logs", containerH.ContainerLogs)
	// C5: Stats
	protected.GET("/containers/:id/stats", containerH.ContainerStats)
	// C6: Create
	protected.POST("/containers", containerH.CreateContainer)
	// C7: Images
	protected.GET("/containers/images", containerH.ListImages)
	protected.POST("/containers/images/pull", containerH.PullImage)
	protected.DELETE("/containers/images/:id", containerH.RemoveImage)
	// C8: Volumes
	protected.GET("/containers/volumes", containerH.ListVolumes)
	protected.POST("/containers/volumes", containerH.CreateVolume)
	protected.DELETE("/containers/volumes/:name", containerH.RemoveVolume)
	// C9: Networks
	protected.GET("/containers/networks", containerH.ListNetworks)
	protected.POST("/containers/networks", containerH.CreateNetwork)
	protected.DELETE("/containers/networks/:id", containerH.RemoveNetwork)
	// C10: Stacks
	protected.GET("/containers/stacks", containerH.ListStacks)
	protected.POST("/containers/stacks", containerH.DeployStack)
	// C11: Templates (no auth — public knowledge)
	r.GET("/api/v1/containers/templates", containerH.ListTemplates)
	// Watchtower
	protected.GET("/containers/watchtower", containerH.ListWatched)
	protected.POST("/containers/watchtower/update", containerH.TriggerUpdate)

	// Tier 6: Monitoring Depth — Prometheus compat + Loki push + ML anomaly
	protected.POST("/prom/write", handler.PromWrite(pool))
	protected.POST("/prom/query", handler.PromQuery(pool))
	protected.POST("/loki/push", handler.LokiPush(pool))
	protected.GET("/loki/query", handler.LokiQuery(pool))
	protected.GET("/anomaly/list", handler.ListAnomalies(pool))
	protected.POST("/anomaly/detect", handler.DetectAnomaly(pool))

	// Tier 6 v2: Synthetics (M7) + Grafana datasource adapter (M4)
	protected.POST("/synthetics", handler.CreateSynthetics(pool))
	protected.GET("/synthetics", handler.ListSynthetics(pool))
	protected.GET("/synthetics/:id", handler.GetSynthetics(pool))
	protected.PATCH("/synthetics/:id", handler.PatchSynthetics(pool))
	protected.DELETE("/synthetics/:id", handler.DeleteSynthetics(pool))
	protected.POST("/synthetics/:id/run", handler.RunSyntheticsNow(pool))
	protected.GET("/synthetics/:id/results", handler.GetSyntheticsResults(pool))
	protected.GET("/grafana/search", handler.GrafanaSearch(pool))
	protected.POST("/grafana/query", handler.GrafanaQuery(pool))

	// Tier 6 v3: Dashboard Builder (M10)
	protected.POST("/dashboards", handler.CreateDashboard(pool))
	protected.GET("/dashboards", handler.ListDashboards(pool))
	protected.GET("/dashboards/:id", handler.GetDashboard(pool))
	protected.PATCH("/dashboards/:id", handler.PatchDashboard(pool))
	protected.DELETE("/dashboards/:id", handler.DeleteDashboard(pool))
	protected.POST("/dashboards/:id/eval", handler.EvalDashboard(pool))
	protected.POST("/dashboards/:id/default", handler.SetDashboardDefault(pool))
	protected.DELETE("/dashboards/:id/default", handler.UnsetDashboardDefault(pool))

	// Tier 6 v4: Real User Monitoring (M8)
	r.POST("/api/v1/rum/events", handler.IngestRUM(pool))   // unauthenticated — tenant_id query param
	protected.GET("/rum/summary", handler.RUMSummary(pool))
	protected.GET("/rum/events", handler.ListRUMEvents(pool))

	return r
}
