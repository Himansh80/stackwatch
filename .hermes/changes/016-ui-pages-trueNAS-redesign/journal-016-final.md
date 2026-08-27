# 016 — TrueNAS workspace redesign — FINAL JOURNAL

**Status:** COMPLETE — 2026-08-27

## Summary

TrueNAS workspace page (`/truenas`) now has a proper Datadog-style visualization:
- 6-card KPI strip at top (clickable, navigates to section)
- Pools section: 3-col health grid with StatusPill + usage bar + fragmentation
- Disks section: 2-col temp chart grid with online/offline StatusPill
- Snapshots section: dataset-grouped timeline with age sparkline + stale detection
- Other 8 sections (Datasets, NFS, SMB, iSCSI, Users, Services, Cloud Sync, Boot Env) keep the existing table view

## What was built

### Phase 6A — KPI strip
- 6 cards: Total hosts / Online hosts / Pools / Active shares / Snapshots / Disks
- Clickable (where it makes sense) — navigates to section
- Status-aware (Online hosts changes accent based on connectivity)
- KPI metrics computed client-side from loaded hosts list + section rows
- New `.tru-kpi-grid` CSS (6-col → 3-col → 2-col responsive)

### Phase 6B — Section visualizations
- **PoolHealthCard** — pool name + health StatusPill + used/total gradient bar + fragmentation % + bytes display
- **DiskTempCard** — disk name + model + StatusPill (temp°C or Offline) + TimeSeriesChart sparkline
- **SnapshotGroup** — dataset-grouped snapshots with count + last-snapshot name + age sparkline + stale detection (>14d = warn)
- Section-aware rendering: pools/disks/snapshots use cards; others use table fallback
- New CSS: `.tru-pool-grid`, `.tru-disk-grid`, `.tru-snap-grid`, `.tru-pool-card`, `.tru-pool-bar`, `.tru-pool-stats`, `.tru-disk-card`, `.tru-snap-group`, `.tru-snap-head`

## Files modified

- `web/src/pages/TrueNASWorkspace.tsx` — 168 → 343 LOC (under 400 cap)
- `web/src/styles/profile.css` — added ~100 lines of TrueNAS-specific CSS

## Verification

| Check | Result |
|-------|--------|
| `tsc --noEmit` | exit 0 |
| `npm run build` | exit 0 (596 modules transformed) |
| `<400 LOC cap` | TrueNASWorkspace 343 LOC |
| Bundle deployed to .115 | ✅ via scp |

## Commits

```
954e124 feat(ui): Phase 6B — TrueNAS Pools/Disks/Snapshots visualizations
b21c108 feat(ui): Phase 6 — TrueNAS workspace KPI strip
```

## Honest gaps

- **Historical metrics**: The disk temp chart is synthesized from `Math.sin(i / 2)` because TrueNAS doesn't expose historical samples in the current API. Once backend adds `/system/diskstats/history` endpoint, switch to real data.
- **Pool fragmentation %**: The API returns `fragmentation` or `frag_percent` but the exact field name varies by TrueNAS version. The `readNumber()` helper checks both keys.
- **Snapshot age threshold**: Hardcoded at 14 days. Should be a per-policy setting eventually.
- **NFS/SMB/iSCSI**: These sections still render as raw tables. Could get similar card treatments but weren't in scope for this change.

## Lessons learned

1. **Section-aware rendering in the same page** is a good pattern for low-effort high-value visual upgrades. No new endpoints needed.
2. **Inline component functions** (PoolHealthCard, DiskTempCard, SnapshotGroup) keep file count low while still modular. Move to `components/truenas/` only if reused across pages.
3. **CSS in shared profile.css** is fine for low-volume additions. Split into `styles/truenas.css` once it exceeds ~200 lines.

## Follow-up

- 017-ui-pages-apm-redesign — APM dashboard polish (charts-heavy)
- 018-ui-pages-incidents-redesign — Incidents + Notifications + Intelligence polish
