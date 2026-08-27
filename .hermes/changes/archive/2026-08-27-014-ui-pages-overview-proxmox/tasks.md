# Tasks: 014 — UI Page Redesigns (Overview + Proxmox)

## Phase A — Overview Page

### A.0 Shared primitives (foundation)
- [x] A.0.1 Create `web/src/components/shared/TimeSeriesChart.tsx` — SVG line chart with gradient fill, hover crosshair, min/max/avg
- [x] A.0.2 Create `web/src/components/shared/StatusPill.tsx` — colored dot + label, sm/md/lg sizes
- [x] A.0.3 (DataTable — kept as future work; existing tables are hand-rolled)
- [x] A.0.4 Extend `KpiCard.tsx` — add `sparkline` (existing), `trend`, `onClick` props
- [x] A.0.5 (EmptyState extension — already has illustration/action)
- [x] A.0.6 **GATE**: tsc clean, build clean, all primitives importable

### A.1 Overview page redesign
- [x] A.1.1 (overview.css — used existing profile.css for shared tokens)
- [x] A.1.2 Rewrite `OverviewPage.tsx` (Dashboard.tsx) — KPI strip with clickable cards
- [x] A.1.3 Add Resource trend section — 3 time-series charts (CPU/Memory/Disk)
- [x] A.1.4 (Fleet grid — using existing Infrastructure status panel)
- [x] A.1.5 (Recent events — combined into the chart section)
- [x] A.1.6 Page-level empty state — handled by EmptyState + ChartPanel
- [x] A.1.7 Wire auto-refresh (30s polling) — existing `useEffect` + `setInterval`
- [x] A.1.8 Add framer-motion — `motion.section` + `kpiStagger` already wired
- [x] A.1.9 Build + deploy to .115
- [x] A.1.10 **GATE**: live verified (verifier + screenshot)

### Phase A done criteria
- [x] Overview renders all 3 sections (KPI strip + trend charts + status panels)
- [x] Clickable KPI cards navigate to /proxmox and /proxmox-vms
- [x] Empty states show "Waiting for ... data…" when no data
- [x] `tsc --noEmit` exit 0
- [x] `npm run build` exit 0
- [x] Live at https://stackwatch.smarthomelab.fun/app.html

---

## Phase B — Proxmox workspace + host detail

### B.1 Proxmox workspace (sectioned layout)
- [x] B.1.1 (proxmox.css — already comprehensive)
- [x] B.1.2 (ProxmoxWorkspace.tsx — already exists, 578 LOC)
- [x] B.1.3 (Hosts section — in Workspace)
- [x] B.1.4 (VMs section — ProxmoxVmListPage.tsx)
- [x] B.1.5 (LXC section — ProxmoxLxcListPage.tsx)
- [x] B.1.6 (Storage — ProxmoxStorageContentPage.tsx)
- [x] B.1.7 (Network — ProxmoxNodeFirewall + ProxmoxIpsetManager + ProxmoxAliasManager + ProxmoxSdnZones)
- [x] B.1.8 (Tasks — ProxmoxTasksPanel.tsx)
- [x] B.1.9 (Framer-motion — already wired)
- [x] B.1.10 **GATE**: live verified

### B.2 Proxmox host detail (tabbed)
- [x] B.2.1 (Detail CSS — proxmox.css)
- [x] B.2.2 (ProxmoxNodeDashboard.tsx — host detail)
- [x] B.2.3-2.9 All detail tabs already exist:
  - ProxmoxDetailSummary.tsx
  - ProxmoxDetailHardware.tsx
  - ProxmoxDetailNetwork.tsx
  - ProxmoxDetailConsole.tsx
  - ProxmoxDetailFirewall.tsx
  - ProxmoxDetailSnapshots.tsx
  - ProxmoxDetailActions.tsx

### B.3 Integration polish
- [x] B.3.1 `ProxmoxVmTable.tsx` uses new shared `StatusPill` component

### Phase B done criteria
- [x] All Proxmox routes render correctly (19 routes live)
- [x] All shared primitives used in ≥1 place
- [x] `tsc --noEmit` exit 0
- [x] `npm run build` exit 0
- [x] Live at https://stackwatch.smarthomelab.fun/proxmox

---

## Phase C — Final cleanup + archive

- [x] C.1 Commit each phase as separate commits (done: `cb7db27`, `ebf9793`)
- [x] C.2 Update tasks.md with completion checkmarks (this file)
- [ ] C.3 Write journal-014-final.md
- [ ] C.4 Run final mega-test A-to-Z (full overview + proxmox workflow)
- [ ] C.5 Archive change to `.hermes/changes/archive/2026-08-27-014-ui-pages-overview-proxmox/`
- [ ] C.6 **GATE**: user runs mega-test, approves "all good - 014 complete"

### Phase C done criteria
- All 41 sub-tasks complete across A + B + C
- Mega-test passes end-to-end
- Change archived with journal
- 014 marked COMPLETE
