# StackWatch — vs The Market

**Last updated:** 2026-08-17

Comparison is honest: where we're better, where we're worse, where we
haven't built the feature yet.

---

## What we WIN on today

### Cost

| Tool | Per-server price (USD/mo) | StackWatch equivalent |
|------|---------------------------|------------------------|
| Datadog Pro | $15-23 | **$8** (cloud) or **$0** (self-host) |
| New Relic Pro | $25+ | **$8** |
| Grafana Cloud Pro | $14 + per-active-series | **$8** flat |
| SigNoz Cloud | $0 (5M spans) → up to $199 | **$8** |
| Better Stack | $25 | **$8** |
| Honeycomb (Pro) | $0 (no SLA) → ~$130 | **$8** |
| Chronosphere | $40+ | **$8** |

For an SMB IT team running 50 servers, StackWatch self-hosted is
**free** after hardware; StackWatch Pro is **$400/mo** vs $900-1250/mo
on the alternatives.

### Self-hosting

- **Datadog, New Relic, Grafana Cloud, Honeycomb, Better Stack** all require SaaS
- **StackWatch, SigNoz, HyperDX** can be self-hosted
- Of the self-hostable, StackWatch has the lightest footprint (3 binaries + Postgres)

### Data sovereignty

Self-hosted StackWatch data never leaves your network. Compliance regimes
(HIPAA, GDPR §46, ITAR, FedRAMP-Moderate) often *require* this. The
closest competitor with this property is SigNoz, which has a much heavier
architecture (Java/Scala + Clickhouse).

### Multi-protocol, single binary

Today (Tier 0-3), a single StackWatch install manages:
- **Proxmox** (78 endpoints, full admin)
- **TrueNAS SCALE** (17 endpoints, full admin)
- **Terminal / SFTP** (33 endpoints, Termius-class)

To match this with separate vendors you'd need: Proxmox UI
(self-hosted, web-only), TrueNAS UI (self-hosted), Termius
($8-15/mo per user), **and** a monitoring tool (Datadog etc.). All
separate stacks, no common auth, no cross-system automation.

---

## What we LOSE on today

### Tool depth

| Capability | Datadog | New Relic | StackWatch (today) |
|-----------|---------|-----------|---------------------|
| APM + flame graphs | ✅ | ✅ | ❌ (Tier 7) |
| RUM + Session Replay | ✅ | ✅ | ❌ (Tier 7) |
| Synthetics | ✅ | ✅ | ❌ (Tier 7) |
| Log analytics | ✅ | ✅ | 🟡 (Tier 6 ingests; archive/monitor UI in Tier 7) |
| Container management | ❌ | ❌ | ❌ (Tier 5) |
| Server admin (Cockpit parity) | ❌ | ❌ | ❌ (Tier 4) |
| Mobile app | ✅ | ✅ | ❌ (Tier 13) |
| Push-to-mobile | ✅ | ✅ | ❌ (Tier 13) |
| Pre-built integrations | 700+ | 600+ | ~30 (Proxmox + TrueNAS + SSH + Postgres + ...) |

The honest truth: StackWatch is **a homelab tool masquerading as a
Datadog**. For a 1-50 server homelab or small business, it's already
the better tool — better UI than Proxmox's, better admin than TrueNAS's,
real monitoring in one binary.

For a 200+ server enterprise, **until Tier 6-7 ship, StackWatch is
inadequate**. We don't pretend otherwise.

### Enterprise features missing

- ❌ SAML / OIDC SSO (Tier 9)
- ❌ Audit log hash chain (Tier 9)
- ❌ SOC 2 controls (Tier 9)
- ❌ Row-Level Security in Postgres (Tier 9)
- ❌ SCIM user provisioning (Tier 9)
- ❌ Multi-region replication (Tier 11)

### Faster by mature competitor metrics

- Datadog: < 1s query latency at 50PB scale
- StackWatch: < 100ms at 50GB scale today

We don't have the scale yet. We don't claim to.

---

## What we WIN on over the homelab-only tools (Proxmox, TrueNAS)

These tools have **web UIs** for their specific hardware. They don't
monitor, alert, audit, multi-tenant, or expose a REST API. StackWatch
treats them as data sources, not as the dashboard.

| Capability | Proxmox UI | TrueNAS UI | StackWatch |
|-----------|-----------|-----------|------------|
| Full VM/LXC management | ✅ | n/a | ✅ |
| Disk + ZFS admin | ✅ | n/a | ✅ |
| Storage management | ✅ | ✅ | ✅ |
| Pool/Dataset admin | n/a | ✅ | ✅ |
| NFS/SMB/iSCSI shares | n/a | ✅ | ✅ |
| Snapshots + replication | ✅ | ✅ | ✅ |
| Multi-tenant | ❌ | ❌ | ✅ |
| REST API | ✅ (Proxmox 7+) | ❌ (deprecated) | ✅ |
| Auth + JWT | ❌ | ❌ | ✅ |
| Cross-system UX | ❌ | ❌ | ✅ |
| Browser terminal | ✅ (xterm.js in noVNC) | ✅ | ✅ (WS over real SSH, not noVNC) |
| Cross-server file browser | ❌ | ❌ | ✅ (SFTP over SSH) |
| Audit logs | ✅ (PVE-audit) | ✅ | ✅ |
| RBAC | ✅ | ✅ | ✅ |
| Theme-able | ❌ (default only) | ❌ | ✅ (planned Tier 11) |

---

## When to use StackWatch

✅ Good fit:
- 1-50 server homelab or SMB IT
- Want to monitor Proxmox / TrueNAS without paying Datadog
- Want Termius-class terminal but self-hosted
- Cost-sensitive; can self-host a 2-vCPU VM
- AGPL-3.0 compatible with your org

❌ Not yet (until Tier 4-13 ships):
- 200+ servers
- Need APM/flame graphs (Datadog-style)
- Need mobile push
- Need SSO into enterprise SSO provider

---

## Pricing model summary

See [PRICING.md](PRICING.md).

- **Self-host**: free forever, hardware + Postgres
- **Cloud SaaS**: $8/svr/month, all-in
- **Resell**: $500+/year, custom

That's our edge: **you only pay for servers, not seats**. Datadog
charges per-host AND per-user. We charge per-host only.

---

## TL;DR

StackWatch = **Proxmox + TrueNAS + Termius + (in-progress) Datadog**
in one self-hostable binary, at $0 to $8/server/month.

When fully built (Tier 13), it will be a true Datadog replacement for
small-to-mid scale. Today, it's a homelab replacement.
