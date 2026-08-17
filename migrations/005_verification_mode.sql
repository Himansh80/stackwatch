-- StackWatch Tier 3.4 — Strict known_hosts verification
-- Adds verification_mode to connections (default = strict)

ALTER TABLE connections
ADD COLUMN IF NOT EXISTS verification_mode TEXT NOT NULL DEFAULT 'strict'
    CHECK (verification_mode IN ('strict', 'insecure'));

-- Index for fast lookups by (tenant_id, host, port) in dialSSH
CREATE INDEX IF NOT EXISTS idx_known_hosts_tenant_host_port
    ON known_hosts (tenant_id, host, port);

-- Make sure the unique constraint exists (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'known_hosts_tenant_host_port_key') THEN
        ALTER TABLE known_hosts ADD CONSTRAINT known_hosts_tenant_host_port_key UNIQUE (tenant_id, host, port);
    END IF;
END$$;
