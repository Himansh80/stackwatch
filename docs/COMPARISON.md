# StackWatch vs Market — Tier 0 Position

**Date:** 2026-08-16

| Tool | Replaces | Tier | Status |
|------|----------|------|--------|
| Datadog | All monitoring + APM + Logs + RUM + Synthetics + Security | 7 | Tier 0 only |
| Proxmox UI | VM/LXC/Storage/Network/Backup | 1 | Next |
| TrueNAS UI | ZFS/Datasets/NFS/SMB/iSCSI | 2 | Planned |
| Portainer | Container management | 5 | Planned |
| Watchtower | Container auto-update | 5 | Planned |
| Termix | Web SSH/RDP/VNC terminal | 3 | Planned |
| Cockpit | Linux server admin | 4 | Planned |
| Netdata | Per-second metrics + ML anomaly | 6 | Planned |
| Prometheus | TSDB + PromQL | 6 | Will embed |
| Grafana | Dashboards + data sources | 6 | Will embed |
| Loki | Log aggregation + LogQL | 6 | Will embed |
| Chrome Remote Desktop | RDP into Windows/Mac/Linux | 3 | Planned |
| Homarr | Homelab dashboard | 10 | Planned |

## Tier 0 = foundation only

We shipped:
- Multi-tenant database schema (tenants, users, api_keys, audit_log)
- JWT auth (login, signup, me, change-password, logout, forgot-password)
- Structured logging with request IDs
- Panic recovery (one panic doesn't kill the server)
- CORS + security headers
- File-size check (CI blocks files > 500 lines)
- Frontend with React + TypeScript + Vite
- GitHub Actions CI for backend + frontend

What you can do today:
- Sign up → log in → change password → log out (via API)
- See structured logs for every request
- Database schema ready for all future tiers

What you can't do yet:
- Add a server, see metrics, view VMs, anything else (next tiers)

## vs competitors (when complete)

| Feature | StackWatch | Datadog | Grafana Cloud | Proxmox UI | Portainer |
|---------|-----------|---------|---------------|------------|-----------|
| Cost | Free self-hosted | $$$ per host | $$ per user | Free | Free CE |
| Single pane | ✅ All tools | ✅ Monitoring only | ✅ Monitoring only | ✅ VMs only | ✅ Containers only |
| Open source | ✅ AGPL-3 | ❌ Proprietary | ❌ Core OSS, Cloud prop | ✅ AGPL-3 | ✅ Zlib |
| Self-hostable | ✅ | ❌ | ✅ OSS only | ✅ | ✅ |
| Multi-tenant | ✅ (Tier 0+) | ✅ | ✅ Orgs | ❌ | ❌ |
| API-first | ✅ | ✅ | ✅ | ✅ | ✅ |
| Mobile app | Tier 13 | ✅ | ❌ | ❌ | ❌ |
| Web terminal | Tier 3 | ❌ | � | ✅ | ❌ |
| File browser | Tier 3 | ❌ | ❌ | ❌ | ❌ |
| Smart device control | ❌ Out of scope | ❌ | ❌ | ❌ | ❌ |
| Modular code | ✅ 500-line rule | ✅ | �️ | ✅ | ⚠️ |

## What we WIN on

- **One interface, one login, every tool** — competitors make you context-switch
- **Strict modular code** — add a feature without breaking anything else
- **Open source, self-hostable** — no vendor lock-in
- **India-market pricing** — Razorpay + INR + ₹99/mo subscription
- **Web terminal + file browser + smart devices** — unique combo no competitor has
- **Built on giants' patterns** — Datadog/Stripe/GitHub file sizes, modular monolith, structured logs

## What we LOSE on (for now)

- **No AI/automation tier yet** — Datadog Bits AI / PagerDuty are far ahead
- **No SOC 2** — Tier 9
- **No mobile app** — Tier 13
- **No production cloud offering** — Tier 11+

## When Tier 1-5 ship (3-4 months)

You'll be able to use StackWatch as your ONLY web UI for:
- All VMs (replace Proxmox UI)
- All storage (replace TrueNAS UI)
- All containers (replace Portainer + Watchtower)
- All terminals (replace Termix + Chrome RD)
- All server admin (replace Cockpit)

That's when the product becomes genuinely useful as a daily driver.

## Pricing model (planned, Tier 11+)

- **Self-hosted personal:** Free
- **Self-hosted business:** $500/year (commercial use)
- **Cloud free:** 1-3 servers
- **Cloud paid:** $8/server/month
- **Cloud enterprise:** Custom (SAML, SOC 2, dedicated support)
