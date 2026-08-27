# Design: 016 — TrueNAS workspace redesign

## Visual language

Consistent with 014 (Overview) + 015 (StatusPill migration). New tokens:
- `.tru-kpi-grid` — 6-col KPI strip (auto-collapse to 3-col at <1100px, 2-col at <700px)
- `.tru-pool-grid` — 4-col pool card grid
- `.tru-disk-card` — disk card with temperature chart
- `.tru-snap-group` — snapshot group with sparkline

## KPI strip (top of page)

```
+--------------------------------------------------------------+
| [+ Add host]                                                  |
+--------------------------------------------------------------+
| [Hosts 3] [Online 1] [Pools 4] [Disks 3] [Shares 0] [Snaps 7] |
| (clickable cards, sparklines below)                          |
+--------------------------------------------------------------+
```

## Pool health grid (Pools section)

```
+--------------------------------------------------------------+
| POOLS                                          4 pools        |
+--------+----------+----------+----------+                    |
| tank   | datapool | log-pool | fastpool |                    |
| OK 87% | OK 23%    | OK 12%   | WARN 95% |                    |
| █████  | ███       | ██       | █████████|                    |
| 1.2T/  | 120G/    | 50G/    | 1.8T/    |                    |
| 1.4T   | 500G     | 400G    | 2.0T     |                    |
+--------+----------+----------+----------+                    |
```

## Disk temperature chart (Disks section)

```
+--------------------------------------------------------------+
| DISKS                                          3 disks       |
+--------------------------------------------------------------+
| sda (1.8TB) ONLINE                          45°C             |
| ╭───────────────────────────────────────────╮              |
| │ (temperature chart)                        │              |
| ╰───────────────────────────────────────────╯              |
| sdb (1.8TB) ONLINE                          42°C             |
| ...                                                          |
+--------------------------------------------------------------+
```

## Snapshot timeline (Snapshots section)

```
+--------------------------------------------------------------+
| SNAPSHOTS                                       7 snapshots  |
+--------------------------------------------------------------+
| tank/home                                                    |
|   auto-2026-08-01  →  auto-2026-08-26  (26d)  ✓  [older]   |
| tank/backup                                                  |
|   weekly-2026-08-01  →  weekly-2026-08-22  (22d)  ✓         |
+--------------------------------------------------------------+
```

## Host management modal (existing)

- Live status badge → StatusPill (already migrated in 015)
- Add connection latency chart (TimeSeriesChart) when host is online
- Add last sync indicator (from heartbeat)

## File structure

```
web/src/pages/
  TrueNASWorkspace.tsx           # Main page (must stay ≤400 LOC)
web/src/components/truenas/      # NEW (if PoolHealthCard needed)
  PoolHealthCard.tsx            # ≤200 LOC
```

If file exceeds 400 LOC, split into:
- `TrueNASWorkspace.tsx` — page shell + KPI strip + section router
- `web/src/components/truenas/PoolHealthCard.tsx` — pool grid
- `web/src/components/truenas/DiskCard.tsx` — disk temperature card
- `web/src/components/truenas/SnapshotGroup.tsx` — snapshot timeline group

## Implementation order

**Phase A:** KPI strip + section navigation polish (~150 LOC added to TrueNASWorkspace)
**Phase B:** PoolHealthCard + DiskCard + SnapshotGroup components
**Phase C:** Module discipline check + archive

Each phase ends with:
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verification at .115
- Per-phase test plan delivered to user
