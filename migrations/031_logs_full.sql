-- Tier 7 — Log Management Full (D3).
--
-- Five new tables power the Datadog-grade log management surface on
-- top of the existing Tier 6 v1 logs table (which stores raw events).
--
--   log_monitors:           alert rules on log queries (count > threshold
--                           over a time window)
--   log_archives:           archive destinations (s3 / gcs / azure / local)
--   log_rehydrations:       rehydration jobs queued against an archive
--   log_retention_policies: hot/cold storage windows per service
--   log_patterns:           detected log patterns (aggregated message
--                           prefixes per service)
--
-- All tables are tenant-scoped. Every query against these tables must
-- filter by tenant_id; archive_id FK CASCADEs on parent archive delete
-- so a rehydration row can never outlive its destination.
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.

-- pgcrypto is required for gen_random_uuid() on older Postgres versions.
-- Postgres 13+ ships it by default but ensure it's installed.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- log_monitors: alert rules on log queries
CREATE TABLE IF NOT EXISTS log_monitors (
  id                        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                 uuid NOT NULL,
  name                      text NOT NULL,
  query                     text NOT NULL,
  threshold_count           int  NOT NULL DEFAULT 1,
  threshold_window_seconds  int  NOT NULL DEFAULT 300,
  severity                  text NOT NULL DEFAULT 'warn',
  enabled                   boolean NOT NULL DEFAULT true,
  created_at                timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS log_monitors_tenant_created_idx
  ON log_monitors (tenant_id, created_at DESC);

-- log_archives: archive destinations
CREATE TABLE IF NOT EXISTS log_archives (
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id           uuid NOT NULL,
  name                text NOT NULL,
  destination_type    text NOT NULL,  -- 's3' | 'gcs' | 'azure' | 'local'
  destination_config  jsonb NOT NULL DEFAULT '{}'::jsonb,
  enabled             boolean NOT NULL DEFAULT true,
  created_at          timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS log_archives_tenant_created_idx
  ON log_archives (tenant_id, created_at DESC);

-- log_rehydrations: rehydration jobs against an archive
CREATE TABLE IF NOT EXISTS log_rehydrations (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  archive_id    uuid NOT NULL REFERENCES log_archives(id) ON DELETE CASCADE,
  start_time    timestamptz NOT NULL,
  end_time      timestamptz NOT NULL,
  status        text NOT NULL DEFAULT 'queued',  -- 'queued' | 'running' | 'completed' | 'failed'
  progress_pct  int  NOT NULL DEFAULT 0,
  created_at    timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS log_rehydrations_tenant_archive_created_idx
  ON log_rehydrations (tenant_id, archive_id, created_at DESC);

-- log_retention_policies: hot/cold storage windows per service
CREATE TABLE IF NOT EXISTS log_retention_policies (
  id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  service   text NOT NULL,
  hot_days  int  NOT NULL DEFAULT 7,
  cold_days int  NOT NULL DEFAULT 90,
  enabled   boolean NOT NULL DEFAULT true,
  created_at timestamptz DEFAULT now(),
  UNIQUE (tenant_id, service)
);
-- Defensive: add created_at to installations that pre-date the column.
ALTER TABLE log_retention_policies
  ADD COLUMN IF NOT EXISTS created_at timestamptz DEFAULT now();

-- log_patterns: detected log patterns (one row per known pattern)
CREATE TABLE IF NOT EXISTS log_patterns (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  service       text NOT NULL,
  pattern       text NOT NULL,
  sample_count  int  NOT NULL DEFAULT 1,
  last_seen_at  timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS log_patterns_tenant_service_idx
  ON log_patterns (tenant_id, service, last_seen_at DESC);
