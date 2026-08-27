# Design: 014 — UI Page Redesigns (Overview + Proxmox)

## Approach

Two-phase speckit:
- **Phase A** — Overview page redesign (1 week of work)
- **Phase B** — Proxmox workspace + host detail (1 week of work)

Shared primitives added in Phase A, used in Phase B (no re-invention).

## Visual language (consistent with 013 design system)

### Color palette (extends tokens.css)
```
--color-bg:           #060914 (background)
--color-surface:      #0d1322 (panels)
--color-surface-2:    #131b30 (raised)
--color-border:       rgba(255,255,255,0.06)
--color-text:         #e8edf5
--color-text-muted:   #94a3b8
--color-text-dim:     #64748b

--color-accent:       #22d3ee (cyan)
--color-accent-2:     #818cf8 (indigo)
--color-success:      #10b981 (green)
--color-warning:      #f59e0b (amber)
--color-danger:       #ef4444 (red)
--color-info:         #3b82f6 (blue)

--color-critical:     #ef4444
--color-warning-sev:  #f59e0b
--color-info-sev:     #3b82f6
--color-ok:           #10b981
```

### Typography
- Display: 32px / 600 / -0.02em (page titles)
- H2: 24px / 600
- H3: 18px / 600
- Body: 14px / 400
- Caption: 12px / 500 / uppercase / +0.04em letter-spacing
- Numeric: tabular-nums, monospace fallback
- Family: Inter, system-ui, -apple-system, sans-serif

### Spacing
4px / 8px / 12px / 16px / 24px / 32px / 48px / 64px

### Radius
6px / 8px / 12px / 16px

### Shadows
```
--shadow-sm: 0 1px 2px rgba(0,0,0,0.2)
--shadow-md: 0 4px 12px rgba(0,0,0,0.3)
--shadow-lg: 0 12px 32px rgba(0,0,0,0.4)
```

### Motion
- Page enter: fadeUp 0.3s ease-out
- Card stagger: 50ms between siblings
- Hover: 150ms ease-out (subtle shadow lift)
- Tap: 0.95 scale for 100ms
- All motion respects `useReducedMotion()`

## Shared primitives

### KpiCard (extend existing)
Already at `web/src/components/shared/KpiCard.tsx`. Extend with:
- Optional `sparkline` prop (number[] for mini chart)
- Optional `trend` prop ('up' | 'down' | 'flat')
- Optional `onClick` for clickable cards

### TimeSeriesChart (NEW)
- Props: `series` (number[]), `labels` (string[]), `color`, `unit`, `min`, `max`
- SVG-based, 240×80 default
- Hover crosshair (timestamp + value)
- Gradient fill below the line
- Min/max/avg below
- Empty state when no data

### StatusPill (NEW)
- Props: `status` ('up' | 'stale' | 'down' | 'unknown'), `label`, `size`
- Colored dot + label
- Sizes: 'sm' (10px text), 'md' (12px), 'lg' (14px)

### DataTable (NEW)
- Props: `columns`, `rows`, `onRowClick`, `sortable`
- Built on semantic `<table>` for a11y
- Sortable column headers (click to toggle asc/desc)
- Hover row highlight
- Empty state slot

### EmptyState (extend existing)
Already at `web/src/components/shared/EmptyState.tsx`. Extend with:
- Optional `illustration` slot (ReactNode — emoji or SVG)
- Polished copy with headline + subhead
- Optional `action` slot for primary CTA button

## Overview page layout

```
+---------------------------------------------------+
| HELLO, Himanshu.          [LIVE pulse] Overview   |
+---------------------------------------------------+
| [Servers up] [Hosts online] [Alerts firing] [Incidents]
| KPI strip: 4 cards, each with sparkline           |
+---------------------------------------------------+
| Resource trend                                     |
| ┌─ CPU ─┐  ┌─ Memory ─┐  ┌─ Disk ─┐               |
| │  24h  │  │   24h    │  │  24h   │               |
| │chart  │  │  chart   │  │ chart  │               |
| └───────┘  └──────────┘  └────────┘               |
+---------------------------------------------------+
| Fleet status (4-col grid of server tiles)          |
| ┌─ foo ─┐  ┌─ bar ─┐  ┌─ baz ─┐  ┌─ qux ─┐         |
| │ 12%   │  │ 4%    │  │ 87%   │  │ 23%   │         |
| └───────┘  └───────┘  └───────┘  └───────┘         |
+---------------------------------------------------+
| Recent events (vertical timeline)                  |
| ● 2m ago  alert_fired  cloud-app   CPU 87%        |
| ● 5m ago  deploy_ok   api-gateway v1.2.3         |
+---------------------------------------------------+
```

## Proxmox workspace layout

```
+---------------------------------------------------+
| Proxmox                       [+ Register host]    |
+---------------------------------------------------+
| Hosts section                                      |
| KPI: Total / Online / Issues / Last sync          |
| Hosts table                                        |
+---------------------------------------------------+
| VMs section                                        |
| KPI: Total / Running / Stopped / Templates         |
| VMs table with action menu                         |
+---------------------------------------------------+
| LXC section  (same pattern)                        |
+---------------------------------------------------+
| Storage section                                    |
| KPI: Total / Healthy / Degraded / Offline         |
| Storages table                                     |
+---------------------------------------------------+
| Network section                                    |
| Network table (bridges / bonds / VLANs)            |
+---------------------------------------------------+
| Tasks section                                      |
| Last 25 tasks (auto-refresh 5s)                   |
+---------------------------------------------------+
```

## Proxmox host detail layout

```
+---------------------------------------------------+
| < Hosts / router                            [actions]
| pve-router.local (192.168.0.107)                  |
| Proxmox VE 8.x · Uptime 27d                       |
+---------------------------------------------------+
| [Summary] [VMs] [LXC] [Storage] [Network] [FW] [Tasks]
|                                                  |
| (active tab content)                              |
+---------------------------------------------------+
```

## File structure

```
web/src/
  pages/
    OverviewPage.tsx           # rewritten (was OverviewPage)
    ProxmoxPage.tsx            # rewritten
    ProxmoxHostDetailPage.tsx  # rewritten
  components/
    shared/
      KpiCard.tsx              # extended (sparkline, trend, onClick)
      TimeSeriesChart.tsx      # NEW
      StatusPill.tsx           # NEW
      DataTable.tsx            # NEW
      EmptyState.tsx           # extended (illustration, action)
  styles/
    pages/
      overview.css             # NEW (page-specific)
      proxmox.css              # NEW (page-specific)
```

Every file ≤400 LOC. Every new component used in ≥2 places.

## Implementation order

**Phase A (Overview):**
1. Build shared primitives (TimeSeriesChart, StatusPill, DataTable)
2. Extend KpiCard with sparkline
3. Rewrite OverviewPage.tsx using primitives
4. Build + deploy + verify

**Phase B (Proxmox):**
1. Rewrite ProxmoxPage.tsx with sectioned layout
2. Rewrite ProxmoxHostDetailPage.tsx with tabs
3. Build + deploy + verify

Each phase ends with:
- `npx tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verification at .115
- Per-phase test plan delivered to user

After all phases: full mega-test A-to-Z.
