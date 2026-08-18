-- Tier 6 — Monitoring Depth
-- Adds metric_points table (storage for Prometheus / Loki / anomaly data)
-- and indices for fast queries.

-- metric_points: one row per (server, metric_name, timestamp) sample.
-- value is double precision (Prometheus stores float64).
-- message is text, NULL by default; only used when metric_name='log' (Loki push).
CREATE TABLE IF NOT EXISTS metric_points (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    server_id    UUID,                           -- NULL allowed for global metrics
    metric_name  TEXT NOT NULL,                  -- e.g. "cpu.user_pct", "log"
    value        DOUBLE PRECISION NOT NULL,
    ts           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    message      TEXT,                           -- only for log entries (Loki)
    labels       JSONB DEFAULT '{}'::jsonb       -- Prometheus-style labels
);

CREATE INDEX IF NOT EXISTS idx_mp_tenant_metric_ts
    ON metric_points (tenant_id, metric_name, ts DESC);

CREATE INDEX IF NOT EXISTS idx_mp_server_ts
    ON metric_points (server_id, ts DESC) WHERE server_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_mp_metric_only_ts
    ON metric_points (metric_name, ts DESC);

-- GIN index on labels for PromQL filter {label="value"}
CREATE INDEX IF NOT EXISTS idx_mp_labels
    ON metric_points USING GIN (labels);

-- Index for log entries (Loki)
CREATE INDEX IF NOT EXISTS idx_mp_logs_ts
    ON metric_points (ts DESC) WHERE metric_name = 'log';

-- Anomaly events: pre-computed Z-score anomalies for fast list queries.
CREATE TABLE IF NOT EXISTS anomaly_events (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    server_id   UUID,
    metric_name TEXT NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    zscore      DOUBLE PRECISION NOT NULL,
    ts          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    severity    TEXT NOT NULL DEFAULT 'warning'  -- 'warning' | 'critical'
);

CREATE INDEX IF NOT EXISTS idx_anomaly_tenant_ts
    ON anomaly_events (tenant_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_anomaly_server_metric_ts
    ON anomaly_events (server_id, metric_name, ts DESC);

-- Synthetics checks (M7 deferred to v2; table created here for forward compat)
CREATE TABLE IF NOT EXISTS synthetics_checks (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    kind         TEXT NOT NULL,                  -- 'http' | 'tcp' | 'icmp'
    target       TEXT NOT NULL,
    interval_sec INT NOT NULL DEFAULT 60,
    timeout_ms   INT NOT NULL DEFAULT 5000,
    enabled      BOOLEAN NOT NULL DEFAULT true,
    last_run_at  TIMESTAMPTZ,
    last_status  TEXT,
    last_ms      INT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_synth_tenant
    ON synthetics_checks (tenant_id, enabled);
