# Tier 14 Phase 14.4 Plan: LXC UI

## Approach

Mirror Phase 14.1 (VM list) + 14.2 (VM detail) + 14.3 (CRUD) for LXC. The backend already supports LXC (`/cluster/resources` returns both types). Just need to:
1. Filter by `type === 'lxc'` in the list query
2. Use `/nodes/:node/lxc/:vmid` instead of `/nodes/:node/qemu/:vmid` for detail
3. Build a 4-step create wizard (templated from existing form patterns)

## Backend changes
ZERO — Tier 1 already provides everything.

## Frontend (8 new files + 1 route registration)

### Step 1: List page (mirror of VM list)
- `web/src/components/proxmox/ProxmoxLxcListPage.tsx` (~200 LOC)
- Reuses ProxmoxKpiStrip + ProxmoxFilterBar + ProxmoxEmptyState + ProxmoxVmTable
- Filter `resources.filter(r => r.type === 'lxc')`

### Step 2: Detail page (mirror of VM detail)
- `web/src/components/proxmox/ProxmoxLxcDetailPage.tsx` (~280 LOC)
- Reuses ProxmoxDetailHeader + ProxmoxDetailTabs + ProxmoxDetailSummary + ProxmoxDetailNetwork + ProxmoxDetailSnapshots + ProxmoxDetailFirewall
- Console tab: placeholder (LXC serial console is Phase 14.7)
- API paths use `/lxc/:vmid` instead of `/qemu/:vmid`

### Step 3: Create wizard (4 steps + reusable template picker)
- `web/src/components/proxmox/ProxmoxLxcCreatePage.tsx` (~200 LOC, orchestrator)
- `web/src/components/proxmox/ProxmoxLxcTemplatePicker.tsx` (~200 LOC)
- `web/src/components/proxmox/ProxmoxLxcCreateStep1Template.tsx` (~100 LOC)
- `web/src/components/proxmox/ProxmoxLxcCreateStep2Resources.tsx` (~180 LOC)
- `web/src/components/proxmox/ProxmoxLxcCreateStep3Network.tsx` (~150 LOC)
- `web/src/components/proxmox/ProxmoxLxcCreateStep4Confirm.tsx` (~150 LOC)

### Step 4: Wire into App.tsx
- 3 new routes: `/proxmox-lxc`, `/proxmox-lxc/new`, `/proxmox-lxc/:hostId/:node/:vmid`
- Sidebar: add "LXC" item under Infrastructure

### Step 5: Styling
- Reuse proxmox.css, proxmox-detail.css, proxmox-detail-dialogs.css
- New: small additions for wizard step bar in proxmox-detail.css

## Order

1. List page
2. Detail page (mirror)
3. Template picker
4. Wizard orchestrator + step 1
5. Step 2 (resources)
6. Step 3 (network)
7. Step 4 (confirm)
8. Wire routes
9. Sidebar nav item
10. tsc + build + deploy
11. Commit

## Modularity discipline

- Every TSX < 400 LOC
- Wizard steps are separate components (each ≤200)
- Reuse existing KPI/Filter/Table/EmptyState

## Risks

- LXC templates list (vztmpl) requires per-storage fetch — may need to iterate over storages
- Auto-detect next VMID: Proxmox doesn't have a clean API for this; user must enter or we pick via /cluster/nextid
- Static IP parsing: validate CIDR format before submit