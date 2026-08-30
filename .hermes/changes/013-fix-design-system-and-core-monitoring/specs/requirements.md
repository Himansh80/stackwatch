# Requirements: Fix Design System Violations + Build Core Monitoring Backend

## Functional Requirements

### FR-1: Design System Compliance (CSS)
- **FR-1.1**: All hardcoded colors in `profile.css` replaced with CSS custom properties from `tokens.css`
- **FR-1.2**: All hardcoded colors in `dashboard.css` replaced with CSS custom properties from `tokens.css`
- **FR-1.3**: All hardcoded colors in `shell.css` replaced with CSS custom properties from `tokens.css`
- **FR-1.4**: All spacing values in all CSS files use the design system scale (4, 8, 12, 16, 20, 24, 32, 40, 48, 64, 80, 96, 128)
- **FR-1.5**: All typography sizes use the design system scale (11, 12, 14, 16, 18, 20, 24, 30, 36, 48, 60)
- **FR-1.6**: All border radius values use the design system scale (4, 6, 8, 12, 16)
- **FR-1.7**: All transition durations use design system tokens (--duration-fast, --duration-base, etc.)
- **FR-1.8**: Mobile breakpoints standardized: 640px (mobile), 1024px (tablet), 1280px (desktop)

### FR-2: Core Monitoring Backend Endpoints
- **FR-2.1**: `GET /api/v1/servers` — List all agent-connected servers with pagination, filtering, sorting
  - Query params: `limit`, `offset`, `search`, `status`, `sort`
  - Response: `{ servers: Server[], total: number }`
  - Server fields: id, name, hostname, ip_address, os, status, last_seen_at, cpu_cores, memory_total, created_at
- **FR-2.2**: `GET /api/v1/fleet/summary` — Aggregate fleet metrics
  - Response: `{ total_servers: number, online: number, offline: number, stale: number, avg_cpu_pct: number, avg_mem_pct: number, avg_disk_pct: number }`
- **FR-2.3**: `GET /api/v1/alerts` — List firing alerts with filtering
  - Query params: `limit`, `offset`, `state` (open/acknowledged/resolved), `severity`, `server_id`
  - Response: `{ alerts: Alert[], total: number }`
- **FR-2.4**: `GET /api/v1/fleet/timeseries` — Historical fleet metrics for sparklines
  - Query params: `metric` (cpu/mem/disk), `window` (1h/6h/24h), `interval` (1m/5m/15m)
  - Response: `{ points: { timestamp: string, value: number }[] }`

### FR-3: Dashboard Frontend Integration
- **FR-3.1**: Dashboard shows 8 KPI cards (4 Proxmox + 4 Fleet)
  - Connected hosts (Proxmox), Compute nodes (Proxmox), Running workloads (Proxmox), API health
  - Total servers (Fleet), Online servers (Fleet), Avg CPU (Fleet), Open alerts (Fleet)
- **FR-3.2**: Resource Trend section shows 4 charts
  - CPU utilization (Proxmox), Connected hosts (Proxmox), Running workloads (Proxmox), Fleet CPU (Fleet)
- **FR-3.3**: Live Telemetry shows both Proxmox workloads AND agent server pressure
- **FR-3.4**: Operations table shows both Proxmox hosts AND agent servers
- **FR-3.5**: Empty states with guidance when no data ("Add your first server" with install command)
- **FR-3.6**: Auto-refresh every 30s for all data sources

### FR-4: Navigation
- **FR-4.1**: Sidebar "Monitoring" section with items:
  - Servers → `/servers`
  - Fleet → `/fleet`
  - Alerts → `/alerts`
  - Alert Rules → `/alert-rules`
  - Notifications → `/notifications`
  - Team → `/team`
- **FR-4.2**: Active item highlighted per design system
- **FR-4.3**: Mobile drawer works for new items

## Non-Functional Requirements

### NFR-1: Performance
- Dashboard initial load < 2s
- API responses < 200ms (p95)
- Fleet summary < 100ms

### NFR-2: Design System
- Zero hardcoded values in component CSS after fix
- All components consume tokens via `var(--*)`
- WCAG AA contrast (4.5:1) for body text

### NFR-3: Responsive
- Works at 320px (mobile), 768px (tablet), 1024px (desktop), 1920px (large desktop)
- Sidebar collapses to drawer at < 768px
- KPI grid: 4-col → 2-col → 1-col

### NFR-4: Accessibility
- Focus-visible on all interactive elements
- Semantic HTML structure
- ARIA labels where needed
- Reduced motion support

## Acceptance Scenarios

### AS-1: Designer reviews CSS
**Given** the CSS files are modified
**When** designer runs a token audit script
**Then** zero hardcoded hex/rgba values found in component CSS

### AS-2: User loads Dashboard with no servers
**Given** user is logged in, no agent servers connected
**When** user navigates to `/dashboard`
**Then** page shows Proxmox data + empty states for Fleet with "Add your first server" CTA

### AS-3: User loads Dashboard with 5 servers
**Given** 5 agent servers reporting (3 online, 2 offline)
**When** user navigates to `/dashboard`
**Then** KPI cards show: Total=5, Online=3, Offline=2, Avg CPU=42%
**And** Resource trend shows Fleet CPU chart with data
**And** Operations table lists all 5 servers with status badges

### AS-4: User navigates to Servers page
**Given** user is on Dashboard
**When** user clicks "Servers" in sidebar
**Then** navigates to `/servers` with server list, search, filter, pagination

### AS-5: Mobile user accesses Dashboard
**Given** viewport is 375px wide
**When** user loads `/dashboard`
**Then** sidebar is drawer, KPI cards stack 1-col, charts full-width, all touch targets ≥ 44px

### AS-6: Rate limited user
**Given** user hits rate limit on dashboard API calls
**When** API returns 429
**Then** Dashboard shows error bar with retry button, preserves last good data