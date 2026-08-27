# Tasks: 020 — Security + CSPM pages polish

## Phase A — SecurityPage

- [ ] A.1 Add rolling history state for KPIs
- [ ] A.2 KPI cards get sparklines (open threats / severity)
- [ ] A.3 Add 2-up TimeSeriesChart row (threat rate + open threats)
- [ ] A.4 **GATE**: security page renders trend chart

## Phase B — CspmPage

- [ ] B.1 Add rolling history state for KPIs
- [ ] B.2 KPI cards get sparklines
- [ ] B.3 Add 2-up TimeSeriesChart row (compliance drift + failing resources)
- [ ] B.4 **GATE**: cspm page renders trend chart

## Phase C — Final polish + archive

- [ ] C.1 Module discipline (≤400 LOC per file)
- [ ] C.2 Commit per-phase
- [ ] C.3 tasks.md updated with checkmarks
- [ ] C.4 Write journal-020-final.md
- [ ] C.5 Archive to `.hermes/changes/archive/2026-08-27-020-ui-pages-security-redesign/`

### Done criteria

- 2 pages use shared primitives consistently
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live deployed at .115
- 020 archived with journal
