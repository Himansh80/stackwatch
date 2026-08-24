# Tasks: Tier 7 Phase 3 — CI/CD Vis + DB Mon + Service Mgmt + Notebook + Team Collab

**Feature**: 005-tier7-phase3
**Spec**: [spec.md](spec.md)
**Status**: Draft

## Phase 0 — Pre-work: routes.go modularization
- [ ] 0.1 Extract part of routes.go to `cmd/api-gateway/routes_protected.go` (the protected group + its middleware setup)
- [ ] 0.2 Verify routes.go ≤ 400 LOC after split
- [ ] 0.3 Verify all existing routes still work

## Phase 1 — CI/CD Visibility (D8)
- [ ] 1.1 Migration `migrations/036_cicd_dbmon.sql` (2 tables: cicd_pipelines, cicd_deployments) — split into 036_cicd.sql + 036_dbmon.sql actually
- [ ] 1.2 Apply migration to prod
- [ ] 1.3 Handlers `handlers_cicd.go` (~250 LOC, 7 routes)
- [ ] 1.4 Register 7 CI/CD routes
- [ ] 1.5 Shared `PipelineCard.tsx` (~80 LOC)
- [ ] 1.6 Page `CicdPage.tsx` (~150 LOC)
- [ ] 1.7 Verify + commit

## Phase 2 — DB Monitoring (D9)
- [ ] 2.1 Migration `migrations/036_dbmon.sql` (3 tables: database_queries, connection_pools, slow_queries view)
- [ ] 2.2 Handlers `handlers_dbmon.go` (~200 LOC, 5 routes)
- [ ] 2.3 Register 5 DB mon routes
- [ ] 2.4 Shared `SlowQueryTable.tsx` (~80 LOC)
- [ ] 2.5 Page `DatabasePage.tsx` (~150 LOC)
- [ ] 2.6 Verify + commit

## Phase 3 — Service Management (D10)
- [ ] 3.1 Migration `migrations/037_servicemgmt.sql` (4 tables: incidents, war_rooms, postmortems, tasks)
- [ ] 3.2 Handlers `handlers_servicemgmt.go` (~350 LOC, 10 routes)
- [ ] 3.3 Register 10 routes
- [ ] 3.4 Shared `IncidentCard.tsx` (~80 LOC)
- [ ] 3.5 Page `IncidentsPage.tsx` (~180 LOC)
- [ ] 3.6 Verify + commit

## Phase 4 — Notebook (D11)
- [ ] 4.1 Migration `migrations/038_notebook.sql` (2 tables: notebooks, notebook_collaborators)
- [ ] 4.2 Handlers `handlers_notebook.go` (~200 LOC, 6 routes)
- [ ] 4.3 Register 6 routes
- [ ] 4.4 Shared `NotebookEditor.tsx` (~120 LOC, simple textarea + markdown render)
- [ ] 4.5 Page `NotebookPage.tsx` (~180 LOC)
- [ ] 4.6 Verify + commit

## Phase 5 — Team/Collab (D12)
- [ ] 5.1 Migration `migrations/038_team.sql` (3 tables: dashboard_shares, timeline_comments, mentions)
- [ ] 5.2 Handlers `handlers_team.go` (~200 LOC, 5 routes)
- [ ] 5.3 Register 5 routes
- [ ] 5.4 Shared `MentionInput.tsx` (~100 LOC)
- [ ] 5.5 Page `SharedDashboardsPage.tsx` (~150 LOC)
- [ ] 5.6 Verify + commit

## Phase 6 — Final verify + deploy + archive
- [ ] 6.1 Final go build + npm run type-check + lint + build all green
- [ ] 6.2 Bundle JS gzipped ≤ baseline + 80KB
- [ ] 6.3 Subagent spec-compliance review
- [ ] 6.4 Subagent code-quality review
- [ ] 6.5 Deploy binary + bundle to .115
- [ ] 6.6 Live 4× verifier PASS
- [ ] 6.7 Archive to .hermes/changes/archive/2026-08-24-005-tier7-phase3/
- [ ] 6.8 Update journal.md
- [ ] 6.9 Tier 7 COMPLETE — ready to start 006 (Tier 8)

## Final success criteria

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 5 user stories pass | [ ] |
| SC-002 | All 30 routes registered + return correct status | [ ] |
| SC-003 | 3 migrations applied to prod | [ ] |
| SC-004 | go build + npm run type-check + lint + build exit 0 | [ ] |
| SC-005 | Bundle JS gzipped ≤ baseline + 80KB | [ ] |
| SC-006 | Live 4× verifier PASS | [ ] |
| SC-007 | All files under 400 LOC (including routes.go + routes_protected.go) | [ ] |
| SC-008 | All shared components used in ≥2 places | [ ] |
| SC-009 | Archive folder created | [ ] |
| SC-010 | journal.md updated | [ ] |
| SC-011 | Tier 7 COMPLETE — ready for Tier 8 | [ ] |