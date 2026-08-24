# Implementation Plan: Tier 7 Phase 3 — Final 5 subtiers

**Feature**: 005-tier7-phase3
**Status**: Ready for tasks

## Approach

Six phases. Phase 0 first (routes.go modularization) so Phase 1-5 can each add their routes without hitting the 400 cap. Then 5 subtier phases (CI/CD, DB Mon, Service Mgmt, Notebook, Team), each end-to-end. Final phase: verify + archive.

## Phase 0 — routes.go split (CRITICAL pre-work)

routes.go is currently 400 LOC at the cap. After Phase 0 it should be ≤200 LOC (extracted middleware + protected group).

Extract:
- `cmd/api-gateway/routes_protected.go` (~200 LOC) — the entire `protected` group setup + middleware + handler registration
- `cmd/api-gateway/routes.go` (~200 LOC) — public + optional + admin groups only

This unblocks all subsequent phases from hitting the 400 cap.

## Phase 1 — CI/CD Visibility (D8)

Same pattern as prior phases. Read spec.md §Phase 1 for details.

## Phase 2 — DB Monitoring (D9)

Read spec.md §Phase 2.

## Phase 3 — Service Management (D10)

Largest scope of this change. 10 routes, 4 tables. Read spec.md §Phase 3.

## Phase 4 — Notebook (D11)

Read spec.md §Phase 4. Use simple textarea + markdown render for the editor (no rich editor complexity).

## Phase 5 — Team/Collab (D12)

Read spec.md §Phase 5. MentionInput uses @-style autocomplete against tenant users.

## Phase 6 — Verify + archive

Same pattern. Tier 7 COMPLETE marker in journal.md.

## Risks

| Risk | Mitigation |
|------|------------|
| routes.go overflow without Phase 0 | Phase 0 must complete first |
| 30 routes = lots of testing | Speckit verifier with parameterized loops |
| Notebook editor complexity | Simple textarea only |
| Incident timeline = complex joins | Use simple denormalized JSON column if needed |

## Done criteria

- [ ] Phase 0: routes.go ≤ 200 LOC, routes_protected.go created
- [ ] All 5 user stories pass
- [ ] Phase 1-5 verifications all green
- [ ] Phase 6 verify-first sweep passes
- [ ] All 30 routes verified live
- [ ] All files under 400 LOC
- [ ] Tier 7 COMPLETE marker in journal
- [ ] Archive created
- [ ] Ready for Tier 8 (Intelligence & Alerting)