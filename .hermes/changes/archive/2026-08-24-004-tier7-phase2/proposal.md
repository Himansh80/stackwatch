# Proposal: Tier 7 Phase 2 — Synthetics Full (D5) + Security (D6) + CSPM (D7)

**Change folder**: `.hermes/changes/004-tier7-phase2/`
**Spec folder**: `specs/004-tier7-phase2/`
**Created**: 2026-08-24
**Status**: Draft → Ready for apply

## Why

Tier 7 = Datadog parity. Phase 1 (003-tier7-datadog-parity) shipped 3 of 11 subtiers: APM (D2), Log Mgmt (D3), RUM (D4). Phase 2 ships the next 3: Synthetics Full, Security, CSPM.

Without these, StackWatch can only react to problems that already happened. Synthetics = proactive monitoring. Security = compliance + threat detection. CSPM = cloud config drift detection.

## What changes

### Backend additions (no removals, no refactors)
- 3 migrations: 033_synthetics.sql, 034_security.sql, 035_cspm.sql
- 4 handler files: synthetics, security, cspm, synthetics_runner (background test executor)
- ~19 new routes registered

### Frontend additions
- 4 shared components: SlaBadge, ThreatCard, ComplianceBar, CspmSeverityBadge
- 3 pages: SyntheticsPage, SecurityPage, CspmPage
- 3 new sidebar nav items

### Bundle impact
- Estimated +50-60KB gzipped JS (3 medium handlers + 3 pages + 4 components)

## Impact

| Area | Impact |
|---|---|
| Backend | +3 migrations, +4 handlers, +19 routes, ~1100 LOC |
| Database | +8 new tables |
| Frontend | +7 files, ~700 LOC |
| Build | +50-60KB JS gzipped max |
| Tests | All existing tests must still pass |
| Docs | journal.md updated; archive folder created |
| Breaking changes | None — purely additive |

## Scope discipline

This change is **Tier 7 Phase 2 only**. 3 of 11 subtiers ship here. The remaining 5 subtiers (CI/CD Vis, DB Mon, Service Mgmt, Notebook, Team Collab) ship in future speckit changes 005-008.

Any subagent that adds features outside the 3 declared subtiers (Synthetics Full, Security, CSPM) is out of scope.

## Done when

- [x] All 3 user stories pass acceptance criteria
- [x] All migrations applied to production DB on `.116`
- [x] All 19 routes return 200 (or appropriate auth-gated 401)
- [ ] 4× back-to-back live verifier PASS
- [ ] Bundle JS gzipped ≤ baseline + 60KB
- [x] All files under 400 LOC
- [x] All motion variants use `useReducedMotion()`
- [ ] Archive folder created at `.hermes/changes/archive/2026-08-24-004-tier7-phase2/`
- [x] journal.md updated
- [ ] Ready to start 005