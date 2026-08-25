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

-- =====================================================================
-- PL2 — Usage Metering (Phase 2)
-- =====================================================================
--
-- Two backing tables for the metering surface that Phase 2 (and every
-- later phase that wants to bill) reads from:
--
--   platform_usage_events      — raw, immutable event log. Every time
--                                something billable happens we INSERT
--                                one row here. The worker (Phase 2
--                                code lives at internal/platform/
--                                usage_meter.go) rolls these up into
--                                the aggregate table every hour.
--
--   platform_usage_aggregates  — hourly rollups keyed by (tenant,
--                                event_kind, bucket_ts). UNIQUE
--                                constraint means the worker's
--                                UPSERT pattern handles out-of-order
--                                events and crash-during-rollup
--                                races gracefully. The DOWNSTREAM
--                                reads (current/history/summary) ONLY
--                                scan this table, never the raw
--                                events, so the billing dashboard stays
--                                fast even at millions of events/day.
--
-- Why immutable events: a usage event represents something that
-- HAPPENED. Allowing UPDATE/DELETE would let an admin tamper with
-- billing history. The handler layer enforces no-PUT/no-DELETE; the
-- schema also omits any UPDATE policy triggers — it's just a plain
-- append-only log. Retention is 365 days configurable (the worker
-- prunes).
--
-- Why numeric(18,4) for quantity: API calls + dashboard panels are
-- naturally integer counts, but storage.gb.hour and bandwidth may be
-- fractional (0.001 GB is meaningful). 4 decimal places is enough
-- for sub-cent precision without ballooning the row size.
--
-- Why a partial index on resource_id: most events DON'T carry a
-- resource_id (login.success has none), so the partial index keeps
-- lookups by resource cheap without indexing every row.

CREATE TABLE IF NOT EXISTS platform_usage_events (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL,
    user_id      uuid,
    event_kind   text NOT NULL,
    quantity     numeric(18,4) NOT NULL DEFAULT 1,
    unit         text NOT NULL DEFAULT 'count',
    resource_id  text,
    metadata     jsonb NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_platform_usage_events_tenant_time
    ON platform_usage_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_usage_events_kind
    ON platform_usage_events(event_kind, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_platform_usage_events_resource
    ON platform_usage_events(resource_id) WHERE resource_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS platform_usage_aggregates (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL,
    event_kind   text NOT NULL,
    bucket_ts    timestamptz NOT NULL,
    sum_quantity numeric(18,4) NOT NULL DEFAULT 0,
    count        integer NOT NULL DEFAULT 0,
    min_quantity numeric(18,4),
    max_quantity numeric(18,4),
    UNIQUE (tenant_id, event_kind, bucket_ts)
);

CREATE INDEX IF NOT EXISTS idx_platform_usage_aggregates_tenant_time
    ON platform_usage_aggregates(tenant_id, bucket_ts DESC);