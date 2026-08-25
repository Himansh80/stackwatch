# Tier 14 — Phase 14.1 Specification: VM List Page

## 1. Goal

A Datadog-style VM list page that replaces the Proxmox web UI's QEMU list view.

## 2. Functional Requirements

### 2.1 Host selector (top of page)
- Dropdown listing all registered Proxmox hosts (`/api/v1/proxmox/hosts`)
- Default: first online host
- If no hosts: render empty-state ("Register your first Proxmox host" CTA)
- If host offline: render warning state with reconnect button

### 2.2 KPI strip (top)
- Total VMs
- Running (green)
- Stopped (gray)
- Paused (amber)
- Cluster CPU % (aggregated across nodes)
- Cluster RAM % (aggregated across nodes)

### 2.3 Filter bar
- Search input (filter by name, vmid, IP)
- Status filter chips (All / Running / Stopped / Paused)
- Node filter dropdown (if multi-node cluster)

### 2.4 VM table
Columns:
- Status (colored pill: green=running, gray=stopped, amber=paused)
- VMID (mono)
- Name (mono, bold)
- Node (chip)
- IP Address (mono)
- CPU % (mini bar + value)
- RAM (used / total, e.g. "4.2 GB / 8.0 GB")
- Uptime (relative: "3d 4h", "5m", etc.)
- Actions dropdown: Start, Stop, Reboot, Shutdown, Console, Migrate

Row interactions:
- Hover: subtle background lift
- Click row: navigate to VM detail page (Phase 14.2)
- Click action: confirm dialog for destructive (stop/reboot/shutdown)

### 2.5 Loading + error + empty states
- Loading: skeleton rows (5-7 placeholder)
- Error: red banner with retry button + raw error message
- Empty: hero card "No VMs found" with illustration

### 2.6 Auto-refresh
- Every 5 seconds (configurable)
- Pause when tab is hidden (visibility API)

## 3. Backend dependencies (already exist)

- `GET /api/v1/proxmox/hosts` — list hosts (Tier 1)
- `GET /api/v1/proxmox/hosts/:id/qemu` — list VMs on a host (Tier 1)
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/:action` — VM action (Tier 1)
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/current` — VM live metrics (Tier 1)

## 4. UI Components (planned)

- `ProxmoxPage.tsx` — page shell (≤400 LOC)
- `ProxmoxHostSelector.tsx` — host dropdown (≤150 LOC)
- `ProxmoxKpiStrip.tsx` — KPI cards (≤150 LOC)
- `ProxmoxFilterBar.tsx` — search + chips (≤150 LOC)
- `ProxmoxVmTable.tsx` — table (≤350 LOC)
- `ProxmoxVmRow.tsx` — single row (≤150 LOC)
- `ProxmoxEmptyState.tsx` — empty + error states (≤150 LOC)
- `ProxmoxVmActions.tsx` — action menu (≤150 LOC)

All under 400 LOC. Split more if needed.

## 5. Animation (framer-motion)

- Page enter: stagger children (50ms each)
- Row hover: scale 1.01, bg lift, 150ms ease-out
- KPI count: animate from 0 to value (spring)
- Action button click: scale 0.95 then 1 (150ms)
- Status pill change: color transition 300ms

## 6. Styling (Datadog-style)

- Background: `#0f172a` (slate-900)
- Surface: `#1e293b` (slate-800)
- Border: `#334155` (slate-700)
- Text primary: `#f1f5f9` (slate-100)
- Text muted: `#94a3b8` (slate-400)
- Accent: `#3b82f6` (blue-500)
- Success: `#10b981` (emerald-500)
- Warning: `#f59e0b` (amber-500)
- Danger: `#ef4444` (red-500)
- Font: system stack (`-apple-system, BlinkMacSystemFont, ...`)
- Mono: `JetBrains Mono` for VMIDs and IPs
- Radius: 8px (cards), 6px (inputs), 4px (chips)
- Spacing: 4 / 8 / 12 / 16 / 24 / 32

## 7. Acceptance criteria

1. Page loads at `/app.html/proxmox`
2. KPI strip shows correct counts from live data
3. Filter chips + search filter the table client-side (no extra requests)
4. Start action on a stopped VM shows toast "Starting..." then status flips to running within 5s
5. Stop action shows confirmation dialog before sending
6. Empty state when host has 0 VMs
7. Error state when host unreachable (e.g. SSH down)
8. Auto-refresh works (visible by status pill changing)
9. Mobile-responsive (table → cards on <768px)
10. Dark theme, framer-motion animations, Datadog aesthetic

## 8. Test plan (user-driven)

After implementation:
- Open page, verify KPI strip
- Filter to "Running", verify table filters
- Search by IP, verify filter
- Click Start on a stopped VM, verify it starts
- Click Stop on a running VM, verify confirm dialog + status flips
- Refresh manually, verify status updates
- Empty host test (register host with no VMs)
- Error state test (kill Proxmox API, see error banner)
- Mobile test (resize to 375px, verify table → cards)
