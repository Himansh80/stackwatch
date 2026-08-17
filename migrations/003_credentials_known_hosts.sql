-- StackWatch Tier 3.2 — Credentials vault + known_hosts
-- Encrypted passwords / key passphrases per-tenant, plus host fingerprint store.

-- credentials: encrypted secrets (passwords, key passphrases, sudo passwords)
-- cipher: AES-GCM. nonce stored alongside ciphertext.
-- encryption_key_id: identifies which master key encrypted this secret (for rotation)
CREATE TABLE credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL CHECK (kind IN ('password', 'key_passphrase', 'sudo_password')),
    ciphertext      BYTEA NOT NULL,
    nonce           BYTEA NOT NULL,
    encryption_key_id TEXT NOT NULL DEFAULT 'v1',
    description     TEXT NOT NULL DEFAULT '',
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX idx_credentials_tenant ON credentials(tenant_id);
CREATE INDEX idx_credentials_tenant_kind ON credentials(tenant_id, kind);

-- known_hosts: SSH host fingerprints (one per host:port)
-- fingerprint: SHA256 colon-separated hex (45 chars after "SHA256:")
-- public_key_der: optional DER bytes for stronger verification
CREATE TABLE known_hosts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    host            TEXT NOT NULL,
    port            INTEGER NOT NULL DEFAULT 22,
    key_type        TEXT NOT NULL,  -- ssh-ed25519, ssh-rsa, ecdsa-sha2-nistp256, etc.
    fingerprint     TEXT NOT NULL,  -- SHA256:base64colon
    comment         TEXT NOT NULL DEFAULT '',
    added_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, host, port)
);

CREATE INDEX idx_known_hosts_tenant ON known_hosts(tenant_id);
CREATE INDEX idx_known_hosts_host ON known_hosts(host);