-- Tier 6 v4 — Real User Monitoring (M8) event storage.
-- One row per RUM event from the browser snippet.

CREATE TABLE IF NOT EXISTS rum_events (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,                  -- 'pageload' | 'jserror' | 'longtask' | 'fetch' | 'xhr'
    url        TEXT,
    value      DOUBLE PRECISION,              -- metric value (ms, etc.)
    attrs      JSONB NOT NULL DEFAULT '{}'::jsonb,
    ts         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rum_tenant_ts
    ON rum_events (tenant_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_rum_kind_ts
    ON rum_events (kind, ts DESC);

-- For slow query filtering
CREATE INDEX IF NOT EXISTS idx_rum_kind_value
    ON rum_events (kind, value DESC) WHERE kind IN ('pageload', 'longtask');
