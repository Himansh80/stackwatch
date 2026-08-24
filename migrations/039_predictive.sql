-- Tier 8 — Intelligence & Alerting (006-tier8-intelligence-alerting)
-- Phase 2: Predictive Alerting (Tier 8.2).
--
-- Adds two tables for the predictive alerting surface: the
-- predictive_alerts event log (fired when a forecast predicts a
-- breach) and the forecast_history accuracy log (model MAPE/RMSE
-- per evaluation, so the UI can show how trustworthy the model is).
--
-- Naming: same `intelligence_` prefix as Phase 1 — keeps the
-- intelligence surface namespaced from Tier 6's streaming anomaly
-- tables. The Phase 1 models/events tables stay untouched.
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS) so re-applying
-- this migration is a no-op. No data migration is needed for the first
-- phase — the tables start empty; the linear_regression math lives in
-- the handler (Phase 2 is simple OLS; no fancy dependency).

-- ---------------------------------------------------------------------------
-- intelligence_predictive_alerts — one row per predicted-breach alert.
--
-- Inserted by POST /predict/forecast when the forecast's p90 (upper
-- 90% percentile) crosses the breach threshold (currently 90% for
-- percentage metrics; the handler treats p90>=90 as the trigger).
--
-- `server_id` is nullable so tenant-wide forecasts (e.g. cluster-wide
-- error rate) work. `confidence` is the model's overall confidence in
-- the forecast (1 / (1 + mape/100)) so the UI can show how sure we
-- are. severity follows the existing 3-tier (info/warning/critical).
--
-- status is one of 'open' / 'acknowledged' / 'resolved'. ack_user_id
-- is stamped from the JWT on POST /predict/ack — never from the body
-- so a user can't ack on behalf of someone else.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS intelligence_predictive_alerts (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric_name          text NOT NULL,
    server_id            uuid,                            -- nullable for tenant-wide forecasts
    predicted_value      double precision NOT NULL,
    predicted_breach_at  timestamptz NOT NULL,
    confidence           double precision NOT NULL,
    severity             text NOT NULL DEFAULT 'info',    -- 'info' | 'warning' | 'critical'
    status               text NOT NULL DEFAULT 'open',    -- 'open' | 'acknowledged' | 'resolved'
    ack_user_id          uuid,
    ack_note             text,
    created_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_intelligence_predictive_alerts_status_breach
    ON intelligence_predictive_alerts(tenant_id, status, predicted_breach_at);

CREATE INDEX IF NOT EXISTS idx_intelligence_predictive_alerts_metric
    ON intelligence_predictive_alerts(tenant_id, metric_name, created_at DESC);

-- ---------------------------------------------------------------------------
-- intelligence_forecast_history — one row per model evaluation.
--
-- Inserted every time a forecast runs and we have a "holdout" set to
-- score against. mape = Mean Absolute Percentage Error; rmse = Root
-- Mean Squared Error. The /predict/accuracy endpoint aggregates these
-- (latest, or AVG over the last 20) so the UI can render "MAPE 12%"
-- and warn the user when confidence drops.
--
-- model_type is one of 'linear_regression' (default) /
-- 'exponential_smoothing' / 'holt_winters' — Phase 2 only writes
-- 'linear_regression'; the others are reserved for future phases
-- but the schema is forward-compatible.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS intelligence_forecast_history (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    metric_name     text NOT NULL,
    server_id       uuid,
    model_type      text NOT NULL DEFAULT 'linear_regression',
    mape            double precision NOT NULL,           -- Mean Absolute Percentage Error
    rmse            double precision NOT NULL,           -- Root Mean Squared Error
    evaluated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_intelligence_forecast_history_metric
    ON intelligence_forecast_history(tenant_id, metric_name, evaluated_at DESC);