# Tasks: 021 — Platform + Database + CI/CD pages polish

## Phase A — PlatformPage

- [ ] A.1 Add StatusPill per plan row (active/trialing/expired)
- [ ] A.2 Add StatusPill per deployment row
- [ ] A.3 **GATE**: platform page renders status pills

## Phase B — DatabasePage

- [ ] B.1 Add rolling history state for KPIs (query rate, slow queries)
- [ ] B.2 KPI sparklines (slow queries + errors)
- [ ] B.3 2-up TimeSeriesChart row
- [ ] B.4 **GATE**: database page renders trend chart

## Phase C — CicdPage

- [ ] C.1 Add rolling history state
- [ ] C.2 KPI sparklines (pipelines + failures)
- [ ] C.3 2-up TimeSeriesChart row
- [ ] C.4 **GATE**: cicd page renders trend chart

## Phase D — Final polish + archive

- [ ] D.1 Module discipline (≤400 LOC per file)
- [ ] D.2 Commit
- [ ] D.3 tasks.md updated
- [ ] D.4 Write journal-021-final.md
- [ ] D.5 Archive to `.hermes/changes/archive/2026-08-27-021-ui-pages-platform-redesign/`

### Done criteria

- 3 pages use shared primitives consistently
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live deployed at .115
- 021 archived with journal
