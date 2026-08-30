# Design: Fix Design System Violations + Build Core Monitoring Backend

## Approach

This change is split into 4 phases that can be executed sequentially:

1. **Phase 1: CSS Design System Compliance** — Fix all hardcoded values in CSS files
2. **Phase 2: Core Monitoring Backend** — Add missing Go endpoints
3. **Phase 3: Dashboard Frontend Integration** — Wire Dashboard.tsx to new endpoints
4. **Phase 4: Navigation & Polish** — Add nav items, empty states, responsive testing

## Architecture Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| **Phase 1 first** | CSS fixes are independent, unblock visual consistency for new components | Frontend may look broken until Phase 1 complete |
| **Extend `server_service.go`** | Already has server CRUD, repo pattern established | Need to add pagination/filtering logic |
| **New `fleet_service.go`** | Aggregation logic is distinct from individual server ops | New file, but cleaner separation |
| **Reuse `KpiCard` component** | Already designed per spec, supports sparklines | Need to ensure all props work for fleet data |
| **Add "Monitoring" nav section** | Clear categorization, follows Datadog pattern | Need to update Sidebar.tsx section rendering |

## Implementation Details

### Phase 1: CSS Design System Compliance

#### Files to Modify:
1. `web/src/styles/profile.css` — **Priority 1** (100+ violations)
2. `web/src/styles/dashboard.css` — **Priority 2** (legacy hardcoded values)
3. `web/src/styles/shell.css` — **Priority 3** (topbar, command palette)

#### Token Mapping Strategy:
| Hardcoded Pattern | Token Replacement |
|-------------------|-------------------|
| `#0e1726`, `#0a101b` | `var(--color-surface)`, `var(--color-bg)` |
| `#fcd34d`, `#f59e0b` | `var(--color-warning)`, `var(--color-warning-hover)` |
| `#5ee6ad`, `#10b981` | `var(--color-success)`, `var(--color-success-hover)` |
| `#ef4444`, `#fca5a5` | `var(--color-error)`, `var(--color-error-hover)` |
| `#67e8f9`, `#22d3ee` | `var(--color-primary-hover)`, `var(--color-primary)` |
| `#8294ae`, `#647994` | `var(--color-text-muted)`, `var(--color-text-subtle)` |
| `#f3f7fd`, `#dbe7f4` | `var(--color-text)`, `var(--color-text)` |
| `rgba(245,158,11,0.14)` | `var(--color-warning-muted)` |
| `rgba(16,185,129,0.12)` | `var(--color-success-muted)` |
| `rgba(239,68,68,0.12)` | `var(--color-error-muted)` |
| `32px`, `36px`, `28px` | `var(--space-8)`, `var(--space-9)` → use `var(--space-8)` or `var(--space-10)` |
| `14px`, `22px`, `18px` | `var(--space-3)` (12px), `var(--space-4)` (16px), `var(--space-5)` (20px) |
| `font-size: 38px` | `var(--text-4xl)` (36px) or `var(--text-5xl)` (48px) |
| `font-size: 13px` | `var(--text-sm)` (12px) or `var(--text-base)` (14px) |
| `padding: 14px 22px` | `var(--space-3) var(--space-4)` |
| `gap: 8px` | `var(--space-2)` |
| `border-radius: 10px` | `var(--radius-lg)` (8px) or `var(--radius-xl)` (12px) |

#### Breakpoint Standardization:
```css
/* Old: scattered values */
/* New: standardized */
@media (max-width: 640px)   { /* mobile */ }
@media (max-width: 1024px)  { /* tablet */ }
@media (max-width: 1280px)  { /* desktop */ }
```

### Phase 2: Core Monitoring Backend

#### New Service: `internal/service/fleet_service.go`
```go
type FleetService struct {
    db repository.DB
}

func (s *FleetService) GetFleetSummary(ctx context.Context, tenantID uuid.UUID) (*FleetSummary, error)
func (s *FleetService) GetFleetTimeseries(ctx context.Context, tenantID uuid.UUID, metric string, window string) ([]TimeseriesPoint, error)
```

#### Extended Service: `internal/service/server_service.go`
```go
func (s *ServerService) ListServers(ctx context.Context, tenantID uuid.UUID, params ListParams) ([]Server, int, error)
func (s *ServerService) GetServer(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*Server, error)
```

#### Handlers: `cmd/api-gateway/handlers_monitoring.go` (new file)
```go
func (s *Server) listServers(c *gin.Context)
func (s *Server) fleetSummary(c *gin.Context)
func (s *Server) listAlerts(c *gin.Context)
func (s *Server) fleetTimeseries(c *gin.Context)
```

#### Routes: `cmd/api-gateway/routes_protected.go`
```go
protected.GET("/servers", s.listServers)
protected.GET("/fleet/summary", s.fleetSummary)
protected.GET("/alerts", s.listAlerts)
protected.GET("/fleet/timeseries", s.fleetTimeseries)
```

#### Database Queries:
- `servers` table: already exists, add indexes on `tenant_id`, `status`, `last_seen_at`
- `metric_points` table: aggregate for fleet summary
- `alerts` table: already exists, filter by tenant + state

### Phase 3: Dashboard Frontend Integration

#### Modified: `web/src/pages/Dashboard.tsx`
```typescript
// Add new state for fleet data
const [fleetSummary, setFleetSummary] = useState<FleetSummary | null>(null);
const [fleetHistory, setFleetHistory] = useState({ cpu: [], mem: [], disk: [] });
const [servers, setServers] = useState<Server[]>([]);

// Add fetch calls in loadDashboard()
const [serversRes, fleetRes, alertsRes, fleetTsRes] = await Promise.all([
  api('GET', '/api/v1/servers').catch(() => null),
  api('GET', '/api/v1/fleet/summary').catch(() => null),
  api('GET', '/api/v1/alerts?state=open').catch(() => null),
  api('GET', '/api/v1/fleet/timeseries?metric=cpu&window=1h').catch(() => null),
]);

// Add 4 new KpiCards for fleet
// Add 4th chart in trend grid for Fleet CPU
// Add combined operations table (Proxmox hosts + agent servers)
```

#### New Components (if needed):
- `web/src/components/dashboard/FleetKpiStrip.tsx` — 4 fleet KPI cards
- `web/src/components/dashboard/ServerList.tsx` — Combined server table

### Phase 4: Navigation & Polish

#### Modified: `web/src/components/sidebar/nav-config.ts`
```typescript
// Add "Monitoring" section
{ label: 'Monitoring', items: [
  { label: 'Servers', href: '/servers', icon: 'server' },
  { label: 'Fleet', href: '/fleet', icon: 'activity' },
  { label: 'Alerts', href: '/alerts', icon: 'bell' },
  { label: 'Alert Rules', href: '/alert-rules', icon: 'shield' },
  { label: 'Notifications', href: '/notifications', icon: 'mail' },
  { label: 'Team', href: '/team', icon: 'users' },
]}
```

#### Modified: `web/src/components/sidebar/Sidebar.tsx`
- Ensure section rendering handles new section
- Mobile drawer works for new items

## Component Patterns to Follow

### KPI Card (from design system)
```tsx
<KpiCard
  label="Total servers"
  value={fleetSummary?.total_servers || 0}
  delta={`${fleetSummary?.online || 0} online`}
  accent="cyan"
  sparkline={fleetHistory.hosts}
  onClick={() => navigate('/servers')}
/>
```

### Server Row (Datadog-style)
```tsx
<div className="dash-host-row" data-tone={server.status === 'online' ? 'good' : server.status === 'stale' ? 'warn' : 'bad'}>
  <StatusPill status={server.status} size="sm" />
  <strong>{server.name}</strong>
  <span>{server.ip_address}</span>
  <span>{formatRelative(server.last_seen_at)}</span>
  <button className="dash-host-kebab">⋮</button>
</div>
```

## Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| CSS changes break layout | Medium | High | Test at all breakpoints, use visual regression |
| Fleet summary query slow | Low | Medium | Add DB indexes, cache for 30s |
| Dashboard over 400 LOC | Medium | Low | Extract sub-components early |
| Mobile nav broken | Low | High | Test drawer on 320px, 375px viewports |
| Rate limit on dashboard APIs | Medium | Medium | Exempt dashboard from rate limit or increase limit |

## Testing Strategy

### Unit Tests (Go)
- `fleet_service_test.go` — summary aggregation, timeseries queries
- `server_service_test.go` — list with filters, pagination

### Integration Tests
- `curl` each new endpoint with auth token
- Verify response shapes match spec

### Frontend Verification
- Load `/dashboard` with 0 servers → empty states show
- Load `/dashboard` with 5 servers → all KPIs populate
- Navigate to `/servers` → list renders, search works
- Mobile viewport (375px) → drawer opens, cards stack
- Check console for 0 errors

## File Structure After Changes

```
web/src/
├── styles/
│   ├── tokens.css          (unchanged - source of truth)
│   ├── DESIGN-SYSTEM.md    (unchanged - documentation)
│   ├── profile.css         (FIXED - all tokens)
│   ├── dashboard.css       (FIXED - all tokens)
│   └── shell.css           (FIXED - all tokens)
├── pages/
│   └── Dashboard.tsx       (EXTENDED - fleet data)
├── components/
│   ├── sidebar/
│   │   ├── nav-config.ts   (EXTENDED - Monitoring section)
│   │   └── Sidebar.tsx     (UNCHANGED - handles new section)
│   ├── dashboard/
│   │   ├── KpiCard.tsx     (UNCHANGED - works for fleet)
│   │   ├── TrendChart.tsx  (UNCHANGED)
│   │   ├── HostList.tsx    (UNCHANGED)
│   │   └── FleetKpiStrip.tsx (NEW - 4 fleet KPIs)
│   └── shared/
│       ├── StatusPill.tsx  (UNCHANGED)
│       └── EmptyState.tsx  (UNCHANGED)
└── lib/
    ├── api.ts              (UNCHANGED)
    └── proxmox.ts          (UNCHANGED)

cmd/api-gateway/
├── handlers_monitoring.go  (NEW - 4 endpoints)
├── handlers_v2.go          (UNCHANGED)
├── routes_protected.go     (EXTENDED - 4 routes)
└── main.go                 (UNCHANGED)

internal/service/
├── server_service.go       (EXTENDED - ListServers)
├── fleet_service.go        (NEW - summary, timeseries)
├── alert_service.go        (UNCHANGED - has ListAlerts)
└── service.go              (EXTENDED - register new services)
```

## Rollback Plan

If issues arise:
1. CSS: Revert `profile.css`, `dashboard.css`, `shell.css` individually
2. Backend: Comment out new routes in `routes_protected.go`
3. Frontend: Revert `Dashboard.tsx` and `nav-config.ts`
4. Services: Don't register `fleet_service` in `service.go`

All changes are additive (new endpoints, new nav items, extended components) — no existing functionality is removed.