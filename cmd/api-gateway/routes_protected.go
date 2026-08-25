package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountProtectedRoutes registers every endpoint that lives behind the
// JWT-protected "/api/v1" group. Splitting this out of routes.go keeps
// the latter under the 400-LOC cap while letting each Tier add routes
// without ballooning either file.
//
// Add new protected routes here in tier order (Tier 0 → Tier 7+). The
// CI/CD webhook receivers are intentionally NOT in this file — they
// live on the public router in routes.go so GitHub / GitLab can POST
// to them without an API token.
//
// Handler ownership:
//   - Tier 0 tenants/users/api-keys         → top-level helper funcs
//   - Tier 1 Proxmox                         → handler.NewProxmoxHandler
//   - Tier 2 TrueNAS                         → truenasProxy()
//   - Tier 3 Terminal / SSH                  → handler.NewTerminalHandler
//   - Tier 4 Server Admin (Cockpit)          → handler.NewAdminHandler
//   - Tier 5 Containers (Portainer)          → handler.NewContainerHandler
//   - Tier 6 Monitoring depth                → top-level helper funcs
//   - Tier 6 v2-v5 synthetics/RUM/tracing    → top-level helper funcs
//   - Tier 7 APM / logs / RUM / synthetics   → top-level helper funcs
//   - Tier 7 security / cspm                 → top-level helper funcs
//   - Tier 7 CI/CD                           → top-level helper funcs (Phase 1)
func mountProtectedRoutes(protected *gin.RouterGroup, pool *db.Pool, webTerminalURL string, synthRunner *handler.SyntheticsRunner) {
	// ---- Tier 0: auth + tenant/user/api-key management ----
	// (authH protected handlers are wired from routes.go where the
	// handler instance lives; protected.GET("/auth/me") etc. are
	// registered by setupAuthProtectedRoutes below in routes.go.)
	protected.GET("/tenants", handler.ListTenants(pool))
	protected.POST("/tenants", handler.CreateTenant(pool))
	protected.GET("/tenants/me", handler.GetTenantMe(pool))
	protected.GET("/tenants/:id", handler.GetTenant(pool))
	protected.PATCH("/tenants/:id", handler.UpdateTenant(pool))

	protected.GET("/users", handler.ListTier0Users(pool))
	protected.GET("/users/me", handler.GetTier0UserMe(pool))
	protected.GET("/users/:id", handler.GetTier0User(pool))
	protected.POST("/users", handler.CreateTier0User(pool))
	protected.PATCH("/users/:id", handler.UpdateTier0User(pool))
	protected.DELETE("/users/:id", handler.DeleteTier0User(pool))

	protected.GET("/api-keys", handler.ListAPIKeys(pool))
	protected.POST("/api-keys", handler.CreateAPIKey(pool))
	protected.POST("/api-keys/:id/revoke", handler.RevokeAPIKey(pool))
	protected.DELETE("/api-keys/:id", handler.DeleteAPIKey(pool))

	// ---- Tier 1: Proxmox VE lifecycle + config + cluster ----
	proxmoxH := handler.NewProxmoxHandler(pool)
	protected.POST("/proxmox/hosts", proxmoxH.CreateHost)
	protected.GET("/proxmox/hosts", proxmoxH.ListHosts)
	protected.GET("/proxmox/hosts/:id", proxmoxH.GetHost)
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
	// Tier 1 completion: templates and cloud-init
	protected.GET("/proxmox/hosts/:id/nodes/:node/templates", proxmoxH.ListTemplates)
	protected.GET("/proxmox/hosts/:id/templates", proxmoxH.ListTemplates)
	protected.POST("/proxmox/hosts/:id/nodes/:node/:kind/:vmid/template", proxmoxH.MarkTemplate)
	protected.POST("/proxmox/hosts/:id/templates/from-vm/:node/:kind/:vmid", proxmoxH.MarkTemplate)
	protected.POST("/proxmox/hosts/:id/nodes/:node/:kind/:vmid/cloud-init", proxmoxH.ApplyCloudInit)
	protected.POST("/proxmox/hosts/:id/templates/:node/:kind/:vmid/cloud-init", proxmoxH.ApplyCloudInit)
	// Tier 1 completion: HA, cluster membership and migration
	protected.GET("/proxmox/hosts/:id/cluster/ha/status", proxmoxH.GetHAStatus)
	protected.GET("/proxmox/hosts/:id/cluster/ha/resources", proxmoxH.ListHAResources)
	protected.POST("/proxmox/hosts/:id/cluster/join", proxmoxH.JoinCluster)
	protected.DELETE("/proxmox/hosts/:id/cluster/leave", proxmoxH.LeaveCluster)
	protected.POST("/proxmox/hosts/:id/nodes/:node/:kind/:vmid/migrate", proxmoxH.MigrateResource)
	// Tier 1 completion: Proxmox RRD monitoring
	protected.GET("/proxmox/hosts/:id/nodes/:node/monitoring", proxmoxH.HostMonitoring)
	protected.GET("/proxmox/hosts/:id/nodes/:node/:kind/:vmid/monitoring", proxmoxH.ResourceMonitoring)
	protected.GET("/proxmox/hosts/:id/host/:node/stats", proxmoxH.HostMonitoring)
	protected.GET("/proxmox/hosts/:id/nodes/:node/:kind/:vmid/stats", proxmoxH.ResourceMonitoring)
	protected.GET("/proxmox/hosts/:id/host/:node/alerts", proxmoxH.MonitoringAlerts)

	// ---- Tier 2: TrueNAS SCALE proxy ----
	protected.Any("/truenas/*path", truenasProxy())

	// ---- Tier 3: Terminal / SSH / SFTP ----
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

	// ---- Tier 4: Server Admin (Cockpit parity) ----
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

	// ---- Tier 5: Containers (Portainer + Watchtower) ----
	containerH := handler.NewContainerHandler(adminH)
	protected.GET("/containers", containerH.ListContainers)
	protected.GET("/containers/:id", containerH.GetContainer)
	protected.POST("/containers/:id/:action", containerH.ContainerAction)
	protected.GET("/containers/:id/logs", containerH.ContainerLogs)
	protected.GET("/containers/:id/stats", containerH.ContainerStats)
	protected.POST("/containers", containerH.CreateContainer)
	protected.GET("/containers/images", containerH.ListImages)
	protected.POST("/containers/images/pull", containerH.PullImage)
	protected.DELETE("/containers/images/:id", containerH.RemoveImage)
	protected.GET("/containers/volumes", containerH.ListVolumes)
	protected.POST("/containers/volumes", containerH.CreateVolume)
	protected.DELETE("/containers/volumes/:name", containerH.RemoveVolume)
	protected.GET("/containers/networks", containerH.ListNetworks)
	protected.POST("/containers/networks", containerH.CreateNetwork)
	protected.DELETE("/containers/networks/:id", containerH.RemoveNetwork)
	protected.GET("/containers/stacks", containerH.ListStacks)
	protected.POST("/containers/stacks", containerH.DeployStack)
	// Watchtower
	protected.GET("/containers/watchtower", containerH.ListWatched)
	protected.POST("/containers/watchtower/update", containerH.TriggerUpdate)

	// ---- Tier 6: Monitoring depth (Prom / Loki / ML) ----
	protected.POST("/prom/write", handler.PromWrite(pool))
	protected.POST("/prom/query", handler.PromQuery(pool))
	protected.POST("/loki/push", handler.LokiPush(pool))
	protected.GET("/loki/query", handler.LokiQuery(pool))
	// /anomaly/list — Tier 6 streaming list (read from the legacy
	// anomaly_events table). Stays here for backward compat with
	// existing Tier 6 dashboards.
	protected.GET("/anomaly/list", handler.ListAnomalies(pool))
	// /anomaly/detect — Tier 8.1 model-backed version is registered
	// in routes_intelligence.go (mountIntelligenceRoutes). The
	// Tier 6 stateless detect endpoint has been superseded; the
	// ml.IsAnomalyValue helper still works for ad-hoc scripts that
	// import the package directly.

	// ---- Tier 6 v2: Synthetics (M7) + Grafana adapter (M4) ----
	protected.POST("/synthetics", handler.CreateSynthetics(pool))
	protected.GET("/synthetics", handler.ListSynthetics(pool))
	protected.GET("/synthetics/:id", handler.GetSynthetics(pool))
	protected.PATCH("/synthetics/:id", handler.PatchSynthetics(pool))
	protected.DELETE("/synthetics/:id", handler.DeleteSynthetics(pool))
	protected.POST("/synthetics/:id/run", handler.RunSyntheticsNow(pool))
	protected.GET("/synthetics/:id/results", handler.GetSyntheticsResults(pool))
	protected.GET("/grafana/search", handler.GrafanaSearch(pool))
	protected.POST("/grafana/query", handler.GrafanaQuery(pool))

	// ---- Tier 6 v3: Dashboards (M10) ----
	protected.POST("/dashboards", handler.CreateDashboard(pool))
	protected.GET("/dashboards", handler.ListDashboards(pool))
	protected.GET("/dashboards/:id", handler.GetDashboard(pool))
	protected.PATCH("/dashboards/:id", handler.PatchDashboard(pool))
	protected.DELETE("/dashboards/:id", handler.DeleteDashboard(pool))
	protected.POST("/dashboards/:id/eval", handler.EvalDashboard(pool))
	protected.POST("/dashboards/:id/default", handler.SetDashboardDefault(pool))
	protected.DELETE("/dashboards/:id/default", handler.UnsetDashboardDefault(pool))

	// ---- Tier 6 v4: RUM ingest (ingest is on public router in routes.go) ----
	protected.GET("/rum/summary", handler.RUMSummary(pool))
	protected.GET("/rum/events", handler.ListRUMEvents(pool))

	// ---- Tier 6 v5: Distributed Tracing (M5) ----
	protected.POST("/traces", handler.IngestSpan(pool))
	protected.POST("/traces/otlp", handler.IngestOTLP(pool))
	protected.GET("/traces", handler.ListTraces(pool))
	protected.GET("/traces/services/summary", handler.TraceServiceSummary(pool))
	protected.GET("/traces/:trace_id", handler.GetTraceByID(pool))

	// ---- Tier 7.1: APM (D2) ----
	protected.POST("/apm/services", handler.RegisterAPMService(pool))
	protected.GET("/apm/services", handler.ListAPMServices(pool))
	protected.GET("/apm/services/:id", handler.GetAPMService(pool))
	protected.GET("/apm/services/:id/flame-graph", handler.APMFlameGraph(pool))
	protected.POST("/apm/traces", handler.IngestAPMTrace(pool))
	protected.GET("/apm/traces/:trace_id", handler.GetAPMTrace(pool))
	protected.POST("/apm/deployments", handler.RecordAPMDeployment(pool))
	protected.GET("/apm/deployments", handler.ListAPMDeployments(pool))
	protected.GET("/apm/service-map", handler.APMServiceMap(pool))

	// ---- Tier 7.2: Logs Full (D3) ----
	protected.POST("/logs/monitors", handler.CreateLogMonitor(pool))
	protected.GET("/logs/monitors", handler.ListLogMonitors(pool))
	protected.PUT("/logs/monitors/:id", handler.UpdateLogMonitor(pool))
	protected.DELETE("/logs/monitors/:id", handler.DeleteLogMonitor(pool))
	protected.POST("/logs/archives", handler.CreateLogArchive(pool))
	protected.GET("/logs/archives", handler.ListLogArchives(pool))
	protected.POST("/logs/archives/:id/rehydrate", handler.TriggerLogRehydration(pool))
	protected.GET("/logs/rehydrations", handler.ListLogRehydrations(pool))
	protected.POST("/logs/retention", handler.SetLogRetention(pool))
	protected.GET("/logs/retention", handler.ListLogRetention(pool))
	protected.GET("/logs/patterns", handler.ListLogPatterns(pool))

	// ---- Tier 7.3: RUM Full (D4) ----
	protected.POST("/rum/web-vitals", handler.IngestRUMWebVital(pool))
	protected.POST("/rum/resource-timings", handler.IngestRUMResource(pool))
	protected.POST("/rum/interactions", handler.IngestRUMInteraction(pool))
	protected.POST("/rum/long-tasks", handler.IngestRUMLongTask(pool))
	protected.POST("/rum/errors", handler.IngestRUMError(pool))
	protected.POST("/rum/heatmap", handler.IngestRUMHeatmap(pool))
	protected.GET("/rum/sessions", handler.ListRUMSessions(pool))
	protected.GET("/rum/sessions/:id", handler.GetRUMSession(pool))
	protected.GET("/rum/sessions/:id/waterfall", handler.GetRUMSessionWaterfall(pool))
	protected.GET("/rum/error-groups", handler.ListRUMErrorGroups(pool))

	// ---- Tier 7.4: Synthetics Full (D5) ----
	protected.POST("/synthetics/locations", handler.CreateSynthLocation(pool))
	protected.GET("/synthetics/locations", handler.ListSynthLocations(pool))
	protected.POST("/synthetics/ci-configs", handler.CreateSynthCIConfig(pool))
	protected.GET("/synthetics/ci-configs", handler.ListSynthCIConfigs(pool))
	protected.POST("/synthetics/webhook", handler.IngestSynthWebhook(pool))
	protected.POST("/synthetics/tests-full", handler.CreateSynthTest(pool, synthRunner))
	protected.GET("/synthetics/tests-full", handler.ListSynthTests(pool))
	protected.GET("/synthetics/tests-full/:test_id", handler.GetSynthTest(pool))
	protected.PUT("/synthetics/tests-full/:test_id", handler.UpdateSynthTest(pool))
	protected.DELETE("/synthetics/tests-full/:test_id", handler.DeleteSynthTest(pool))
	protected.POST("/synthetics/tests-full/:test_id/run", handler.RunSynthTestNow(pool, synthRunner))
	protected.GET("/synthetics/tests-full/:test_id/sla", handler.GetSynthTestSLA(pool))
	protected.GET("/synthetics/tests-full/:test_id/results", handler.GetSynthTestResults(pool))

	// ---- Tier 7.5: Security (D6) ----
	protected.GET("/security/threats", handler.ListSecurityThreats(pool))
	protected.GET("/security/audit-trails", handler.ListSecurityAuditTrails(pool))
	protected.GET("/security/compliance", handler.ListSecurityCompliance(pool))
	protected.GET("/security/siem", handler.ListSecuritySIEM(pool))

	// ---- Tier 7.6: CSPM (D7) ----
	protected.GET("/cspm/resources", handler.ListCSPMResources(pool))
	protected.GET("/cspm/findings", handler.ListCSPMFindings(pool))

	// ---- Tier 7.7: CI/CD Visibility (D8) — Phase 1 ----
	protected.GET("/cicd/pipelines", handler.ListCICDPipelines(pool))
	protected.POST("/cicd/pipelines", handler.CreateCICDPipeline(pool))
	protected.GET("/cicd/pipelines/:id", handler.GetCICDPipeline(pool))
	protected.GET("/cicd/deployments", handler.ListCICDDeployments(pool))
	protected.POST("/cicd/deployments", handler.CreateCICDDeployment(pool))

	// ---- Tier 7.8: DB Monitoring (D9) — Phase 2 ----
	protected.GET("/database/slow-queries", handler.ListDBMonSlowQueries(pool))
	protected.GET("/database/queries/top", handler.ListDBMonTopQueries(pool))
	protected.GET("/database/query-explain", handler.ExplainDBMonQuery(pool))
	protected.GET("/database/connection-pool", handler.ListDBMonConnectionPools(pool))
	protected.POST("/database/connection-pool", handler.IngestDBMonConnectionPool(pool))
	protected.POST("/database/query-stats", handler.IngestDBMonQueryStats(pool))

	// ---- Tier 7.9: Service Management (D10) — Phase 3 ----
	// 10 routes extracted to routes_incidents.go (mountIncidentRoutes)
	// so routes_protected.go stays under the 400-LOC cap.
	mountIncidentRoutes(protected, pool)

	// ---- Tier 7.10: Notebook (D11) — Phase 4 ----
	// 6 routes extracted to routes_notebook.go (mountNotebookRoutes)
	// so routes_protected.go stays under the 400-LOC cap.
	mountNotebookRoutes(protected, pool)

	// ---- Tier 7.11: Team/Collab (D12) — Phase 5 (FINAL) ----
	// 5 routes extracted to routes_team.go (mountTeamRoutes) — this
	// is the final route group for Tier 7. After this phase ships
	// every Tier 7 subtier has its routes mounted.
	mountTeamRoutes(protected, pool)

	// Tier 8 routes live in routes_intelligence.go.
	mountIntelligenceRoutes(protected, pool)

	// Tier 9: 32 routes live in routes_enterprise.go.
	mountEnterpriseRoutes(protected, pool)

	// Tier 10: Homelab routes in routes_homelab.go.
	mountHomelabRoutes(protected, pool)
}