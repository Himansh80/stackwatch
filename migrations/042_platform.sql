-- Tier 11 Phase 1 — Platform & Commerce (009-tier11-platform-commerce)
--
-- Adds the first backing pieces for Push-Button Deploy (PL1):
--
--   deploy_install_tokens — one-time-use tokens that authorise a
--   freshly-installed Linux box to register itself with a tenant.
--
--   users.is_platform_admin — flag distinguishing the new platform-wide
--   admin role (separate from per-tenant `super_admin`) so Phase 5
--   (Backup/Restore) and Phase 6 (Multi-Region/HA) can gate their
--   admin-only routes without leaking into per-tenant super_admin.
--
-- Why two pieces in Phase 1: PL1 only needs deploy_install_tokens to
-- ship the install flow; the is_platform_admin ALTER is shipped here
-- too because every subsequent phase (PL2 metering, PL5 backup,
-- PL8 health) will query the flag and we don't want Phase 5/6 to
-- add a separate migration just for the column.
--
-- Future phases of this same change extend 042_platform.sql:
--
--   Phase 2 (PL2 — Usage Metering):
--     usage_events + usage_daily_rollups
--   Phase 3 (PL3 — Self-Service Signup):
--     no new tables (extends existing users + tenants)
--   Phase 4 (PL4 — Tenant Limits):
--     plan_limits
--   Phase 5 (PL5 — Backup/Restore):
--     platform_backups
--   Phase 6 (PL6 — Multi-Region / HA):
--     platform_regions + platform_replicas
--   Phase 7 (PL7 — Rate Limiting):
--     no new tables (in-memory state)
--   Phase 8 (PL8 — Platform Health):
--     platform_health_samples
--
-- All DDL is idempotent (CREATE TABLE IF NOT EXISTS / CREATE
-- INDEX IF NOT EXISTS / ALTER TABLE … ADD COLUMN IF NOT EXISTS)
-- so re-applying this file is a no-op. No destructive ALTERs
-- land here — Phase 1 ships only the one new table + one
-- non-destructive ALTER.
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

-- =====================================================================
-- users.is_platform_admin — platform-wide admin flag (Tier 11 role split)
-- =====================================================================
--
-- Existing `users.role` carries per-tenant roles (super_admin,
-- admin, operator, viewer). Tier 11 introduces a SEPARATE flag
-- for the new platform-wide admin (PL5 backup, PL6 multi-region,
-- PL8 health — all need a role that sees across tenants). The
-- boolean lives on `users` rather than a new join table because
-- the platform-admin surface is per-user (a single person can be
-- platform-admin on this StackWatch instance) and we don't need
-- audit / delegation today.
--
-- Idempotent ADD COLUMN IF NOT EXISTS so re-applying is a no-op.
-- Default false — existing rows are not platform admins, by design.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_platform_admin BOOLEAN NOT NULL DEFAULT false;