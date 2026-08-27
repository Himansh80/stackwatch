# Proposal: 020 — Security + CSPM pages polish

## Why

Both `/security` and `/cspm` are the security/enterprise surfaces. They already have:
- KPI strip + tab navigation
- Threat cards + compliance rules

But:
- No time-series visualization for threat rate / compliance drift
- KPI cards lack sparklines
- No severity timeline

This change adds:
- Sparklines on threat/compliance KPIs
- Trend chart for threats over time + compliance drift
- Better severity visualization in threat cards

## What changes

### 1. SecurityPage polish
- KPI cards get sparklines (24 polls)
- 2-up TimeSeriesChart row: Threat rate / Open threats
- Severity pills in threat cards (already migrated in 015)

### 2. CspmPage polish
- KPI cards get sparklines
- 2-up TimeSeriesChart row: Compliance drift / Failing resources
- Severity pills in finding cards

## Scope guardrails

- **2 pages only** (SecurityPage, CspmPage).
- **No new endpoints.**
- **Module discipline** — every file ≤400 LOC.
- **Reuse primitives.**

## Out of scope

- New endpoints
- Other pages

## Impact

| Area | Impact |
|------|--------|
| Files modified | 2 (SecurityPage.tsx, CspmPage.tsx) |
| Breaking | No |
| Risk | Visual regression. Mitigated by per-phase test plan. |

## Workflow

- Speckit: proposal → spec → design → tasks → build → archive
- Per-phase test plan

## Acceptance

- SecurityPage: KPI sparklines + threat trend chart
- CspmPage: KPI sparklines + compliance drift chart
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at .115

## Rollback

- Revert single commit "feat(ui): 020 Security + CSPM polish"
