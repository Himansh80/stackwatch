# Tier 14 — Phase 14.1 Plan: VM List Page

## Frontend (new files in `web/src/pages/proxmox/`)

### Step 1: Page shell + routing
- `web/src/pages/proxmox/ProxmoxPage.tsx` — root page, hosts routing
- Add route `/app.html/proxmox` in router

### Step 2: Sub-components (in `web/src/components/proxmox/`)
- `ProxmoxHostSelector.tsx` (≤150 LOC)
- `ProxmoxKpiStrip.tsx` (≤150 LOC)
- `ProxmoxFilterBar.tsx` (≤150 LOC)
- `ProxmoxVmTable.tsx` (≤350 LOC)
- `ProxmoxVmRow.tsx` (≤150 LOC)
- `ProxmoxVmActions.tsx` (≤150 LOC)
- `ProxmoxEmptyState.tsx` (≤150 LOC)

### Step 3: Styling
- `web/src/styles/proxmox.css` — page-specific styles
- Reuse existing dashboard tokens (`colors.ts` + `polish.css` if exists)

### Step 4: Animations
- `framer-motion` — check if already installed; install if not
- Add staggered enter, hover lift, action feedback

## Backend (no new endpoints)

Use existing Tier 1 endpoints:
- `GET /api/v1/proxmox/hosts` — list
- `GET /api/v1/proxmox/hosts/:id/qemu` — VMs on host
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/:action` — action

## DB

No new tables.

## Tests

- Live curl: `GET /api/v1/proxmox/hosts` → 200 with 1 host
- Live curl: `GET /api/v1/proxmox/hosts/<id>/qemu` → 200 with VM list
- Manual: open page, exercise each filter, each action

## Order of implementation

1. Page shell + route
2. Host selector
3. KPI strip
4. Filter bar
5. VM table
6. VM row
7. VM actions
8. Empty + error states
9. Animations
10. Mobile responsive
11. Live test

## Modular discipline

- Every file under 400 LOC
- Reuse existing components (KpiCard, StatusPill if they exist)
- No new deps unless strictly needed (framer-motion if not installed)

## Commit strategy

- Commit after each step if it produces visible progress (host selector, KPI strip, etc.)
- Final commit: "feat(tier14): Phase 14.1 Proxmox VM list page"
