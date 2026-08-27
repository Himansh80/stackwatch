# Tasks: 019 — Synthetics + Logs + RUM pages polish

## Phase A — SyntheticsPage

- [x] A.1 Fix misindented StatusPill JSX (line 300)
- [x] A.2 Per-row StatusPill shows enabled/paused (pass/fail needs results endpoint — approximation via sla_uptime_pct >= 99.5)
- [x] A.3 KPI sparklines for passing/failing counts
- [x] A.4 TimeSeriesChart for pass/fail over time (2 panels)
- [x] A.5 **GATE**: synthetics page renders new trend chart + sparklines

## Phase B — LogsFullPage

- [x] B.1 Fix dead tab state (`_setTab` → `setTab`)
- [x] B.2 Add tab navigation UI (was missing entirely — page had 5 tabs but no way to switch)
- [x] B.3 Add KPI strip (Errors / Warnings / Info / Debug)
- [x] B.4 StatusPill per tab label
- [x] B.5 **GATE**: logs page has working tabs + KPI strip

## Phase C — RumFullPage

- [x] C.1 Already shipped in Phase 5 (session-level StatusPill)
- [x] C.2 Verified — page renders correctly

## Phase D — Final polish + archive

- [x] D.1 Module discipline: SyntheticsPage 366 LOC, LogsFullPage 175 LOC (both under 400)
- [x] D.2 Commit (Phase 9)
- [x] D.3 tasks.md updated with checkmarks
- [x] D.4 journal-019-final.md
- [x] D.5 Archive to `.hermes/changes/archive/2026-08-27-019-ui-pages-synthetics-redesign/`

### Done criteria

- 3 pages use shared primitives consistently ✓
- `tsc --noEmit` exit 0 ✓
- `npm run build` exit 0 ✓
- Live deployed at .115 ✓
- 019 archived with journal ✓

**019 is COMPLETE.**
