# Implementation Plan: Tier 9 — Security & Enterprise

**Feature**: 007-tier9-security-enterprise
**Status**: Draft

## Approach

Seven phases. Phase 0 first (modular split + foundation) so Phase 1-6 can each add their routes without hitting the 400 cap. Then 6 subtier phases (SSO, SCIM, RBAC, Audit, Compliance, Enterprise Tenants), each end-to-end. Final phase: verify + archive.

## Phase 0 — Modular pre-work

**Why**: routes_protected.go is currently 396 LOC. Adding 32 new routes would push it past 500 LOC. We need a split BEFORE Phase 1.

Tasks:
- 0.1 Create `cmd/api-gateway/routes_enterprise.go` (~60 LOC) — empty placeholder `mountEnterpriseRoutes()` function that returns nil. We'll add the actual route registrations in Phase 1+ as we go.
- 0.2 Modify `cmd/api-gateway/routes_protected.go` — add a single line call `mountEnterpriseRoutes(protected, pool)` at the end of `mountProtectedRoutes`.
- 0.3 Verify `go build ./cmd/api-gateway` and `go vet` still exit 0
- 0.4 Live check: existing routes still return 401.

After Phase 0: routes_protected.go grows slightly (1 line call); routes_enterprise.go is empty waiting for Phase 1.

## Phase 1 — SSO Foundation (Tier 9.1)

Tasks:
1.1 Migration `migrations/040_enterprise.sql` — `sso_providers` + `sso_connections` tables
1.2 Apply migration to prod
1.3 Handlers split into 2 files:
   - `handlers_sso.go` (~250 LOC, 8 routes)
   - `handlers_sso_types.go` (~60 LOC)
1.4 Register 8 routes via `mountEnterpriseRoutes`
1.5 Shared `SsoProviderCard.tsx` (~80 LOC, Datadog-style)
1.6 Add SSO section to EnterprisePage (Phase 6 will unify)
1.7 Verify + commit Phase 1

## Phase 2 — SCIM Provisioning (Tier 9.2)

Tasks:
2.1 Migration 040 — `scim_tokens` + `scim_sync_log` tables
2.2 Handlers:
   - `handlers_scim.go` (~200 LOC, 6 routes)
   - `handlers_scim_types.go` (~60 LOC)
2.3 Register 6 routes
2.4 Shared `ScimTokenTable.tsx` (~80 LOC)
2.5 Add SCIM section to EnterprisePage
2.6 Verify + commit Phase 2

## Phase 3 — Advanced RBAC (Tier 9.3)

Tasks:
3.1 Migration 040 — `rbac_roles` table
3.2 Handlers:
   - `handlers_rbac.go` (~250 LOC, 6 routes)
   - `handlers_rbac_types.go` (~60 LOC)
3.3 Register 6 routes
3.4 Shared `RbacRoleEditor.tsx` (~80 LOC)
3.5 Add RBAC section to EnterprisePage
3.6 Verify + commit Phase 3

## Phase 4 — Audit Log Retention + Export (Tier 9.4)

Tasks:
4.1 Migration 040 — `audit_log_archive` table
4.2 Handlers:
   - `handlers_audit_archive.go` (~200 LOC, 4 routes)
   - `handlers_audit_archive_types.go` (~60 LOC)
4.3 Register 4 routes
4.4 Shared `AuditArchiveCard.tsx` (~80 LOC)
4.5 Add audit section to EnterprisePage
4.6 Verify + commit Phase 4

## Phase 5 — Compliance Reports (Tier 9.5)

Tasks:
5.1 Migration 040 — `compliance_reports` table
5.2 Handlers:
   - `handlers_compliance.go` (~200 LOC, 5 routes)
   - `handlers_compliance_types.go` (~60 LOC)
5.3 Register 5 routes
5.4 Shared `ComplianceReportCard.tsx` (~80 LOC)
5.5 Add compliance section to EnterprisePage
5.6 Verify + commit Phase 5

## Phase 6 — Enterprise Tenants + Org Settings (Tier 9.6)

Tasks:
6.1 Migration 040 — `parent_org_id` column added to `tenants` table (idempotent ALTER)
6.2 Handlers:
   - `handlers_enterprise_tenants.go` (~150 LOC, 3 routes)
   - `handlers_enterprise_tenants_types.go` (~50 LOC)
6.3 Register 3 routes
6.4 Shared `OrgSettingsForm.tsx` (~80 LOC)
6.5 Final unify: EnterprisePage becomes 6-tab dashboard (SSO / SCIM / RBAC / Audit / Compliance / Orgs)
6.6 Add Enterprise link to AppSidebar
6.7 Verify + commit Phase 6

## Phase 7 — Final verify + deploy + archive

Tasks:
7.1 Final go build + npm run type-check + lint + build all green
7.2 Bundle JS gzipped ≤ baseline + 90 KB
7.3 Deploy binary to `.115` (use skill `stackwatch-build-deploy`)
7.4 Live 4× verifier PASS (curl all 32 routes, expect 401)
7.5 Archive to `.hermes/changes/archive/2026-08-24-007-tier9-security-enterprise/`
7.6 Update journal.md with TIER 9 COMPLETE marker
7.7 Tier 9 ready for Tier 10 (Homelab Dashboard)

## Risks

| Risk | Mitigation |
|------|------------|
| routes_protected.go overflow | Phase 0 split (routes_enterprise.go) BEFORE Phase 1 starts |
| SAML/OIDC complexity | Sign state with HMAC; manual JWT validation for OIDC |
| SCIM spec is verbose | Minimum subset (Users + Groups CRUD); extend later |
| RBAC check could be slow | Cache effective permissions per user (in-memory TTL) |
| Audit archive size | Compress with gzip; chunk into 100k-event files |
| Compliance reports access | Inherit from caller's tenant + role |

## Done criteria

- [ ] Phase 0: routes_enterprise.go created, placeholder ready
- [ ] Phases 1-6: each subtier passes verification + commits cleanly
- [ ] All 32 routes registered
- [ ] All files < 400 LOC
- [ ] Tier 9 COMPLETE marker in journal
- [ ] Archive created
- [ ] Ready for Tier 10 (Homelab Dashboard)