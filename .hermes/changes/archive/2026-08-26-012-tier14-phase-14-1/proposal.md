# Tier 14 — Proxmox Full Web UI

## Why this tier

StackWatch already has the **Proxmox API** (Tier 1, 28 endpoints via `internal/client/proxmox/`). What's missing is the **web UI** — the daily-use screens that would replace the Proxmox web UI itself.

Right now, the operator opens Proxmox's own web UI to:
- See the list of VMs
- Start/stop a VM
- View VM details
- Read the console

This tier builds those screens inside StackWatch.

## Scope (Phase 14 only)

- **Phase 14.1: VM list page** (this speckit change) — list, filter, start/stop
- **Phase 14.2: VM detail page** — tabs: Summary, Config, Network, Console, Firewall, Snapshots, Backup
- **Phase 14.3: VM create page** — form with wizard, hardware spec, OS ISO selection
- **Phase 14.4: LXC list + detail + create** (mirror of 14.1-14.3 for CT)
- **Phase 14.5: Node dashboard** — per-node overview (CPU/mem/disk + subsys + DNS + time + syslog + updates)
- **Phase 14.6: Storage content browser** — ISO/template/backup uploads
- **Phase 14.7: Firewall** — rules + aliases + IPSets
- **Phase 14.8: SDN** — zones + VNets + subnets
- **Phase 14.9: Users + API tokens + permissions + pools**
- **Phase 14.10: Tasks + Backup + Replication + Cloud-Init + HA + Certificates**

## Why speckit

Per user direction (2026-08-25): strict speckit workflow (proposal → spec → plan → tasks → checklist → build → archive) for every tier/feature. Modular discipline enforced.

## Why modular

Every Go file < 400 LOC. Every TSX file < 400 LOC. Split by domain. Single responsibility per file.

## Why Datadog style

Dark theme, KPI cards with colored top stripes, status pills, severity badges, smooth animations (framer-motion), professional polish.

## Why user-test workflow

User said: "build one thing → test it like a human, tester, customer → you test → fix → next feature". Every feature is a complete vertical slice (backend endpoint + frontend UI + live verify) before moving on.

## Tech stack

- Frontend: React (existing web/src/pages/) + framer-motion + ui/ux-pro-max principles + 21st.dev components
- Backend: existing Tier 1 (no new endpoints for Phase 14.1; uses GET /proxmox/hosts/:id/qemu)
- DB: existing (no new tables for Phase 14.1)
- No new dependencies (framer-motion may be added in Phase 14.2)

## Out of scope (later phases)

- LXC (Phase 14.4)
- Node dashboard (Phase 14.5)
- Create VM wizard (Phase 14.3)
- Anything else listed above

## Success criteria for Phase 14.1

- User can see VM list with name, status, IP, CPU%, RAM%, uptime
- User can start/stop/reboot/shutdown a VM from the UI
- User can filter by status (running/stopped/paused)
- User can search by name/vmid/IP
- UI is dark-themed, animated, professional (Datadog-style)
- Live data fetched from existing `/api/v1/proxmox/hosts/:id/qemu`
- Status changes reflected within 2 seconds (manual refresh + auto-poll)
- Empty state when no hosts registered
- Error state when host unreachable
