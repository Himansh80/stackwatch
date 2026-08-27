# Spec: 014 — UI Page Redesigns (Overview + Proxmox)

## Functional requirements

### Overview page (`/overview`)

#### 1. KPI strip (top of page)
- **Servers up**: count of servers with status=up AND last_seen < 5min, shown as `{count} of {total}` with percentage
- **Hosts online**: count of distinct hostnames with heartbeats in last 5min (Proxmox + TrueNAS)
- **Alerts firing**: count of alerts in state=open, colored by severity (red if any critical, amber if any warning, green if zero)
- **Active incidents**: count of incidents in state=acknowledged OR open, colored by severity
- Each card: colored top stripe (cyan/green/amber/red based on metric type), big numeric value, sublabel, mini sparkline (last 7 days of that metric)
- Click any card → navigates to relevant page (/servers, /alerts, /incidents)

#### 2. Resource trend (middle of page)
- 3 time-series charts: CPU / Memory / Disk
- Time-range pills: 1h / 6h / 24h / 7d (default 24h)
- Each chart:
  - SVG-based, hover crosshair with timestamp + value
  - Gradient fill (cyan for CPU, violet for memory, amber for disk)
  - Min/max/avg stats below the chart
  - Y-axis label (e.g., "% CPU")
- Click chart → expands to full page with the same time range
- Empty state ("Waiting for live resource data") when no metrics

#### 3. Fleet status grid (lower middle)
- Grid of server tiles (4 columns on desktop, 2 on tablet, 1 on mobile)
- Each tile: server name (top), CPU% (large numeric), MEM% (small), status dot (up/stale/down)
- Tile click → navigates to /servers/{id}
- Color-coded:
  - Green border on left for online
  - Yellow for stale
  - Red for down
  - Gray for unknown

#### 4. Recent events (bottom)
- Vertical timeline (last 50 events)
- Each event: timestamp (relTime), severity dot, message preview, host
- Severity colors: red (critical), amber (warning), blue (info), gray (debug)
- Hover on event → shows full event JSON in tooltip
- "View all events →" link → /alerts

### Proxmox workspace (`/proxmox`)

#### 1. Hosts section (top)
- KPI strip: Total hosts / Online / Issues / Last sync
- Table of hosts: name, IP, version, status, last_seen
- Each row → /proxmox-hosts/:id
- "+ Register host" button → modal with host URL + token fields

#### 2. VMs section
- KPI strip: Total VMs / Running / Stopped / Templates
- Table: VMID, name, host, status (running/stopped/paused), CPU%, MEM%, Uptime, Actions
- Actions menu per row: Start / Stop / Shutdown / Reset / Migrate / Console
- Filter chips: All hosts / Selected host / Show templates
- Search bar (VMID / name)

#### 3. LXC section
- Same pattern as VMs but for containers

#### 4. Storage section
- KPI strip: Total storages / Healthy / Degraded / Offline
- Table: name, type, host, used, total, used%, status, actions
- Click row → /storage detail (modal or expand)

#### 5. Network section
- Table of bridges / bonds / VLANs
- Each row: name, type, host, attached interfaces, status

#### 6. Tasks section
- Last 25 tasks: UPID, type, host, status, started, duration
- Auto-refresh every 5s
- Filter by status (running / failed / completed)

### Proxmox host detail (`/proxmox-hosts/:id`)

#### Tabs:
- **Summary**: KPI strip (CPU/MEM/Disk) + node info table + uptime
- **VMs**: full VM table filtered to this host
- **LXC**: full LXC table filtered to this host
- **Storage**: storage list filtered to this host
- **Network**: network interfaces filtered to this host
- **Firewall**: rules + aliases + IPSets
- **Tasks**: tasks filtered to this host

## Non-functional requirements

- **Performance**: Overview page TTI < 1.5s on cached load
- **Accessibility**: All clickable elements keyboard-navigable; ARIA labels on icons; contrast ratio ≥4.5:1
- **Mobile**: 768px breakpoint — KPI strip becomes 2x2 grid, charts stack, tables become cards
- **Animations**: Page enter (fadeUp), card stagger (kpiStagger), hover (subtle shadow lift)
- **Data refresh**: Auto-refresh every 30s (server-side timestamps)
- **Empty states**: Every list has a polished empty state with illustration + CTA

## Acceptance

- Overview page renders all 4 sections (KPI + charts + fleet + events) without errors
- Proxmox workspace shows all 6 sections (Hosts / VMs / LXC / Storage / Network / Tasks)
- Proxmox host detail has 7 tabs, all functional
- All shared primitives (KpiCard, TimeSeriesChart, StatusPill, DataTable, EmptyState) used in ≥2 places
- `npx tsc --noEmit` exit 0
- `npm run build` exit 0
- Live at https://stackwatch.smarthomelab.fun/overview and /proxmox
- Hard-refresh (Ctrl+Shift+R) renders new design

## Out of scope

- Per-page settings/preferences persistence
- Real-time WebSocket streaming (uses 30s polling instead)
- Custom dashboard builder
- Mobile native app

## Test plan format

Per-phase test plan (after Phase A — overview, Phase B — proxmox):
1. Numbered test cases
2. Expected vs actual
3. Edge cases (empty state, error, mobile)
4. "What to tell me" outcome

Mega-test at end (full Overview + Proxmox A-to-Z) — user's acceptance gate.
