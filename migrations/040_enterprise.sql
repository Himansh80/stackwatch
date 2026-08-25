-- Tier 9 — Security & Enterprise (007-tier9-security-enterprise)
-- Phase 1: SSO Foundation (Tier 9.1).
--
-- Adds the two tables that back the SSO surface — one row per
-- tenant-configured IdP integration (sso_providers) and one row per
-- successful user↔IdP-subject link (sso_connections).
--
-- Naming: sso_* (NOT intelligence_*, NOT tier9_*) so future phases
-- (SCIM / RBAC / Audit / Compliance / Enterprise tenants) can add
-- their tables to this same migration without colliding.
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS / CREATE
-- INDEX IF NOT EXISTS) so re-applying is a no-op. No data migration
-- is needed for Phase 1 — the tables start empty.
--
-- Security:
--   * sso_providers.config stores the IdP-specific config (OIDC
--     client_id/secret/discovery_url OR SAML metadata + entity_id).
--     Sensitive fields (client_secret, SAML signing keys) MUST be
--     encrypted at the application layer via internal/handler
--     encryptSecret (AES-GCM, CREDENTIALS_MASTER_KEY env) before
--     being inserted — see handlers_sso.go's CreateSSOProvider.
--     This column is opaque ciphertext to the DB.
--   * sso_connections.subject is the IdP's stable user identifier
--     (OIDC `sub` claim OR SAML NameID). It's stable per IdP per
--     user but is NOT a secret; treat as PII only (no email).

-- ---------------------------------------------------------------------------
-- sso_providers — one row per tenant-configured IdP integration.
--
-- `type` is 'oidc' (Okta / Azure AD / Google Workspace / Auth0 / etc.)
-- or 'saml' (legacy IdPs that don't speak OIDC). We enforce the enum
-- at the DB level so a hand-crafted INSERT can't smuggle in junk.
--
-- `name` is the operator-chosen display name ("Okta corp", "Azure AD",
-- "Google Workspace engineering") — never shown to end users, only
-- on the admin EnterprisePage.
--
-- `config` is a jsonb blob. The shape depends on `type`:
--   oidc: { "client_id": "...", "client_secret_ciphertext": "...",
--           "client_secret_nonce": "...",
--           "discovery_url": "https://...",
--           "redirect_uri": "https://...",
--           "scopes": ["openid","email","profile"],
--           "key_id": "v1" }
--   saml: { "metadata_xml": "..." /* raw XML, may be huge */,
--           "metadata_url": "..."  /* alternative to metadata_xml */,
--           "entity_id": "...",
--           "sso_url": "...",
--           "x509_cert_ciphertext": "..." /* encrypted PEM */,
--           "x509_cert_nonce": "...",
--           "key_id": "v1" }
-- The _ciphertext/_nonce pairs are the AES-GCM output of
-- internal/handler.encryptSecret and are opaque to the DB.
--
-- `enabled=false` is the SOFT-DELETE state — DELETE /sso/providers/:id
-- sets this rather than removing the row, so we keep the audit trail
-- (which users used this provider) intact.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sso_providers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('oidc', 'saml')),
    name        TEXT NOT NULL,
    config      JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Lookups always filter by tenant_id first, then by enabled (UI list).
CREATE INDEX IF NOT EXISTS idx_sso_providers_tenant
    ON sso_providers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sso_providers_tenant_enabled
    ON sso_providers(tenant_id, enabled);

-- ---------------------------------------------------------------------------
-- sso_connections — one row per user↔IdP-subject link.
--
-- The UNIQUE (provider_id, subject) constraint enforces "one user
-- per IdP subject per provider" — the JIT-provisioning handler looks
-- up a connection by this pair on every callback.
--
-- The (user_id) index supports the "list my SSO connections" endpoint
-- (GET /sso/connections) and the per-user "last used" sweep.
--
-- `last_used_at` is bumped on every successful callback so the UI
-- can show "last sign-in via SSO: 2 hours ago".
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sso_connections (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id  UUID NOT NULL REFERENCES sso_providers(id) ON DELETE CASCADE,
    subject      TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sso_connections_unique
    ON sso_connections(provider_id, subject);
CREATE INDEX IF NOT EXISTS idx_sso_connections_user
    ON sso_connections(user_id);
CREATE INDEX IF NOT EXISTS idx_sso_connections_tenant
    ON sso_connections(tenant_id);

-- ---------------------------------------------------------------------------
-- Tier 9.2 (Phase 2) — SCIM Provisioning.
--
-- Adds the two tables that back the SCIM 2.0 user lifecycle surface —
-- one row per tenant-issued SCIM bearer token (scim_tokens) and one
-- row per successful/failed IdP-driven SCIM operation (scim_sync_log).
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS / CREATE
-- INDEX IF NOT EXISTS) so re-applying this migration is a no-op.
--
-- Security:
--   * scim_tokens.token_hash is the bcrypt hash (cost 10) of the
--     plaintext bearer token. The plaintext is generated by the
--     handler via crypto/rand (32 bytes, base64url-encoded) and is
--     returned to the caller EXACTLY ONCE in the POST /scim/tokens
--     response body. After that the only way to authenticate is to
--     send the plaintext and have the handler bcrypt-compare it
--     against token_hash. We never store plaintext and never log it.
--   * scim_sync_log is the audit trail for every SCIM operation.
--     status='success' on a clean op; status='error' + error_message
--     on a failure. Useful for SOC2 / ISO27001 auditors who need to
--     prove who provisioned / deprovisioned whom.
-- ---------------------------------------------------------------------------

-- ---------------------------------------------------------------------------
-- scim_tokens — one row per tenant-issued SCIM bearer token.
--
-- The token is the IdP-issued credential (Okta / Azure AD / Google
-- Workspace) that lets the IdP POST to /scim/v2/Users etc. We generate
-- the plaintext server-side via crypto/rand, return it ONCE on
-- creation, and only ever store the bcrypt hash. This mirrors the
-- security model of api_keys (Tier 0) so operators recognize the
-- pattern.
--
-- `scopes` is a text[] of allowed operations:
--   users:read   — read users via GET /scim/v2/Users
--   users:write  — create / update / disable users
--   groups:read  — read groups (Phase 2 stub; full Group support later)
--   groups:write — modify groups (Phase 2 stub; full Group support later)
--
-- `expires_at` is optional. NULL = no expiry (the IdP can rotate by
-- deleting this row + creating a new one). Setting a non-NULL value
-- means the handler rejects the token after that timestamp.
--
-- `last_used_at` is bumped on each successful SCIM auth so the UI
-- can show "last used: 12 minutes ago".
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scim_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    scopes       TEXT[] NOT NULL DEFAULT '{}',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- All queries filter by tenant_id first (UI list endpoint).
CREATE INDEX IF NOT EXISTS idx_scim_tokens_tenant
    ON scim_tokens(tenant_id);

-- ---------------------------------------------------------------------------
-- scim_sync_log — one row per SCIM operation (audit trail).
--
-- `op` is the lifecycle action the IdP attempted:
--   create  — POST /scim/v2/Users        (provision new user)
--   update  — PUT  /scim/v2/Users/:id    (modify existing user)
--   delete  — DELETE /scim/v2/Users/:id  (soft-disable user)
--   list    — GET /scim/v2/Users         (browse users)
--
-- `external_id` is the IdP-side stable user identifier (SCIM
-- `externalId`). We store it as text (not uuid) because IdPs use
-- their own identifiers (Okta = `00uxxx`, Azure = GUIDs, etc.).
--
-- `resource_type` is 'User' for Phase 2 (Phase 3+ may add 'Group').
--
-- `status` is 'success' on a clean op; 'error' on any failure with
-- `error_message` populated. We deliberately do NOT log the request
-- body — it can contain PII (emails, names) and we don't want the
-- audit log to become a PII store.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS scim_sync_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    op             TEXT NOT NULL,
    external_id    TEXT,
    resource_type  TEXT NOT NULL,
    status         TEXT NOT NULL,
    error_message  TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The (tenant_id, created_at DESC) index supports the
-- "GET /api/v1/enterprise/scim/sync-log" endpoint which filters
-- by tenant and sorts by recency. Composite index avoids a sort.
CREATE INDEX IF NOT EXISTS idx_scim_sync_log_tenant_time
    ON scim_sync_log(tenant_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Tier 9.3 (Phase 3) — Advanced RBAC.
--
-- Adds the two tables that back the custom-roles surface — one row per
-- tenant-defined or built-in role (rbac_roles) and one row per user↔role
-- assignment (user_role_assignments).
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS / CREATE
-- INDEX IF NOT EXISTS) so re-applying this migration is a no-op.
--
-- Built-in roles (Admin / Operator / Viewer / Billing) are NOT inserted
-- here. They are LAZY-SEEDED on the first GET /api/v1/enterprise/rbac/roles
-- call per tenant by handlers_rbac.go::seedBuiltinRolesForTenant — this
-- avoids a slow multi-tenant up-front seed when there are many tenants.
--
-- Security:
--   * permissions is a text[] of "<resource>:<verb>" tokens
--     (e.g., 'servers:read', 'alerts:write'). The allowlist is defined
--     in handlers_rbac_types.go::builtinPermissions — handlers reject
--     any string not on the list at INSERT time so a hand-crafted
--     payload can't smuggle in junk like 'admin:*' or 'system:root'.
--   * is_builtin=true roles are READ-ONLY — the custom-role CRUD
--     handlers in handlers_rbac.go refuse PATCH/DELETE on built-in
--     rows. Only their name + permissions are populated; description
--     is operator-supplied.
--   * The UNIQUE (tenant_id, name) constraint on rbac_roles enforces
--     "one role name per tenant" — built-in roles occupy the four
--     names Admin / Operator / Viewer / Billing per tenant so custom
--     roles can't collide with built-ins.
--   * The UNIQUE (user_id, role_id) constraint on user_role_assignments
--     makes POST idempotent (handler re-INSERT returns the existing row
--     via ON CONFLICT DO NOTHING + RETURNING).
-- ---------------------------------------------------------------------------

-- ---------------------------------------------------------------------------
-- rbac_roles — one row per tenant-defined role.
--
-- `name` is operator-chosen for custom roles; built-in tenants get
-- the four names Admin / Operator / Viewer / Billing. UNIQUE per
-- tenant so the same operator can have "Admin" in multiple tenants
-- without collision (rare but legal).
--
-- `permissions` is the union of `<resource>:<verb>` strings granted
-- by this role. The handler enforces membership in builtinPermissions
-- so we never write garbage.
--
-- `is_builtin=true` means the row was seeded by the platform and
-- cannot be modified or deleted by tenant admins. Built-in rows
-- always have a non-empty permissions array.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS rbac_roles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT,
    permissions  TEXT[] NOT NULL DEFAULT '{}',
    is_builtin   BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

-- Every query in handlers_rbac.go filters by tenant_id first.
CREATE INDEX IF NOT EXISTS idx_rbac_roles_tenant
    ON rbac_roles(tenant_id);

-- ---------------------------------------------------------------------------
-- user_role_assignments — many-to-many user↔role join.
--
-- `assigned_at` is the moment the handler bound the user to the role.
-- We don't store an `assigned_by` for Phase 3 (no audit trail on
-- assignment) — that's a future Tier 9.x audit-log concern and would
-- add a column for negligible value here. The audit of "who can do
-- what right now" is derivable from this table on its own.
--
-- ON DELETE CASCADE on role_id mirrors the lifecycle of rbac_roles:
-- when a custom role is deleted, its assignments vanish with it. We
-- do NOT cascade from users — a user deletion must be handled by the
-- existing Tier 7 user-delete flow (which already cleans up its own
-- rows); if it doesn't, the FK constraint will surface the bug loudly
-- instead of silently losing audit data.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS user_role_assignments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL,
    role_id      UUID NOT NULL REFERENCES rbac_roles(id) ON DELETE CASCADE,
    assigned_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, role_id)
);

-- Supports the "list my roles" / "list users with role X" lookups in
-- the check endpoint and the user-role-assignments panel.
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user
    ON user_role_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role
    ON user_role_assignments(role_id);
-- Composite (tenant_id, user_id) supports the per-tenant check query.
CREATE INDEX IF NOT EXISTS idx_user_role_assignments_tenant_user
    ON user_role_assignments(tenant_id, user_id);
