# Proposal: 018 — Incidents + Intelligence pages polish

## Why

Both pages already have KPI strips + tab navigation but:
- **IncidentsPage**: 3 KPI cards lack sparklines; incidents are listed as cards with no chart of incident rate over time; severity dropdown is the only filter
- **IntelligencePage**: 5-tab dispatcher, but the Anomalies / Predictions / Correlations / Noise sections all have their own internal card layout that's inconsistent

This change adds:
- Sparklines + time-series trend chart to IncidentsPage
- Unifies the IntelligencePage internal section renderers with shared StatusPill + TimeSeriesChart

## What changes

### 1. IncidentsPage — KPI sparklines + trend chart
- KPI cards (Open / Sev1 / MTTR) get sparklines (last 24 polls)
- New 2-up TimeSeriesChart row: Incident rate per day + Resolution time trend
- Severity filter dropdown stays (now with StatusPill icon per severity)

### 2. IncidentsPage — Tab navigation fixed
- Currently `tab` state is set but the render always shows "all" incidents
- Wire tabs so each tab shows its slice (open / acknowledged / resolved / all)
- Each tab gets a StatusPill in the section header

### 3. IntelligencePage — Section polish
- Anomalies tab: anomaly cards use StatusPill for severity
- Predictions tab: prediction cards get sparkline of historical accuracy
- Correlations tab: correlation cards get StatusPill for severity (high/med/low)
- Noise tab: noise rules use StatusPill for enable/disable
- All sections get consistent TimeSeriesChart where applicable

## Scope guardrails

- **IncidentsPage + IntelligencePage only.** Notifications + Notebooks + Collaboration are separate.
- **No new endpoints.** Use existing data.
- **Module discipline** — every file ≤400 LOC.
- **Reuse primitives** (StatusPill, KpiCard with sparkline, TimeSeriesChart).

## Out of scope

- New endpoints
- Other pages
- Mobile app

## Impact

| Area | Impact |
|------|--------|
| Files modified | 2 (IncidentsPage.tsx, IntelligencePage.tsx) |
| Breaking | No |
| Risk | Visual regression. Mitigated by per-phase test plan. |

## Workflow

- Speckit: proposal → spec → design → tasks → build → archive
- Per-phase test plan
- Reuse primitives

## Acceptance

- IncidentsPage has KPI sparklines + trend chart row
- IncidentsPage tabs work (Open / Acknowledged / Resolved / All)
- IntelligencePage sections use StatusPill + sparklines consistently
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at .115

## Rollback

- Revert single commit "feat(ui): 018 Incidents + Intelligence polish"
