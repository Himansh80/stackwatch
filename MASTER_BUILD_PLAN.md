# StackWatch — Master Build Plan (Complete)

**Date:** 2026-08-16  
**Status:** Tier 0 ✅ COMPLETE — live verified 82/82 PASS (2026-08-18); Tier 1 is next
**Project:** `~/HermesProjects/stackwatch`

---

## Tier 0 — FOUNDATION (Session 1)

**Steps:** 7  
**Files created:** ~25

### 0.1 Repo init
- `git init` in `stackwatch/`
- `git remote add origin git@github.com:himanshukhandelwal/stackwatch.git`
- `.gitignore` (no `.bak`, `node_modules/`, `dist/`, `bin/`)
- `LICENSE` (AGPL-3)
- `README.md` (project overview)

### 0.2 Go project base
- `go.mod` (module: `github.com/stackwatch/platform`)
- `cmd/api-gateway/main.go` (minimal HTTP server with /health)
- `internal/kernel/kernel.go` (shared types: Tenant, User, Pagination, Error)
- `internal/db/db.go` (pgx pool wrapper)
- `internal/middleware/recover.go` (panic recovery)
- `internal/middleware/requestid.go` (request ID)
- `internal/middleware/cors.go`
- `internal/middleware/logging.go` (structured slog)

### 0.3 Database schema
- `migrations/000_init_sql.sql` (tenants, users, audit_log)
- `migrations/001_users.sql` (id, email, password_hash, role, tenant_id, created_at)
- `migrations/002_tenants.sql` (id, name, plan, status)

### 0.4 Auth (minimal)
- `internal/auth/jwt.go` (JWT issue + verify)
- `internal/auth/hash.go` (bcrypt)
- `internal/handler/auth.go` (POST /auth/login, POST /auth/signup, GET /auth/me)
- Tests: `internal/auth/jwt_test.go`

### 0.5 Frontend base
- `web/package.json` (Vite + React + TypeScript)
- `web/src/main.tsx` (entry)
- `web/src/App.tsx` (router)
- `web/src/pages/Login.tsx` (login form)
- `web/src/pages/Dashboard.tsx` (placeholder)
- `web/src/lib/api.ts` (axios client)
- `web/vite.config.ts`

### 0.6 CI
- `.github/workflows/backend-ci.yml` (go build, test, lint, file-size check)
- `.github/workflows/frontend-ci.yml` (npm install, build, lint)
- `.golangci.yml` (lint config)
- `.file-size-check.sh` (block PRs with files > 500 lines)

### 0.7 Documentation
- `docs/INDEX.md`
- `docs/INSTALL.md`
- `docs/USER-GUIDE.md` (skeleton)
- `docs/ARCHITECTURE.md` (skeleton)
- `docs/TESTING.md`
- `docs/COMPARISON.md`

**Verify after Tier 0:**
1. `git push origin main` works
2. CI green
3. `go build ./...` passes
4. `npm run build` in `web/` passes
5. `curl -X POST localhost:8080/auth/login -d '{"email":"test@test.com","password":"test"}'` returns 200
6. Frontend loads at http://localhost:5173

---

## Tier 1 — PROXMOX FULL REPLACEMENT (3-4 sessions)

**Goal:** Replace Proxmox Web UI. Add Proxmox host → manage all VMs from StackWatch.

### 1.1 Proxmox connector (cmd/proxmox-connector)
- `cmd/proxmox-connector/main.go` (HTTP service :8087)
- `internal/client/proxmox/proxmox.go` (REST API client)
- `internal/service/proxmox_service.go` (business logic)
- `internal/handler/proxmox_host.go` (CRUD hosts)
- Routes: `/api/v1/proxmox/hosts`, `/hosts/:id/test`

### 1.2 VM lifecycle (P1)
- `internal/handler/proxmox_vm.go` (list/get/create/update/delete)
- `internal/handler/proxmox_vm_action.go` (start/stop/reboot/shutdown/suspend/resume)
- Endpoints: GET/POST/DELETE `/api/v1/proxmox/:host/vms`, PATCH `/vms/:vmid`, POST `/vms/:vmid/action`

### 1.3 LXC lifecycle (P2)
- `internal/handler/proxmox_lxc.go` (list/get/create/delete/start/stop)
- Endpoints: same pattern as VM

### 1.4 Storage (P3)
- `internal/handler/proxmox_storage.go` (ZFS/LVM/Ceph/directory)
- Endpoints: GET/POST `/api/v1/proxmox/:host/storage`, `/storage/:id/zfs`, `/storage/:id/dataset`

### 1.5 Networking (P4)
- `internal/handler/proxmox_network.go` (Bridges/VLANs/Bonds/Firewall)
- Endpoints: GET/POST `/api/v1/proxmox/:host/network/{bridges,vlans,bonds,firewall}`

### 1.6 Backup (P5)
- `internal/handler/proxmox_backup.go` (create/restore/schedule)
- Endpoints: POST `/api/v1/proxmox/:host/vms/:vmid/backup`, `/restore`

### 1.7 Templates (P6)
- `internal/handler/proxmox_templates.go` (template library, cloud-init)
- Endpoints: GET `/api/v1/proxmox/templates`, POST `/templates/from-vm/:vmid`

### 1.8 HA & Cluster (P7)
- `internal/handler/proxmox_cluster.go` (cluster join/leave, HA, quorum, migration)
- Endpoints: GET/POST `/api/v1/proxmox/cluster/{join,leave,status,ha,migrate}`

### 1.9 Monitoring (P8)
- `internal/handler/proxmox_monitoring.go` (per-VM CPU/mem/disk/net, host utilization, alerts)
- Endpoints: GET `/api/v1/proxmox/:host/vms/:vmid/stats`, `/host/:id/stats`

### Frontend (Tier 1)
- `web/src/pages/ProxmoxHosts.tsx` (list Proxmox hosts)
- `web/src/pages/ProxmoxHostDetail.tsx` (host overview)
- `web/src/pages/ProxmoxVMs.tsx` (VM list + actions)
- `web/src/pages/ProxmoxVMDetail.tsx` (VM console + stats)
- `web/src/pages/ProxmoxLXC.tsx`
- `web/src/pages/ProxmoxStorage.tsx`
- `web/src/pages/ProxmoxNetwork.tsx`
- `web/src/pages/ProxmoxBackup.tsx`
- `web/src/components/VMTable.tsx` (reusable)
- `web/src/components/VMActions.tsx` (start/stop/etc)

---

## Tier 2 — TRUENAS SCALE FULL REPLACEMENT (3-4 sessions)

**Goal:** Replace TrueNAS SCALE UI. Add TrueNAS host → manage all storage from StackWatch.

### 2.1 TrueNAS connector (cmd/truenas-connector)
- `cmd/truenas-connector/main.go` (HTTP service :8088)
- `internal/client/truenas/truenas.go` (REST API v2 client)
- `internal/service/truenas_service.go`
- `internal/handler/truenas_host.go` (CRUD hosts)

### 2.2 ZFS Pool (T1)
- `internal/handler/truenas_pool.go` (create/mirror/RAID-Z, add vdev, import, destroy, status)
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/pools`

### 2.3 Dataset (T2)
- `internal/handler/truenas_dataset.go` (create/resize/delete, recordsize/compression/quota)
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/datasets`

### 2.4 NFS (T3)
- `internal/handler/truenas_nfs.go`
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/nfs`

### 2.5 SMB (T4)
- `internal/handler/truenas_smb.go`
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/smb`

### 2.6 iSCSI (T5)
- `internal/handler/truenas_iscsi.go` (extents, targets, associated targets)
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/iscsi/{extents,targets,associated-targets}`

### 2.7 Snapshots (T6)
- `internal/handler/truenas_snapshot.go`
- `internal/handler/truenas_replication.go`
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/snapshots`, `/replications`

### 2.8 Disk Health (T7)
- `internal/handler/truenas_disk.go` (SMART, status, replacement)
- Endpoints: GET `/api/v1/truenas/:host/disks`, `/disks/:name/smart`, `/disks/:name/scrub`

### 2.9 Users (T8)
- `internal/handler/truenas_user.go` (users, groups, ACL)
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/users`, `/groups`, `/acls`

### 2.10 System (T9)
- `internal/handler/truenas_system.go` (boot envs, update, services)
- Endpoints: GET `/api/v1/truenas/:host/system/{boot,update,services}`

### 2.11 Cloud Sync (T10)
- `internal/handler/truenas_cloud_sync.go` (S3/GCS/Azure/Backblaze)
- Endpoints: GET/POST/DELETE `/api/v1/truenas/:host/cloud-sync`

### Frontend (Tier 2)
- `web/src/pages/TrueNASHosts.tsx`
- `web/src/pages/TrueNASHostDetail.tsx`
- `web/src/pages/TrueNASPools.tsx`
- `web/src/pages/TrueNASDatasets.tsx`
- `web/src/pages/TrueNASNFS.tsx`
- `web/src/pages/TrueNASSMB.tsx`
- `web/src/pages/TrueNASiSCSI.tsx`
- `web/src/pages/TrueNASSnapshots.tsx`
- `web/src/pages/TrueNASDisks.tsx`
- `web/src/pages/TrueNASUsers.tsx`
- `web/src/pages/TrueNASSystem.tsx`
- `web/src/pages/TrueNASCloudSync.tsx`

---

## Tier 3 — REMOTE ACCESS (Chrome RD + Termix)  (1-2 sessions)

**Goal:** Open any machine from StackWatch → full desktop in browser.

### 3.1 Remote access service (cmd/remote-access)
- `cmd/remote-access/main.go` (HTTP service :8089)
- `internal/client/guacamole/guacamole.go` (Apache Guacamole client)
- `internal/service/remote_access_service.go`
- `internal/handler/remote_connection.go` (CRUD connections)
- `internal/handler/remote_session.go` (open/close sessions)

### 3.2 Connection CRUD (R2)
- `internal/handler/remote_connection_rdp.go`
- `internal/handler/remote_connection_vnc.go`
- `internal/handler/remote_connection_ssh.go`
- `internal/handler/remote_connection_telnet.go`
- Endpoints: GET/POST/DELETE `/api/v1/remote/connections/{rdp,vnc,ssh,telnet}`

### 3.3 Remote desktop viewer (R3)
- Frontend: `web/src/pages/RemoteDesktop.tsx` (canvas-based viewer)
- WebSocket bridge: `/api/v1/ws/remote/:connection_id`
- Multi-monitor, clipboard sync, fullscreen

### 3.4 Session features (R4)
- Recording (.cast), screenshot, reconnect, observer mode
- `internal/handler/remote_session_record.go`

### 3.5 Credential vault (R5)
- `internal/handler/remote_credentials.go` (encrypted storage)
- Per-user access, rotation, audit log

### 3.6 SSH tunneling (R6)
- `internal/handler/remote_tunnel.go` (local/remote port forwarding)
- Health monitoring, auto-reconnect

### 3.7 SSH multi-tab (R7)
- `web/src/components/SSHTabs.tsx` (split up to 4 panels)

---

## Tier 4 — SERVER ADMIN (Cockpit parity)  (2-3 sessions)

**Goal:** When user opens any server, see full admin panel.

### 4.1 Agent extensions (cmd/agent)
- `cmd/agent/services.go` (systemd via D-Bus)
- `cmd/agent/storage.go` (lsblk, mount, SMART)
- `cmd/agent/network.go` (interfaces, routes, DNS, firewalld)
- `cmd/agent/processes.go` (process list)
- `cmd/agent/updates.go` (apt/dnf available updates)
- `cmd/agent/logs.go` (journald, /var/log)
- `cmd/agent/users.go` (system users, SSH keys, sudo)
- `cmd/agent/timers.go` (cron + systemd timers)
- `cmd/agent/performance.go` (per-second I/O wait, OOM events)

### 4.2 Handler layer
- `internal/handler/admin_services.go` (S1)
- `internal/handler/admin_storage.go` (S2)
- `internal/handler/admin_network.go` (S3)
- `internal/handler/admin_processes.go` (S4)
- `internal/handler/admin_updates.go` (S5)
- `internal/handler/admin_logs.go` (S6)
- `internal/handler/admin_users.go` (S7)
- `internal/handler/admin_timers.go` (S8)
- `internal/handler/admin_performance.go` (S9)

### 4.3 Frontend
- `web/src/pages/ServerAdmin.tsx` (parent page)
- `web/src/pages/admin/Services.tsx` (S1)
- `web/src/pages/admin/Storage.tsx` (S2)
- `web/src/pages/admin/Network.tsx` (S3)
- `web/src/pages/admin/Processes.tsx` (S4)
- `web/src/pages/admin/Updates.tsx` (S5)
- `web/src/pages/admin/Logs.tsx` (S6)
- `web/src/pages/admin/Users.tsx` (S7)
- `web/src/pages/admin/Timers.tsx` (S8)
- `web/src/pages/admin/Performance.tsx` (S9)

---

## Tier 5 — CONTAINER MANAGEMENT (Portainer + Watchtower)  (2 sessions)

**Goal:** Open container host → manage Docker from StackWatch.

### 5.1 Container service (cmd/container-mgmt)
- `cmd/container-mgmt/main.go` (HTTP service :8090)
- `internal/client/docker/docker.go` (Docker socket client)
- `internal/service/container_service.go`
- `internal/handler/container.go` (C1-C4)
- `internal/handler/container_action.go` (C2)
- `internal/handler/container_console.go` (C3)
- `internal/handler/container_logs.go` (C4)
- `internal/handler/container_stats.go` (C5)
- `internal/handler/container_create.go` (C6)
- `internal/handler/container_image.go` (C7)
- `internal/handler/container_volume.go` (C8)
- `internal/handler/container_network.go` (C9)
- `internal/handler/stack.go` (C10)
- `internal/handler/template.go` (C11)
- `internal/handler/watchtower.go` (auto-update)

### 5.2 Frontend
- `web/src/pages/Containers.tsx` (list)
- `web/src/pages/ContainerDetail.tsx` (console + logs + stats)
- `web/src/pages/ContainerCreate.tsx`
- `web/src/pages/Images.tsx`
- `web/src/pages/Volumes.tsx`
- `web/src/pages/Networks.tsx`
- `web/src/pages/Stacks.tsx`
- `web/src/pages/Templates.tsx`

---

## Tier 6 — MONITORING DEPTH (Netdata + Prom + Grafana + Loki)  (3-4 sessions)

**Goal:** Per-second metrics + anomalies + Prometheus compat + log aggregation.

### 6.1 Ingest depth (already exists, extend)
- `cmd/agent/collectors/per_second.go` (M1)
- WS stream endpoint already exists

### 6.2 ML anomaly (M2)
- `pkg/ml/anomaly.go` (z-score, k-means)
- `cmd/analytics/anomaly.go` (service)
- `internal/handler/anomaly.go` (CRUD endpoints)

### 6.3 Prometheus compat (M3)
- `cmd/analytics/prometheus.go` (remote_write endpoint, PromQL API)
- `internal/handler/prom_query.go`
- `internal/handler/prom_write.go`

### 6.4 Grafana integration (M4)
- `internal/handler/grafana_datasource.go` (StackWatch as Grafana source)
- `internal/handler/grafana_annotation.go`

### 6.5 Tracing (M5)
- `internal/handler/trace.go` (OTLP receiver)
- Storage schema + flame graph UI

### 6.6 Log management (M6)
- `cmd/analytics/loki.go` (Loki-compatible API)
- `internal/handler/log_pipeline.go` (grok/regex)
- `internal/handler/log_pattern.go`
- `internal/handler/log_archive.go` (S3 cold storage)

### 6.7 Synthetics (M7)
- `cmd/synthetics/main.go` (HTTP/TCP/ping checks)
- `internal/handler/synthetics.go`

### 6.8 RUM (M8)
- `web/src/rum.ts` (JS snippet for customer sites)
- `internal/handler/rum.go` (ingest + query)

### 6.9 Network monitoring (M9)
- `internal/handler/network.go` (per-NIC, NetFlow/sFlow)

### 6.10 Dashboard builder (M10)
- `web/src/pages/DashboardBuilder.tsx` (drag-and-drop)
- `web/src/components/widgets/TimeSeries.tsx`
- `web/src/components/widgets/Heatmap.tsx`
- `web/src/components/widgets/Geomap.tsx`
- `web/src/components/widgets/TopList.tsx`
- `web/src/components/widgets/KPI.tsx`

---

## Tier 7 — DATADOG FULL PLATFORM (4-5 sessions)

**Goal:** Parity with Datadog APM + RUM + Synthetics + Security + Notebooks.

### 7.1 APM (D2)
- Distributed tracing (parent-child)
- Service map
- Deployment tracking
- Code-level profiling

### 7.2 Log Management full (D3)
- Infinite log explorer
- Live tail
- Archiving + rehydration
- Pipelines, monitors

### 7.3 RUM full (D4)
- Session replay
- Performance tracking
- Error tracking
- Resource timing
- Heatmaps

### 7.4 Synthetics full (D5)
- API tests (HTTP/gRPC)
- Browser tests (Selenium-like)
- Private locations
- CI/CD integration

### 7.5 Security (D6)
- Threat detection
- Audit trails
- Compliance rules
- SIEM integration

### 7.6 CSPM (D7)
- AWS/GCP/Azure security posture
- Misconfiguration scanning

### 7.7 CI/CD Visibility (D8)
- Pipeline tracking
- Test tracking
- Deployment correlation

### 7.8 DB Monitoring (D9)
- Query performance
- Slow query log
- Explain plans
- Connection pooling

### 7.9 Service Management (D10)
- Incident declaration
- War room
- Timeline
- Post-mortem
- Task assignment

### 7.10 Notebook (D11)
- Collaborative markdown docs
- Embedded graphs/timeline

### 7.11 Team/Collab (D12)
- Dashboards shared with view/edit perms
- Annotation mentions
- Timeline comments

---

## Tier 8 — INTELLIGENCE & ALERTING (2-3 sessions)

**Goal:** Datadog + PagerDuty parity.

### 8.1 Composite alerts (I1)
### 8.2 Anomaly alerting (I2)
### 8.3 Forecasting (I3)
### 8.4 Incident management (I4)
### 8.5 On-call & escalation (I5)
### 8.6 Alert correlation (I6)
### 8.7 SLO/SLI tracking (I7)
### 8.8 Maintenance windows (I8)
### 8.9 Scheduled reports (I9)

---

## Tier 9 — SECURITY & ENTERPRISE (2 sessions)

### 9.1 RLS to production (SE1)
### 9.2 SAML/SSO (SE2)
### 9.3 RBAC granular (SE3)
### 9.4 Audit log UI (SE4)
### 9.5 API key scopes (SE5)
### 9.6 2FA enforcement (SE6)
### 9.7 Secrets management (SE7)
### 9.8 SOC2 (SE8)
### 9.9 IP allowlist (SE9)
### 9.10 Network performance (SE10)

---

## Tier 10 — HOMELAB DASHBOARD (Homarr)  (1 session)

### 10.1 Widgets (H1)
### 10.2 Service status (H2)
### 10.3 Notes/Todo (H3)
### 10.4 Calendar (H4)
### 10.5 Download stats (H5)
### 10.6 Media server (H6)
### 10.7 Search (H7)
### 10.8 Per-user (H8)
### 10.9 RSS (H9)
### 10.10 Task scheduler (H10)

---

## Tier 11 — PLATFORM & COMMERCE (1-2 sessions)

### 11.1 Push-button deploy (PL1)
### 11.2 Usage metering (PL2)
### 11.3 Self-service signup (PL3)
### 11.4 Tenant limits (PL4)
### 11.5 Backup/restore (PL5)
### 11.6 Multi-region/HA (PL6)
### 11.7 Rate limiting (PL7)
### 11.8 Platform health (PL8)

---

## Tier 12 — DOCS & GTM (1 session)

### 12.1 COMPARISON.md (vs all 12 tools)
### 12.2 INSTALL/USER-GUIDE/ARCHITECTURE/TESTING
### 12.3 Pricing/packaging
### 12.4 Marketing site

---

## Tier 13 — MOBILE (1 session)

### 13.1 React Native app (MO1)
### 13.2 Push notifications (MO2)

---

## Effort Summary

| Tier | Effort | Priority |
|------|--------|----------|
| 0 | 1 session | HIGHEST (blocks all) |
| 1 | 3-4 sessions | Highest ROI |
| 3 | 1-2 sessions | Chrome RD replacement |
| 4 | 2-3 sessions | Cockpit replacement |
| 5 | 2 sessions | Portainer replacement |
| 2 | 3-4 sessions | TrueNAS replacement |
| 6 | 3-4 sessions | Datadog beginning |
| 7 | 4-5 sessions | Datadog parity |
| 8-13 | 12-15 sessions | Complete the platform |

**Total: 26-35 sessions** (each session = 1-4 hours of focused work)

---

## Next Step (USER ACTION)

**Say "start tier 0"** and I'll build the foundation: git repo, base Go project, base React project, DB schema, auth, single dummy endpoint, CI, docs. Estimated 1 focused session. I'll verify with live tests, then ask you to test, then move to Tier 1.
