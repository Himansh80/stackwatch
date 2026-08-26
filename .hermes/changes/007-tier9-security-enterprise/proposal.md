# Proposal: Tier 9 — Security & Enterprise

**Change folder**: `.hermes/changes/007-tier9-security-enterprise/`
**Spec folder**: `specs/007-tier9-security-enterprise/`
**Created**: 2026-08-24
**Status**: Draft

## Why

Tiers 0-8 shipped core infrastructure: monitoring (T1-6), observability (T7), and intelligence (T8). Tier 9 adds the **enterprise layer** that large organizations require:

- SSO so employees use their corporate identity (Okta, Azure AD, Google Workspace) instead of separate passwords
- SCIM provisioning so IT can sync users from their IdP automatically
- Advanced RBAC so admins can define custom roles with fine-grained permissions
- Audit log retention so SOC2/ISO27001/HIPAA auditors can prove who did what
- Compliance reports that generate evidence packages for auditors
- Enterprise tenant management for organizations with multiple business units

Per MASTER_BUILD_PLAN.md: "Tier 9 — Security & Enterprise (2 sessions)". This proposal estimates 1 change folder with 7 phases.

## What changes

### Backend additions (purely additive)

**7 new tables**:
- `sso_providers` — OIDC/SAML config per tenant (id, tenant_id, type 'oidc'|'saml', name, config jsonb, enabled, created_at)
- `sso_connections` — per-user SSO link (id, user_id, provider_id, subject text — IdP's stable user ID, created_at)
- `scim_tokens` — tenant API tokens for SCIM provisioning (id, tenant_id, name, token_hash, scopes text[], expires_at, last_used_at, created_at)
- `scim_sync_log` — audit of SCIM operations (id, tenant_id, op 'create'|'update'|'delete', external_id, status, error, created_at)
- `rbac_roles` — custom roles per tenant (id, tenant_id, name, description, permissions text[], created_at)
- `audit_log_archive` — long-term audit storage (id, tenant_id, batch_id, event_count, compressed_payload bytea, period_start, period_end, created_at)
- `compliance_reports` — generated reports (id, tenant_id, framework 'soc2'|'iso27001'|'hipaa', period_start, period_end, status 'pending'|'running'|'completed'|'failed', artifact_url, created_at)

**~32 new routes**:
- Phase 1 (SSO): 8 routes (list providers + create + update + delete + initiate + callback + disconnect + test)
- Phase 2 (SCIM): 6 routes (list tokens + create + revoke + SCIM /Users endpoint + /Groups endpoint + sync log)
- Phase 3 (RBAC): 6 routes (list roles + create + update + delete + assign to user + check permission)
- Phase 4 (Audit): 4 routes (list archives + create archive + download archive + export query results)
- Phase 5 (Compliance): 5 routes (list reports + generate report + status + download + schedule recurring)
- Phase 6 (Enterprise tenants): 3 routes (list sub-orgs + create + update org settings)

### Frontend additions (Datadog style)

- 1 new top-level page: `EnterprisePage.tsx` (Tabbed UI for SSO, SCIM, RBAC, Compliance)
- 6 shared components: `SsoProviderCard`, `ScimTokenTable`, `RbacRoleEditor`, `AuditArchiveCard`, `ComplianceReportCard`, `OrgSettingsForm`
- 4 extracted page sections: `SsoSection`, `ScimSection`, `RbacSection`, `ComplianceSection`
- AppSidebar: "Enterprise" entry under operations section

### Modular discipline maintained

- All new files ≤400 LOC
- Handlers split by domain (SSO + SCIM + RBAC + Audit + Compliance + Enterprise tenants)
- Routes split into `routes_enterprise.go` (NEW, similar to routes_intelligence.go)
- New migration: `040_enterprise.sql` (7 tables + 4 indexes)
- No new dependencies (use existing crypto/saml packages + go-oidc if available; fallback to manual JWT validation if not)
- Tenant_id isolation enforced everywhere
- All migrations idempotent

## Impact

| Area | Impact |
|---|---|
| Backend | +1 migration, +6 handler files, +32 routes, ~2200 LOC |
| Database | +7 new tables + 4 indexes |
| Frontend | +6 components + 1 page + 4 sections, ~1200 LOC |
| Build | +60-90 KB JS gzipped |
| Tests | All existing tests must still pass |
| Docs | journal.md updated; archive folder created |
| Breaking changes | None — purely additive |

## Enterprise parity

Mirrors common SaaS enterprise features:
- **Okta / Azure AD / Google Workspace SSO** via OIDC + SAML
- **SCIM 2.0** user/group lifecycle management
- **Custom RBAC roles** with permission inheritance
- **Audit log archival** with S3-compatible storage (configurable)
- **Compliance reports** for SOC2 / ISO27001 / HIPAA
- **Multi-org tenant hierarchy** for enterprises with multiple business units

## Scope discipline

This is Tier 9 ONLY. Out of scope:
- Tier 10 (Homelab Dashboard) — Homarr replacement
- Tier 11 (Platform & Commerce) — billing, quotas, Stripe
- Tier 12 (Docs & GTM) — customer docs, marketing
- Tier 13 (Mobile) — React Native app

Any subagent that adds features from these tiers is out of scope.

## Done when

- [ ] All 6 user stories pass acceptance criteria
- [ ] All 32 routes registered + return correct status codes
- [ ] Migration 040 applied to prod
- [ ] Bundle JS gzipped ≤ baseline + 90 KB
- [ ] All files under 400 LOC
- [ ] Archive folder created
- [ ] journal.md updated with TIER 9 COMPLETE marker
- [ ] Tier 9 ready for Tier 10 (Homelab Dashboard)