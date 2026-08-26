# Tier 14 Phase 14.4 — Specification: LXC UI

## 1. URL pattern

- `/proxmox-lxc` — list
- `/proxmox-lxc/:hostId/:node/:vmid` — detail
- `/proxmox-lxc/new` — create wizard

## 2. List page layout (mirror of /proxmox-vms)

- Host selector (top)
- KPI strip: Total / Running / Stopped / Paused / Cluster CPU / Cluster RAM
- Filter bar: search + status chips
- Table (desktop) / cards (mobile): Status / VMID / Type / Name / Node / IP / RAM / CPU / Uptime / Actions

## 3. Detail page layout (mirror of /proxmox-vms/:id/:node/:vmid)

- Header: breadcrumb + name + status pill + ID/Type/Node/Uptime + actions
- Tabs: Summary / Resources / Network / Snapshots / Firewall / Console
- Summary: status + uptime + CPU + RAM + Disk + Tags
- Resources: cores / memory / disk / arch / ostype / boot
- Network: same as VM (QEMU agent interfaces)
- Snapshots: same as VM (CRUD)
- Firewall: same as VM (CRUD)
- Console: placeholder (LXC uses serial console via pct, defer to Phase 14.7)

## 4. Create wizard layout (`/proxmox-lxc/new`)

4-step wizard with progress bar at top:

### Step 1: Template
- Dropdown of available LXC templates (from `/storage/:storage/content?content=vztmpl`)
- Shows: name, size, version
- Click → selects

### Step 2: Resources
- VMID (auto-fill next free)
- Hostname
- Cores (1-32)
- Memory MB (128-65536)
- Swap MB (0 = disabled)
- Disk GB
- Root storage (dropdown of available storages)

### Step 3: Network
- Bridge (dropdown: vmbr0, vmbr1, etc.)
- IPv4 (DHCP / static IP+CIDR+gateway)
- IPv6 (DHCP / SLAAC / static)
- VLAN tag (optional)

### Step 4: Confirm
- Review all settings
- "Create container" button → POST /lxc
- On success → redirect to `/proxmox-lxc/:hostId/:node/:newVmid`
- On error → stay on step 4 with error message

## 5. State management

- Wizard state in `ProxmoxLxcCreatePage` only (no URL persistence)
- Template + Resources + Network fields collected across steps
- "Back" buttons on steps 2-4; "Next" on 1-3
- Final submit on step 4

## 6. Acceptance criteria

1. List loads with LXC containers only (filtered)
2. KPI strip shows LXC counts
3. Filter chips work
4. Click row → detail page
5. Detail tabs all work
6. Wizard step 1 lists templates from API
7. Wizard step 2-3 save state across navigation
8. Wizard step 4 creates LXC and redirects
9. Empty states everywhere (no templates / no containers)
11. Mobile-responsive throughout

## 7. Backend dependencies (all existing in Tier 1)

- `GET /api/v1/proxmox/hosts/:id/qemu` → list VMs+LXC (filter client-side)
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/lxc/:vmid` → config
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/lxc` → create
- `DELETE .../lxc/:vmid` → delete
- `POST .../lxc/:vmid/status/:action` → lifecycle
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/storage/:storage/content` → templates list

## 8. Out of scope

- LXC console (serial terminal via pct) → Phase 14.7
- LXC migration → Phase 14.3 already covers VM migrate, LXC needs same UI (deferred)
- Bulk operations → Phase 14.10