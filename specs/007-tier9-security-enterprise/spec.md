# Feature Specification: Tier 9 — Security & Enterprise

**Feature Branch**: `007-tier9-security-enterprise`
**Created**: 2026-08-24
**Status**: Draft

## Overview

Tier 9 layers enterprise-grade security on top of Tiers 0-8. After this ships, StackWatch can be sold to large organizations that require SSO, SCIM, custom RBAC, audit archival, and compliance reporting.

## User Stories

### Story 1 — SSO Foundation (Tier 9.1, P1 priority)

**As an** IT administrator
**I want** my team to log in with corporate SSO (Okta, Azure AD, Google Workspace)
**So that** we don't need separate passwords and benefit from our IdP's MFA

**WhyWhy this priority**: Without SSO, large enterprises won't buy. This is the #1 enterprise gate.

**Independent test**: Configure a mock OIDC provider, click "Login with SSO", verify the user is created/logged in.

**Acceptance scenarios**:
1. **Given** a tenant admin
   **When** `POST /api/v1/enterprise/sso/providers` is called with `{type: 'oidc', name: 'Okta', client_id, client_secret, discovery_url}`
   **Then** an `sso_providers` row is created with the config encrypted
2. **Given** a user visits the login page
   **When** they click "Login with Okta"
   **Then** they are redirected to the IdP's authorization endpoint (initiated via `GET /api/v1/enterprise/sso/initiate?provider_id=X`)
3. **Given** the IdP redirects back with a code
   **When** `GET /api/v1/enterprise/sso/callback?code=X&state=Y` is called
   **Then** the API exchanges the code for tokens, identifies the user, and either creates the user (JIT provisioning) or logs in the existing user
4. **Given** a user is logged in via SSO
   **When** `GET /api/v1/enterprise/sso/connections` is called
   **Then** the user's SSO connection is returned (subject + provider)
5. **Given** a tenant admin wants to remove SSO
   **When** `DELETE /api/v1/enterprise/sso/providers/:id` is called
   **Then** the provider is disabled (soft delete, not hard delete)
6. **Given** SSO is misconfigured
   **When** `POST /api/v1/enterprise/sso/test` is called with `{provider_id: X}`
   **Then** the discovery URL is fetched and validated; result is returned with errors if any
7. **Given** SAML is configured (not OIDC)
   **When** a user initiates SSO via `GET /api/v1/enterprise/sso/initiate?provider_id=X`
   **Then** a SAML AuthnRequest is generated and the user is redirected to the IdP's SSO endpoint
8. **Given** SAML response is posted to callback
   **When** the SAML assertion is validated (signature + audience + NotOnOrAfter)
   **Then** the user is identified and logged in

---

### Story 2 — SCIM Provisioning (Tier 9.2, P1 priority)

**As an** IT administrator
**I want** users and groups in our IdP to auto-sync to StackWatch
**So that** onboarding/offboarding happens automatically

**WhyWhy this priority**: Manual user creation doesn't scale. SCIM is the de-facto enterprise standard.

**Independent test**: Configure SCIM token, hit `/scim/v2/Users` with a POST, verify the user is created.

**Acceptance scenarios**:
1. **Given** a tenant admin
   **When** `POST /api/v1/enterprise/scim/tokens` is called with `{name: 'Okta SCIM', scopes: ['users:read', 'users:write', 'groups:write']}`
   **Then** an `scim_tokens` row is created; plaintext token returned ONCE for the admin to copy
2. **Given** a SCIM token exists
   **When** the IdP POSTs to `/scim/v2/Users` with a SCIM user payload
   **Then** the user is created in StackWatch (mapped from externalId + userName to user.email + user.full_name)
3. **Given** a user exists in StackWatch
   **When** the IdP PUTs to `/scim/v2/Users/:id`
   **Then** the user is updated (active → is_active, name → full_name, etc.)
4. **Given** a user is deprovisioned in the IdP
   **When** the IdP DELETEs `/scim/v2/Users/:id`
   **Then** the StackWatch user is soft-disabled (is_active=false, NOT hard-deleted, to preserve audit trail)
5. **Given** SCIM operations have happened
   **When** `GET /api/v1/enterprise/scim/sync-log` is called
   **Then** the recent SCIM operations are returned (timestamp + op + external_id + status)
6. **Given** a token is leaked
   **When** `DELETE /api/v1/enterprise/scim/tokens/:id` is called
   **Then** the token is revoked (last_used_at stays; expires_at unchanged; auth fails for all future requests)

---

### Story 3 — Advanced RBAC (Tier 9.3, P2 priority)

**As a** tenant admin
**I want** to create custom roles with specific permissions
**So that** I can grant least-privilege access to my team

**WhyWhy this priority**: Default roles (admin/operator/viewer) aren't enough for complex orgs.

**Independent test**: Create a role with permission "servers:read", assign to a user, verify they can read but not write servers.

**Acceptance scenarios**:
1. **Given** a tenant admin
   **When** `POST /api/v1/enterprise/rbac/roles` is called with `{name: 'ReadOnly', permissions: ['servers:read', 'dashboards:read']}`
   **Then** an `rbac_roles` row is created
2. **Given** a custom role exists
   **When** `PUT /api/v1/enterprise/rbac/roles/:id` is called with updated permissions
   **Then** the role's permissions are updated; all assignments inherit the new permissions
3. **Given** a custom role is no longer needed
   **When** `DELETE /api/v1/enterprise/rbac/roles/:id` is called
   **Then** the role is removed; users with only this role lose access
4. **Given** a user needs a custom role
   **When** `POST /api/v1/enterprise/rbac/users/:user_id/roles` is called with `{role_id: X}`
   **Then** the user is assigned to the role (additive; existing roles preserved)
5. **Given** a user has multiple roles
   **When** `GET /api/v1/enterprise/rbac/check?permission=servers:write&user_id=X` is called
   **Then** returns `{allowed: true|false, matching_roles: [...]}` based on cumulative permissions
6. **Given** a tenant has many roles
   **When** `GET /api/v1/enterprise/rbac/roles` is called
   **Then** all roles are returned (built-in + custom, with permission counts)

---

### Story 4 — Audit Log Retention + Export (Tier 9.4, P1 priority)

**As a** compliance officer
**I want** to archive audit logs for 7+ years and export them on demand
**So that** we can prove compliance with SOC2 / ISO27001 / HIPAA

**WhyWhy this priority**: Audit retention is a hard requirement for regulated industries.

**Independent test**: Trigger 100 audit events, run archive job, verify archive contains all 100.

**Acceptance scenarios**:
1. **Given** audit events have accumulated
   **When** `POST /api/v1/enterprise/audit/archive` is called with `{period_start, period_end}`
   **Then** an archive job runs in the background; status returns 'pending' → 'running' → 'completed'
2. **Given** an archive is completed
   **When** `GET /api/v1/enterprise/audit/archives` is called
   **Then** all archives are returned (id, period, event_count, size_bytes, created_at)
3. **Given** an archive exists
   **When** `GET /api/v1/enterprise/audit/archives/:id/download` is called
   **Then** the archive file is downloaded (gzip-compressed JSON, Content-Disposition: attachment)
4. **Given** an auditor needs filtered logs
   **When** `POST /api/v1/enterprise/audit/export` is called with `{event_type, start_time, end_time, format: 'csv'}`
   **Then** a CSV file is returned with filtered events

---

### Story 5 — Compliance Reports (Tier 9.5, P2 priority)

**As a** compliance officer
**I want** pre-built reports for SOC2 / ISO27001 / HIPAA
**So that** I don't have to write custom queries for each audit cycle

**WhyWhy this priority**: Pre-built reports save dozens of hours per audit cycle.

**Independent test**: Generate a SOC2 report, verify it includes access logs + change logs + encryption status.

**Acceptance scenarios**:
1. **Given** a tenant admin
   **When** `POST /api/v1/enterprise/compliance/reports` is called with `{framework: 'soc2', period_start, period_end}`
   **Then** a report job is queued; status returns 'pending'
2. **Given** a report is running
   **When** `GET /api/v1/enterprise/compliance/reports/:id` is called
   **Then** status returns 'running' (with progress if available) or 'completed' with artifact_url
3. **Given** a report is completed
   **When** `GET /api/v1/enterprise/compliance/reports/:id/download` is called
   **Then** the report PDF is downloaded (cover page + sections + evidence appendix)
4. **Given** a tenant wants recurring reports
   **When** `POST /api/v1/enterprise/compliance/schedule` is called with `{framework, frequency: 'quarterly', recipients: [email]}`
   **Then** the schedule is created; reports will be auto-generated on the cadence
5. **Given** a tenant has reports
   **When** `GET /api/v1/enterprise/compliance/reports` is called
   **Then** all reports are returned with status + artifact links

---

### Story 6 — Enterprise Tenants + Org Settings (Tier 9.6, P3 priority)

**As a** large enterprise admin
**I want** to organize users into sub-orgs / business units
**So that** billing + access control map to my org structure

**WhyWhy this priority**: Multi-org is needed for Fortune 500 customers.

**Independent test**: Create a parent org, create 2 sub-orgs, verify users are scoped to sub-orgs.

**Acceptance scenarios**:
1. **Given** a tenant has multiple business units
   **When** `POST /api/v1/enterprise/orgs` is called with `{name: 'Engineering', parent_org_id: X}`
   **Then** a sub-org is created with its own tenant_id
2. **Given** a sub-org exists
   **When** `GET /api/v1/enterprise/orgs` is called
   **Then** the org hierarchy is returned (parent + children + users per org)
3. **Given** an org has custom settings
   **When** `PATCH /api/v1/enterprise/orgs/:id/settings` is called with `{theme: 'dark', default_dashboard: 'engineering', ...}`
   **Then** the org settings are updated; new users inherit them

---

## Out of scope

- Tier 10 (Homelab Dashboard) — Homarr replacement, drag-drop widgets
- Tier 11 (Platform & Commerce) — billing, quotas, Stripe integration
- Tier 12 (Docs & GTM) — customer-facing docs, marketing landing
- Tier 13 (Mobile) — React Native app

Any subagent that adds features from these tiers is out of scope.

## Done criteria (measured at the END of Tier 9)

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 6 user stories pass their acceptance scenarios | [ ] |
| SC-002 | 32 routes registered + return correct status codes | [ ] |
| SC-003 | Migration 040 applied to prod | [ ] |
| SC-004 | go build + npm run type-check + lint + build exit 0 | [ ] |
| SC-005 | Bundle JS gzipped ≤ baseline + 90 KB | [ ] |
| SC-006 | Live 4× verifier PASS | [ ] |
| SC-007 | All files under 400 LOC | [ ] |
| SC-008 | Archive folder created | [ ] |
| SC-009 | journal.md updated with TIER 9 COMPLETE marker | [ ] |
| SC-010 | Tier 9 ready for Tier 10 (Homelab Dashboard) | [ ] |

## Risks

| Risk | Mitigation |
|------|------------|
| routes_protected.go will overflow (currently 396 LOC; +32 routes would push it past 500) | Phase 0 split (routes_enterprise.go NEW) BEFORE Phase 1 starts |
| SAML/OIDC requires careful state management | Use signed state parameter + Redis-backed session (in-memory is fine for MVP) |
| SCIM spec is verbose | Implement minimum subset (Users + Groups create/update/delete); add more later |
| RBAC check could be slow if it queries DB on every request | Cache effective permissions per user (in-memory, with TTL) |
| Audit archive could grow huge | Compress with gzip; chunk into 100k-event files |
| Compliance reports need careful access controls | Reports inherit from caller's tenant + role |

## Modular approach (binding for all phases)

- **One concern per file.** Handlers split into 6 files: sso + scim + rbac + audit + compliance + enterprise_tenants.
- **Routes split.** Create `routes_enterprise.go` (NEW) similar to routes_intelligence.go.
- **Shared components only.** Enterprise-style components in `web/src/components/shared/`.
- **No new dependencies.** Use existing tokens (`--surface`, `--border`, `--accent`, `--green`, `--amber`, `--red`).
- **Tenant_id isolation.** Every query filters by claims.TenantID.
- **Idempotent migrations.** `CREATE TABLE IF NOT EXISTS`.

## Notes for implementers

- Build env gotcha: ALWAYS use `export GOOS=linux; export GOARCH=amd64; go build -o ./<local>` (not `GOOS=linux go build -o /tmp/...` which silently fails).
- After completing each phase, ALWAYS commit. Add "git commit hash (mandatory)" to output format.
- Use the existing `auth` package patterns (RequireAuth, tenantIDFromContext, IsSuperAdmin).
- For SAML signature validation, use Go's `crypto` package — no new dependencies.
- For OIDC, use `golang.org/x/oauth2` + manual JWT validation (already available).