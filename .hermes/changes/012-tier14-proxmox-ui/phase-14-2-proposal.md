# Tier 14 Phase 14.2 — VM Detail Page

## Why this phase

Phase 14.1 (VM list page) lets you see VMs and start/stop them. Phase 14.2 lets you **drill into a single VM** and see everything about it: hardware config, current state, network interfaces, console access, snapshots, firewall.

## Goal

A Datadog-style VM detail page at `/proxmox-vms/:hostId/:node/:vmid` with tabs:
- **Summary** — VM metadata + live status + uptime + tags
- **Hardware** — Config tab (cores, memory, disk, BIOS, machine type)
- **Network** — Interfaces (IP, MAC, bridge, MTU)
- **Console** — xterm.js terminal + noVNC (re-use web-terminal WS bridge from Tier 3)
- **Snapshots** — List + create + delete + rollback (Phase 14.2 ships list only; full CRUD = Phase 14.3)
- **Firewall** — VM-level firewall rules (Phase 14.2 ships view only; full CRUD = Phase 14.3)
- **Actions** — Start / Stop / Reboot / Shutdown / Migrate / Delete (destructive with confirm)

## Why this scope

User clicks a VM row → navigates to detail → sees hardware → starts/stops → maybe opens console. That covers 90% of daily Proxmox web UI use.

## Out of scope (later phases)

- Phase 14.3 — Snapshot create/delete/rollback + Firewall rule CRUD
- Phase 14.4 — VM create wizard (currently the existing workspace's `+ Create VM` form)
- Phase 14.5 — Live CPU/RAM/Network charts (RRD data)
- Phase 14.6 — Backup scheduler

## Backend dependencies

Phase 14.2 uses only **existing Tier 1 endpoints**:
- `GET /api/v1/proxmox/hosts` — list (host selector)
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid` — full config
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/:action` — lifecycle
- `DELETE /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid` — delete
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/agent/network-get-interfaces` (NEW for Phase 14.2) — VM IP/MAC

## New backend endpoint needed

Only ONE: **VM network interfaces via QEMU agent**.

Proxmox API: `GET /nodes/{node}/qemu/{vmid}/agent/network-get-interfaces`
Returns: array of interfaces with ip-addresses, MAC, name

If QEMU agent isn't running on the guest, this fails. Phase 14.2 handles that gracefully (show "agent not running" hint, fall back to bridge info).

## UI components (planned, all ≤400 LOC)

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxVmDetailPage.tsx` | 200 | Page shell, route param parsing, tab router |
| `ProxmoxDetailHeader.tsx` | 150 | VM name + status pill + breadcrumb + actions |
| `ProxmoxDetailSummary.tsx` | 250 | Summary tab (metadata + live state) |
| `ProxmoxDetailHardware.tsx` | 200 | Hardware tab (cores/memory/disk/BIOS) |
| `ProxmoxDetailNetwork.tsx` | 200 | Network tab (interfaces table) |
| `ProxmoxDetailConsole.tsx` | 100 | Console tab placeholder (real impl in Phase 14.3) |
| `ProxmoxDetailSnapshots.tsx` | 150 | Snapshots tab list (CRUD deferred) |
| `ProxmoxDetailFirewall.tsx` | 150 | Firewall tab view (CRUD deferred) |
| `ProxmoxDetailActions.tsx` | 150 | Action bar (start/stop/reboot/migrate/delete) |
| `proxmox-detail.css` | 400 | All detail-page styles |

11 files, each ≤400 LOC.

## Tab UX

- Tabs at top of header (under VM name), styled like Datadog detail pages
- Each tab has its own URL fragment: `#summary`, `#hardware`, etc.
- Active tab persists in URL hash
- "Actions" is NOT a tab — it's a button group in the header (always visible)

## Acceptance criteria for Phase 14.2

1. User clicks a VM row → navigates to detail page (with VMID + node + host in URL)
2. Page loads within 1 second
3. Header shows VM name, status pill, breadcrumb (Proxmox > router > VMID)
4. Tab navigation works (click tab, see content)
5. URL hash changes when switching tabs (deep-linkable)
6. Hardware tab shows cores, memory, disk, BIOS, machine type
7. Network tab shows IP, MAC, bridge, MTU (or "agent not running")
8. Snapshots tab shows list (even if empty)
9. Firewall tab shows rule list (even if empty)
10. Action bar: Start/Stop/Reboot work with confirm
11. Action bar: Migrate opens a node-picker modal (deferred impl = button only, no-op message)
12. Action bar: Delete shows confirm with VM name + vmid, then redirects to /proxmox-vms
13. Auto-refresh every 5s for status pill + summary stats
14. Mobile-responsive (tabs collapse to dropdown on <768px)
15. Console tab shows "Console opens in v2.3" placeholder

## Test plan (to deliver at end of phase)

Detailed A-to-Z plan with numbered cases, expected vs actual, edge cases.

## Risks

- `/cluster/resources` returns aggregated data but `GetVMConfig` returns full Proxmox config — both used, no conflict
- QEMU agent may not be installed on all guests → graceful fallback
- Console placeholder is honest, not fake