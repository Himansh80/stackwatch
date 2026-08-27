# Tasks: 019 — Synthetics + Logs + RUM pages polish

## Phase A — SyntheticsPage

- [ ] A.1 Fix misindented StatusPill JSX (line 300)
- [ ] A.2 Add per-test actual result StatusPill (pass/fail from results endpoint)
- [ ] A.3 Add KPI sparklines for passing/failing counts
- [ ] A.4 Add TimeSeriesChart for pass rate over time
- [ ] A.5 **GATE**: synthetics page has trend chart + proper status

## Phase B — LogsFullPage

- [ ] B.1 Add KPI strip (logs/sec, errors, warnings, info)
- [ ] B.2 Add StatusPill per tab label (LogSearch, Monitors, Archives, Retention)
- [ ] B.3 **GATE**: logs page has KPI strip + tab StatusPills

## Phase C — RumFullPage

- [ ] C.1 Add StatusPill per session row (healthy/degraded/broken)
- [ ] C.2 Status calculation: errors=0 + LCP<2.5s = healthy, errors<5 + LCP<4s = degraded, else broken
- [ ] C.3 **GATE**: rum page shows session health

## Phase D — Final polish + archive

- [ ] D.1 Module discipline (≤400 LOC per file)
- [ ] D.2 Commit per-phase
- [ ] D.3 tasks.md updated with checkmarks
- [ ] D.4 Write journal-019-final.md
- [ ] D.5 Archive to `.hermes/changes/archive/2026-08-27-019-ui-pages-synthetics-redesign/`

### Done criteria

- Synthetics/Logs/RUM all use shared primitives consistently
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live deployed at .115
- 019 archived with journal
