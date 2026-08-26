# Tier 14 Phase 14.4 — LXC Container Support (list + detail + create wizard)

## Why this phase

Tier 1 already has LXC backend endpoints (Create, GetConfig, UpdateConfig, Delete, Lifecycle, etc.) and the `/cluster/resources` endpoint returns both VMs and LXC. The current `/proxmox-vms` page filters by `type === 'qemu'`. LXC containers are invisible.

Phase 14.4 adds a full LXC UI surface: list page, detail page (mirror of VM detail), create wizard.

## Goals

1. **LXC list page** at `/proxmox-lxc` — Datadog-style grid of containers with start/stop/reboot
2. **LXC detail page** at `/proxmox-lxc/:hostId/:node/:vmid` — same tab structure as VM detail
3. **Create LXC wizard** — multi-step form (template → resources → network → confirm)

## Scope (this phase)

- Reuse the same component pattern as `/proxmox-vms`
- Backend already returns LXC in `/cluster/resources` (just filtered out by frontend)
- Create LXC: use existing `POST /proxmox/hosts/:id/nodes/:node/lxc` (Tier 1)

## Out of scope (later phases)

- Phase 14.5 — VM create wizard (mirror of LXC create)
- Phase 14.6 — VM/LXC migrate between nodes (already partially done in 14.3)
- Phase 14.7 — Node dashboard (CPU/RAM per node)
- Phase 14.8 — Storage content browser (ISOs / templates / backups)
- Phase 14.9 — SDN + Firewall + IPSets
- Phase 14.10 — Task panel + Replication + HA + Certificates + ACME

## Backend dependencies

ZERO new backend endpoints. Tier 1 already provides:
- `GET /api/v1/proxmox/hosts/:id/qemu` → returns VMs+LXC (filter client-side)
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/lxc` → create LXC
- `GET .../nodes/:node/lxc/:vmid` → config
- `DELETE .../nodes/:node/lxc/:vmid` → delete
- `POST .../nodes/:node/lxc/:vmid/status/:action` → lifecycle

## UI components (planned, all ≤400 LOC)

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxLxcListPage.tsx` | 200 | List of LXC containers (filter type === 'lxc') |
| `ProxmoxLxcListPage.css` | 0 (reuse proxmox.css) | — |
| `ProxmoxLxcDetailPage.tsx` | 280 | Mirror of VM detail but for LXC |
| `ProxmoxLxcCreatePage.tsx` | 350 | Multi-step wizard |
| `ProxmoxLxcCreateStep1Template.tsx` | 150 | Choose template |
| `ProxmoxLxcCreateStep2Resources.tsx` | 200 | Cores + RAM + disk |
| `ProxmoxLxcCreateStep3Network.tsx` | 150 | Bridge + IP + VLAN |
| `ProxmoxLxcCreateStep4Confirm.tsx` | 150 | Review + submit |
| `ProxmoxLxcTemplatePicker.tsx` | 200 | Reusable component |

## Routes

- `/proxmox-lxc` — list page
- `/proxmox-lxc/:hostId/:node/:vmid` — detail page (mirror of VM detail)
- `/proxmox-lxc/new` — create wizard

## Acceptance criteria

1. `/proxmox-lxc` lists all LXC across selected host (filter by type === 'lxc')
2. KPI strip + filter chips + search work like VM list
3. Click LXC → detail page (same tabs as VM detail)
4. Detail page shows: Summary, Resources, Network, Snapshots, Firewall, Console (CT console is xterm.js, not noVNC — deferred)
5. Create wizard: pick template → set resources → network → review → submit
6. After create: redirect to detail page with new LXC
7. Mobile-responsive (cards on <768px)
8. Console tab for LXC: placeholder (real serial console is Phase 14.7+)

## Modularity discipline

- All TSX < 400 LOC
- Reuse ProxmoxKpiStrip, ProxmoxFilterBar, ProxmoxEmptyState, ProxmoxVmActions (or wrap them generically)
- Wizard = 4 small step components (each ≤200 LOC)