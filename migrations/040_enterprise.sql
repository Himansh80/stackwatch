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
