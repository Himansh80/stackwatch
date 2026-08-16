# StackWatch — Research & Feature Inventory

**Date:** 2026-08-16  
**Method:** Web search per tool, capture every feature, group by tier

---

## 1. Datadog (Tier 7)

Source: Datadog.com, G2, Capterra, DASH 2026 announcements

### Infrastructure Monitoring
- Real-time server metrics (CPU, memory, disk, network, load)
- Process-level monitoring (top processes by CPU/memory)
- Disk I/O monitoring (read/write ops, throughput)
- Network monitoring (bandwidth, latency, packets, errors)
- GPU monitoring (NVIDIA, AMD)
- Temperature sensors
- Container monitoring (Docker, Kubernetes)
- VM monitoring (VMware, Hyper-V)
- Cloud integration (AWS, Azure, GCP metadata)
- Auto-discovery of new services/servers
- Server inventory & asset management

### Dashboards & Visualization
- Custom drag-and-drop dashboards
- Pre-built dashboard templates
- Time-series graphs (line, area, bar, heatmap, scatter)
- Gauge widgets (single value with thresholds)
- Top lists (top CPU consumers, top memory users)
- Host maps (geographic visualization)
- Service maps (dependency graph)
- Note/text widgets
- iframe widgets
- Dashboard sharing (public links, embed)
- Dashboard versioning & history
- Fullscreen kiosk mode
- Dark/light theme

### Alerting & Incident Management
- Threshold alerts (static: CPU > 80%)
- Anomaly detection (ML-based)
- Composite alerts (AND/OR logic)
- Multi-alert (alert if N of M hosts trigger)
- Alert scheduling (maintenance windows, business hours)
- Alert grouping (by cluster, service, region)
- Alert lifecycle (triggered → acknowledged → resolved)
- Incident timeline & collaboration
- Runbook automation (auto-remediation)
- SLO/SLA monitoring & error budgets
- Forecasting alerts (predict disk full in 7 days)

### Notification Channels (15+)
- Email (SMTP)
- Slack
- Discord
- Telegram
- PagerDuty / OpsGenie
- Microsoft Teams
- Webhook (custom HTTP)
- SMS (Twilio)
- Pushover / Pushbullet
- ntfy / Gotify
- Google Chat
- VictorOps

### Log Management
- Log collection & ingestion
- Live log tailing
- Log search (full-text, regex, filters)
- Log parsing & processing pipelines
- Log patterns (automatic clustering)
- Log archives (S3, cold storage)
- Log metrics (generate metrics from logs)
- Log-based alerts
- Correlation (logs ↔ metrics ↔ traces)

### APM & Distributed Tracing
- Service performance monitoring
- Distributed trace collection
- Service dependency map
- Endpoint-level breakdown
- Database query analysis
- External service tracking
- Code hotspots (slowest functions)

### Real User Monitoring (RUM)
- Page load performance (FCP, LCP, FID, CLS)
- Session replay
- Error tracking
- User journey tracking
- Performance by country/device/browser
- Core Web Vitals

### Synthetic Monitoring
- HTTP(S) endpoint checks
- TCP/UDP port checks
- DNS resolution checks
- SSL certificate expiry monitoring
- Browser tests (Selenium-like)
- API tests (multi-step workflows)
- Ping monitoring (ICMP)
- Global probe locations

### Security & Compliance
- Threat detection
- Audit logging
- Compliance reports (SOC 2, ISO 27001)
- Vulnerability scanning
- File integrity monitoring
- Anomaly detection for security

### AI & Automation (DASH 2026)
- Bits AI Dev Agent (autonomous debugging)
- Bits AI Ops Agent (auto-remediate)
- AI-powered anomaly detection
- Forecasting
- Noise reduction
- Auto-triage
- ChatOps integration
- Natural language query

### Status Pages
- Public status pages (custom domain)
- Component-level status
- Incident communication
- Historical uptime
- Subscriber notifications
- Custom branding

### API & Integrations
- Full REST API (~600 endpoints)
- Webhooks
- API keys management
- Rate limiting
- Terraform provider
- CLI tool
- Data export (CSV, JSON, PDF)
- SDKs (Python, Go, JS, Ruby, Java, .NET)

### Multi-tenancy & Team
- RBAC
- Team/organization management
- Audit log
- SSO (SAML, OAuth)
- API key scoping
- Usage quotas & billing

---

## 2. Portainer (Tier 5)

Source: portainer.io, GitHub, Capterra

### Container Management
- Container list (status, image, ports, CPU/mem, uptime)
- Container lifecycle (start/stop/restart/remove/pause/resume/kill)
- Container console (exec)
- Container logs (real-time stream)
- Container stats (CPU/mem/net/blkio)
- Container create (image, env, volumes, ports, network, limits)
- Image management (pull/remove, list)
- Volume management (create/remove/prune)
- Network management (create/remove/connect/disconnect)
- Compose / Stacks (deploy from UI)
- Templates (1-click deploy)
- Multi-orchestrator (Docker, Swarm, K8s, ACI)
- Podman support
- RBAC (admin/standard/read-only)
- Environment-scoped permissions
- Audit logging
- External authentication
- Edge fleet management

### Editions
- **CE (Community):** Free, basic features
- **BE (Business):** Multi-team RBAC, external auth, edge fleet, audit logs, fleet governance policies

---

## 3. Watchtower (Tier 5)

Source: github.com/containrrr/watchtower

### Features
- Auto-update running containers
- Pull new image from Docker Hub / private registry
- Gracefully shutdown + restart with same options
- Schedule updates (cron)
- Scope by label / name
- Monitor-only mode (notify, don't update)
- Cleanup old images
- Notifications (email, Slack, Shoutrrr, Gotify, Ntfy, Discord, Telegram, Microsoft Teams)
- HTTP API for manual triggering
- Pre/post-update hooks
- Rolling restarts
- Group containers

---

## 4. Termix (Tier 3)

Source: github.com/Termix-SSH/Termix

### Features
- Web-based SSH terminal
- RDP (Remote Desktop)
- VNC
- Telnet
- Tab-based navigation (multiple sessions)
- Split-screen (up to 4 panels)
- SSH tunneling (local/remote port forwarding)
- Remote file management
- Docker management (in v2)
- Native apps (Windows, macOS, Linux)
- Mobile apps (iOS, Android)
- Session sharing (multi-user)
- Customization (themes, fonts, sizes)
- Credential vault
- Host inventory
- Connection history

---

## 5. Netdata (Tier 6)

Source: netdata.cloud, Wikipedia, GitHub

### Features
- Real-time per-second monitoring
- Zero-configuration (auto-detects metrics)
- Auto-generated dashboards
- Edge ML anomaly detection (k-means clustering)
- 800+ integrations
- Logs collection + analysis
- Alerts (Email, Slack, PagerDuty, etc.)
- AI root cause reporting
- Distributed architecture (parent + child)
- Cloud or self-hosted
- Metric streaming (Kafka, Prometheus remote write)
- High-resolution (1s) data
- Real-time charts
- Custom dashboards
- Free, open-source

---

## 6. Prometheus (Tier 6)

Source: prometheus.io, DeepWiki

### Features
- Time-series database (TSDB)
- Pull-based metrics collection
- PromQL query language
- Multi-dimensional time series
- Service discovery (DNS, Consul, K8s, file-based)
- Alerting rules
- Recording rules
- Alertmanager (deduplication, grouping, routing)
- Remote write (long-term storage)
- Remote read
- Federation (multi-tier)
- Exporters (Node, MySQL, PostgreSQL, etc.)
- Built-in expression browser
- Web API
- Storage: local disk / S3 / GCS
- Compression
- Backfill tool
- Self-monitoring

---

## 7. Grafana (Tier 6)

Source: grafana.com, GitHub

### Features
- 100+ data sources (Prometheus, Loki, Elasticsearch, etc.)
- Multi-source queries (Mix data sources in single panel)
- Visualization types: graph, stat, gauge, table, geomap, heatmap, bar, pie, bar gauge, candlestick, flowchart, canvas, logs
- Transformations (math, regex, join, filter)
- Variables/template queries
- Alerting (visual rules, contact points)
- Annotation API
- Dashboard sharing (public link, embed)
- Dashboard versioning
- Kiosk mode
- Fullscreen mode
- Dark/light theme
- Plugins (data sources, panels, apps)
- Provisioning (yaml config)
- User roles (admin, editor, viewer)
- SSO (SAML, OAuth, LDAP)
- Audit logs
- Reporting (PDF scheduled)
- Enterprise: data sources limit, reporting, custom branding

---

## 8. Cockpit (Tier 4)

Source: cockpit-project.org

### Features
- Web-based server admin UI
- Service management (systemd)
- Storage (lsblk, mount, SMART, ZFS, LVM)
- Network (interfaces, routes, DNS, firewalld)
- Process management (top, kill)
- Software updates (apt, dnf)
- Log viewer (journald, /var/log)
- User management (users, SSH keys, sudo)
- Scheduled tasks (cron, systemd timers)
- Performance (1s metrics, OOM)
- SELinux management
- Kernel crash dump (kdump)
- Diagnostics (sos report)
- Multi-server dashboard
- Web terminal (cockpit-shell)
- Applications (add-ons): 389 Directory Server, NFS, Samba, KRB5, OSTree, machines, PCP, metrics, dashboard

---

## 9. Grafana Loki (Tier 6)

Source: grafana.com/loki, DeepWiki

### Features
- Log aggregation system
- LogQL query language (similar to PromQL)
- Multi-tenancy
- Cost-effective (no full-text index)
- Index by labels only
- Storage: S3, GCS, Azure Blob, local
- Modes: monolithic, simple scalable, microservices
- Promtail + Alloy for log shipping
- Pipeline stages (parse, regex, JSON, template)
- Alerting rules
- Recording rules
- Retention policies
- Sample/filter by service
- Grafana integration
- Explore UI
- Log-based metrics
- Kubernetes native

---

## 10. Chrome Remote Desktop (Tier 3)

Source: digi-tools.info, Capterra

### Features
- Cross-platform (Windows, macOS, Linux, Android, iOS)
- File transfer
- Multi-monitor support
- Clipboard sync
- Session recording
- Unattended access
- Access control
- Session timeout
- Customizable settings
- High resolution support
- Audio support (limited)
- Mobile access
- Security protocols (TLS, PIN)
- Connection management
- Keyboard shortcut handling
- 3D graphics support

---

## 11. Proxmox VE (Tier 1)

Source: proxmox.com, PVE admin guide

### Features
- KVM virtualization
- LXC containers
- Web UI (single pane of glass)
- Cluster management
- High availability (HA)
- Live migration
- SDN (Software Defined Networking)
- Firewall (per-VM, datacenter)
- Storage: ZFS, Ceph, iSCSI, NFS, LVM, local
- Backup (full/incremental, schedule, restore)
- Snapshots
- Templates (cloud-init, linked clones)
- Live migration
- Roles & permissions
- API tokens
- Multi-factor auth
- Two-factor (TOTP)
- Software-defined storage (Ceph)
- Software-defined networking (OVS, Linux bonds)
- VM console (VNC, noVNC)
- Container console
- Real-time monitoring
- Historical reporting
- Syslog viewer
- Cluster Resource Scheduler (CRS)
- Update management (APT repositories)
- License (AGPL, free for commercial)

---

## 12. TrueNAS Scale (Tier 2)

Source: truenas.com, TrueNAS Guide

### Features
- ZFS filesystem (checksum, self-healing)
- RAID-Z1/2/3
- Pools (mirror, RAIDZ, add vdev, import, destroy)
- Datasets (recordsize, compression, quota)
- Zvols (for iSCSI)
- NFS shares
- SMB shares (ACL, recycle bin, shadow copies)
- iSCSI (extents, targets, associated targets)
- Snapshots (manual, scheduled)
- Replication (push/pull, local, remote)
- Cloud sync (S3, GCS, Azure, Backblaze, Google Drive)
- SMART monitoring
- Disk replacement
- SCRUB / resilver
- User management (local, AD, LDAP)
- Group management
- Dataset ACL (POSIX, NFSv4)
- API keys
- Boot environments
- Apps (Docker-based, Docker Compose)
- VMs (KVM)
- System dataset
- Update management
- Services (NFS, SMB, iSCSI, SSH, FTP)
- Reporting (CPU, mem, disk, network)
- Alerts (email, Slack, etc.)
- Web UI
- API (REST v2)
- WebSocket events
- TrueCommand (multi-system management)

---

## 13. Homarr (Tier 10)

Source: homarr.dev

### Features
- Drag-and-drop dashboard
- 40+ app integrations (Sonarr, Radarr, Plex, etc.)
- Customizable widgets
- Service status (UP/DOWN/DEGRADED)
- Custom notes / todo
- Calendar / agenda
- Download client stats (qBittorrent, Transmission)
- Media server integration (Plex, Jellyfin, Emby)
- Search bar (global)
- Per-user dashboards
- RSS feed widget
- Task scheduler
- Self-hosted
- Light/dark theme
- Multi-language
- Mobile-friendly
- PWA (installable)

---

## Coverage Summary

| Area | Total features across all tools | StackWatch Tier |
|------|--------------------------------|-----------------|
| **Infrastructure monitoring** | 30+ | 1, 6 |
| **Container management** | 25+ | 5 |
| **Logs** | 15+ | 6 |
| **APM / Tracing** | 10+ | 7 |
| **RUM** | 10+ | 7 |
| **Synthetic monitoring** | 8+ | 7 |
| **Alerting** | 15+ | 8 |
| **Notification channels** | 15+ | 8 |
| **Proxmox** | 30+ | 1 |
| **TrueNAS** | 25+ | 2 |
| **Remote access** | 10+ | 3 |
| **Server admin** | 15+ | 4 |
| **Dashboards** | 20+ | 6, 7 |
| **AI/ML** | 10+ | 7, 8 |
| **Security** | 15+ | 9 |
| **Mobile** | 10+ | 13 |
| **Status pages** | 8+ | 7 |
| **Multi-tenancy / RBAC** | 10+ | 9, 11 |
| **Homelab dashboard** | 10+ | 10 |

**Total features to ship: ~300+ across all tiers**

---

## Strategy

**Tier 0** locks the foundation (modular structure, CI, auth, base UI).  
**Tier 1-5** are the immediate "replace the tools you use today" set.  
**Tier 6-7** is the Datadog-level monitoring.  
**Tier 8-13** are the polish + commerce + mobile.

Each tier is built **one module at a time**, **fully tested**, then **user tested**, then **next module**.
