-- StackWatch Tier 1 — Proxmox full replacement schema
-- Multi-tenant: every row belongs to a tenant_id

-- proxmox_hosts: registered Proxmox nodes (one per cluster/standalone)
CREATE TABLE IF NOT EXISTS proxmox_hosts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    base_url        TEXT NOT NULL,
    api_token       TEXT NOT NULL,  -- encrypted at rest in Tier 9
    verify_tls      BOOLEAN NOT NULL DEFAULT false,
    node_name       TEXT,          -- optional: specific node if known
    status          TEXT NOT NULL DEFAULT 'unknown',
    last_check_at   TIMESTAMPTZ,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_proxmox_hosts_tenant ON proxmox_hosts(tenant_id);

-- proxmox_nodes: cluster nodes discovered from a host
CREATE TABLE IF NOT EXISTS proxmox_nodes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id         UUID NOT NULL REFERENCES proxmox_hosts(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'unknown',
    uptime_seconds  BIGINT NOT NULL DEFAULT 0,
    cpu_count       INTEGER NOT NULL DEFAULT 0,
    cpu_usage       REAL NOT NULL DEFAULT 0,
    mem_total       BIGINT NOT NULL DEFAULT 0,
    mem_used        BIGINT NOT NULL DEFAULT 0,
    disk_total      BIGINT NOT NULL DEFAULT 0,
    disk_used       BIGINT NOT NULL DEFAULT 0,
    ssl_fingerprint TEXT,
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(host_id, name)
);

CREATE INDEX IF NOT EXISTS idx_proxmox_nodes_host ON proxmox_nodes(host_id);

-- proxmox_vms: VMs discovered across all hosts
CREATE TABLE IF NOT EXISTS proxmox_vms (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id         UUID NOT NULL REFERENCES proxmox_hosts(id) ON DELETE CASCADE,
    node_name       TEXT NOT NULL,
    vmid            INTEGER NOT NULL,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'qemu',  -- qemu or lxc
    status          TEXT NOT NULL DEFAULT 'unknown',
    cpu_count       INTEGER NOT NULL DEFAULT 0,
    cpu_usage       REAL NOT NULL DEFAULT 0,
    mem_total       BIGINT NOT NULL DEFAULT 0,
    mem_used        BIGINT NOT NULL DEFAULT 0,
    disk_total      BIGINT NOT NULL DEFAULT 0,
    disk_used       BIGINT NOT NULL DEFAULT 0,
    net_in          BIGINT NOT NULL DEFAULT 0,
    net_out         BIGINT NOT NULL DEFAULT 0,
    uptime_seconds  BIGINT NOT NULL DEFAULT 0,
    template        BOOLEAN NOT NULL DEFAULT false,
    tags            TEXT,
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(host_id, vmid)
);

CREATE INDEX IF NOT EXISTS idx_proxmox_vms_host ON proxmox_vms(host_id);
CREATE INDEX IF NOT EXISTS idx_proxmox_vms_status ON proxmox_vms(status);

-- proxmox_storage: storage pools per host
CREATE TABLE IF NOT EXISTS proxmox_storage (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id         UUID NOT NULL REFERENCES proxmox_hosts(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'unknown',
    total_bytes     BIGINT NOT NULL DEFAULT 0,
    used_bytes      BIGINT NOT NULL DEFAULT 0,
    avail_bytes     BIGINT NOT NULL DEFAULT 0,
    usage_pct       REAL NOT NULL DEFAULT 0,
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(host_id, name)
);

CREATE INDEX IF NOT EXISTS idx_proxmox_storage_host ON proxmox_storage(host_id);
