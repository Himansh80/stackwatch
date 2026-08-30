# Tasks: Fix Design System Violations + Build Core Monitoring Backend

## Phase 1: CSS Design System Compliance

### 1.1 Fix profile.css (100+ violations)
- [ ] 1.1.1 Audit all hardcoded colors in profile.css — map to tokens
- [ ] 1.1.2 Replace all hardcoded hex/rgba colors with `var(--color-*)` tokens
- [ ] 1.1.3 Replace all non-scale spacing with `var(--space-*)` tokens
- [ ] 1.1.4 Replace all non-scale font sizes with `var(--text-*)` tokens
- [ ] 1.1.5 Replace all non-scale border-radius with `var(--radius-*)` tokens
- [ ] 1.1.6 Replace all hardcoded transitions with `var(--duration-*)` + `var(--ease-*)`
- [ ] 1.1.7 Standardize mobile breakpoints to 640px / 1024px / 1280px
- [ ] 1.1.8 Verify: `grep -E '#[0-9a-f]{6}|rgba?\(|[0-9]+px(?![;}])' profile.css` returns only tokens

### 1.2 Fix dashboard.css
- [ ] 1.2.1 Audit hardcoded colors in dashboard.css
- [ ] 1.2.2 Replace with tokens (focus on legacy `.dash-*` classes)
- [ ] 1.2.3 Verify no hardcoded values remain

### 1.3 Fix shell.css
- [ ] 1.3.1 Audit topbar clock, command palette, weather widget styles
- [ ] 1.3.2 Replace with tokens
- [ ] 1.3.3 Verify no hardcoded values remain

### 1.4 Design System Verification
- [ ] 1.4.1 Create token audit script: `scripts/audit-tokens.sh`
- [ ] 1.4.2 Run audit on all CSS files — must pass with 0 violations
- [ ] 1.4.3 Visual regression: load all pages at 320/768/1024/1920px

## Phase 2: Core Monitoring Backend

### 2.1 Database Schema / Indexes
- [ ] 2.1.1 Add index on `servers.tenant_id` (if missing)
- [ ] 2.1.2 Add index on `servers.status` (if missing)
- [ ] 2.1.3 Add index on `servers.last_seen_at` (if missing)
- [ ] 2.1.4 Add index on `metric_points.server_id + ts` for fleet queries
- [ ] 2.1.5 Add index on `alerts.tenant_id + state` (if missing)

### 2.2 Create Fleet Service
- [ ] 2.2.1 Create `internal/service/fleet_service.go` with FleetSummary + Timeseries methods
- [ ] 2.2.2 Implement `GetFleetSummary` — aggregate from metric_points + servers
- [ ] 2.2.3 Implement `GetFleetTimeseries` — bucketed queries for sparklines
- [ ] 2.2.4 Add unit tests for fleet_service.go

### 2.3 Extend Server Service
- [ ] 2.3.1 Add `ListServers` method to `internal/service/server_service.go`
- [ ] 2.3.2 Implement pagination, filtering, sorting
- [ ] 2.3.3 Add unit tests for ListServers

### 2.4 Create Monitoring Handlers
- [ ] 2.4.1 Create `cmd/api-gateway/handlers_monitoring.go`
- [ ] 2.4.2 Implement `listServers` handler
- [ ] 2.4.3 Implement `fleetSummary` handler
- [ ] 2.4.4 Implement `listAlerts` handler (reuse alert_service)
- [ ] 2.4.5 Implement `fleetTimeseries` handler

### 2.5 Register Routes
- [ ] 2.5.1 Add 4 routes to `cmd/api-gateway/routes_protected.go`
- [ ] 2.5.2 Register FleetService and ensure ServerService in `internal/service/service.go`
- [ ] 2.5.3 Build: `go build ./cmd/api-gateway`
- [ ] 2.5.4 Run tests: `go test ./internal/service/...`

### 2.6 Deploy & Verify Backend
- [ ] 2.6.1 Deploy binary to .115
- [ ] 2.6.2 Test endpoints with curl + JWT:
  - [ ] `GET /api/v1/servers` → 200, correct shape
  - [ ] `GET /api/v1/fleet/summary` → 200, correct shape
  - [ ] `GET /api/v1/alerts` → 200, correct shape
  - [ ] `GET /api/v1/fleet/timeseries?metric=cpu&window=1h` → 200, correct shape
- [ ] 2.6.3 Verify 4-pass test (run verifier 4x)

## Phase 3: Dashboard Frontend Integration

### 3.1 Update Dashboard.tsx
- [ ] 3.1.1 Add fleet state: `fleetSummary`, `fleetHistory`, `servers`
- [ ] 3.1.2 Add API calls in `loadDashboard()` for 4 new endpoints
- [ ] 3.1.3 Add error handling for fleet endpoints (don't break Proxmox)
- [ ] 3.1.4 Add 4 fleet KPI cards to KPI strip (Total, Online, Avg CPU, Open Alerts)
- [ ] 3.1.5 Add 4th chart (Fleet CPU) to trend grid
- [ ] 3.1.6 Merge Proxmox hosts + agent servers in Operations table
- [ ] 3.1.7 Update Live Telemetry to show both Proxmox workloads + agent pressure
- [ ] 3.1.8 Add empty states for fleet sections when no servers

### 3.2 Create FleetKpiStrip Component (optional)
- [ ] 3.2.1 Create `web/src/components/dashboard/FleetKpiStrip.tsx`
- [ ] 3.2.2 Use KpiCard with fleet data
- [ ] 3.2.3 Add click handlers to navigate to `/servers`, `/alerts`

### 3.3 Verify Frontend
- [ ] 3.3.1 Build: `npm run type-check && npm run lint && npm run build`
- [ ] 3.3.2 Deploy to .115: `scp web/dist/* root@.115:/opt/stackwatch/web/public/`
- [ ] 3.3.3 Hard refresh browser (Ctrl+Shift+R)
- [ ] 3.3.4 Test `/dashboard` with 0 servers → shows empty states
- [ ] 3.3.5 Test `/dashboard` with servers → KPIs populate
- [ ] 3.3.6 Check console for 0 errors

## Phase 4: Navigation & Polish

### 4.1 Add Navigation Items
- [ ] 4.1.1 Add "Monitoring" section to `web/src/components/sidebar/nav-config.ts`
- [ ] 4.1.2 Verify Sidebar.tsx renders new section correctly
- [ ] 4.1.3 Test mobile drawer with new items

### 4.2 Polish & Empty States
- [ ] 4.2.1 Ensure empty states show "Add your first server" with install command
- [ ] 4.2.2 Verify all KPI cards use `tabular-nums` for values
- [ ] 4.2.3 Verify hover/focus/active states on all new interactive elements
- [ ] 4.2.4 Test at 320px, 375px, 768px, 1024px, 1280px, 1920px

### 4.3 Final Verification
- [ ] 4.3.1 Run design system token audit on ALL CSS
- [ ] 4.3.2 Run full dashboard flow: login → dashboard → servers → alerts → fleet
- [ ] 4.3.3 Verify 4-pass backend test still passes
- [ ] 4.3.4 Document any known issues

## Done Criteria
- [ ] All 4 CSS files pass token audit (0 hardcoded values)
- [ ] 4 new backend endpoints return 200 with correct shapes
- [ ] Dashboard shows Proxmox + Fleet data
- [ ] Navigation has Monitoring section with 6 items
- [ ] Empty states guide user to add servers
- [ ] Responsive at all breakpoints
- [ ] 0 console errors
- [ ] 4-pass backend verification passes
- [ ] All Go tests pass
- [ ] All TypeScript checks pass