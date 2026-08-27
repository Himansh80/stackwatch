# Proposal: 017 — APM pages polish

## Why

ApmPage already has:
- 4-card KPI strip (Services / Traces/min / Errors/min / P95 global)
- Service cards (TraceSummary)
- Deployments table

But:
- KPI cards aren't clickable
- No time-series chart for traces/errors over time
- Service list uses single-line cards with no sparkline for trend
- ApmServicePage (detail) lacks a back-link in the header

This change applies the design system + adds a single time-series chart for the global trace rate (mini sparkline per KPI card).

## What changes

### 1. KPI strip — clickable + sparklines
- Services (indigo) — navigates to service list
- Traces/min (cyan) — sparkline of last 24 data points
- Errors/min (red) — sparkline of error rate over time
- P95 global (amber) — sparkline of p95 latency over time

### 2. Time-series trend chart
- Add a 3-up TimeSeriesChart row below the KPI strip
- (a) Traces/min (b) Errors/min (c) P95 latency — each chart has its own color

### 3. ApmServicePage polish
- Back-link at top to return to /apm
- Service status badge (StatusPill) based on error_rate
- Service health card (KpiCard grid): req/sec, err%, p95

## Scope guardrails

- **APM pages only.** Synthetics, Logs, RUM stay unchanged.
- **No new endpoints.** Use existing data sources.
- **Module discipline** — every file ≤400 LOC.
- **Same data sources** — `/api/v1/apm/services`, `/api/v1/apm/services/:id`, `/api/v1/apm/deployments`.

## Out of scope

- New APM endpoints (e.g. historical trace timeseries)
- Other pages
- Mobile app

## Impact

| Area | Impact |
|------|--------|
| Files modified | 2 (ApmPage.tsx, ApmServicePage.tsx) |
| Breaking | No |
| Risk | Visual regression on APM surface. Mitigated by per-phase test plan. |

## Workflow

- Speckit: proposal → spec → design → tasks → build → archive
- Per-phase test plan
- Reuse primitives (StatusPill, KpiCard with onClick, TimeSeriesChart)

## Acceptance

- APM dashboard has KPI strip + 3-up trend chart row
- ApmServicePage has back-link + service status badge
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at https://stackwatch.smarthomelab.fun/apm

## Rollback

- Revert the single commit "feat(ui): 017 APM pages polish"
