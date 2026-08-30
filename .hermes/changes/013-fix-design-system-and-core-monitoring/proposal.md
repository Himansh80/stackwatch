# Proposal: Fix Design System Violations + Build Core Monitoring Backend

## Why

**Problem 1: Design System Non-Compliance**
- 100+ hardcoded hex/rgba values in `profile.css` violating the token system
- Non-scale spacing values (14px, 22px, 28px, etc.) throughout CSS files
- Mixed typography scales (13px, 38px, etc. not in token scale)
- Components don't follow the design system patterns (KPI cards, buttons, inputs)
- Dashboard looks "not good and similar" — lacks Datadog/Linear/Stripe polish

**Problem 2: Core Monitoring Missing from Dashboard**
- Dashboard only shows Proxmox data — no agent servers, no fleet metrics, no alerts
- Missing backend endpoints: `/api/v1/servers`, `/api/v1/fleet/summary`, `/api/v1/alerts`, `/api/v1/fleet/timeseries`
- User perceives "blank screen" because there's no meaningful data

**Problem 3: Navigation Gaps**
- Sidebar missing core monitoring items: Servers, Fleet, Alerts, Alert Rules, Notifications, Team

## What Changes

### Phase 1: Design System Compliance (CSS)
1. **Fix `profile.css`** — Replace ALL hardcoded colors with tokens, ALL spacing with scale values
2. **Fix `dashboard.css`** — Replace legacy hardcoded colors, standardize spacing
3. **Fix `shell.css`** — Tokenize topbar clock, command palette, weather widget
4. **Standardize mobile breakpoints** across all CSS files (640px, 1024px, 1280px)

### Phase 2: Core Monitoring Backend (Go)
1. **Add server listing endpoint** — `GET /api/v1/servers` with pagination/filtering
2. **Add fleet summary endpoint** — `GET /api/v1/fleet/summary` (aggregate metrics)
3. **Add alerts endpoint** — `GET /api/v1/alerts` with state filtering
4. **Add fleet timeseries endpoint** — `GET /api/v1/fleet/timeseries` for sparklines

### Phase 3: Dashboard Integration (Frontend)
1. **Update `Dashboard.tsx`** — Fetch and display agent server data alongside Proxmox
2. **Add new KPI cards** — Servers (up/down), Fleet CPU/Memory/Disk
3. **Add resource trend charts** — Fleet CPU, Memory, Disk
4. **Add operations table** — Agent servers with status, last seen, actions

### Phase 4: Navigation & Polish
1. **Add monitoring nav items** to sidebar (Servers, Fleet, Alerts, Alert Rules, Notifications, Team)
2. **Add empty states with guidance** — "Add your first server" with install command
3. **Verify design system compliance** across all pages

## Impact

| Area | Files Affected | Breaking Changes | Migration Needed |
|------|----------------|------------------|------------------|
| CSS/Design | `profile.css`, `dashboard.css`, `shell.css` | No | No |
| Backend API | `handlers_v2.go`, `routes_protected.go`, `server_service.go`, `fleet_service.go` | No (new endpoints) | No |
| Frontend Pages | `Dashboard.tsx`, `nav-config.ts`, `Sidebar.tsx` | No | No |
| Components | `KpiCard.tsx`, `SkeletonCard.tsx`, new components | No | No |

## Architecture Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Use existing `server_service.go` for `/servers` | Already has server CRUD logic | Need to add list with pagination |
| Create new `fleet_service.go` for aggregation | Separation of concerns, reusable | New service file |
| Extend `Dashboard.tsx` not replace | Preserve Proxmox integration | More complex component |
| Reuse `KpiCard` for fleet metrics | Consistency, DRY | May need sparkline props |
| Add nav items under "Monitoring" section | Clear categorization | May need section label |

## Risks

- **CSS changes may break visual layout** — Mitigation: Test at all breakpoints, use design system tokens
- **Backend endpoints need DB queries** — Mitigation: Use existing repository patterns, add indexes if needed
- **Dashboard becomes complex** — Mitigation: Extract sub-components, keep under 400 LOC
- **Mobile responsiveness** — Mitigation: Test at 320px, 768px, 1024px, 1920px