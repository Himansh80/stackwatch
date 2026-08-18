-- Tier 6 v5 — Distributed Tracing (M5) storage.
--
-- A trace is a tree of spans. Each span has a parent (except the root span).
-- - trace_id: 16-byte hex (32 chars), identifies the whole trace
-- - span_id:  8-byte hex (16 chars), identifies this span
-- - parent_id: span_id of the parent (empty for root)
-- - kind:     'server' | 'client' | 'internal' | 'producer' | 'consumer'
--
-- Trace ingestion accepts simplified JSON shape:
--   { trace_id, span_id, parent_id, service, name, kind, duration_ms, status, ts, attributes }
-- Also accepts OTLP-format payload (resource_spans -> scope_spans -> spans).

CREATE TABLE IF NOT EXISTS traces (
    id           BIGSERIAL PRIMARY KEY,
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trace_id     TEXT         NOT NULL,
    span_id      TEXT         NOT NULL,
    parent_id    TEXT         DEFAULT '',
    service      TEXT         NOT NULL,
    name         TEXT         NOT NULL,
    kind         TEXT         NOT NULL DEFAULT 'internal',
    duration_ms  INT          NOT NULL DEFAULT 0,
    status       TEXT         NOT NULL DEFAULT 'ok',
    ts           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    attributes   JSONB        NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS traces_tenant_trace_idx ON traces (tenant_id, trace_id);
CREATE INDEX IF NOT EXISTS traces_tenant_service_idx ON traces (tenant_id, service, ts);
CREATE INDEX IF NOT EXISTS traces_tenant_ts_idx ON traces (tenant_id, ts DESC);
