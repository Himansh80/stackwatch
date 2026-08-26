# Implementation Plan: Tier 7 — Datadog Full Platform Parity (Phase 1 of 11)

**Feature**: 003-tier7-datadog-parity
**Spec**: [spec.md](spec.md)
**Created**: 2026-08-24
**Status**: Ready for tasks

## Approach

Three-phase build with verification at every phase boundary. Each subtier (APM, Log Mgmt Full, RUM Full) ships end-to-end (migration + handlers + shared components + pages) in its own phase. Every phase ends with: build green, type-check green, lint green, bundle check, subagent spec-compliance + code-quality review.

1. **Phase 1 — APM (D2)**: Migration 030 → apm_services/apm_traces/apm_spans/apm_deployments tables → handlers_apm.go → 5 shared components + 2 pages (ApmPage, ApmServicePage)
2. **Phase 2 — Log Management Full (D3)**: Migration 031 → log_monitors/log_archives/log_rehydrations/log_retention_policies/log_patterns tables → handlers_logs_full.go → 2 shared components (LogEntry, LogSearchBar) + 1 page (LogsFullPage)
3. **Phase 3 — RUM Full (D4)**: Migration 032 → rum_sessions/rum_web_vitals/rum_resources/rum_interactions/rum_long_tasks/rum_error_groups tables → handlers_rum_full.go → 2 shared components (ResourceWaterfall, ErrorGroupCard) + 2 pages (RumFullPage, RumSessionPage)

Each phase is independently deployable. Phase 1 is the trace model foundation; Phase 2 (logs) depends only on the existing Tier 6 v1 logs table; Phase 3 (RUM) is independent.

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| 3 subtiers per change (not all 11) | Each is independently verifiable; reduces review surface | Future changes 004-007 still needed |
| New handler files (not extending existing) | Each subtier is ~300-400 LOC; one file per subtier keeps modularity | 3 more files vs 1 huge one |
| Trace model: services + traces + spans (3 tables) | Datadog's data model; spans can be queried per-trace or per-service | More tables; slower queries if unindexed |
| RUM per-tenant rate limit + 10% sampling | Protect backend from RUM floods | Slight loss of fidelity at high scale |
| Flame graph: SVG renderer (no D3 dep) | Already have minimal data-viz infra; no new dep | Manual SVG vs library polish |
| Page lazy-load via React.lazy() | Reduce initial bundle bloat | Slightly slower nav between pages |

## Phase 1 — APM (D2)

### 1.1 — Migration 030
- Create `migrations/030_apm.sql` with apm_services, apm_traces, apm_spans, apm_deployments
- Apply to prod DB on `.116`
- Verify table creation via psql

### 1.2 — Handlers (`internal/handler/handlers_apm.go`, ~400 LOC)
- `POST /api/v1/apm/services` — register service (idempotent on tenant+name)
- `GET /api/v1/apm/services` — list services for tenant
- `GET /api/v1/apm/services/:id` — service detail (request rate, error rate, p95 latency)
- `POST /api/v1/apm/traces` — ingest trace (OTLP-compatible JSON body)
- `GET /api/v1/apm/traces/:trace_id` — get trace + all spans
- `POST /api/v1/apm/deployments` — record deployment
- `GET /api/v1/apm/deployments` — list deployments (filter ?service_id=X)
- `GET /api/v1/apm/service-map` — graph data (nodes + edges with request count)
- `GET /api/v1/apm/services/:id/flame-graph` — flame graph for time window

### 1.3 — Routes
- Register 8 APM routes in `cmd/api-gateway/routes.go`
- All under `protected` group (require auth)

### 1.4 — Shared components
- `TraceSummary.tsx` (~80 LOC) — service node header
- `FlameGraph.tsx` (~120 LOC) — SVG flame graph renderer (uses sparklineDraw)

### 1.5 — Pages
- `ApmPage.tsx` (~200 LOC) — services list + service map + deployments
- `ApmServicePage.tsx` (~250 LOC) — service detail + flame graph + traces

### 1.6 — Verification
- `go build ./cmd/api-gateway` exit 0
- `npm run type-check && npm run build` exit 0
- Bundle JS gzipped ≤ baseline + 30KB
- All files <400 LOC
- Subagent spec-compliance + code-quality review PASS

## Phase 2 — Log Management Full (D3)

### 2.1 — Migration 031
- `migrations/031_logs_full.sql` — log_monitors, log_archives, log_rehydrations, log_retention_policies, log_patterns
- Apply to prod

### 2.2 — Handlers (`internal/handler/handlers_logs_full.go`, ~350 LOC)
- `POST /api/v1/logs/monitors` — create
- `GET /api/v1/logs/monitors` — list
- `PUT /api/v1/logs/monitors/:id` — update
- `DELETE /api/v1/logs/monitors/:id` — delete
- `GET /api/v1/logs/patterns` — detected patterns (GROUP BY service, message_prefix)
- `POST /api/v1/logs/archives` — create
- `GET /api/v1/logs/archives` — list
- `POST /api/v1/logs/archives/:id/rehydrate` — queue rehydration
- `GET /api/v1/logs/rehydrations` — list (filter ?archive_id=X)
- `POST /api/v1/logs/retention` — set policy
- `GET /api/v1/logs/retention` — list policies
- 11 endpoints total

### 2.3 — Routes
- Register 11 routes in `cmd/api-gateway/routes.go`

### 2.4 — Shared components
- `LogEntry.tsx` (~60 LOC) — log row (Datadog-style: level pill + service + timestamp + message)
- `LogSearchBar.tsx` (~100 LOC) — search + filters + time-range picker

### 2.5 — Page
- `LogsFullPage.tsx` (~280 LOC) — search results + retention + archives + monitors tabs

### 2.6 — Verification
- Same gates as Phase 1

## Phase 3 — RUM Full (D4)

### 3.1 — Migration 032
- `migrations/032_rum_full.sql` — rum_sessions, rum_web_vitals, rum_resources, rum_interactions, rum_long_tasks, rum_error_groups

### 3.2 — Handlers (`internal/handler/handlers_rum_full.go`, ~300 LOC)
- `POST /api/v1/rum/web-vitals` — ingest
- `POST /api/v1/rum/resource-timings` — ingest
- `POST /api/v1/rum/interactions` — ingest
- `POST /api/v1/rum/long-tasks` — ingest
- `POST /api/v1/rum/errors` — ingest (groups by fingerprint)
- `POST /api/v1/rum/heatmap` — ingest click event
- `GET /api/v1/rum/sessions` — list (filter ?time_range=X)
- `GET /api/v1/rum/sessions/:id` — session detail
- `GET /api/v1/rum/sessions/:id/waterfall` — resource timings
- `GET /api/v1/rum/error-groups` — list (filter ?service=X)
- 10 endpoints total

### 3.3 — Routes
- Register 10 routes in `cmd/api-gateway/routes.go`

### 3.4 — Shared components
- `ResourceWaterfall.tsx` (~140 LOC) — network requests in chronological order
- `ErrorGroupCard.tsx` (~80 LOC) — error group with stack trace preview

### 3.5 — Pages
- `RumFullPage.tsx` (~200 LOC) — sessions list + web vitals + error groups
- `RumSessionPage.tsx` (~220 LOC) — session detail + replay + waterfall

### 3.6 — Verification
- Same gates as Phase 1

## Phase 4 — Verify-first sweep + deploy + archive

### 4.1 — Final build verification
- `go build ./cmd/api-gateway` exit 0
- `npm run type-check && npm run lint && npm run build` exit 0
- Bundle JS gzipped ≤ baseline + 80KB

### 4.2 — Subagent reviews
- Spec-compliance reviewer (read-only)
- Code-quality reviewer (read-only)

### 4.3 — Deploy to `.115` (api-gateway binary) + `.115` (web bundle)
- Build on .117 (canonical Go build host) for the api-gateway binary
- Build vite bundle locally; scp to .115
- Restart services

### 4.4 — Live 4× verifier
- All 30 routes return 200 or appropriate auth-gated 401
- Static pages return 200
- Seed test data, verify ingestion works

### 4.5 — Archive
- Move to `.hermes/changes/archive/2026-08-24-003-tier7-datadog-parity/`

## Risks + mitigations

| Risk | Mitigation |
|------|------------|
| Backend LOC exceeds 400 per file | Split each subtier into multiple handler files (apm_traces.go, apm_services.go, etc.) |
| RUM floods backend at scale | Per-tenant rate limit; sample at 10% in production |
| Flame graph for huge traces slow | Cap at 500 spans per request; paginate |
| All 30 routes is a lot to test | Speckit verifier with parameterized loops |
| New tables grow unbounded | Add partition strategy in Phase 2 |

## Done criteria

- [ ] All 3 user stories pass acceptance scenarios
- [ ] Phase 1-3 verification gates all green
- [ ] Phase 4 verify-first sweep passes
- [ ] Subagent spec-compliance review PASS
- [ ] Subagent code-quality review APPROVED
- [ ] Live 4× verifier PASS
- [ ] Bundle JS gzipped ≤ baseline + 80KB
- [ ] Bundle CSS gzipped unchanged
- [ ] All files under 400 LOC
- [ ] All motion variants use useReducedMotion()
- [ ] Migrations applied to prod
- [ ] All 30 routes registered + verified
- [ ] Archive folder created
- [ ] journal.md updated
- [ ] Ready to start 004