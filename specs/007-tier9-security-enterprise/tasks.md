# Tasks: Tier 9 — Security & Enterprise

**Feature**: 007-tier9-security-enterprise
**Spec**: [spec.md](spec.md)
**Status**: Draft

## Phase 0 — Modular pre-work

- [ ] 0.1 Create `cmd/api-gateway/routes_enterprise.go` (~60 LOC) — empty placeholder `mountEnterpriseRoutes()` function
- [ ] 0.2 Modify `cmd/api-gateway/routes_protected.go` — add `mountEnterpriseRoutes(protected, pool)` call at end of `mountProtectedRoutes`
- [ ] 0.3 Verify `go build ./cmd/api-gateway` exit 0
- [ ] 0.4 Verify `go vet ./cmd/api-gateway` exit 0
- [ ] 0.5 Confirm all existing routes still return 401

## Phase 1 — SSO Foundation (Tier 9.1)

- [ ] 1.1 Migration `migrations/040_enterprise.sql` — table `sso_providers` (id, tenant_id, type 'oidc'|'saml', name, config jsonb, enabled default true, created_at). Index on (tenant_id).
- [ ] 1.2 Migration `migrations/040_enterprise.sql` — table `sso_connections` (id, tenant_id, user_id, provider_id FK, subject text NOT NULL, created_at, last_used_at). UNIQUE (provider_id, subject). Index on (user_id).
- [ ] 1.3 Apply migration to prod
- [ ] 1.4 Handler `internal/handler/handlers_sso_types.go` (~60 LOC) — types + allowed providers
- [ ] 1.5 Handler `internal/handler/handlers_sso.go` (~250 LOC, 8 routes):
  - GET /api/v1/enterprise/sso/providers — list providers
  - POST /api/v1/enterprise/sso/providers — create (OIDC: client_id, secret, discovery_url; SAML: metadata_xml or metadata_url)
  - PATCH /api/v1/enterprise/sso/providers/:id — update config
  - DELETE /api/v1/enterprise/sso/providers/:id — soft delete (enabled=false)
  - GET /api/v1/enterprise/sso/initiate?provider_id=X — start OIDC/SAML flow (redirects to IdP)
  - GET /api/v1/enterprise/sso/callback?code=X&state=Y — OIDC callback (exchange code for token, identify user, JIT create if needed, issue JWT)
  - POST /api/v1/enterprise/sso/callback — SAML POST binding (validate signature, audience, NotOnOrAfter)
  - GET /api/v1/enterprise/sso/connections — list caller's SSO connections
  - POST /api/v1/enterprise/sso/test — test provider config (fetch discovery_url or SAML metadata)
- [ ] 1.6 Register 8 routes via `mountEnterpriseRoutes`
- [ ] 1.7 Shared `web/src/components/shared/SsoProviderCard.tsx` (~80 LOC):
  - Props: provider, onEdit, onDelete, onTest
  - Layout: card with provider name + type badge + status (enabled/disabled) + actions
  - Click expand for OIDC config (client_id, discovery_url) or SAML config (metadata_url, entity_id)
- [ ] 1.8 Create `web/src/components/SsoSection.tsx` (~200 LOC):
  - Header: "Single Sign-On"
  - Provider list (cards)
  - "+ Add provider" button → form modal (type select, name, OIDC/SAML-specific fields)
- [ ] 1.9 Verify all gates green (go + npm)
- [ ] 1.10 Commit Phase 1

## Phase 2 — SCIM Provisioning (Tier 9.2)

- [ ] 2.1 Migration 040 — table `scim_tokens` (id, tenant_id, name, token_hash UNIQUE, scopes text[] default '{}', expires_at, last_used_at, created_at). Index on (tenant_id).
- [ ] 2.2 Migration 040 — table `scim_sync_log` (id, tenant_id, op 'create'|'update'|'delete', external_id, status 'success'|'error', error_message, created_at). Index on (tenant_id, created_at DESC).
- [ ] 2.3 Apply migration
- [ ] 2.4 Handler `handlers_scim_types.go` (~60 LOC) — SCIM payload types
- [ ] 2.5 Handler `handlers_scim.go` (~200 LOC, 6 routes):
  - GET /api/v1/enterprise/scim/tokens — list tokens (no plaintext)
  - POST /api/v1/enterprise/scim/tokens — create (returns plaintext ONCE)
  - DELETE /api/v1/enterprise/scim/tokens/:id — revoke
  - GET /api/v1/enterprise/scim/sync-log — recent operations (filter ?since=X)
  - **SCIM endpoints** (separate path, public with token auth):
    - POST /scim/v2/Users — create user
    - PUT /scim/v2/Users/:id — update user
    - DELETE /scim/v2/Users/:id — soft-disable user
    - GET /scim/v2/Users — list
    - POST /scim/v2/Groups — create group (optional)
- [ ] 2.6 Register 4 enterprise routes + 5 SCIM routes
- [ ] 2.7 Shared `web/src/components/shared/ScimTokenTable.tsx` (~80 LOC):
  - Props: tokens, onRevoke
  - Table: name | scopes | last_used_at | expires_at | actions
  - "Create token" button → modal showing plaintext once
- [ ] 2.8 Add `web/src/components/ScimSection.tsx` (~200 LOC) to EnterprisePage
- [ ] 2.9 Verify all gates green
- [ ] 2.10 Commit Phase 2

## Phase 3 — Advanced RBAC (Tier 9.3)

- [ ] 3.1 Migration 040 — table `rbac_roles` (id, tenant_id, name UNIQUE per tenant, description, permissions text[] default '{}', is_builtin bool default false, created_at). Index on (tenant_id).
- [ ] 3.2 Migration 040 — table `user_role_assignments` (id, tenant_id, user_id, role_id FK CASCADE, assigned_at). UNIQUE (user_id, role_id).
- [ ] 3.3 Apply migration
- [ ] 3.4 Handler `handlers_rbac_types.go` (~60 LOC) — role types + permission allowlist
- [ ] 3.5 Handler `handlers_rbac.go` (~250 LOC, 6 routes):
  - GET /api/v1/enterprise/rbac/roles — list (built-in + custom)
  - POST /api/v1/enterprise/rbac/roles — create custom role
  - PATCH /api/v1/enterprise/rbac/roles/:id — update permissions
  - DELETE /api/v1/enterprise/rbac/roles/:id — delete (only custom; built-in protected)
  - POST /api/v1/enterprise/rbac/users/:user_id/roles — assign role (body: {role_id})
  - DELETE /api/v1/enterprise/rbac/users/:user_id/roles/:role_id — unassign
  - GET /api/v1/enterprise/rbac/check?permission=X&user_id=Y — check if user has permission
- [ ] 3.6 Register 6 routes
- [ ] 3.7 Shared `web/src/components/shared/RbacRoleEditor.tsx` (~80 LOC):
  - Props: role, onSave
  - Permission multi-select (grouped: servers:*, dashboards:*, alerts:*, etc.)
  - Built-in roles are read-only
- [ ] 3.8 Add `web/src/components/RbacSection.tsx` (~200 LOC) to EnterprisePage
- [ ] 3.9 Verify all gates green
- [ ] 3.10 Commit Phase 3

## Phase 4 — Audit Log Retention + Export (Tier 9.4)

- [ ] 4.1 Migration 040 — table `audit_log_archive` (id, tenant_id, batch_id text UNIQUE, event_count int, period_start, period_end, compressed_payload bytea, size_bytes bigint, created_at). Index on (tenant_id, created_at DESC).
- [ ] 4.2 Apply migration
- [ ] 4.3 Handler `handlers_audit_archive_types.go` (~60 LOC) — archive types
- [ ] 4.4 Handler `handlers_audit_archive.go` (~200 LOC, 4 routes):
  - GET /api/v1/enterprise/audit/archives — list archives (filter ?since=X)
  - POST /api/v1/enterprise/audit/archive — create archive job (body: {period_start, period_end}). Returns job_id. Background goroutine compresses and inserts.
  - GET /api/v1/enterprise/audit/archives/:id/download — download archive (gzip + Content-Disposition)
  - POST /api/v1/enterprise/audit/export — query + return CSV (body: {event_type, start_time, end_time, format: 'csv'|'json'})
- [ ] 4.5 Register 4 routes
- [ ] 4.6 Shared `web/src/components/shared/AuditArchiveCard.tsx` (~80 LOC):
  - Props: archive, onDownload
  - Card: period range | event count | size | download button
- [ ] 4.7 Add `web/src/components/AuditSection.tsx` (~200 LOC) to EnterprisePage
- [ ] 4.8 Verify all gates green
- [ ] 4.9 Commit Phase 4

## Phase 5 — Compliance Reports (Tier 9.5)

- [ ] 5.1 Migration 040 — table `compliance_reports` (id, tenant_id, framework 'soc2'|'iso27001'|'hipaa', period_start, period_end, status 'pending'|'running'|'completed'|'failed', artifact_url, scheduled_id nullable, created_at, completed_at). Index on (tenant_id, framework, created_at DESC).
- [ ] 5.2 Migration 040 — table `compliance_schedules` (id, tenant_id, framework, frequency 'monthly'|'quarterly'|'yearly', recipients text[], enabled default true, next_run_at, created_at).
- [ ] 5.3 Apply migration
- [ ] 5.4 Handler `handlers_compliance_types.go` (~60 LOC) — report types
- [ ] 5.5 Handler `handlers_compliance.go` (~200 LOC, 5 routes):
  - GET /api/v1/enterprise/compliance/reports — list reports
  - POST /api/v1/enterprise/compliance/reports — generate (body: {framework, period_start, period_end}). Background goroutine assembles PDF.
  - GET /api/v1/enterprise/compliance/reports/:id — status
  - GET /api/v1/enterprise/compliance/reports/:id/download — download PDF
  - POST /api/v1/enterprise/compliance/schedule — create recurring schedule (body: {framework, frequency, recipients})
  - GET /api/v1/enterprise/compliance/schedules — list schedules
  - DELETE /api/v1/enterprise/compliance/schedules/:id — delete
- [ ] 5.6 Register 5 routes
- [ ] 5.7 Shared `web/src/components/shared/ComplianceReportCard.tsx` (~80 LOC):
  - Props: report, onDownload
  - Card: framework | period | status | download button
- [ ] 5.8 Add `web/src/components/ComplianceSection.tsx` (~200 LOC) to EnterprisePage
- [ ] 5.9 Verify all gates green
- [ ] 5.10 Commit Phase 5

## Phase 6 — Enterprise Tenants + Org Settings (Tier 9.6) — Final feature phase

- [ ] 6.1 Migration 040 — `ALTER TABLE tenants ADD COLUMN IF NOT EXISTS parent_org_id uuid REFERENCES tenants(id) ON DELETE SET NULL` (idempotent)
- [ ] 6.2 Migration 040 — `ALTER TABLE tenants ADD COLUMN IF NOT EXISTS settings jsonb NOT NULL DEFAULT '{}'` (idempotent)
- [ ] 6.3 Apply migration
- [ ] 6.4 Handler `handlers_enterprise_tenants.go` (~150 LOC, 3 routes):
  - GET /api/v1/enterprise/orgs — list org hierarchy (parent + children)
  - POST /api/v1/enterprise/orgs — create sub-org (body: {name, parent_org_id, settings})
  - PATCH /api/v1/enterprise/orgs/:id/settings — update settings
- [ ] 6.5 Register 3 routes
- [ ] 6.6 Shared `web/src/components/shared/OrgSettingsForm.tsx` (~80 LOC):
  - Props: org, onSave
  - Theme selector + default dashboard + default landing page
- [ ] 6.7 Build unified `web/src/pages/EnterprisePage.tsx` (~150 LOC, 6 tabs: SSO / SCIM / RBAC / Audit / Compliance / Orgs)
- [ ] 6.8 AppSidebar: add "Enterprise" link under operations section
- [ ] 6.9 Verify all gates green
- [ ] 6.10 Commit Phase 6

## Phase 7 — Final verify + deploy + archive

- [ ] 7.1 Final go build + npm run type-check + lint + build all green
- [ ] 7.2 Bundle JS gzipped ≤ baseline + 90 KB
- [ ] 7.3 Deploy binary to `.115` (use skill `stackwatch-build-deploy`)
- [ ] 7.4 Live 4× verifier PASS (curl all 32 routes, expect 401)
- [ ] 7.5 Archive to `.hermes/changes/archive/2026-08-24-007-tier9-security-enterprise/`
- [ ] 7.6 Update journal.md with TIER 9 COMPLETE marker
- [ ] 7.7 Tier 9 ready for Tier 10 (Homelab Dashboard)

## Final success criteria

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 6 user stories pass acceptance scenarios | [ ] |
| SC-002 | 32 routes registered + return correct status codes | [ ] |
| SC-003 | Migration 040 applied to prod | [ ] |
| SC-004 | go build + npm run type-check + lint + build exit 0 | [ ] |
| SC-005 | Bundle JS gzipped ≤ baseline + 90 KB | [ ] |
| SC-006 | Live 4× verifier PASS | [ ] |
| SC-007 | All files under 400 LOC | [ ] |
| SC-008 | Archive folder created | [ ] |
| SC-009 | journal.md updated with TIER 9 COMPLETE marker | [ ] |
| SC-010 | Tier 9 ready for Tier 10 (Homelab Dashboard) | [ ] |