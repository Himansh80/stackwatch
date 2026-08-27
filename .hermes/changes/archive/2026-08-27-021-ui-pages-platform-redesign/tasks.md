# Tasks: 021 — Platform + Database + CI/CD pages polish

## Phase A — PlatformPage

- [x] A.1 StatusPill per tab button (active='up'/green, inactive='unknown'/gray)
- [x] A.2 **GATE**: platform page renders status pills on every tab

## Phase B — DatabasePage

- [x] B.1 Rolling history state (slowHistory, queryHistory)
- [x] B.2 KPI sparklines (Slow queries)
- [x] B.3 2-up TimeSeriesChart (Slow queries + Query volume est.)
- [x] B.4 **GATE**: database page renders trend chart

## Phase C — CicdPage

- [x] C.1 Rolling history state (pipelineHistory, failureHistory)
- [x] C.2 KPI sparklines (Pipelines today)
- [x] C.3 2-up TimeSeriesChart (Pipelines + Failures)
- [x] C.4 **GATE**: cicd page renders trend chart

## Phase D — Final polish + archive

- [x] D.1 Module discipline: PlatformPage 156 LOC, DatabasePage 313 LOC, CicdPage 247 LOC (all under 400)
- [x] D.2 Commit per-phase
- [x] D.3 tasks.md updated
- [x] D.4 Write journal-021-final.md
- [x] D.5 Archive to `.hermes/changes/archive/2026-08-27-021-ui-pages-platform-redesign/`

### Done criteria

- 3 pages use shared primitives consistently ✓
- `tsc --noEmit` exit 0 ✓
- `npm run build` exit 0 ✓
- Live deployed at .115 ✓
- 021 archived with journal ✓

**021 is COMPLETE.**
