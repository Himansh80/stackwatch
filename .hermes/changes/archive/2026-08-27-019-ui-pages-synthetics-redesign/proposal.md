# Proposal: 019 — Synthetics + Logs + RUM pages polish

## Why

These three pages are the last major observability surfaces:
- **SyntheticsPage** has a KPI strip + tests table, but the row StatusPill is misindented (cosmetic) and only shows enabled/paused — not actual passing/failing
- **LogsFullPage** is small (143 LOC), already has tab dispatcher
- **RumFullPage** has session list with error count

This change:
- SyntheticsPage: fix StatusPill alignment + add per-test status (from results endpoint) + TimeSeriesChart for pass rate
- LogsFullPage: add StatusPill per log level + KPI strip
- RumFullPage: add StatusPill per session health

## What changes

### 1. SyntheticsPage — proper test status
- Per-row StatusPill shows actual test result (up/down/stale) based on `/synthetics/results/:test_id`
- KPIs get sparklines (passing/failing over time)
- New trend chart row: pass rate over time
- Fix the misindented StatusPill JSX

### 2. LogsFullPage — KPI strip + level StatusPills
- 4-card KPI strip (logs/sec, errors, warnings, info)
- Tab labels get StatusPill per level

### 3. RumFullPage — session health StatusPills
- StatusPill per session row (healthy/degraded/broken based on errors + LCP)

## Scope guardrails

- **3 pages only** (Synthetics, Logs, RUM).
- **No new endpoints.** Use existing data sources.
- **Module discipline** — every file ≤400 LOC.
- **Reuse primitives.**

## Out of scope

- New endpoints
- Other pages

## Impact

| Area | Impact |
|------|--------|
| Files modified | 3 (SyntheticsPage, LogsFullPage, RumFullPage) |
| Breaking | No |
| Risk | Visual regression. Mitigated by per-phase test plan. |

## Workflow

- Speckit: proposal → spec → design → tasks → build → archive
- Per-phase test plan

## Acceptance

- SyntheticsPage: KPI sparklines + trend chart + proper per-test status
- LogsFullPage: KPI strip + tab StatusPills
- RumFullPage: session health StatusPills
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at .115

## Rollback

- Revert single commit "feat(ui): 019 Synthetics/Logs/RUM polish"
