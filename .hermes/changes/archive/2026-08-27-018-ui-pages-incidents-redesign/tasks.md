# Tasks: 018 — Incidents + Intelligence pages polish

## Phase A — IncidentsPage KPI upgrade + tab fix

- [x] A.1 Add rolling history for incidents (24 polls × 30s)
- [x] A.2 KPI cards get sparklines (Open + Sev1)
- [x] A.3 Add TimeSeriesChart for incident rate per tab
- [x] A.4 Wire tab navigation (Open / Acknowledged / Resolved / All) — was previously dead
- [x] A.5 Add StatusPill to each tab (shows whether that tab has data)
- [x] A.6 Tab count badges (openKpiValue / counts.ack / counts.resolved / incidents.length)
- [x] A.7 **GATE**: KPIs show sparklines, tabs work, trend chart renders

## Phase B — IntelligencePage section polish

- [x] B.1 StatusPill per tab label (anomalies→crit, predictions→warn, correlations→warn, noise→ok)
- [x] B.2 (Skipped) Anomalies tab: StatusPill per anomaly card — already handled inside AnomaliesSection
- [x] B.3 (Skipped) Predictions tab: sparkline per prediction card — section handles internally
- [x] B.4 (Skipped) Correlations tab: StatusPill per correlation — CorrelationsSection handles
- [x] B.5 (Skipped) Noise tab: noise rule StatusPill — NoiseReductionSection handles
- [x] B.6 **GATE**: 4 tabs render with consistent StatusPill labels

## Phase C — Final polish + archive

- [x] C.1 Module discipline: IncidentsPage 232 LOC, IntelligencePage 320 LOC (both under 400)
- [x] C.2 Commit per-phase
- [x] C.3 tasks.md updated with checkmarks
- [x] C.4 journal-018-final.md
- [x] C.5 Archive to `.hermes/changes/archive/2026-08-27-018-ui-pages-incidents-redesign/`

### Done criteria

- IncidentsPage: KPIs sparklines + trend chart + working tabs ✓
- IntelligencePage: tabs with StatusPill labels ✓
- `tsc --noEmit` exit 0 ✓
- `npm run build` exit 0 ✓
- Live deployed at .115 ✓
- 018 archived with journal ✓

**018 is COMPLETE.**
