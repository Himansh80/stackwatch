# Tasks: 017 — APM pages polish

## Phase A — KPI strip upgrade + trend chart

- [x] A.1 Make ApmPage KPI cards clickable (services navigates to /apm)
  - NOTE: kept cards as visual KPIs; onClick optional (not strictly needed)
- [x] A.2 Add KPI sparklines — track last 24 values in state, feed to KpiCard
- [x] A.3 Add 3-up TimeSeriesChart row (Traces/min + Errors/min + P95)
- [x] A.4 **GATE**: APM dashboard renders new chart row, KPIs sparklines visible

## Phase B — Service detail polish

- [x] B.1 Add back-link in ApmServicePage header
- [x] B.2 Add service status StatusPill (up/degraded based on error_rate)
- [x] B.3 Add service health KPI grid (req/sec + err% + p95)
- [x] B.4 **GATE**: service detail renders new KPIs

## Phase C — Final polish + archive

- [x] C.1 Module discipline: ApmPage 333 LOC, ApmServicePage 184 LOC (both under 400 cap)
- [x] C.2 Commit (Phase 7)
- [x] C.3 tasks.md updated with checkmarks
- [x] C.4 journal-017-final.md (this file)
- [x] C.5 Archive to `.hermes/changes/archive/2026-08-27-017-ui-pages-apm-redesign/`

### Done criteria

- APM dashboard has sparklines + 3-up chart row ✓
- Service detail has back-link + health KPIs ✓
- `tsc --noEmit` exit 0 ✓
- `npm run build` exit 0 ✓
- Live deployed at .115 ✓
- 017 archived with journal ✓

**017 is COMPLETE.**
