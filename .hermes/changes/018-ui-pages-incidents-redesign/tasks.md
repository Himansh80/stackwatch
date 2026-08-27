# Tasks: 018 — Incidents + Intelligence pages polish

## Phase A — IncidentsPage KPI upgrade + tab fix

- [ ] A.1 Add rolling history for incidents (24 polls × 30s)
- [ ] A.2 KPI cards get sparklines (Open / Sev1 / MTTR)
- [ ] A.3 Add TimeSeriesChart for incident rate per day
- [ ] A.4 Wire tab navigation (Open / Acknowledged / Resolved / All) — currently broken
- [ ] A.5 Add StatusPill to the section header (showing current tab status)
- [ ] A.6 **GATE**: KPIs show sparklines, tabs work, trend chart renders

## Phase B — IntelligencePage section polish

- [ ] B.1 Anomalies tab: StatusPill per anomaly card (sev-based)
- [ ] B.2 Predictions tab: prediction cards get sparkline of historical accuracy
- [ ] B.3 Correlations tab: correlation cards use StatusPill for severity
- [ ] B.4 Noise tab: noise rule rows use StatusPill for enable/disable
- [ ] B.5 Verify all 5 tabs render with consistent design
- [ ] B.6 **GATE**: sections render with consistent primitives

## Phase C — Final polish + archive

- [ ] C.1 Module discipline (≤400 LOC per file)
- [ ] C.2 Commit per-phase
- [ ] C.3 tasks.md updated with checkmarks
- [ ] C.4 Write journal-018-final.md
- [ ] C.5 Archive to `.hermes/changes/archive/2026-08-27-018-ui-pages-incidents-redesign/`

### Done criteria

- IncidentsPage + IntelligencePage use shared primitives consistently
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live deployed at .115
- 018 archived with journal
