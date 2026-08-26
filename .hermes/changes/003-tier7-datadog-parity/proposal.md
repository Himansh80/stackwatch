# Proposal: Tier 7 — Datadog Full Platform Parity (Phase 1 of 11)

**Change folder**: `.hermes/changes/003-tier7-datadog-parity/`
**Spec folder**: `specs/003-tier7-datadog-parity/`
**Created**: 2026-08-24
**Status**: Draft → Ready for apply

## Why

StackWatch today has Tiers 0-6 (foundation, Proxmox, TrueNAS, remote access, server admin, containers, monitoring depth). Per the master plan, Tier 7 is Datadog Full Platform parity — the competitive moat. We need this for the sell-to-market position.

Phase 1 of Tier 7 ships 3 of 11 subtiers (the foundation trace + log + RUM models). Without these, every subsequent Tier 7 feature (DB monitoring, CI/CD visibility, Service Mgmt, Notebook, Team Collab) is crippled.

## What changes

### Backend additions (no removals, no refactors of existing code)
- 3 new migrations: `030_apm.sql`, `031_logs_full.sql`, `032_rum_full.sql`
- 3 new handler files: `handlers_apm.go`, `handlers_logs_full.go`, `handlers_rum_full.go`
- ~30 new routes registered in `cmd/api-gateway/routes.go`

### Frontend additions (no removals)
- 6 new shared components in `web/src/components/shared/`
- 5 new pages: `ApmPage`, `ApmServicePage`, `LogsFullPage`, `RumFullPage`, `RumSessionPage`

### Bundle impact
- Estimated +60-80KB gzipped JS (3 large handlers + 5 pages + 6 components)
- CSS unchanged (reuse existing tokens + motion)

## Impact

| Area | Impact |
|---|---|
| Backend | +3 migrations, +3 handlers, +30 routes, ~1050 LOC |
| Database | +9 new tables |
| Frontend | +11 files (6 shared + 5 pages), ~1730 LOC |
| Build | +60-80KB JS gzipped max |
| Tests | All existing tests must still pass; new tests per handler |
| Docs | journal.md updated; archive folder created |
| Breaking changes | None — purely additive |

## Scope discipline

This change is **Phase 1 of Tier 7 only**. 3 of 11 subtiers ship here. The remaining 8 subtiers (Synthetics Full, CSPM, CI/CD Vis, DB Monitoring, Service Management, Notebook, Team/Collab) ship in future speckit changes 004-007.

Any subagent that adds features outside the 3 declared subtiers (APM, Log Mgmt Full, RUM Full) is out of scope.

## Done when

- [ ] All 3 user stories pass acceptance criteria
- [ ] All migrations applied to production DB on `.116`
- [ ] All 30 routes return 200 (or appropriate auth-gated 401)
- [ ] 4× back-to-back live verifier PASS (with seeded super_admin)
- [ ] Bundle JS gzipped ≤ baseline + 80KB
- [ ] All files under 400 LOC
- [ ] All motion variants use `useReducedMotion()`
- [ ] Archive folder created at `.hermes/changes/archive/2026-08-24-003-tier7-datadog-parity/`
- [ ] journal.md updated with session entry
- [ ] Ready to start 004 (Tier 7.4-7.6: Synthetics + Security + CSPM)