# Feature Specification: Tier 7 — Datadog Full Platform Parity (Phase 1 of 11)

**Feature Branch**: `003-tier7-datadog-parity`
**Created**: 2026-08-24
**Status**: Draft
**Source change folder**: `.hermes/changes/003-tier7-datadog-parity/`
**Master plan reference**: `MASTER_BUILD_PLAN.md` §"Tier 7 — Datadog Full Platform (4-5 sessions)"

**Input**: User requirement — StackWatch must be feature-equivalent to Datadog + Grafana + Better Stack + New Relic. This change ships the first 3 of 11 Tier 7 subtiers as Phase 1 of the multi-session Tier 7 work.

## Goal

Bring StackWatch to Datadog-grade observability feature parity. Phase 1 ships 3 of 11 Tier 7 subtiers, each independently verifiable:

1. **7.1 APM (D2)** — Distributed Tracing: trace model, services, flame graphs, deployments. Foundation for everything else (DB monitoring, Service Mgmt, CI/CD Vis, Notebook all depend on a trace model).
2. **7.2 Log Management Full (D3)** — Real log pipeline: retention policies, archives, rehydration, log monitors, log patterns.
3. **7.3 RUM Full (D4)** — Real User Monitoring: web vitals, resource timings, interactions, long tasks, error groups, session replay.

## Out of scope (will be future speckit changes 004-007)

- 7.4 Synthetics Full (D5) — advanced multi-step browser tests + SLA
- 7.5 Security (D6) — threat detection, audit trails, compliance
- 7.6 CSPM (D7) — cloud security posture management
- 7.7 CI/CD Visibility (D8)
- 7.8 DB Monitoring (D9)
- 7.9 Service Management (D10) — incidents, war rooms, postmortems, tasks
- 7.10 Notebook (D11)
- 7.11 Team/Collab (D12) — shared dashboards, mentions, timeline comments

## User Scenarios & Testing *(mandatory)*

### User Story 1 — APM service map (Priority: P1)

A user opens StackWatch → APM → Service Map. The map shows service nodes (one per registered service) with edges representing trace flow. Hovering a node shows the service name, request rate, error rate, p95 latency. Clicking a node opens its flame graph.

**Why P1**: APM is the foundation. Without a trace model, every other observability feature (DB monitoring, CI/CD, Service Mgmt, Notebook) is crippled.

**Independent test**: Register two services via `POST /api/v1/apm/services`. Send 5 traces from service A→B and 2 traces from service B→C. Open APM service map — verify 3 nodes + 2 edges render.

**Acceptance scenarios**:

1. **Given** a user has registered at least one service, **when** they open APM, **then** the service map renders all services as nodes with edges showing trace flow.
2. **Given** the user clicks a service node, **when** the flame graph loads, **then** it shows the slowest span with its breakdown.
3. **Given** the user opens APM Deployments, **when** they create a deployment, **then** the deployment appears in the deployments list.

### User Story 2 — Log Management (Priority: P1)

A user opens Logs. They see a search bar with time-range picker, source filter, level filter, and a virtualized list of log entries with live tail. They can save searches, create monitors, define retention policies, and trigger archive rehydration.

**Why P1**: Log Mgmt is core observability. The existing Tier 6 v1 has basic `logs/query` — Tier 7.2 adds the full Datadog-grade management surface.

**Independent test**: Create a retention policy with 30-day hot + 365-day cold storage. Create a log monitor on `level=error AND service=api`. Verify the monitor fires when a matching log arrives.

**Acceptance scenarios**:

1. **Given** the user has logs in the system, **when** they search with `service=api level=error`, **then** matching log entries appear within 1 second.
2. **Given** the user has log archives configured, **when** they trigger rehydration for the last 7 days, **then** a rehydration job is queued and its status is queryable.
3. **Given** the user creates a log monitor on `count(level=error) > 10 over 5m`, **when** the threshold is breached, **then** an alert fires.

### User Story 3 — RUM full (Priority: P1)

A user installs the RUM snippet on their website. Real users send web vitals (LCP, FID, CLS), resource timings, JS errors, slow network requests, and interaction events. The user opens RUM → Sessions, sees a list of real sessions with replay, and can drill into any session's resource waterfall + error groups.

**Why P1**: RUM completes the front-end observability story. Tier 6 v4 added the basic RUM collector (JS, MyHumana). Tier 7.3 adds the full Datadog RUM surface (web vitals, errors, session replay).

**Independent test**: Send a batch of web vitals + errors via `POST /rum/web-vitals` + `POST /rum/errors`. Open RUM page. Verify the data appears in the resource waterfall + error groups table.

**Acceptance scenarios**:

1. **Given** the RUM snippet is installed, **when** a real user loads it, **then** web vitals (LCP, FID, CLS, FCP, TTFB) are sent to `/api/v1/rum/web-vitals` and persisted.
2. **Given** a JS error occurs in the browser, **when** the RUM collector catches it, **then** the error appears in `/api/v1/rum/error-groups` with stack trace + occurrence count.
3. **Given** the user has sessions with RUM data, **when** they open a session, **then** the resource waterfall shows all network requests + JS errors in chronological order.

## Design system additions (this change)

### New design tokens
No new CSS tokens required — all reuse existing `--accent`, `--green`, `--amber`, `--red`, `--indigo`, `--violet`, `--shadow-sm`, `--motion-page`.

### New motion variants
No new motion variants — reuse `pageEnter`, `kpiEnter`, `kpiStagger`, `paletteEnter`, `statusPulse`, `sparklineDraw`, `shimmer`.

### New shared components

- `web/src/components/shared/TraceSummary.tsx` — service map node header
- `web/src/components/shared/FlameGraph.tsx` — flame graph SVG renderer
- `web/src/components/shared/LogEntry.tsx` — log entry row (Datadog-style)
- `web/src/components/shared/LogSearchBar.tsx` — search + filters + time-range
- `web/src/components/shared/ResourceWaterfall.tsx` — RUM waterfall
- `web/src/components/shared/ErrorGroupCard.tsx` — error group with stack

### New pages (frontend)

- `web/src/pages/ApmPage.tsx` — APM overview (services, service map, deployments)
- `web/src/pages/ApmServicePage.tsx` — service detail (flame graph, traces)
- `web/src/pages/LogsFullPage.tsx` — log management (search, retention, archives, monitors)
- `web/src/pages/RumFullPage.tsx` — RUM overview (sessions, web vitals, errors)
- `web/src/pages/RumSessionPage.tsx` — session detail (replay, waterfall)

## Database additions (migrations)

### Migration 030: APM tables
```sql
-- services: registered APM services
CREATE TABLE apm_services (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  name text NOT NULL,
  language text,
  framework text,
  environment text,
  created_at timestamptz DEFAULT now(),
  UNIQUE (tenant_id, name)
);

-- traces: incoming trace summaries (one row per root span)
CREATE TABLE apm_traces (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  service_id uuid NOT NULL REFERENCES apm_services(id),
  trace_id text NOT NULL,
  root_span_id text NOT NULL,
  duration_us BIGINT NOT NULL,
  status text NOT NULL DEFAULT 'ok',
  started_at timestamptz NOT NULL,
  ingested_at timestamptz DEFAULT now()
);
CREATE INDEX ON apm_traces (tenant_id, started_at DESC);
CREATE INDEX ON apm_traces (tenant_id, service_id, started_at DESC);

-- spans: individual spans (flame graph source)
CREATE TABLE apm_spans (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  trace_id text NOT NULL,
  span_id text NOT NULL,
  parent_span_id text,
  service_id uuid NOT NULL REFERENCES apm_services(id),
  name text NOT NULL,
  duration_us BIGINT NOT NULL,
  status text NOT NULL DEFAULT 'ok',
  started_at timestamptz NOT NULL,
  attributes jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX ON apm_spans (tenant_id, trace_id);
CREATE INDEX ON apm_spans (tenant_id, service_id, started_at DESC);

-- deployments: deployment markers (linked to services)
CREATE TABLE apm_deployments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  service_id uuid NOT NULL REFERENCES apm_services(id),
  environment text NOT NULL DEFAULT 'production',
  version text NOT NULL,
  commit_sha text,
  deployed_at timestamptz DEFAULT now(),
  rolled_back boolean DEFAULT false
);
CREATE INDEX ON apm_deployments (tenant_id, service_id, deployed_at DESC);
```

### Migration 031: Log Management tables
```sql
-- log_monitors: alert rules on log queries
CREATE TABLE log_monitors (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  name text NOT NULL,
  query text NOT NULL,
  threshold_count int NOT NULL DEFAULT 1,
  threshold_window_seconds int NOT NULL DEFAULT 300,
  severity text NOT NULL DEFAULT 'warn',
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz DEFAULT now()
);

-- log_archives: archive destinations
CREATE TABLE log_archives (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  name text NOT NULL,
  destination_type text NOT NULL,  -- 's3' | 'gcs' | 'azure' | 'local'
  destination_config jsonb NOT NULL DEFAULT '{}'::jsonb,
  enabled boolean NOT NULL DEFAULT true,
  created_at timestamptz DEFAULT now()
);

-- log_rehydrations: rehydration jobs
CREATE TABLE log_rehydrations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  archive_id uuid NOT NULL REFERENCES log_archives(id),
  start_time timestamptz NOT NULL,
  end_time timestamptz NOT NULL,
  status text NOT NULL DEFAULT 'queued',  -- 'queued' | 'running' | 'completed' | 'failed'
  progress_pct int NOT NULL DEFAULT 0,
  created_at timestamptz DEFAULT now()
);

-- log_retention_policies: hot/cold storage
CREATE TABLE log_retention_policies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  service text NOT NULL,
  hot_days int NOT NULL DEFAULT 7,
  cold_days int NOT NULL DEFAULT 90,
  enabled boolean NOT NULL DEFAULT true,
  UNIQUE (tenant_id, service)
);

-- log_patterns: detected log patterns
CREATE TABLE log_patterns (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  service text NOT NULL,
  pattern text NOT NULL,
  sample_count int NOT NULL DEFAULT 1,
  last_seen_at timestamptz DEFAULT now()
);
```

### Migration 032: RUM tables
```sql
-- rum_sessions: per-page-view or per-tab sessions
CREATE TABLE rum_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  user_id text,
  browser text,
  os text,
  device text,
  country text,
  url text NOT NULL,
  started_at timestamptz NOT NULL,
  duration_ms BIGINT,
  page_views int DEFAULT 0,
  errors int DEFAULT 0,
  attributes jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX ON rum_sessions (tenant_id, started_at DESC);

-- rum_web_vitals: per-page-load LCP/FID/CLS/FCP/TTFB
CREATE TABLE rum_web_vitals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  url text NOT NULL,
  metric text NOT NULL,  -- 'lcp' | 'fid' | 'cls' | 'fcp' | 'ttfb'
  value double precision NOT NULL,
  rating text,  -- 'good' | 'needs-improvement' | 'poor'
  ts timestamptz DEFAULT now()
);
CREATE INDEX ON rum_web_vitals (tenant_id, ts DESC);

-- rum_resources: network requests
CREATE TABLE rum_resources (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  url text NOT NULL,
  resource_type text,
  duration_ms BIGINT NOT NULL,
  size_bytes BIGINT,
  status int,
  ts timestamptz DEFAULT now()
);
CREATE INDEX ON rum_resources (tenant_id, ts DESC);

-- rum_interactions: clicks/inputs/scrolls
CREATE TABLE rum_interactions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  action text NOT NULL,
  target text,
  duration_ms BIGINT,
  ts timestamptz DEFAULT now()
);

-- rum_long_tasks: tasks > 50ms
CREATE TABLE rum_long_tasks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  duration_ms BIGINT NOT NULL,
  ts timestamptz DEFAULT now()
);

-- rum_error_groups: unique errors with stack traces
CREATE TABLE rum_error_groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  fingerprint text NOT NULL,
  message text NOT NULL,
  stack_trace text,
  service text,
  source text,
  occurrence_count int NOT NULL DEFAULT 1,
  first_seen_at timestamptz DEFAULT now(),
  last_seen_at timestamptz DEFAULT now(),
  UNIQUE (tenant_id, fingerprint)
);
```

## Backend API additions

### APM endpoints (handlers_apm.go)
- `POST /api/v1/apm/services` — register a service
- `GET /api/v1/apm/services` — list services
- `GET /api/v1/apm/services/:id` — service detail (request rate, error rate, p95)
- `GET /api/v1/apm/services/:id/flame-graph` — flame graph for a time window
- `POST /api/v1/apm/traces` — ingest a trace (OTLP-compatible)
- `GET /api/v1/apm/traces/:trace_id` — get trace details
- `POST /api/v1/apm/deployments` — record a deployment
- `GET /api/v1/apm/deployments` — list deployments (filter by service)
- `GET /api/v1/apm/service-map` — graph data (nodes + edges)

### Logs Full endpoints (handlers_logs_full.go)
- `GET /api/v1/logs/search` — search logs with filters (already exists as /logs/query, alias to /logs/search)
- `POST /api/v1/logs/monitors` — create log monitor
- `GET /api/v1/logs/monitors` — list log monitors
- `GET /api/v1/logs/patterns` — detected log patterns
- `POST /api/v1/logs/archives` — create archive
- `GET /api/v1/logs/archives` — list archives
- `POST /api/v1/logs/archives/:id/rehydrate` — trigger rehydration
- `GET /api/v1/logs/rehydrations` — list rehydrations
- `POST /api/v1/logs/retention` — set retention policy
- `GET /api/v1/logs/retention` — get retention policies

### RUM Full endpoints (handlers_rum_full.go)
- `POST /api/v1/rum/web-vitals` — ingest web vital
- `POST /api/v1/rum/resource-timings` — ingest resource timing
- `POST /api/v1/rum/interactions` — ingest interaction
- `POST /api/v1/rum/long-tasks` — ingest long task
- `POST /api/v1/rum/errors` — ingest error
- `POST /api/v1/rum/heatmap` — ingest click heatmap event
- `GET /api/v1/rum/sessions` — list sessions
- `GET /api/v1/rum/sessions/:id` — session detail
- `GET /api/v1/rum/sessions/:id/waterfall` — resource waterfall
- `GET /api/v1/rum/error-groups` — list error groups

## Files this change modifies (estimated)

### Backend
- `migrations/030_apm.sql` (NEW)
- `migrations/031_logs_full.sql` (NEW)
- `migrations/032_rum_full.sql` (NEW)
- `internal/handler/handlers_apm.go` (NEW, ~400 LOC)
- `internal/handler/handlers_logs_full.go` (NEW, ~350 LOC)
- `internal/handler/handlers_rum_full.go` (NEW, ~300 LOC)
- `cmd/api-gateway/routes.go` (+30 routes)

### Frontend
- `web/src/components/shared/TraceSummary.tsx` (NEW, ~80 LOC)
- `web/src/components/shared/FlameGraph.tsx` (NEW, ~120 LOC)
- `web/src/components/shared/LogEntry.tsx` (NEW, ~60 LOC)
- `web/src/components/shared/LogSearchBar.tsx` (NEW, ~100 LOC)
- `web/src/components/shared/ResourceWaterfall.tsx` (NEW, ~140 LOC)
- `web/src/components/shared/ErrorGroupCard.tsx` (NEW, ~80 LOC)
- `web/src/pages/ApmPage.tsx` (NEW, ~200 LOC)
- `web/src/pages/ApmServicePage.tsx` (NEW, ~250 LOC)
- `web/src/pages/LogsFullPage.tsx` (NEW, ~280 LOC)
- `web/src/pages/RumFullPage.tsx` (NEW, ~200 LOC)
- `web/src/pages/RumSessionPage.tsx` (NEW, ~220 LOC)

### Total: 3 migrations, 3 handler files (~1050 LOC), 6 shared components (~580 LOC), 5 pages (~1150 LOC), 30 routes — ~2780 LOC total.

## Risks

| Risk | Mitigation |
|------|------------|
| Flame graph rendering for large traces | Cap trace spans per request at 500; paginate if more |
| RUM ingest volume at scale | Per-tenant rate limit; sample at 10% in production |
| Log monitor eval performance | Eval only enabled monitors, every 30s, with per-monitor circuit breaker |
| Trace ID cardinality explosion | Hash service+endpoint to a fingerprint; cap stored trace attributes |
| Page bundle bloat | All new pages lazy-loaded via React.lazy() |

## Done criteria

- [ ] All 3 user stories pass acceptance scenarios
- [ ] Phase 1 verification gates all green
- [ ] Phase 1 subagent spec-compliance PASS
- [ ] Phase 1 subagent code-quality APPROVED
- [ ] Live 4× verifier PASS (with seeded super_admin)
- [ ] Bundle JS gzipped ≤ baseline + 80KB
- [ ] Bundle CSS gzipped unchanged
- [ ] All files under 400 LOC
- [ ] All motion variants use useReducedMotion()
- [ ] Migrations 030/031/032 applied to prod
- [ ] 30 routes registered + verified
- [ ] All 6 new shared components used in ≥2 places
- [ ] All 5 new pages navigable via sidebar
- [ ] Archive folder created
- [ ] journal.md updated
- [ ] Ready to start 004 (Tier 7.4-7.6: Synthetics + Security + CSPM)