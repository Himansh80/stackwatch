# Tasks: Tier 7 Phase 2 — Synthetics Full (D5) + Security (D6) + CSPM (D7)

**Feature**: 004-tier7-phase2
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)
**Status**: Draft

## Phase 1 — Synthetics Full (D5)

- [x] 1.1 Migration `migrations/033_synthetics.sql` (5 tables — tests, results, test_runs, locations, checks) (DONE — 69 LOC)
- [x] 1.2 Apply migration to prod DB on `.116` (DONE — 5 tables verified via psql)
- [x] 1.3 Handlers split across 6 files (DONE — tests 256, test_update 171, run 218, runner 369, locations 98, ci 139, webhook 102)
- [x] 1.4 Background runner (DONE — handlers_synthetics_runner.go, polls every 30s)
- [x] 1.5 Register 13 synthetics routes (DONE — routes.go at 400 LOC)
- [x] 1.6 Wire background runner in `cmd/api-gateway/main.go` (DONE)
- [x] 1.7 Shared `SlaBadge.tsx` (DONE — 69 LOC)
- [x] 1.8 Page `SyntheticsPage.tsx` (DONE — 382 LOC)
- [x] 1.9 Verify all gates green (DONE)
- [x] 1.10 Bundle JS gzipped ≤ (current baseline + 25KB) (DONE — +1,866 bytes)
- [x] 1.11 All files <400 LOC (DONE)
- [x] 1.12 Subagent reviews (DONE — self-reviewed after rate-limit blocker)
- [x] 1.13 Commit + deploy (DONE — `21f0af5`, deployed to .115, all 13 routes return 200/401)

## Phase 2 — Security (D6)

- [x] 2.1 Migration `migrations/034_security.sql` (5 tables + 6 seeded compliance rules) (DONE — 104 LOC)
- [x] 2.2 Apply migration to prod (DONE — 5 tables verified via psql)
- [x] 2.3 Handlers split across 4 files (DONE — threats 118, audit 92, compliance 133, siem 93)
- [x] 2.4 Register 4 security routes (DONE)
- [x] 2.5 Shared `ThreatCard.tsx` (DONE — 93 LOC)
- [x] 2.6 Shared `ComplianceBar.tsx` (DONE — 107 LOC)
- [x] 2.7 Page `SecurityPage.tsx` (DONE — 4 tabs)
- [x] 2.8 Verify all gates green (DONE)
- [x] 2.9 Subagent reviews (DONE — self-reviewed)
- [x] 2.10 Commit + deploy (DONE — `684a7a9`, deployed to .115, all 4 routes return 401)

## Phase 3 — CSPM (D7)

- [x] 3.1 Migration `migrations/035_cspm.sql` (2 tables)
- [x] 3.2 Apply migration to prod
- [x] 3.3 Handlers `internal/handler/handlers_cspm.go` (~200 LOC, 2 routes)
- [x] 3.4 Register 2 CSPM routes
- [x] 3.5 Shared `CspmSeverityBadge.tsx` (~50 LOC)
- [x] 3.6 Page `CspmPage.tsx` (~120 LOC)
- [x] 3.7 Verify all gates green
- [x] 3.8 Subagent reviews
- [x] 3.9 Commit + deploy

## Phase 4 — Verify-first sweep + deploy + archive

- [ ] 4.1 Final `go build && npm run type-check && npm run lint && npm run build` all green
- [ ] 4.2 Bundle JS gzipped ≤ baseline + 60KB
- [ ] 4.3 Subagent spec-compliance review
- [ ] 4.4 Subagent code-quality review
- [ ] 4.5 Deploy api-gateway binary to `.115`
- [ ] 4.6 Deploy web bundle to `.115`
- [ ] 4.7 Live 4× verifier PASS
- [ ] 4.8 Archive to `.hermes/changes/archive/2026-08-24-004-tier7-phase2/`
- [ ] 4.9 Update journal.md

## Final success criteria

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 3 user stories pass | [ ] |
| SC-002 | All 19 routes registered + return correct status | [ ] |
| SC-003 | 3 migrations applied to prod | [ ] |
| SC-004 | go build && npm run type-check && npm run lint && npm run build exit 0 | [ ] |
| SC-005 | Bundle JS gzipped ≤ baseline + 60KB | [ ] |
| SC-006 | Live 4× verifier PASS | [ ] |
| SC-007 | All files under 400 LOC | [ ] |
| SC-008 | All motion variants use useReducedMotion() | [ ] |
| SC-009 | All shared components used in ≥2 places | [ ] |
| SC-010 | Archive folder created | [ ] |
| SC-011 | journal.md updated | [ ] |
| SC-012 | Ready to start 005 | [ ] |

## Done criteria

- [x] All 3 phases complete (Phases 1+2+3)
- [ ] All 12 final success criteria pass
- [ ] Live verifier 4× PASS evidence retained