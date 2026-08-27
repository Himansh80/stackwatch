# Proposal: 016 — TrueNAS workspace redesign

## Why

The TrueNAS workspace page (`/truenas`) already has 11 sections (Overview, Pools, Datasets, NFS, SMB, iSCSI, Snapshots, Disks, Users, System Services, Cloud Sync) and a host management modal. But:

- No KPI strip — jumps straight into a JSON-RPC response table
- No time-series visualization — connections/disk activity just dump raw rows
- Status badges are inconsistent — some use `StatusPill` (migrated in 015), some inline spans
- No per-pool/disk usage chart — ZFS pool health is rendered as a JSON dump

This change applies the new design system (TimeSeriesChart, clickable KpiCard, StatusPill) and adds proper KPIs/charts so the TrueNAS workspace feels like a first-class observability surface.

## What changes

### 1. KPI strip at top (clickable cards)
- **Total hosts** — count of registered TrueNAS hosts
- **Online hosts** — count with status=online (green StatusPill inline)
- **Pool health** — degraded/healthy/offline counts
- **Disk count** — total + failures
- **Active shares** — NFS + SMB + iSCSI combined
- **Snapshots** — total + total size

Each card: accent stripe, sparkline of last 24h metric, click → navigate to section.

### 2. Pool health visualization
- For the Pools section, replace the JSON dump with a grid of pool cards
- Each card shows: name, used/total bar (CSS-only), fragmentation %, health StatusPill
- Click pool card → expand to show datasets underneath

### 3. Disk health visualization
- For the Disks section, make a temperature chart per disk (TimeSeriesChart with `values=temperatures[]`)
- StatusPill for each disk (online/offline/error)
- Compact table view as alternative

### 4. Snapshot timeline
- For the Snapshots section, group by dataset
- Each dataset row shows recent snapshot ages (sparkline of age-in-days)
- StatusPill for "healthy" / "no-recent-snapshots"

### 5. Host health card (existing host-management modal upgrade)
- Live status badge (StatusPill)
- Connection latency chart (TimeSeriesChart)
- Last sync indicator

## Scope guardrails

- **TrueNAS workspace page only.** Other pages (Proxmox, Billing, etc.) ship in their own speckits.
- **No new endpoints.** Same backend, better rendering.
- **Use existing primitives** (TimeSeriesChart, StatusPill, KpiCard). No new shared components.
- **Module discipline** — every file ≤400 LOC. The TrueNAS page is currently 168 LOC — must stay under 400 even after redesign.
- **Same data sources** — `/api/v1/truenas/*` endpoints (via truenas-connector sidecar).

## Out of scope

- New TrueNAS endpoints
- Other pages
- Mobile app
- Documentation updates

## Impact

| Area | Impact |
|------|--------|
| Files modified | 1 (`web/src/pages/TrueNASWorkspace.tsx`) |
| Files added | 1 optional (`web/src/components/truenas/PoolHealthCard.tsx` if 400 LOC exceeded) |
| Breaking | No |
| Risk | Visual regression on TrueNAS surface. Mitigated by per-phase test plan. |

## Workflow

- Speckit: proposal → spec → design → tasks → checklist → build → archive
- Per-phase test plan (after each phase)
- Final mega-test A-to-Z
- Reuse primitives from 014 (TimeSeriesChart, StatusPill, KpiCard)

## Acceptance

- TrueNAS workspace renders KPI strip at top with 6 clickable cards
- Pools section shows health grid (no JSON dump)
- Disks section shows temperature chart
- Snapshots section shows timeline grouped by dataset
- Host management modal shows live status + latency chart
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at https://stackwatch.smarthomelab.fun/truenas

## Rollback

- Revert the single commit "feat(ui): 016 TrueNAS workspace redesign"
