-- Tier 7 Phase 3 — DB Monitoring (D9)
--
-- Three database surfaces mirror the Datadog Database Monitoring
-- feature: per-query timing ingest, per-host connection pool
-- snapshots, and an aggregated slow-query view that lets the UI
-- surface top offenders without re-running the full aggregation
-- client-side.
--
-- - database_queries:       one row per observed query. duration_ms
--                           is the wall-clock time the agent spent
--                           in the DB; rows_examined / rows_returned
--                           come from the engine's plan when available
--                           and are 0 otherwise.
-- - database_connection_pools:
--                           one row per pool snapshot. We never
--                           delete from this table — older rows are
--                           the only history we have on pool growth.
-- - database_slow_queries:  MATERIALIZED VIEW over database_queries
--                           keyed by query_hash. Refresh is fire-
--                           and-forget on each slow-queries GET so
--                           the surface stays "almost live" without
--                           paying the aggregation cost on every
--                           insert.
--
-- All tables tenant-scoped. No FKs (per spec; tenant scoping is
-- enforced at handler layer).
--
-- Idempotent: every CREATE uses IF NOT EXISTS. REFRESH on the
-- view is wrapped in a DO block so the migration itself does not
-- require any data to be present.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- database_queries: one row per observed query.
-- Database values (enforced at API edge):
--   'postgres' | 'mysql' | 'mariadb' | 'mongodb'
CREATE TABLE IF NOT EXISTS database_queries (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       uuid NOT NULL,
  database        text NOT NULL,
  query_hash      text NOT NULL,
  query_text      text NOT NULL,
  duration_ms     BIGINT NOT NULL,
  rows_examined   BIGINT NOT NULL DEFAULT 0,
  rows_returned   BIGINT NOT NULL DEFAULT 0,
  user_id         text,
  application_name text,
  ts              timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS database_queries_tenant_ts_idx
  ON database_queries (tenant_id, ts DESC);
CREATE INDEX IF NOT EXISTS database_queries_tenant_db_hash_idx
  ON database_queries (tenant_id, database, query_hash);

-- database_connection_pools: pool snapshots over time.
CREATE TABLE IF NOT EXISTS database_connection_pools (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL,
  database     text NOT NULL,
  host         text NOT NULL,
  port         int  NOT NULL DEFAULT 5432,
  pool_size    int  NOT NULL DEFAULT 0,
  active       int  NOT NULL DEFAULT 0,
  idle         int  NOT NULL DEFAULT 0,
  waiting      int  NOT NULL DEFAULT 0,
  checked_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS database_connection_pools_tenant_host_checked_idx
  ON database_connection_pools (tenant_id, database, host, port, checked_at DESC);

-- database_slow_queries: aggregated view over database_queries.
-- Uniqueness is on (tenant_id, database, query_hash) so the view
-- remains small even if millions of raw rows are inserted.
CREATE MATERIALIZED VIEW IF NOT EXISTS database_slow_queries AS
SELECT
  tenant_id,
  database,
  query_hash,
  ANY_VALUE(query_text)                       AS query_text,
  COUNT(*)                                    AS total_count,
  SUM(duration_ms)                            AS total_duration_ms,
  MAX(duration_ms)                            AS max_duration_ms,
  (SUM(duration_ms)::numeric / NULLIF(COUNT(*), 0))::BIGINT
                                               AS avg_duration_ms,
  MAX(ts)                                     AS last_seen
FROM database_queries
GROUP BY tenant_id, database, query_hash;

-- Unique index is required for REFRESH ... CONCURRENTLY; without it
-- Postgres refuses the non-blocking refresh.
CREATE UNIQUE INDEX IF NOT EXISTS database_slow_queries_pk_idx
  ON database_slow_queries (tenant_id, database, query_hash);

-- First-time refresh (no data yet, so this is a no-op cost-wise).
-- Wrapped in DO so the migration succeeds even if the view already
-- exists with a different shape (the CREATE above is IF NOT EXISTS).
DO $$
BEGIN
  BEGIN
    REFRESH MATERIALIZED VIEW database_slow_queries;
  EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'first refresh skipped: %', SQLERRM;
  END;
END$$;