# Tier 14 Phase 14.7 Plan

## Approach
Two new pages: Node Dashboard (5 cards) + Storage Content (file list).

## Frontend (3 new files + 1 new CSS)

### ProxmoxNodeDashboard.tsx (~280 LOC)
- KPI strip (reuse pattern from VM list)
- Disks card (table)
- ZFS pools card (table or empty)
- Network card (table)
- Services card (table)
- Auto-refresh every 30s (slow refresh — host state changes slowly)
- Error handling: each section handles its own failure

### ProxmoxNodeKpiStrip.tsx (~130 LOC)
- Reusable KPI strip (CPU/RAM/Disk/Load/Uptime)
- Animated count-up

### ProxmoxStorageContentPage.tsx (~250 LOC)
- Storage info header
- File list table with type icons (ISO/template/backup)
- Delete button per row (with confirm modal)
- Upload form (URL field for now, drag-drop in Phase 14.8)

### proxmox-node.css (~200 LOC)
- Card layout (2-col grid)
- KPI strip styles
- Service status pills

## Order
1. KPI strip
2. Node dashboard
3. Storage content page
4. CSS
5. Routes
6. Sidebar links
7. tsc + build + deploy
8. Commit

## Modularity discipline
- Every TSX < 400 LOC
- Reuse px-kpi-strip pattern
- Each section as separate card
