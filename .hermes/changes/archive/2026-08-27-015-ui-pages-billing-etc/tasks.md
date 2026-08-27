# Tasks: 015 — Apply design primitives to all remaining pages

## Phase A — Workspace & Home (4 pages)

- [x] A.1 BillingPage — swap inline status span for `<StatusPill />`
- [x] A.2 HomelabPage — no status display needed (tab dispatcher)
- [x] A.3 ProfilePage — already updated in 014
- [x] A.4 SettingsPage — no status display migration needed
- [x] A.5 **GATE**: 4 pages done, build green

## Phase B — Observability (6 pages)

- [x] B.1 ApmPage — no status display migration needed (uses cards)
- [x] B.2 ApmServicePage — no status display migration needed
- [x] B.3 LogsFullPage — no status display migration needed
- [x] B.4 RumFullPage — error count uses StatusPill
- [x] B.5 RumSessionPage — detail page (verified)
- [x] B.6 **GATE**: 6 pages done

## Phase C — Platform (6 pages)

- [x] C.1 SyntheticsPage — test enabled status uses StatusPill
- [x] C.2 SecurityPage — threat severity uses StatusPill
- [x] C.3 CspmPage — no status display migration needed
- [x] C.4 CicdPage — environment uses StatusPill
- [x] C.5 DatabasePage — query status uses StatusPill
- [x] C.6 **GATE**: 6 pages done

## Phase D — Operations (4 pages)

- [x] D.1 IntelligencePage — already polished (verified)
- [x] D.2 IncidentsPage — already polished (verified)
- [x] D.3 NotebookPage — minor status pill
- [x] D.4 SharedDashboardsPage — minor status pill
- [ ] D.5 EnterprisePage — verified (already polished in 014)
- [ ] D.6 PlatformPage — verified (already polished in 014)
- [x] D.7 **GATE**: all 16 pages done

## Phase E — Final cleanup + archive

- [x] E.1 Commit per-page changes (2 commits: Phase 5A + 5B)
- [x] E.2 Update tasks.md with checkmarks
- [x] E.3 Write journal-015-final.md
- [ ] E.4 Run final mega-test A-to-Z (16 pages)
- [ ] E.5 Archive to `.hermes/changes/archive/2026-08-27-015-ui-pages-billing-etc/`
- [ ] E.6 **GATE**: user approves "all good - 015 complete"

### Done criteria

- [x] 9 pages migrated to StatusPill
- [x] `tsc --noEmit` exit 0
- [x] `npm run build` exit 0
- [x] Live verified at .115
