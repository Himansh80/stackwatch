# StackWatch — Feature Inventory

**Last updated:** 2026-08-17

What's actually live TODAY, versus what's still in the plan.

---

## ✅ Tier 0 — Auth (DONE, verified 10/10)

- Email + password login with bcrypt-12
- Signup (admin-gated)
- Forgot-password / reset-password
- Me / logout / change-password
- Refresh token re-issue
- Per-tenant data scoping
- User CRUD (super_admin only)
- Tenant CRUD
- API key CRUD (sha256-stored, plain shown once)

---

## ✅ Tier 1 — Proxmox (DONE)

- Cluster + node management
- VM lifecycle (create → start → stop → reboot → delete)
- LXC lifecycle (same)
- Storage CRUD + content + download/upload URLs
- Network CRUD (Linux bridge, VLAN, bond)
- Firewall CRUD (rules, groups, IPsets, aliases)
- Disks list + ZFS pool create/destroy
- Users + API tokens CRUD
- Tasks: list / status / log / stop
- ISCSI list + content URLs
- Pools CRUD + cluster resources/status/info
- Backup schedule CRUD + immediate run + vzdump
- Certificates CRUD + ACME accounts/plugins/directories/challenge-schema

**Verified live:** 78 routes registered, 6/6 in cross-check verifier.

---

## ✅ Tier 2 — TrueNAS SCALE 25.10 (DONE)

JSON-RPC over WebSocket sidecar at `:8088`. Endpoints:

- Pools (zfs list)
- Datasets (zfs list)
- NFS shares
- SMB shares
- iSCSI extents / targets / associated targets
- Snapshots
- Replications
- Disks
- Users / Groups
- System info / services / boot environments
- Cloud sync credentials / tasks

**Verified live:** 17 endpoints, 4 back-to-back runs each clean.

---

## ✅ Tier 3 — Remote Access / Termix (DONE)

- SSH connections CRUD + test
- Connection groups
- Per-connection verification + auth config
- SFTP file browser (read / write / mkdir / delete / rename / stat)
- Encrypted credentials vault
- Known-hosts trust list (MITM protection)
- SSH key CRUD (RSA 2048/4096, Ed25519, ECDSA)
- WebSocket terminal (PTY bridge, real SSH, real `xterm.js`-compatible)

**Verified live:** HTTP handlers registered + tested; PTY bridge in
`cmd/web-terminal/`.

---

## ⏳ Tier 4 — Server Admin / Cockpit (PENDING)

- Services (systemd)
- Storage (lsblk, mount, SMART)
- Network (interfaces, routes, DNS, firewalld)
- Processes
- Updates (apt/dnf)
- Logs (journald, /var/log)
- Users (system users, sudo)
- Timers (cron + systemd)
- Performance (I/O wait, OOM events)

**Plan:** 2-3 sessions. NOT STARTED.

---

## ⏳ Tier 5 — Containers / Portainer (PENDING)

- List containers / images / networks / volumes
- Container lifecycle
- Compose deploy / up / down
- Watchtower-style auto-update

## ⏳ Tier 6 — Monitoring Depth (PENDING)

- Per-second metrics
- OpenTelemetry ingestion
- Prom/Grafana/Loki exports
- WebSocket streams

## ⏳ Tier 7 — Datadog parity (PENDING)

- APM + flamegraphs + profiles
- Log Management + monitors + archives
- RUM + sessions + replay
- Synthetics + CI integration
- Security + CSPM + SIEM
- Incident + war room + post-mortem
- DB Monitoring
- Service Catalog + Notebooks
- Team collaboration

## ⏳ Tier 8 — Alerting (PARTIAL)

- 11 notification channel types (email/webhook/slack/telegram/ntfy/discord/teams/pagerduty/opsgenie/sms/dev)
- Sample alert engine rule evaluator (for cpu.user_pct)
- ML anomaly detection (z-score)
- LLM Triage (qwen2.5 Ollama local)

## ⏳ Tier 9 — Security (PENDING)

- Row-Level Security (RLS)
- SAML/OIDC SSO
- IP allowlist
- SOC 2 controls
- Tamper-evident audit log (hash chain)

## ⏳ Tier 10 — Homelab Dashboard (PENDING)

- Hardware inventory page
- Discovery from Nmap / SNMP

## ⏳ Tier 11 — Platform & Commerce (PENDING)

- Stripe checkout
- Razorpay checkout
- Bank transfer annual contracts
- Self-service plan changes

## ⏳ Tier 12 — Docs & GTM (PARTIAL)

- ✅ This document + 13 others
- ⏳ Dedicated marketing landing page HTML
- ⏳ Pricing comparison live

## ⏳ Tier 13 — Mobile (PENDING)

- React Native (or Flutter TBD) mobile app
- FCM / APNS push
- Status pages viewable in-app

---

## Summary

| Tier | Live | Plan |
|------|------|------|
| 0 | ✅ | ✅ |
| 1 | ✅ | ✅ |
| 2 | ✅ | ✅ |
| 3 | ✅ | ✅ |
| 4 | ❌ | ✅ |
| 5 | ❌ | ✅ |
| 6 | ❌ | ✅ |
| 7 | ❌ | ✅ |
| 8 | 🟡 | ✅ |
| 9 | ❌ | ✅ |
| 10 | ❌ | ✅ |
| 11 | ❌ | ✅ |
| 12 | 🟡 | ✅ |
| 13 | ❌ | ✅ |

**Coverage: 4/14 tiers complete, 2 partial, 8 pending.**
