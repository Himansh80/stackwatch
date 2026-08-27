# Tasks: 014 — UI Page Redesigns (Overview + Proxmox)

## Phase A — Overview Page

### A.0 Shared primitives (foundation)
- [ ] A.0.1 Create `web/src/components/shared/TimeSeriesChart.tsx` — SVG line chart with gradient fill, hover crosshair, min/max/avg
- [ ] A.0.2 Create `web/src/components/shared/StatusPill.tsx` — colored dot + label, sm/md/lg sizes
- [ ] A.0.3 Create `web/src/components/shared/DataTable.tsx` — sortable semantic table, hover row, empty state slot
- [ ] A.0.4 Extend `KpiCard.tsx` — add `sparkline`, `trend`, `onClick` props
- [ ] A.0.5 Extend `EmptyState.tsx` — add `illustration` and `action` slots
- [ ] A.0.6 **GATE**: tsc clean, build clean, all 5 components importable

### A.1 Overview page redesign
- [ ] A.1.1 Add `web/src/styles/pages/overview.css` — page-specific layout
- [ ] A.1.2 Rewrite `OverviewPage.tsx` — Datadog-style KPI strip (4 cards with sparklines)
- [ ] A.1.3 Add Resource trend section — 3 time-series charts (CPU/Memory/Disk) with time-range pills
- [ ] A.1.4 Add Fleet status grid — 4-col server tiles (CPU/MEM/status dot)
- [ ] A.1.5 Add Recent events section — vertical timeline with severity dots + relTime
- [ ] A.1.6 Add page-level empty state when no servers connected
- [ ] A.1.7 Wire auto-refresh (30s polling) with `setInterval` cleanup
- [ ] A.1.8 Add framer-motion enter animation (pageEnter + kpiStagger)
- [ ] A.1.9 Build + deploy to .115
- [ ] A.1.10 **GATE**: live verify, deliver Phase A test plan

### Phase A done criteria
- All 14 sub-tasks complete
- Overview renders all 4 sections without errors
- Time-range pills work (1h/6h/24h/7d)
- Sparklines show 7-day history per metric
- Empty states are polished (no raw "no data" text)
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live at https://stackwatch.smarthomelab.fun/overview
- Mobile (<768px) lays out cleanly

---

## Phase B — Proxmox workspace + host detail

### B.1 Proxmox workspace (sectioned layout)
- [ ] B.1.1 Add `web/src/styles/pages/proxmox.css` — page-specific layout
- [ ] B.1.2 Rewrite `ProxmoxPage.tsx` — sectioned layout (Hosts / VMs / LXC / Storage / Network / Tasks)
- [ ] B.1.3 Hosts section: KPI strip + table + register-host modal
- [ ] B.1.4 VMs section: KPI strip + filterable table + actions menu
- [ ] B.1.5 LXC section: same pattern as VMs
- [ ] B.1.6 Storage section: KPI strip + table
- [ ] B.1.7 Network section: table
- [ ] B.1.8 Tasks section: auto-refresh every 5s
- [ ] B.1.9 Wire framer-motion (pageEnter + sectionStagger)
- [ ] B.1.10 **GATE**: live verify, deliver Phase B test plan

### B.2 Proxmox host detail (tabbed)
- [ ] B.2.1 Add `web/src/styles/pages/proxmox-host-detail.css`
- [ ] B.2.2 Rewrite `ProxmoxHostDetailPage.tsx` — 7 tabs (Summary / VMs / LXC / Storage / Network / Firewall / Tasks)
- [ ] B.2.3 Summary tab: KPI strip + node info table + uptime
- [ ] B.2.4 VMs tab: filtered VM table
- [ ] B.2.5 LXC tab: filtered LXC table
- [ ] B.2.6 Storage tab: filtered storage list
- [ ] B.2.7 Network tab: filtered network interfaces
- [ ] B.2.8 Firewall tab: rules + aliases + IPSets
- [ ] B.2.9 Tasks tab: filtered task list with auto-refresh
- [ ] B.2.10 Build + deploy + verify

### Phase B done criteria
- All 21 sub-tasks complete
- All 7 tabs on host detail render correctly
- All 6 sections on workspace render correctly
- All shared primitives used in ≥2 places
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live at https://stackwatch.smarthomelab.fun/proxmox and /proxmox-hosts/:id
- Lighthouse a11y ≥90 on Overview

---

## Phase C — Final cleanup + archive

- [ ] C.1 Commit each phase as separate commits
- [ ] C.2 Update tasks.md with completion checkmarks
- [ ] C.3 Write journal-014-final.md
- [ ] C.4 Run final mega-test A-to-Z (full overview + proxmox workflow)
- [ ] C.5 Archive change to `.hermes/changes/archive/2026-08-27-014-ui-pages-overview-proxmox/`
- [ ] C.6 **GATE**: user runs mega-test, approves "all good - 014 complete"

### Phase C done criteria
- All 41 sub-tasks complete across A + B + C
- Mega-test passes end-to-end
- Change archived with journal
- 014 marked COMPLETE
