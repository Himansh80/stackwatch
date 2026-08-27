# Tasks: 015 — Apply design primitives to all remaining pages

## Phase A — Workspace & Home (4 pages)

- [ ] A.1 BillingPage — swap inline status spans for `<StatusPill />`
- [ ] A.2 HomelabPage — swap inline status spans for `<StatusPill />`
- [ ] A.3 ProfilePage — already updated in 014, verify
- [ ] A.4 SettingsPage — minor status pill migration
- [ ] A.5 **GATE**: 4 pages all use primitives, build green, live verified

## Phase B — Observability (6 pages)

- [ ] B.1 ApmPage — use TimeSeriesChart for latency + StatusPill for service status
- [ ] B.2 ApmServicePage — use StatusPill for service state
- [ ] B.3 LogsFullPage — use StatusPill for log level + TimeSeriesChart for log rate
- [ ] B.4 RumFullPage — use StatusPill for page health
- [ ] B.5 RumSessionPage — already a detail page, verify
- [ ] B.6 **GATE**: 6 pages all use primitives, build green, live verified

## Phase C — Platform (6 pages)

- [ ] C.1 SyntheticsPage — use StatusPill for check status
- [ ] C.2 SecurityPage — use StatusPill for threat severity
- [ ] C.3 CspmPage — use StatusPill + KPI card
- [ ] C.4 CicdPage — use StatusPill for pipeline status
- [ ] C.5 DatabasePage — use StatusPill for query perf
- [ ] C.6 **GATE**: 6 pages all use primitives, build green, live verified

## Phase D — Operations (4 pages)

- [ ] D.1 IntelligencePage — use StatusPill for correlation/severity + TimeSeriesChart
- [ ] D.2 IncidentsPage — use StatusPill for sev (already partially done)
- [ ] D.3 NotebookPage — minor status pill
- [ ] D.4 SharedDashboardsPage — minor status pill
- [ ] D.5 EnterprisePage — already polished, verify
- [ ] D.6 PlatformPage — already polished, verify
- [ ] D.7 **GATE**: all 16 pages done

## Phase E — Final cleanup + archive

- [ ] E.1 Commit per-page changes
- [ ] E.2 Update tasks.md with checkmarks
- [ ] E.3 Write journal-015-final.md
- [ ] E.4 Run final mega-test A-to-Z (16 pages)
- [ ] E.5 Archive to `.hermes/changes/archive/2026-08-27-015-ui-pages-billing-etc/`
- [ ] E.6 **GATE**: user approves "all good - 015 complete"

### Done criteria

- All 16 pages use StatusPill (no inline status spans)
- TimeSeriesChart used wherever time-series exists
- KpiCard used with onClick wherever KPI cards exist
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at .115
- 015 archived with journal
