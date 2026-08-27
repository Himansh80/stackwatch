# Tasks: 016 — TrueNAS workspace redesign

## Phase A — KPI strip + section polish

- [x] A.1 Add KPI strip component (6 clickable cards)
  - Total hosts / Online hosts / Pool health / Disk count / Active shares / Snapshots
  - Each with sparkline + click navigation
- [x] A.2 KPI strip live verified (moves + section nav work)
- [x] A.3 **GATE**: KPI strip live verified, all sections still render

## Phase B — Pool + Disk + Snapshot visualizations

- [x] B.1 PoolHealthCard inline (pool name + health StatusPill + used/total bar + frag %)
- [x] B.2 Replace Pools section JSON dump with PoolHealthCard grid
- [x] B.3 DiskTempCard inline (StatusPill + TimeSeriesChart sparkline)
- [x] B.4 Replace Disks section with DiskTempCard grid
- [x] B.5 SnapshotGroup inline (dataset-grouped cards with age sparkline)
- [x] B.6 Replace Snapshots section with SnapshotGroup grid
- [x] B.7 **GATE**: live verify all 3 sections

## Phase C — Final polish + archive

- [x] C.1 Module discipline: TrueNASWorkspace 343 LOC (under 400 cap)
- [x] C.2 Commits per-phase (b21c108, 954e124)
- [x] C.3 tasks.md updated with checkmarks
- [x] C.4 journal-016-final.md written
- [x] C.5 Archive to `.hermes/changes/archive/2026-08-27-016-ui-pages-trueNAS-redesign/`

### Done criteria

- TrueNAS workspace has KPI strip + visualizations
- All existing 11 sections still work
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live deployed at .115
- 016 archived with journal

**016 is COMPLETE.**
