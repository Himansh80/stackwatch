-- Tier 11 Phase 1 — Platform & Commerce (009-tier11-platform-commerce)
--
-- Adds the first backing table for Push-Button Deploy (PL1):
--
--   deploy_install_tokens — one-time-use tokens that authorise a
--   freshly-installed Linux box to register itself with a tenant.
--
-- Future phases of this same change add their tables here:
--
--   Phase 2 (PL2 — Usage Metering):
--     usage_events + usage_daily_rollups
--
--   Phase 3 (PL3 — Self-Service Signup):
--     no new tables (extends existing users + tenants)
--
--   Phase 4 (PL4 — Tenant Limits):
--     plan_limits
--
--   Phase 5 (PL5 — Backup/Restore):
--     platform_backups
--
--   Phase 6 (PL6 — Multi-Region / HA):
--     platform_regions + platform_replicas
--
--   Phase 7 (PL7 — Rate Limiting):
--     no new tables (in-memory state)
--
--   Phase 8 (PL8 — Platform Health):
--     platform_health_samples
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS / CREATE
-- INDEX IF NOT EXISTS) so re-applying this file is a no-op. No
-- destructive ALTERs land here — Phase 1 ships only this one
-- new table.
--
-- Why deploy_install_tokens exists:
--   A tenant admin clicks "+ Generate install token" in the UI
--   (POST /api/v1/platform/deploy/install-token). The handler
--   creates a 32-byte random plaintext token (prefixed "swi_"
--   so it's grep-able), bcrypt-hashes it (cost 10), and stores
--   the hash + expires_at (now + 1 hour) + tenant_id. The
--   plaintext is returned ONCE to the admin.
--
--   The admin then runs the generated curl one-liner on a fresh
--   Linux box, which hits
--   GET /api/v1/platform/deploy/install-script?token=…&backend=…
--   (PUBLIC — no JWT, because the box has no JWT yet). The
--   endpoint bcrypt-looks up the token, bakes it into the bash
--   snippet, marks the token `used_at = now()` + `used_by_ip`,
--   and returns the script. Replays return 410 Gone.
--
--   Why bcrypt (not SHA-256): matching the Tier 9 SCIM token
--   pattern (handlers_scim_types.go::generateSCIMToken). Both
--   SCIM tokens and install tokens are "a secret the operator
--   pastes into a remote system" — same threat model, same
--   hashing policy.

CREATE TABLE IF NOT EXISTS deploy_install_tokens (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   uuid NOT NULL,
    token_hash  text NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz,
    used_by_ip  text,
    label       text
);

-- Tenant-scoped lookup is the hot path: "list this tenant's tokens"
-- for the UI table. Per-tenant index keeps that scan tiny.
CREATE INDEX IF NOT EXISTS idx_deploy_install_tokens_tenant
    ON deploy_install_tokens(tenant_id);

-- Partial index on unused tokens by expiry: the install-script
-- endpoint fetches only "not used + not expired" rows when a
-- tenant queries recent activity, and the cleanup worker (future)
-- will scan this same partial range to garbage-collect.
CREATE INDEX IF NOT EXISTS idx_deploy_install_tokens_expires
    ON deploy_install_tokens(expires_at) WHERE used_at IS NULL;