-- StackWatch Tier 3.1 — Terminal (Termius replacement) schema
-- SSH keys, saved connections, connection history

-- ssh_keys: PKI keys users can use to authenticate to target hosts
-- private_key is stored encrypted at rest (AES-256-GCM) in Tier 9
-- For now we store as plain text (will be encrypted in Tier 3.2)
CREATE TABLE IF NOT EXISTS ssh_keys (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    fingerprint     TEXT NOT NULL,                -- SHA256:...
    public_key      TEXT NOT NULL,                -- "ssh-ed25519 AAAA..."
    private_key     TEXT NOT NULL,                -- PEM-encoded (encrypted at rest in Tier 3.2)
    key_type        TEXT NOT NULL DEFAULT 'ed25519',  -- 'rsa', 'ed25519', 'ecdsa'
    passphrase      TEXT NOT NULL DEFAULT '',     -- encrypted at rest
    comment         TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name),
    UNIQUE(tenant_id, fingerprint)
);

CREATE INDEX IF NOT EXISTS idx_ssh_keys_tenant_id ON ssh_keys(tenant_id);

-- connections: saved SSH connections (host + port + user + key)
-- We don't store passwords here — passwords go to credentials table (Tier 3.2)
CREATE TABLE IF NOT EXISTS connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,                -- "prod-router", "dev-cluster"
    host            TEXT NOT NULL,                -- "192.168.0.107" or "router.local"
    port            INTEGER NOT NULL DEFAULT 22,
    user_           TEXT NOT NULL DEFAULT 'root', -- quoted: user is reserved in SQL
    ssh_key_id      UUID REFERENCES ssh_keys(id) ON DELETE SET NULL,
    group_name      TEXT NOT NULL DEFAULT 'default',
    tags            TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    color           TEXT NOT NULL DEFAULT '#3b82f6',  -- for UI label
    icon            TEXT NOT NULL DEFAULT 'server',
    last_connected_at TIMESTAMPTZ,
    last_connected_status TEXT,  -- 'success' | 'failed' | 'timeout'
    total_connections INTEGER NOT NULL DEFAULT 0,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_connections_tenant_id ON connections(tenant_id);
CREATE INDEX IF NOT EXISTS idx_connections_group ON connections(tenant_id, group_name);
CREATE INDEX IF NOT EXISTS idx_connections_tags ON connections USING GIN(tags);

-- connection_history: append-only log of connection attempts
CREATE TABLE IF NOT EXISTS connection_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id   UUID REFERENCES connections(id) ON DELETE SET NULL,
    host            TEXT NOT NULL,                -- denormalized for historical queries
    port            INTEGER NOT NULL DEFAULT 22,
    user_           TEXT NOT NULL,
    status          TEXT NOT NULL,                -- 'success' | 'failed' | 'timeout' | 'refused'
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    error_message   TEXT NOT NULL DEFAULT '',
    source_ip       TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_history_tenant_id ON connection_history(tenant_id);
CREATE INDEX IF NOT EXISTS idx_history_user_id ON connection_history(user_id);
CREATE INDEX IF NOT EXISTS idx_history_connection_id ON connection_history(connection_id);
CREATE INDEX IF NOT EXISTS idx_history_started_at ON connection_history(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_history_status ON connection_history(tenant_id, status);
