# Feature Specification: Tier 7 Phase 3 — CI/CD Vis (D8) + DB Mon (D9) + Service Mgmt (D10) + Notebook (D11) + Team Collab (D12)

**Feature Branch**: `005-tier7-phase3`
**Created**: 2026-08-24
**Status**: Draft
**Source change folder**: `.hermes/changes/005-tier7-phase3/`

**Input**: Ship the final 5 of 11 Tier 7 subtiers.

## Goal

Complete Tier 7 Datadog parity with:
1. **7.7 CI/CD Visibility (D8)** — Pipeline runs, deployments linked to APM services
2. **7.8 DB Monitoring (D9)** — Query performance, slow queries, connection pool stats
3. **7.9 Service Management (D10)** — Incidents, war rooms, postmortems, on-call tasks
4. **7.10 Notebook (D11)** — Collaborative runbooks with markdown
5. **7.11 Team/Collab (D12)** — Shared dashboards, mentions, timeline comments

## Out of scope

This completes Tier 7. Tiers 8-12 are future speckit changes.

## Database additions (migrations)

### Migration 036: CI/CD + DB Mon
- `cicd_pipelines` — id, tenant_id, provider ('github'|'gitlab'|'jenkins'|'circleci'), repo, branch, commit_sha, status, started_at, finished_at, duration_ms
- `cicd_deployments` — id, tenant_id, pipeline_id (FK), service_id (FK to apm_services), environment, version
- `database_queries` — id, tenant_id, database, query_hash, query_text, duration_ms, rows_examined, timestamp. Index on (tenant_id, duration_ms DESC)
- `database_slow_queries` — view or materialized view over database_queries with avg/p95/max
- `connection_pools` — id, tenant_id, database, host, port, pool_size, active, idle, waiting, last_check_at

### Migration 037: Service Management
- `incidents` — id, tenant_id, title, description, severity, status ('open'|'acknowledged'|'resolved'), commander_id (uuid), started_at, resolved_at
- `war_rooms` — id, tenant_id, incident_id (FK), channel_url, participants (jsonb), created_at
- `postmortems` — id, tenant_id, incident_id (FK), content (text), author_id, published_at
- `tasks` — id, tenant_id, incident_id (FK, nullable), title, assignee_id (uuid, nullable), status ('todo'|'in_progress'|'done'), due_at, created_at

### Migration 038: Notebook + Team
- `notebooks` — id, tenant_id, title, content (text — markdown), author_id, last_edited_at, created_at
- `notebook_collaborators` — notebook_id, user_id, role ('viewer'|'editor')
- `dashboard_shares` — id, dashboard_id, user_id, permission ('view'|'edit')
- `timeline_comments` — id, tenant_id, incident_id (FK, nullable), user_id, body (text), created_at
- `mentions` — id, tenant_id, mentioned_user_id, mentioning_user_id, context_type, context_id, read_at

## Backend API additions (~30 routes)

### CI/CD (handlers_cicd.go)
- `GET /api/v1/cicd/pipelines` — list (filter ?repo=X&status=failed)
- `POST /api/v1/cicd/pipelines` — create pipeline record
- `GET /api/v1/cicd/pipelines/:id` — detail
- `GET /api/v1/cicd/deployments` — list (filter ?service_id=X&environment=prod)
- `POST /api/v1/cicd/deployments` — create deployment (links to APM service)
- `POST /api/v1/cicd/webhook/github` — GitHub webhook receiver
- `POST /api/v1/cicd/webhook/gitlab` — GitLab webhook receiver

### DB Monitoring (handlers_dbmon.go)
- `GET /api/v1/database/slow-queries` — top 50 slowest queries
- `GET /api/v1/database/query-explain` — explain plan for a query hash
- `GET /api/v1/database/connection-pool` — current pool stats
- `POST /api/v1/database/query-stats` — ingest query stats (from agent)
- `GET /api/v1/database/queries/top` — most-frequent queries

### Service Management (handlers_servicemgmt.go)
- `GET /api/v1/incidents` — list
- `POST /api/v1/incidents` — create
- `GET /api/v1/incidents/:id` — detail
- `POST /api/v1/incidents/:id/acknowledge` — acknowledge
- `POST /api/v1/incidents/:id/resolve` — resolve
- `POST /api/v1/incidents/:id/war-room` — create war room
- `POST /api/v1/incidents/:id/postmortem` — create postmortem
- `GET /api/v1/incidents/:id/tasks` — list tasks
- `POST /api/v1/incidents/:id/tasks` — create task
- `PUT /api/v1/incidents/:id/tasks/:task_id` — update task

### Notebook (handlers_notebook.go)
- `GET /api/v1/notebooks` — list
- `POST /api/v1/notebooks` — create
- `GET /api/v1/notebooks/:id` — detail
- `PUT /api/v1/notebooks/:id` — update content
- `POST /api/v1/notebooks/:id/collaborators` — add collaborator
- `DELETE /api/v1/notebooks/:id/collaborators/:user_id` — remove

### Team/Collab (handlers_team.go)
- `GET /api/v1/dashboards/shared` — list shared dashboards
- `POST /api/v1/dashboards/share` — share a dashboard with user
- `GET /api/v1/annotations/mentions` — list my mentions
- `POST /api/v1/timeline/comments` — add comment to incident
- `GET /api/v1/timeline/:incident_id/comments` — get comments for incident

## Files this change modifies (estimated)

### Backend (~1500 LOC across 5 handler files)
- 3 migrations (036_cicd_dbmon.sql, 037_servicemgmt.sql, 038_notebook_team.sql)
- 5 handler files (cicd, dbmon, servicemgmt, notebook, team)
- ~30 routes
- routes.go will exceed 400 LOC — will need to extract to `routes_protected.go` (following master plan's modularization from the spec)

### Frontend (~1000 LOC across 5 components + 5 pages)
- 5 shared components: PipelineCard, SlowQueryTable, IncidentCard, NotebookEditor, MentionInput
- 5 pages: CicdPage, DatabasePage, IncidentsPage, NotebookPage, SharedDashboardsPage
- AppSidebar: +5 nav items (lots of new nav)

## Risks

| Risk | Mitigation |
|------|------------|
| routes.go at 400 cap, will need to extract to routes_protected.go | Plan for that early |
| 30 routes is a lot to test | Speckit verifier with parameterized loops |
| Notebook editor complexity | Use simple textarea + markdown render, no rich editor |
| Incident timeline complexity | Reuse existing audit_log if possible |

## Done criteria

- [ ] All 5 user stories pass acceptance scenarios
- [ ] 30 routes registered + verified live
- [ ] 3 migrations applied to prod
- [ ] All files under 400 LOC (split routes.go if needed)
- [ ] Bundle JS gzipped ≤ (current + 80KB)
- [ ] Archive folder created
- [ ] journal.md updated
- [ ] Tier 7 COMPLETE — ready for Tier 8 (Intelligence & Alerting)