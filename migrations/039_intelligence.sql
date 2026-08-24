-- Tier 8 — Intelligence & Alerting (006-tier8-intelligence-alerting)
-- Phase 1: ML Anomaly Detection.
--
-- Adds two tables for the per-tenant ML model registry and the event
-- log of detected anomalies. The Welford + EWMA math itself lives in
-- the existing internal/ml.Detector (Welford-style running mean &
-- variance + exponentially-weighted moving average) — these tables
-- just persist the trained state and the event stream so the API can
-- surface "list models" / "list events" and operators can ack noise.
--
-- Naming: we prefix with `intelligence_` to avoid colliding with the
-- existing Tier 6 streaming anomaly_events table (created in
-- migrations/008_monitoring_depth.sql) which has a different shape
-- (BIGSERIAL id, value/zscore only, no model_id FK, no ack fields).
-- The two coexist — Tier 6 captures raw streaming detections, Tier 8
-- captures trained-model detections.
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS) so re-applying
-- this migration is a no-op. No data migration is needed for the first
-- phase — the tables start empty.

-- ---------------------------------------------------------------------------
-- intelligence_anomaly_models — one row per (tenant, metric, server).
--
-- `server_id` is nullable so a single tenant-wide model can exist for
-- metrics that aren't per-host (e.g. cluster-wide error rate). The
-- UNIQUE constraint on (tenant_id, metric_name, server_id) ensures a
-- POST /anomaly/train "upserts" the row for that key without dupes.
--
-- model_type is 'welford' (default; Welford + EWMA from internal/ml)
-- or 'ewma' (EWMA-only, lighter on memory). Phase 1 only writes
-- 'welford' — 'ewma' is reserved for Phase 2 (predictive alerting).
--
-- mean / variance are the Welford running stats. ewma is the smoothed
-- value (used by Phase 1 to compute a more recent baseline for
-- detection). sample_count is the number of training points the
-- model has seen so far — useful for "model needs more data" hints
-- in the UI.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS intelligence_anomaly_models (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric_name     text NOT NULL,
    server_id       uuid,                            -- nullable for tenant-wide models
    model_type      text NOT NULL DEFAULT 'welford', -- 'welford' | 'ewma'
    mean            double precision NOT NULL DEFAULT 0,
    variance        double precision NOT NULL DEFAULT 0,
    ewma            double precision NOT NULL DEFAULT 0,
    last_trained_at timestamptz NOT NULL DEFAULT now(),
    sample_count    bigint NOT NULL DEFAULT 0,
    UNIQUE (tenant_id, metric_name, server_id)
);

CREATE INDEX IF NOT EXISTS idx_intelligence_anomaly_models_lookup
    ON intelligence_anomaly_models(tenant_id, metric_name);

-- ---------------------------------------------------------------------------
-- intelligence_anomaly_events — one row per detection.
--
-- Inserted by POST /anomaly/detect when the running z-score crosses
-- the caller's threshold (default 3σ). `model_id` is FK CASCADE so
-- deleting a model nukes its events (defense in depth against
-- orphaned rows; we never want events pointing to a deleted model).
--
-- observed_value is what the caller passed in; expected_range_low /
-- expected_range_high are the model's "±threshold·stddev" interval
-- at detection time (cached so the UI can render the band without
-- re-running the math).
--
-- acknowledged / ack_note / ack_user_id support the ack workflow:
--   POST /anomaly/ack sets acknowledged=true, ack_user_id=JWT,
--   ack_note=operator's note. The UI shows acked events with a green
--   checkmark badge and dims them in the list.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS intelligence_anomaly_events (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    model_id             uuid REFERENCES intelligence_anomaly_models(id) ON DELETE CASCADE,
    server_id            uuid,
    metric_name          text NOT NULL,
    anomaly_score        double precision NOT NULL,
    severity             text NOT NULL DEFAULT 'info', -- 'info' | 'warning' | 'critical'
    observed_value       double precision NOT NULL,
    expected_range_low   double precision NOT NULL,
    expected_range_high  double precision NOT NULL,
    ts                   timestamptz NOT NULL DEFAULT now(),
    acknowledged         boolean NOT NULL DEFAULT false,
    ack_note             text,
    ack_user_id          uuid
);

CREATE INDEX IF NOT EXISTS idx_intelligence_anomaly_events_tenant_ts
    ON intelligence_anomaly_events(tenant_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_intelligence_anomaly_events_severity
    ON intelligence_anomaly_events(tenant_id, severity, ts DESC);

CREATE INDEX IF NOT EXISTS idx_intelligence_anomaly_events_model
    ON intelligence_anomaly_events(model_id, ts DESC);
