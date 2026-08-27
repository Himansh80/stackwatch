# Proposal: 021 — Platform + Database + CI/CD pages polish

## Why

These three pages are the platform/operations surfaces:
- **PlatformPage** — billing, deployments, signup, regions
- **DatabasePage** — query monitoring
- **CicdPage** — pipelines + deployments

All three have KPI strips already. This change:
- Adds sparklines + trend charts where missing
- Adds StatusPill per item row
- Keeps all 3 pages within the 400 LOC cap

## What changes

### 1. PlatformPage polish (147 LOC)
- StatusPill per plan row (active/trialing/expired)
- StatusPill per deployment row (success/failed/running)

### 2. DatabasePage polish (288 LOC)
- KPI sparklines (slow queries, errors)
- 2-up TimeSeriesChart (query rate + slow query rate)

### 3. CicdPage polish (235 LOC)
- KPI sparklines (pipelines, failures)
- 2-up TimeSeriesChart (pipeline runs + failure rate)

## Scope guardrails

- **3 pages only.**
- **No new endpoints.**
- **Module discipline** — every file ≤400 LOC.

## Out of scope

- New endpoints
- Other pages

## Impact

| Area | Impact |
|------|--------|
| Files modified | 3 |
| Breaking | No |

## Workflow

- Speckit: proposal → spec → design → tasks → build → archive

## Acceptance

- All 3 pages use shared primitives consistently
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at .115

## Rollback

- Revert single commit "feat(ui): 021 Platform + Database + CI/CD polish"
