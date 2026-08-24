-- Tier 7 — RUM Full (D4) — Datadog-grade Real User Monitoring tables.
--
-- Six tables that together capture the full browser-side observability
-- surface: per-session metadata, web vitals (LCP/FID/CLS/FCP/TTFB),
-- network resource timings, user interactions, long tasks > 50ms, and
-- grouped JS errors with fingerprint-based de-duplication.
--
-- All tables are tenant-scoped (tenant_id NOT NULL). Every time-bucketed
-- table has a (tenant_id, <time_col> DESC) index to keep the common
-- dashboard queries cheap.
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-applying this file
-- against a partially-migrated DB is safe.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- rum_sessions: one row per browser session (or page-view, depending
-- on the SDK's emit cadence). session_id is opaque text — the SDK
-- generates it client-side. attributes is a free-form jsonb bag for
-- browser/OS/country/etc. that aren't first-class columns.
CREATE TABLE IF NOT EXISTS rum_sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  user_id text,
  browser text,
  os text,
  device text,
  country text,
  url text NOT NULL,
  started_at timestamptz NOT NULL,
  duration_ms bigint,
  page_views int DEFAULT 0,
  errors int DEFAULT 0,
  attributes jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS rum_sessions_tenant_started_idx
  ON rum_sessions (tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS rum_sessions_tenant_session_idx
  ON rum_sessions (tenant_id, session_id);

-- rum_web_vitals: per-page-load performance metrics. metric is one of
-- lcp/fid/cls/fcp/ttfb (validated by the ingest handler against an
-- allow-list, not by the DB). rating is good/needs-improvement/poor
-- and follows the Web Vitals threshold partitions.
CREATE TABLE IF NOT EXISTS rum_web_vitals (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  url text NOT NULL,
  metric text NOT NULL,
  value double precision NOT NULL,
  rating text,
  ts timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS rum_web_vitals_tenant_ts_idx
  ON rum_web_vitals (tenant_id, ts DESC);
CREATE INDEX IF NOT EXISTS rum_web_vitals_tenant_session_idx
  ON rum_web_vitals (tenant_id, session_id);

-- rum_resources: one row per network request observed by the SDK.
-- resource_type is one of document/script/stylesheet/image/xhr/fetch/
-- font/other. duration_ms is mandatory (positive integer from the
-- PerformanceResourceTiming API).
CREATE TABLE IF NOT EXISTS rum_resources (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  url text NOT NULL,
  resource_type text,
  duration_ms bigint NOT NULL,
  size_bytes bigint,
  status int,
  ts timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS rum_resources_tenant_ts_idx
  ON rum_resources (tenant_id, ts DESC);
CREATE INDEX IF NOT EXISTS rum_resources_tenant_session_idx
  ON rum_resources (tenant_id, session_id);

-- rum_interactions: clicks, inputs, scrolls, etc. The SDK tags each
-- interaction with the action name + target selector so session-replay
-- can correlate them with DOM events.
CREATE TABLE IF NOT EXISTS rum_interactions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  action text NOT NULL,
  target text,
  duration_ms bigint,
  ts timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS rum_interactions_tenant_ts_idx
  ON rum_interactions (tenant_id, ts DESC);
CREATE INDEX IF NOT EXISTS rum_interactions_tenant_session_idx
  ON rum_interactions (tenant_id, session_id);

-- rum_long_tasks: tasks the browser flagged as > 50ms (the standard
-- "long task" threshold). Useful for spotting jank that doesn't show
-- up in network timings.
CREATE TABLE IF NOT EXISTS rum_long_tasks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  session_id text NOT NULL,
  duration_ms bigint NOT NULL,
  ts timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS rum_long_tasks_tenant_ts_idx
  ON rum_long_tasks (tenant_id, ts DESC);

-- rum_error_groups: de-duplicated error bucket. The SDK computes a
-- fingerprint (typically hash of stack-trace top frames) and the
-- ingest handler UPSERTs on (tenant_id, fingerprint) — bumping
-- occurrence_count and refreshing last_seen_at on every hit.
CREATE TABLE IF NOT EXISTS rum_error_groups (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  fingerprint text NOT NULL,
  message text NOT NULL,
  stack_trace text,
  service text,
  source text,
  occurrence_count int NOT NULL DEFAULT 1,
  first_seen_at timestamptz DEFAULT now(),
  last_seen_at timestamptz DEFAULT now(),
  UNIQUE (tenant_id, fingerprint)
);
CREATE INDEX IF NOT EXISTS rum_error_groups_tenant_count_idx
  ON rum_error_groups (tenant_id, occurrence_count DESC);
CREATE INDEX IF NOT EXISTS rum_error_groups_tenant_service_idx
  ON rum_error_groups (tenant_id, service);
