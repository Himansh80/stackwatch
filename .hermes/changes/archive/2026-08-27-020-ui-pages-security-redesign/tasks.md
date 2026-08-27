# Tasks: 020 — Security + CSPM pages polish

## Phase A — SecurityPage

- [x] A.1 Add rolling history state for KPIs (threatCountHistory, criticalHistory)
- [x] A.2 KPI cards get sparklines (Open threats + Critical)
- [x] A.3 4-card KPI strip (Open threats / Critical / High / Total) — added High + Total too
- [x] A.4 Add 2-up TimeSeriesChart row (Threat count + Critical severity)
- [x] A.5 **GATE**: security page renders trend chart

## Phase B — CspmPage

- [x] B.1 Add rolling history state for KPIs (resourceHistory, findingsHistory)
- [x] B.2 KPI cards get sparklines
- [x] B.3 Add 2-up TimeSeriesChart row (Tracked resources + Findings)
- [x] B.4 **GATE**: cspm page renders trend chart

## Phase C — Final polish + archive

- [x] C.1 Module discipline: SecurityPage 363 LOC, CspmPage 250 LOC (both under 400)
- [x] C.2 Commit per-phase
- [x] C.3 tasks.md updated with checkmarks
- [x] C.4 Write journal-020-final.md
- [x] C.5 Archive to `.hermes/changes/archive/2026-08-27-020-ui-pages-security-redesign/`

### Done criteria

- 2 pages use shared primitives consistently ✓
- `tsc --noEmit` exit 0 ✓
- `npm run build` exit 0 ✓
- Live deployed at .115 ✓
- 020 archived with journal ✓

**020 is COMPLETE.**
