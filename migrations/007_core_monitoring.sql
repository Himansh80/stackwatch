-- StackWatch Core Monitoring — Servers + Agents + Metrics + Ingest
-- Tier: Foundation (Tier 0 extension)

-- agents: registered monitoring agents
CREATE TABLE IF NOT EXISTS agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    server_id       UUID, -- nullable until registration completes
    ingest_key      TEXT NOT NULL UNIQUE,
    key_hash        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, active, stale, offline
    version         TEXT,
    os              TEXT,
    arch            TEXT,
    labels          JSONB NOT NULL DEFAULT '{}',
    last_seen_at    TIMESTAMPTZ,
    last_heartbeat  TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_agents_server_id ON agents(server_id);
CREATE INDEX IF NOT EXISTS idx_agents_status ON agents(status);

-- servers: monitored infrastructure nodes
CREATE TABLE IF NOT EXISTS servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_id        UUID REFERENCES agents(id) ON DELETE SET NULL,
    name            TEXT NOT NULL,
    hostname        TEXT,
    ip_address      INET,
    os              TEXT,
    os_version      TEXT,
    arch            TEXT,
    kernel_version  TEXT,
    cpu_cores       INTEGER,
    cpu_model       TEXT,
    memory_total    BIGINT,
    disk_total      BIGINT,
    tags            JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'unknown', -- up, down, stale, unknown
    last_seen_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_servers_tenant_id ON servers(tenant_id);
CREATE INDEX IF NOT EXISTS idx_servers_agent_id ON servers(agent_id);
CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status);

-- metric_points: time-series data from agents
CREATE TABLE IF NOT EXISTS metric_points (
    id              BIGSERIAL PRIMARY KEY,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    server_id       UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    metric_name     TEXT NOT NULL,
    metric_value    DOUBLE PRECISION NOT NULL,
    labels          JSONB NOT NULL DEFAULT '{}',
    ts              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_metric_points_tenant_id ON metric_points(tenant_id);
CREATE INDEX IF NOT EXISTS idx_metric_points_server_id ON metric_points(server_id);
CREATE INDEX IF NOT EXISTS idx_metric_points_metric_name ON metric_points(metric_name);
CREATE INDEX IF NOT EXISTS idx_metric_points_ts ON metric_points(ts DESC);

-- Hypertable for TimescaleDB (if available)
-- SELECT create_hypertable('metric_points', 'ts', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);

-- ingest_tokens: pre-shared keys for agent enrollment
CREATE TABLE IF NOT EXISTS ingest_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    token_hash      TEXT NOT NULL UNIQUE,
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    max_uses        INTEGER, -- NULL = unlimited
    used_count      INTEGER NOT NULL DEFAULT 0,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingest_tokens_tenant_id ON ingest_tokens(tenant_id);

-- agent_enrollment: one-time enrollment records
CREATE TABLE IF NOT EXISTS agent_enrollment (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    server_id       UUID REFERENCES servers(id) ON DELETE SET NULL,
    ingest_token_id UUID REFERENCES ingest_tokens(id) ON DELETE SET NULL,
    agent_ip        INET,
    agent_info      JSONB NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, completed, failed
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_enrollment_tenant_id ON agent_enrollment(tenant_id);

-- updated_at triggers
DROP TRIGGER IF EXISTS trg_agents_updated_at ON agents;
CREATE TRIGGER trg_agents_updated_at BEFORE UPDATE ON agents FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_servers_updated_at ON servers;
CREATE TRIGGER trg_servers_updated_at BEFORE UPDATE ON servers FOR EACH ROW EXECUTE FUNCTION set_updated_at();