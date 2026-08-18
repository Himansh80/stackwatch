-- Tier 6 v2 — Synthetics results history
-- One row per check run; tracks success/failure/timeout over time.

CREATE TABLE IF NOT EXISTS synthetics_results (
    id           BIGSERIAL PRIMARY KEY,
    check_id     UUID NOT NULL REFERENCES synthetics_checks(id) ON DELETE CASCADE,
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status       TEXT NOT NULL,        -- 'success' | 'failure' | 'timeout'
    response_ms  INT,                  -- HTTP RTT or TCP connect time
    status_code  INT,                  -- HTTP status (NULL for TCP/ICMP)
    error        TEXT,                 -- error message if failure
    ts           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sr_check_ts
    ON synthetics_results (check_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_sr_tenant_ts
    ON synthetics_results (tenant_id, ts DESC);

CREATE INDEX IF NOT EXISTS idx_sr_status_ts
    ON synthetics_results (status, ts DESC) WHERE status != 'success';
