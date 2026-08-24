# Proposal: Tier 7 Phase 3 — CI/CD Vis + DB Mon + Service Mgmt + Notebook + Team Collab

**Change folder**: `.hermes/changes/005-tier7-phase3/`
**Spec folder**: `specs/005-tier7-phase3/`
**Created**: 2026-08-24
**Status**: Draft

## Why

This is the final Phase 3 of Tier 7 (Datadog Full Platform Parity). After this ships, Tier 7 is COMPLETE — StackWatch has all 11 Tier 7 subtiers matching Datadog's feature surface.

## What changes

### Backend additions (no removals)
- 3 migrations: 036_cicd_dbmon, 037_servicemgmt, 038_notebook_team
- 5 handler files (cicd, dbmon, servicemgmt, notebook, team)
- ~30 new routes
- routes.go will overflow 400 cap → extract part of route registration to `routes_protected.go` (following the master plan's modularization from the spec)

### Frontend additions
- 5 shared components (PipelineCard, SlowQueryTable, IncidentCard, NotebookEditor, MentionInput)
- 5 pages (CicdPage, DatabasePage, IncidentsPage, NotebookPage, SharedDashboardsPage)
- 5 new sidebar nav items

### Bundle impact
- Estimated +70-80KB gzipped JS

## Impact

| Area | Impact |
|---|---|
| Backend | +3 migrations, +5 handlers, +30 routes, ~1500 LOC |
| Database | +12 new tables |
| Frontend | +10 files, ~1000 LOC |
| Build | +70-80KB JS gzipped max |
| Tests | All existing tests must still pass |
| Docs | journal.md updated; archive folder created |
| Breaking changes | None — purely additive |

## Scope discipline

This is the FINAL phase of Tier 7. No further Tier 7 subtiers after this. Tiers 8-12 are future changes.

Any subagent that adds features outside the 5 declared subtiers (CI/CD Vis, DB Mon, Service Mgmt, Notebook, Team Collab) is out of scope.

## Done when

- [ ] All 5 user stories pass acceptance criteria
- [ ] All migrations applied to prod
- [ ] All 30 routes return 200/401
- [ ] Bundle JS gzipped ≤ baseline + 80KB
- [ ] All files under 400 LOC (split routes.go if needed)
- [ ] Archive folder created
- [ ] journal.md updated
- [ ] Tier 7 COMPLETE — ready to start Tier 8 (Intelligence & Alerting)