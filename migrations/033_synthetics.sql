-- Tier 7 Phase 2 — Synthetics Full (D5)
--
-- Three tables for proactive monitoring (HTTP/TCP/ICMP/browser/multi-step tests,
-- run results, and test locations). All tenant-scoped.
--
-- - synthetics_tests:        one row per synthetic test definition
-- - synthetics_test_runs:    one row per test run outcome (FK CASCADE on test delete)
-- - synthetics_locations:    one row per location a tenant can dispatch tests from
--
-- NOTE: results table is named `synthetics_test_runs` (not `synthetics_results`)
-- to avoid colliding with the Tier 6 v2 `synthetics_results` table (created in
-- migration 009, currently holding 13k+ rows of legacy check history and used
-- by the Tier 6 v2 synthetics scheduler). The Phase 1 spec called for
-- `synthetics_results` but the existing schema already owns that name; renaming
-- the legacy table would require modifying Tier 6 v2 code, which is out of
-- scope for this change. Documented as a deviation in journal.md.
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.
-- pgcrypto is required for gen_random_uuid() on older Postgres; newer ships it
-- by default but we keep the ensure-installed line for safety.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- synthetics_tests: a test definition (type + target + SLA).
CREATE TABLE IF NOT EXISTS synthetics_tests (
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id         uuid NOT NULL,
  name              text NOT NULL,
  type              text NOT NULL,                 -- 'http' | 'tcp' | 'icmp' | 'browser' | 'multi_step'
  url               text NOT NULL,
  method            text DEFAULT 'GET',
  body              text,
  headers           jsonb DEFAULT '{}'::jsonb,
  assertions        jsonb DEFAULT '[]'::jsonb,
  interval_seconds  int  NOT NULL DEFAULT 300,
  timeout_ms        int  NOT NULL DEFAULT 30000,
  locations         text[] DEFAULT '{}',
  sla_uptime_pct    decimal DEFAULT 99.9,
  sla_response_ms   int  DEFAULT 1000,
  enabled           boolean DEFAULT true,
  created_at        timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS synthetics_tests_tenant_enabled_idx
  ON synthetics_tests (tenant_id, enabled);

-- synthetics_test_runs: one row per test execution outcome.
CREATE TABLE IF NOT EXISTS synthetics_test_runs (
  id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id                uuid NOT NULL,
  test_id                  uuid NOT NULL REFERENCES synthetics_tests(id) ON DELETE CASCADE,
  started_at               timestamptz NOT NULL,
  duration_ms              int  NOT NULL,
  status                   text NOT NULL,            -- 'pass' | 'fail' | 'timeout' | 'error'
  response_code            int,
  error_message            text,
  assertions_passed        int  DEFAULT 0,
  assertions_failed        int  DEFAULT 0,
  response_body_excerpt    text
);
CREATE INDEX IF NOT EXISTS synthetics_test_runs_tenant_test_started_idx
  ON synthetics_test_runs (tenant_id, test_id, started_at DESC);

-- synthetics_locations: a logical "from where" — region/zone/agent.
CREATE TABLE IF NOT EXISTS synthetics_locations (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL,
  name        text NOT NULL,
  region      text NOT NULL,
  enabled     boolean DEFAULT true,
  UNIQUE (tenant_id, name)
);