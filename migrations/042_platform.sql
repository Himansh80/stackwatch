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

-- =====================================================================
-- PL3 — Self-Service Signup (Phase 3)
-- =====================================================================
--
-- One backing table for the public self-service signup flow:
--
--   platform_signups — pending-verification records for every
--   /api/v1/platform/signup attempt. A row lives in one of three
--   states:
--
--     'pending_verification' — created on POST /signup, awaits the
--       email-token click. Carries verification_token (32-byte hex
--       plaintext) until verify; bcrypt-hash-at-rest is overkill for
--       a one-shot verification link the operator will paste within
--       minutes (not stored credentials).
--
--     'verified' — POST /signup/verify matched the token, INSERTed
--       the tenant + admin user, and set tenant_id + admin_user_id
--       + verified_at. Subsequent /verify calls with the same token
--       return 409 (the row is locked to its tenant).
--
--     'rejected' — reserved for Phase 5 (PL5 — manual moderation
--       queue when abuse is detected); today no code path sets this.
--
-- Why plaintext verification_token (vs. bcrypt):
--   The email link itself is the only path that ever reads this
--   column (one-shot, < 24h validity), so we trade a hash for a
--   one-row SELECT by `verification_token = $1`. The partial index
--   on verification_token WHERE NOT NULL keeps that lookup O(1).
--
-- Why tenant_id + admin_user_id are nullable:
--   The row is INSERTed BEFORE the tenant exists — POST /signup
--   only knows the email + name + organization. /verify is what
--   creates the tenant + user and stamps the FKs. Storing them
--   here gives the dashboard a single source of truth ("signup
--   attempts, including those that never completed") without a
--   second table.
--
-- Status enum is text (not a Postgres enum) so a future Phase can
-- add 'rate_limited' / 'pending_review' / 'deleted' without an
-- ALTER TYPE — the partial indexes on (status, created_at) and
-- (email) cover the dashboard's three common queries.
CREATE TABLE IF NOT EXISTS platform_signups (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid,
    admin_user_id      uuid,
    email              text NOT NULL,
    password_hash      text NOT NULL,
    full_name          text NOT NULL,
    organization_name  text NOT NULL,
    verification_token text,
    verified_at        timestamptz,
    signup_ip          text,
    signup_user_agent  text,
    status             text NOT NULL DEFAULT 'pending_verification',
    rejected_reason    text,
    created_at         timestamptz NOT NULL DEFAULT now(),
    completed_at       timestamptz
);

-- "Show me the latest N pending signups for the dashboard" — drives
-- the PL3 super-admin widget that lands with Phase 8 (PlatformPage).
CREATE INDEX IF NOT EXISTS idx_platform_signups_status
    ON platform_signups(status, created_at DESC);

-- "Is this email already taken / already pending?" — used by both
-- POST /signup (to short-circuit on conflict) and the future
-- GET /signup/check-email. Plain btree on lowercased email would
-- be nicer; functional indexes need a migration we don't need today.
CREATE INDEX IF NOT EXISTS idx_platform_signups_email
    ON platform_signups(email);

-- Token lookup is the /verify hot path. Partial index (NOT NULL)
-- keeps it small — verified / rejected rows have NULL token.
CREATE INDEX IF NOT EXISTS idx_platform_signups_token
    ON platform_signups(verification_token) WHERE verification_token IS NOT NULL;

-- =====================================================================
-- PL4 — Tenant Limits (Phase 4)
-- =====================================================================
--
-- Two backing tables for the per-tenant plan + catalog surface:
--
--   platform_plan_definitions — the catalog. One row per plan (free,
--                                starter, pro, enterprise). Seeded
--                                on api-gateway boot from
--                                internal/auth/seed_plans.go via
--                                ON CONFLICT (name) DO NOTHING so
--                                reboots are idempotent.
--
--                                Pricing + caps are stored as plain
--                                columns (no JSONB) so the billing
--                                KPI strip can render them as
--                                integer cells without a JSON parse
--                                on every dashboard load. Adding a
--                                new "feature flag" column is a
--                                non-destructive ALTER — same
--                                idempotent rule as every other
--                                table here.
--
--   platform_tenant_limits    — per-tenant plan assignment + custom
--                                overrides + suspend state. UNIQUE
--                                on tenant_id so the UPSERT pattern
--                                in handlers_platform_limits.go
--                                creates the row on first PATCH if
--                                missing — there's no separate
--                                "onboarding" step.
--
-- Why a custom_overrides JSONB column (vs. extra columns):
--   The platform-admin surface lets a sales rep bump ONE limit
--   on a paying customer without rewriting the plan definition.
--   custom_overrides is a sparse map keyed by field name
--   ("max_servers", "data_retention_days", etc.) merged ON TOP of
--   the plan defaults at read time — see
--   internal/platform/limits.go::EffectiveLimits. Merging (not
--   replacing) preserves every other override the operator
--   previously set.
--
-- Why suspend columns on the limits table (not on tenants):
--   A future suspension (Phase 5+) may freeze plan UPGRADES but
--   keep READ access; a different suspension (payment failed)
--   freezes EVERYTHING. Modeling suspend state next to the plan
--   keeps these semantics explicit instead of overloading the
--   tenants.status field, which is used by Tier 0 for soft-delete.
--
-- trial_ends_at supports the "try Pro for 14 days" flow that
-- PL4 spec mentions; the auto-revert-to-free logic lives in the
-- retention worker (Phase 4 ships the column; a future polish
-- pass adds the cron-style auto-revert).

CREATE TABLE IF NOT EXISTS platform_plan_definitions (
    id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    text NOT NULL UNIQUE,
    display_name            text NOT NULL,
    monthly_price_cents     integer NOT NULL DEFAULT 0,
    currency                text NOT NULL DEFAULT 'USD',
    max_servers             integer NOT NULL DEFAULT 0,
    max_alerts              integer NOT NULL DEFAULT 0,
    max_dashboards          integer NOT NULL DEFAULT 0,
    max_team_members        integer NOT NULL DEFAULT 0,
    data_retention_days     integer NOT NULL DEFAULT 0,
    metrics_retention_days  integer NOT NULL DEFAULT 0,
    storage_gb_limit        bigint NOT NULL DEFAULT 0,
    api_calls_per_minute    integer NOT NULL DEFAULT 0,
    features                text[] NOT NULL DEFAULT '{}',
    is_builtin              boolean NOT NULL DEFAULT true,
    sort_order              integer NOT NULL DEFAULT 0,
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now()
);

-- "List plans in display order" is the public /limits/definitions
-- hot path. Sort_order is the pricing-page ordering knob (free
-- first, enterprise last) so a single composite index keeps the
-- sort cheap even as we add bespoke plans.
CREATE INDEX IF NOT EXISTS idx_platform_plan_definitions_sort
    ON platform_plan_definitions(sort_order ASC, name ASC);

CREATE TABLE IF NOT EXISTS platform_tenant_limits (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL UNIQUE,
    plan_name           text NOT NULL,
    custom_overrides    jsonb NOT NULL DEFAULT '{}',
    trial_ends_at       timestamptz,
    suspended_at        timestamptz,
    suspend_reason      text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- "Show me every tenant on plan X" — drives the Phase 8 PL8
-- /health/tenants/top endpoint and the PL4 future migration
-- tool ("how many tenants did we move to pro last month?").
CREATE INDEX IF NOT EXISTS idx_platform_tenant_limits_plan
    ON platform_tenant_limits(plan_name);

-- =====================================================================
-- Tier 11.5: Backup / Restore (Phase 5 — PL5)
--
-- Per proposal.md §"Risks" §"Backup encryption":
--   "Backup encryption — encrypt .tar.gz with AES-256-GCM using
--    a tenant-derived key from the master key."
--
-- Two backing tables:
--
--   platform_backups     — one row per completed (or in-flight)
--                          backup. Stores metadata + SHA-256 of
--                          both the plaintext and the ciphertext
--                          for tamper detection. The ENCRYPTED
--                          blob lives on disk under
--                          /opt/stackwatch/backups/{tenant_id}/
--                          {backup_id}.tar.gz.enc (mode 0600) —
--                          the row NEVER stores the ciphertext
--                          itself, only its path + sizes + the
--                          two checksums. Decryption key is
--                          re-derived per request from the env
--                          master key + the tenant_id (HKDF-
--                          SHA256, see backup_crypto.go).
--
--   platform_backup_jobs — one row per tenant scheduling config.
--                          UNIQUE on tenant_id so the
--                          upsert-on-first-save pattern (no
--                          separate "create schedule" endpoint)
--                          works cleanly. The BackupSchedulerWorker
--                          scans this table every hour and
--                          triggers CreateBackupAndEncrypt per
--                          due job.
--
-- Why we store sha256_plaintext AND sha256_ciphertext:
--   The plaintext hash is "what we intended to back up" — a
--   fresh hash on every restore verifies the decryption
--   matched the original. The ciphertext hash is "what's on
--   disk right now" — it catches a corrupted or partially-
--   written file BEFORE we even attempt decryption (cheaper
--   than the AES-GCM tag check). Together they form a
--   defense-in-depth: filesystem corruption → ciphertext
--   mismatch; bit-flip during decryption → GCM tag failure.
--
-- Why backup_kind defaults to 'manual':
--   The handler POST /backup/create always sets 'manual'; the
--   scheduler sets 'scheduled' so an operator can grep the
--   log to see which path produced each row. The reserved
--   word 'system' is left for a future backup-of-system-tables
--   hook (not in PL5 scope).
--
-- Why expires_at + a partial index:
--   The retention enforcement worker (see
--   internal/platform/backup_retention.go in a follow-up
--   phase) will DELETE expired rows and unlink the file.
--   The partial index on expires_at WHERE NOT NULL keeps
--   that scan cheap as we add a column for "delete when
--   older than retention_days".

CREATE TABLE IF NOT EXISTS platform_backups (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL,
    created_by_user_id  uuid,
    backup_kind         text NOT NULL DEFAULT 'manual',
    status              text NOT NULL DEFAULT 'pending',
    file_path           text,
    file_size_bytes     bigint NOT NULL DEFAULT 0,
    uncompressed_bytes  bigint NOT NULL DEFAULT 0,
    table_count         integer NOT NULL DEFAULT 0,
    row_count           bigint NOT NULL DEFAULT 0,
    encryption_algo     text NOT NULL DEFAULT 'aes-256-gcm',
    sha256_plaintext    text,
    sha256_ciphertext   text,
    started_at          timestamptz NOT NULL DEFAULT now(),
    completed_at        timestamptz,
    error_message       text,
    expires_at          timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now()
);

-- "Show me this tenant's backups newest first" — the hot path
-- for GET /platform/backup/list. Composite index because the
-- dashboard renders the list on every page load.
CREATE INDEX IF NOT EXISTS idx_platform_backups_tenant
    ON platform_backups(tenant_id, created_at DESC);

-- Partial index on in-flight rows only — the scheduler
-- pulls "pending/running" backups to surface stuck jobs in
-- the health dashboard (Phase 8 PL8). Far fewer rows than
-- the full table, so the partial index stays small.
CREATE INDEX IF NOT EXISTS idx_platform_backups_status
    ON platform_backups(status) WHERE status IN ('pending', 'running');

-- Partial index for the future retention sweep — every
-- non-null expires_at row is a candidate for cleanup.
CREATE INDEX IF NOT EXISTS idx_platform_backups_expires
    ON platform_backups(expires_at) WHERE expires_at IS NOT NULL;

-- platform_backup_jobs — per-tenant schedule. UNIQUE on
-- tenant_id because every tenant can have at most one
-- active schedule row (changing the schedule is an UPSERT,
-- not a new row). The last_backup_id FK surfaces "last
-- successful run" without a join across platform_backups.
CREATE TABLE IF NOT EXISTS platform_backup_jobs (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL UNIQUE,
    frequency       text NOT NULL DEFAULT 'daily',
    retention_days  integer NOT NULL DEFAULT 7,
    enabled         boolean NOT NULL DEFAULT true,
    last_run_at     timestamptz,
    last_run_status text,
    last_backup_id  uuid REFERENCES platform_backups(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- "Find all due jobs" is the BackupSchedulerWorker hot path.
-- A composite (enabled, last_run_at NULLS FIRST) lets it
-- pull every job that needs a run with a single index scan.
CREATE INDEX IF NOT EXISTS idx_platform_backup_jobs_enabled
    ON platform_backup_jobs(enabled, last_run_at NULLS FIRST);

-- ============================================================================
-- Tier 11 Phase 6 — Multi-Region / HA (PL6)
-- ============================================================================
--
-- Adds the 2 tables backing the multi-region / HA surface:
--
--   platform_regions — region catalog. One row per StackWatch
--                      instance the platform knows about (primary
--                      sites, replica sites, standby / DR sites).
--                      `code` is the URL-safe lowercase handle
--                      (`us-east-1`, `eu-west-2`) that callers
--                      reference; `region_kind` ('primary' |
--                      'replica' | 'standby') drives the dashboard
--                      tile color. The last_health_* columns are
--                      written by the probe path (GET
--                      /platform/regions/health) — a fire-and-forget
--                      GET against `endpoint_url` measures latency
--                      and surfaces the result in the row.
--
--   platform_region_replicas — replication topology. One row per
--                      (primary, replica) pairing. Today Phase 6
--                      catalogs the rows but does NOT run a
--                      replication worker (the `/regions/replication`
--                      write endpoint is deferred to a follow-up
--                      phase; the platform_admin dashboard still
--                      needs the topology to render the failover
--                      preview).
--
-- Why platform-wide (no tenant_id):
--   Regions are a platform-admin concept. A given tenant's data
--   may live on N regions but the REGION LIST is global — every
--   super_admin sees the same catalog. Tenant scoping enters via
--   the existing `tenants.region_code` column added in Phase 6
--   follow-up; the catalog itself is intentionally shared.
--
-- Idempotency: every CREATE uses IF NOT EXISTS so re-running the
-- migration (the .115 deploy script runs 042 on every boot) is a
-- no-op rather than a destructive error.

CREATE TABLE IF NOT EXISTS platform_regions (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code              text NOT NULL UNIQUE,
    display_name      text NOT NULL,
    region_kind       text NOT NULL DEFAULT 'primary',
    endpoint_url      text NOT NULL,
    is_active         boolean NOT NULL DEFAULT true,
    last_health_at    timestamptz,
    last_health_status text,
    last_health_latency_ms integer,
    last_health_error text,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- "List all regions on the dashboard" — the hot path for GET
-- /platform/regions. Composite index because the dashboard
-- renders the list on every page load; the is_active filter
-- drops deactivated regions from the default view.
CREATE INDEX IF NOT EXISTS idx_platform_regions_active
    ON platform_regions(is_active, code);

-- "Find regions needing a probe" — the partial index keeps the
-- /platform/regions/health worker hot (Phase 8 PL8 will move the
-- probe into a scheduler; today Phase 6 triggers the probe
-- on-demand inside the POST handler).
CREATE INDEX IF NOT EXISTS idx_platform_regions_health
    ON platform_regions(last_health_at NULLS FIRST)
    WHERE is_active = true;

CREATE TABLE IF NOT EXISTS platform_region_replicas (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_region_id uuid NOT NULL REFERENCES platform_regions(id) ON DELETE CASCADE,
    replica_region_id uuid NOT NULL REFERENCES platform_regions(id) ON DELETE CASCADE,
    replication_kind  text NOT NULL DEFAULT 'streaming',
    replication_lag_ms integer NOT NULL DEFAULT 0,
    last_sync_at      timestamptz,
    is_active         boolean NOT NULL DEFAULT true,
    created_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (primary_region_id, replica_region_id)
);

-- "Find all replicas of a primary" — the failover-preview hot
-- path (a future /platform/regions/replication endpoint will
-- surface this list).
CREATE INDEX IF NOT EXISTS idx_platform_region_replicas_primary
    ON platform_region_replicas(primary_region_id) WHERE is_active = true;