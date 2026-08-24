-- Tier 7 Phase 3 — CI/CD Visibility (D8)
--
-- Two tables for the CI/CD observability surface. Mirrors the Datadog
-- pipelines / deployments model: a pipeline is a single run (one SHA,
-- one branch, one status), and a deployment is the act of pushing an
-- artifact from that pipeline into a target environment linked to an
-- APM service so we can correlate deploys with downstream regressions.
--
-- - cicd_pipelines:    one row per CI/CD run, regardless of outcome.
--                      Provider enum is text to keep the migration
--                      simple; runtime handler enforces the allowed set.
-- - cicd_deployments:  one row per (pipeline, environment, service)
--                      deployment. service_id FK CASCADEs so deleting
--                      an APM service cleans up its deployment history.
--
-- All tables are tenant-scoped; every query honors tenant_id from JWT.
-- Webhook receivers (POST /cicd/webhook/{github,gitlab}) write into
-- these tables using the tenant resolved from repo slug lookup.
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.
-- pgcrypto is required for gen_random_uuid(); keep the ensure line
-- even though most installs already have it.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- cicd_pipelines: a single CI/CD run.
-- Provider values (enforced at API edge):
--   'github' | 'gitlab' | 'jenkins' | 'circleci'
-- Status values (enforced at API edge):
--   'pending' | 'running' | 'success' | 'failed' | 'cancelled'
CREATE TABLE IF NOT EXISTS cicd_pipelines (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL,
  provider     text NOT NULL,
  repo         text NOT NULL,
  branch       text NOT NULL DEFAULT '',
  commit_sha   text NOT NULL DEFAULT '',
  status       text NOT NULL DEFAULT 'pending',
  started_at   timestamptz NOT NULL DEFAULT now(),
  finished_at  timestamptz,
  duration_ms  BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS cicd_pipelines_tenant_started_idx
  ON cicd_pipelines (tenant_id, started_at DESC);

-- cicd_deployments: an environment deployment from a pipeline, linked
-- to an APM service so deploys surface in service detail pages.
CREATE TABLE IF NOT EXISTS cicd_deployments (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL,
  pipeline_id  uuid NOT NULL REFERENCES cicd_pipelines(id) ON DELETE CASCADE,
  service_id   uuid NOT NULL REFERENCES apm_services(id) ON DELETE CASCADE,
  environment  text NOT NULL DEFAULT 'production',
  version      text NOT NULL DEFAULT '',
  deployed_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS cicd_deployments_tenant_service_deployed_idx
  ON cicd_deployments (tenant_id, service_id, deployed_at DESC);