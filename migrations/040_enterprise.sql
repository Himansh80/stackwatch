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
