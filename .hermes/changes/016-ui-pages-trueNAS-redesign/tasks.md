# Tasks: 016 — TrueNAS workspace redesign

## Phase A — KPI strip + section polish

- [ ] A.1 Add KPI strip component (6 clickable cards)
  - Total hosts / Online hosts / Pool health / Disk count / Active shares / Snapshots
  - Each with sparkline + click navigation
- [ ] A.2 Move existing host-management section to bottom of page
- [ ] A.3 **GATE**: KPI strip live verified, all sections still render

## Phase B — Pool + Disk + Snapshot visualizations

- [ ] B.1 Create `web/src/components/truenas/PoolHealthCard.tsx` (if needed)
  - Pool name + health StatusPill + used/total bar
- [ ] B.2 Replace Pools section JSON dump with PoolHealthCard grid
- [ ] B.3 Replace Disks section with temperature chart (TimeSeriesChart) + StatusPill per disk
- [ ] B.4 Replace Snapshots section with timeline (grouped by dataset, sparkline of ages)
- [ ] B.5 **GATE**: live verify all 3 sections

## Phase C — Final polish + archive

- [ ] C.1 Verify module discipline (every file ≤400 LOC)
- [ ] C.2 Commit per-phase
- [ ] C.3 Update tasks.md with checkmarks
- [ ] C.4 Write journal-016-final.md
- [ ] C.5 Run final mega-test A-to-Z
- [ ] C.6 Archive to `.hermes/changes/archive/2026-08-27-016-ui-pages-trueNAS-redesign/`
- [ ] C.7 **GATE**: user approves "all good - 016 complete"

### Done criteria

- TrueNAS workspace has KPI strip + visualizations
- All existing 11 sections still work
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at .115
- 016 archived with journal
