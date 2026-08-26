# Tier 14 Phase 14.7 — Node Dashboard + Storage Content Browser

## Why this phase

The Proxmox web UI has a "Node" view showing per-node CPU/mem/disk/load/uptime plus a "Storage content" browser for ISOs/templates/backups. Phase 14.7 adds both, scoped to the selected Proxmox host.

## Goals

1. **Node dashboard** at `/proxmox/nodes/:hostId/:node` — KPIs, resource list, services, network
2. **Storage content browser** at `/proxmox/storage/:hostId/:node/:storage` — file list with upload/delete
3. Wire sidebar links

## Backend

ZERO new endpoints — all use Tier 1:
- `GET /api/v1/proxmox/hosts/:id/nodes` — list nodes
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/network` — network interfaces
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/disks/list` — disk list
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/disks/zfs` — ZFS pools
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/services` — service list
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/storage/:storage/content` — storage content
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/storage/:storage/content` — upload (URL form, not file upload)
- `DELETE .../storage/:storage/content` — delete file

## UI components (planned, all ≤400 LOC)

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxNodeDashboard.tsx` | 280 | KPI strip + sections (disks/zfs/network/services) |
| `ProxmoxNodeKpiStrip.tsx` | 130 | Reusable node KPI strip |
| `ProxmoxStorageContentPage.tsx` | 250 | Storage content list + upload + delete |
| `proxmox-node.css` | 200 | Node dashboard styles |

## Routes

- `/proxmox/nodes/:hostId/:node` — node dashboard
- `/proxmox/storage/:hostId/:node/:storage` — storage content

## Acceptance criteria

1. `/proxmox/nodes/:hostId/:node` shows KPI strip + 4 cards (Disks, ZFS, Network, Services)
2. `/proxmox/storage/:hostId/:node/:storage` shows file list with type icons
3. Empty states for both
4. Errors handled (e.g. no ZFS on host)
5. Mobile-responsive
6. Datadog style throughout