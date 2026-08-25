# Tier 14 Phase 14.1 Requirements Checklist

## Functional
- [ ] **F1** Host selector dropdown with all registered Proxmox hosts
- [ ] **F2** KPI strip (Total / Running / Stopped / Paused / Cluster CPU% / Cluster RAM%)
- [ ] **F3** Search bar (name / vmid / IP)
- [ ] **F4** Status filter chips (All / Running / Stopped / Paused)
- [ ] **F5** VM table with columns: Status / VMID / Name / Node / IP / CPU% / RAM / Uptime / Actions
- [ ] **F6** VM actions: Start / Stop / Reboot / Shutdown
- [ ] **F7** Confirmation dialog for destructive actions
- [ ] **F8** Loading state (skeleton rows)
- [ ] **F9** Empty state (no hosts / no VMs)
- [ ] **F10** Error state (host unreachable)
- [ ] **F11** Auto-refresh every 5 seconds
- [ ] **F12** Pause auto-refresh when tab hidden
- [ ] **F13** Mobile-responsive (table → cards on <768px)
- [ ] **F14** Dark theme (Datadog-style)

## Animation (framer-motion)
- [ ] **A1** Page enter: staggered children (50ms each)
- [ ] **A2** Row hover: scale 1.01 + bg lift (150ms ease-out)
- [ ] **A3** KPI count: animate from 0 to value (spring)
- [ ] **A4** Action button click: scale 0.95 → 1 (150ms)
- [ ] **A5** Status pill change: color transition (300ms)

## Backend
- [ ] **B1** `GET /api/v1/proxmox/hosts` returns ≥1 host (verify with curl)
- [ ] **B2** `GET /api/v1/proxmox/hosts/:id/qemu` returns VM list (verify)
- [ ] **B3** `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/:action` works (verify)

## Quality
- [ ] **Q1** Every TSX file < 400 LOC
- [ ] **Q2** No TypeScript errors
- [ ] **Q3** No browser console errors
- [ ] **Q4** No new dependencies unless required (framer-motion if not installed)
- [ ] **Q5** Reuses existing components (KpiCard, StatusPill) where possible
- [ ] **Q6** Color tokens from existing `colors.ts`/`polish.css`

## User-acceptance
- [ ] **U1** Open `/app.html/proxmox`, verify KPI strip renders live data
- [ ] **U2** Filter to "Running", verify table filters
- [ ] **U3** Search by IP, verify filter
- [ ] **U4** Start a stopped VM, verify status flips
- [ ] **U5** Stop a running VM, verify confirm dialog
- [ ] **U6** Empty host test
- [ ] **U7** Error state test (kill Proxmox API)
- [ ] **U8** Mobile test (375px)
