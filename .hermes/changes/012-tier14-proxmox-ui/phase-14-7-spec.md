# Tier 14 Phase 14.7 — Specification: Node Dashboard + Storage Content

## 1. URL pattern
- `/proxmox/nodes/:hostId/:node` — node dashboard
- `/proxmox/storage/:hostId/:node/:storage` — storage content

## 2. Node Dashboard layout

Top:
- Breadcrumb (Proxmox > {node})
- Heading with node name + status badge
- KPI strip: CPU% / RAM used/total / Disk used/total / Load avg / Uptime

Cards below in 2-col grid:
- **Disks** — table: model, size, type (HDD/SSD/NVMe), usage
- **ZFS pools** — table: name, size, used, free, frag% (empty state if no ZFS)
- **Network** — table: iface, MAC, IPv4/CIDR, MTU, speed
- **Services** — table: name, state (running/stopped), autostart (yes/no)

## 3. Storage Content browser

Top:
- Breadcrumb (Proxmox > {node} > Storage: {storage})
- Heading with storage name + content types (iso, vztmpl, images, backup, snippets)
- Upload area (drag-and-drop placeholder for Phase 14.8) + URL upload form
- Table: volid | format | size | parent | date | Actions (delete)

## 4. State

- Each page fetches on mount + every 30s (slow refresh — these change less)
- Loading + error + empty states everywhere

## 5. Acceptance criteria

1. KPI strip shows live data
2. Disks table shows all physical disks
3. ZFS table shows pools (empty if none)
4. Network table shows interfaces
5. Services table shows systemd services
6. Storage content table shows files with type icons
7. Delete works (with confirm)
8. Empty states everywhere
9. Mobile-responsive
10. Dark theme + Datadog style
