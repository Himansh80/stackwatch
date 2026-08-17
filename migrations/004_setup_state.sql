-- StackWatch Tier 0.5 — Install mode + setup wizard
-- Tracks whether a self-hosted instance has completed first-run setup.

CREATE TABLE setup_state (
    id                 BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),  -- only one row allowed
    install_mode       TEXT NOT NULL CHECK (install_mode IN ('cloud', 'self-hosted')),
    is_complete        BOOLEAN NOT NULL DEFAULT FALSE,
    admin_user_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    setup_domain       TEXT NOT NULL DEFAULT '',
    tls_mode           TEXT NOT NULL DEFAULT 'self_signed' CHECK (tls_mode IN ('self_signed', 'lets_encrypt', 'custom', 'external')),
    tls_cert_path      TEXT NOT NULL DEFAULT '',
    tls_key_path       TEXT NOT NULL DEFAULT '',
    public_url         TEXT NOT NULL DEFAULT '',
    completed_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The single-row table: ensure id=TRUE always
CREATE UNIQUE INDEX idx_setup_state_singleton ON setup_state ((TRUE));

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION trg_setup_state_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER setup_state_updated_at
BEFORE UPDATE ON setup_state
FOR EACH ROW
EXECUTE FUNCTION trg_setup_state_updated_at();

-- Seed: default to cloud mode, not complete
INSERT INTO setup_state (install_mode, is_complete)
VALUES ('cloud', FALSE)
ON CONFLICT DO NOTHING;
