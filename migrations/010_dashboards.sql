-- Tier 6 v3 — Dashboard persistence
-- Single table with JSONB layout for panel definitions.
-- Each dashboard has 0-20 panels; each panel is a JSON object
-- describing its type (timeseries/stat/table), query (PromQL-lite),
-- title, and grid position.

CREATE TABLE IF NOT EXISTS dashboards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    layout      JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_default  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_dashboards_tenant
    ON dashboards (tenant_id, updated_at DESC);

-- Only one default dashboard per tenant.
CREATE UNIQUE INDEX IF NOT EXISTS idx_dashboards_default_per_tenant
    ON dashboards (tenant_id) WHERE is_default;
