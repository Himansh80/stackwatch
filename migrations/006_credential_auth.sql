-- StackWatch Tier 3.6 — Credentials-driven auth in dialSSH
-- Connections can now authenticate via password or key+passphrase (looked up in credentials vault).

ALTER TABLE connections
ADD COLUMN IF NOT EXISTS auth_method TEXT NOT NULL DEFAULT 'key'
    CHECK (auth_method IN ('key', 'password', 'key_with_passphrase'));

-- Optional credential reference (looked up from credentials table)
ALTER TABLE connections
ADD COLUMN IF NOT EXISTS credential_id UUID REFERENCES credentials(id) ON DELETE SET NULL;
