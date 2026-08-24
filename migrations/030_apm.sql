-- Tier 7 — APM (D2) — Distributed Tracing foundation.
--
-- Trace model mirrors Datadog: services + traces + spans + deployments.
-- - apm_services:    a registered service (name + language + framework + env)
-- - apm_traces:      one row per trace (root span summary)
-- - apm_spans:       individual spans (flame graph source)
-- - apm_deployments: deployment markers linked to a service
--
-- All tables are tenant-scoped. Every query against these tables must
-- filter by tenant_id; service_id FKs CASCADE on parent service delete.
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.

-- pgcrypto is required for gen_random_uuid() on older Postgres versions.
-- Postgres 13+ ships it by default but ensure it's installed.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- services: registered APM services
CREATE TABLE IF NOT EXISTS apm_services (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL,
  name        text NOT NULL,
  language    text,
  framework   text,
  environment text,
  created_at  timestamptz DEFAULT now(),
  UNIQUE (tenant_id, name)
);

-- traces: trace summaries (one row per root span)
CREATE TABLE IF NOT EXISTS apm_traces (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  service_id    uuid NOT NULL REFERENCES apm_services(id) ON DELETE CASCADE,
  trace_id      text NOT NULL,
  root_span_id  text NOT NULL,
  duration_us   BIGINT NOT NULL,
  status        text NOT NULL DEFAULT 'ok',
  started_at    timestamptz NOT NULL,
  ingested_at   timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS apm_traces_tenant_started_idx
  ON apm_traces (tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS apm_traces_tenant_service_started_idx
  ON apm_traces (tenant_id, service_id, started_at DESC);

-- spans: individual spans (flame graph source)
CREATE TABLE IF NOT EXISTS apm_spans (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      uuid NOT NULL,
  trace_id       text NOT NULL,
  span_id        text NOT NULL,
  parent_span_id text,
  service_id     uuid NOT NULL REFERENCES apm_services(id) ON DELETE CASCADE,
  name           text NOT NULL,
  duration_us    BIGINT NOT NULL,
  status         text NOT NULL DEFAULT 'ok',
  started_at     timestamptz NOT NULL,
  attributes     jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS apm_spans_tenant_trace_idx
  ON apm_spans (tenant_id, trace_id);
CREATE INDEX IF NOT EXISTS apm_spans_tenant_service_started_idx
  ON apm_spans (tenant_id, service_id, started_at DESC);

-- deployments: deployment markers
CREATE TABLE IF NOT EXISTS apm_deployments (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL,
  service_id  uuid NOT NULL REFERENCES apm_services(id) ON DELETE CASCADE,
  environment text NOT NULL DEFAULT 'production',
  version     text NOT NULL,
  commit_sha  text,
  deployed_at timestamptz DEFAULT now(),
  rolled_back boolean DEFAULT false
);
CREATE INDEX IF NOT EXISTS apm_deployments_tenant_service_deployed_idx
  ON apm_deployments (tenant_id, service_id, deployed_at DESC);
