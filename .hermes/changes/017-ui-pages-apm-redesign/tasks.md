# Tasks: 017 — APM pages polish

## Phase A — KPI strip upgrade + trend chart

- [ ] A.1 Make ApmPage KPI cards clickable (services navigates to /apm)
- [ ] A.2 Add KPI sparklines — track last 24 values in state, feed to KpiCard
- [ ] A.3 Add 3-up TimeSeriesChart row (Traces/min + Errors/min + P95)
- [ ] A.4 **GATE**: APM dashboard renders new chart row, KPIs clickable

## Phase B — Service detail polish

- [ ] B.1 Add back-link in ApmServicePage header
- [ ] B.2 Add service status StatusPill (up/degraded based on error_rate)
- [ ] B.3 Add service health KPI grid (req/sec + err% + p95)
- [ ] B.4 **GATE**: service detail renders new KPIs

## Phase C — Final polish + archive

- [ ] C.1 Verify module discipline (≤400 LOC per file)
- [ ] C.2 Commit per-phase
- [ ] C.3 Update tasks.md with checkmarks
- [ ] C.4 Write journal-017-final.md
- [ ] C.5 Archive to `.hermes/changes/archive/2026-08-27-017-ui-pages-apm-redesign/`
- [ ] C.6 **GATE**: user approves

### Done criteria

- APM dashboard has sparklines + 3-up chart row
- Service detail has back-link + health KPIs
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live deployed at .115
- 017 archived with journal
