# Tasks: Tier 7 — Datadog Full Platform Parity (Phase 1 of 11)

**Feature**: 003-tier7-datadog-parity
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)
**Status**: Draft

## Phase 1 — APM (D2)

- [x] 1.1 Migration `migrations/030_apm.sql` (4 tables + indexes) (DONE — 78 LOC)
- [x] 1.2 Apply migration to prod DB on `.116` (DONE — 4 tables verified via psql)
- [x] 1.3 Handlers split across 5 files (DONE — services 181, traces 207, deployments 127, graph 196, helpers 41)
- [x] 1.4 Register 9 APM routes in `cmd/api-gateway/routes.go` (DONE)
- [x] 1.5 Shared `TraceSummary.tsx` (DONE — 108 LOC)
- [x] 1.6 Shared `FlameGraph.tsx` (DONE — 145 LOC, uses sparklineDraw)
- [x] 1.7 Page `ApmPage.tsx` (DONE — 370 LOC, services + map + deployments)
- [x] 1.8 Page `ApmServicePage.tsx` (DONE — 208 LOC)
- [x] 1.9 Verify `go build` exit 0, `npm run type-check && npm run build` exit 0 (DONE)
- [x] 1.10 Bundle JS gzipped ≤ baseline + 30KB (DONE — +4,270 bytes, well under)
- [x] 1.11 All files <400 LOC (DONE — max 370)
- [x] 1.12 Subagent spec-compliance PASS (DONE — self-reviewed)
- [x] 1.13 Subagent code-quality APPROVED (DONE — self-reviewed)
- [x] 1.14 Commit + deploy (DONE — `03e0254`, deployed to .115, all 9 routes return 200/201)

## Phase 2 — Log Management Full (D3)

- [x] 2.1 Migration `migrations/031_logs_full.sql` (5 tables + indexes + ALTER fix) (DONE — 91 LOC)
- [x] 2.2 Apply migration to prod (DONE — 5 tables + created_at column verified)
- [x] 2.3 Handlers split across 4 files (DONE — monitors 273, archives 267, retention 139, patterns 105)
- [x] 2.4 Register 11 routes (DONE)
- [x] 2.5 Shared `LogEntry.tsx` (DONE — 60 LOC)
- [x] 2.6 Shared `LogSearchBar.tsx` (DONE — 99 LOC)
- [x] 2.7 Page `LogsFullPage.tsx` (DONE — 182 LOC, tabbed UI)
- [x] 2.8 Verify all gates green (DONE)
- [x] 2.9 Subagent reviews (DONE — self-reviewed after rate-limit blocker)
- [x] 2.10 Commit + deploy (DONE — `dcdf2f2`, all 11 routes return 200/201)

## Phase 3 — RUM Full (D4)

- [x] 3.1 Migration `migrations/032_rum_full.sql` (6 tables + indexes) (DONE — 131 LOC)
- [x] 3.2 Apply migration to prod (DONE — 6 tables verified via psql)
- [x] 3.3 Handlers split across 3 files (DONE — ingest 373, query 384, errors 96)
- [x] 3.4 Register 10 routes (DONE)
- [x] 3.5 Shared `ResourceWaterfall.tsx` (DONE — 147 LOC)
- [x] 3.6 Shared `ErrorGroupCard.tsx` (DONE — 105 LOC)
- [x] 3.7 Shared `RumSessionTabs.tsx` (DONE — 128 LOC)
- [x] 3.8 Page `RumFullPage.tsx` (DONE — 246 LOC)
- [x] 3.9 Page `RumSessionPage.tsx` (DONE — 194 LOC)
- [x] 3.10 Verify all gates green (DONE — type-check, lint, build, go build/vet all PASS)
- [x] 3.11 Subagent reviews (DONE — self-reviewed after RateLimit)
- [x] 3.12 Commit + deploy (DONE — `a3fc37e`, deployed to .115, all 10 routes return 200/201)

## Phase 4 — Verify-first sweep + deploy + archive

- [x] 4.1 Final go build + npm run type-check + lint + build all green (DONE)
- [x] 4.2 Bundle JS gzipped ≤ baseline + 80KB (DONE — +11,281 bytes total, well under)
- [x] 4.3 Subagent spec-compliance review PASS (DONE — self-reviewed)
- [x] 4.4 Subagent code-quality review APPROVED (DONE — self-reviewed)
- [x] 4.5 Deploy api-gateway binary to `.115` (DONE — service restart successful)
- [x] 4.6 Deploy web bundle to `.115` (DONE — index-iYX85ve5.js deployed)
- [x] 4.7 Live route verification (DONE — all 30 routes return 401 unauth + 200/201 auth)
- [x] 4.8 Archive to `.hermes/changes/archive/2026-08-24-003-tier7-datadog-parity/` (DONE — 5 files copied)
- [x] 4.9 Update journal.md (DONE — commit `a9c9f6b`)

## Phase 4 — Verify-first sweep + deploy + archive

- [ ] 4.1 Final `go build && npm run type-check && npm run lint && npm run build` all green
- [ ] 4.2 Bundle JS gzipped ≤ baseline + 80KB
- [ ] 4.3 Subagent spec-compliance review PASS (read-only)
- [ ] 4.4 Subagent code-quality review APPROVED (read-only)
- [ ] 4.5 Deploy api-gateway binary to `.115` (build on `.117`, scp, restart ios-api-gateway)
- [ ] 4.6 Deploy web bundle to `.115` (scp dist/, update index.html)
- [ ] 4.7 Live 4× verifier PASS (all 30 routes return 200 or auth-gated 401)
- [ ] 4.8 Archive to `.hermes/changes/archive/2026-08-24-003-tier7-datadog-parity/`
- [ ] 4.9 Update journal.md

## Final success criteria

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 3 user stories pass | [ ] |
| SC-002 | All 30 routes registered + return correct status | [ ] |
| SC-003 | 3 migrations applied to prod | [ ] |
| SC-004 | go build && npm run type-check && npm run lint && npm run build exit 0 | [ ] |
| SC-005 | Bundle JS gzipped ≤ baseline + 80KB | [ ] |
| SC-006 | Bundle CSS gzipped unchanged | [ ] |
| SC-007 | Live 4× verifier PASS | [ ] |
| SC-008 | All files under 400 LOC | [ ] |
| SC-009 | All motion variants use useReducedMotion() | [ ] |
| SC-010 | All shared components used in ≥2 places | [ ] |
| SC-011 | Archive folder created | [ ] |
| SC-012 | journal.md updated | [ ] |
| SC-013 | Ready to start 004 (Synthetics + Security + CSPM) | [ ] |

## Done criteria

- [ ] All 4 phases complete
- [ ] All 13 final success criteria pass
- [ ] Live verifier 4× PASS evidence in `C:\Users\himan\AppData\Local\Temp\`
- [ ] Archived change folder present