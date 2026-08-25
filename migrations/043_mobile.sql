-- Tier 13 Phase 0 — Mobile (speckit change 011-tier13-mobile)
-- Three NEW tables (intentionally NOT prefixed with sw_ — parallel to the
-- legacy sw_push_devices/sw_push_log tables used by web-terminal).

-- push_devices: registered push targets per (tenant, user)
CREATE TABLE IF NOT EXISTS push_devices (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL,
    user_id         uuid NOT NULL,
    platform        text NOT NULL CHECK (platform IN ('android','ios','web')),
    fcm_token       text NOT NULL,
    app_version     text,
    device_name     text,
    last_active_at  timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    enabled         boolean NOT NULL DEFAULT true,
    UNIQUE (tenant_id, platform, fcm_token)
);
CREATE INDEX IF NOT EXISTS idx_push_devices_user
    ON push_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_push_devices_tenant
    ON push_devices(tenant_id);

-- push_log: dispatch history (append-only)
CREATE TABLE IF NOT EXISTS push_log (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL,
    device_id       uuid NOT NULL REFERENCES push_devices(id) ON DELETE CASCADE,
    alert_id        uuid,
    title           text NOT NULL,
    body            text NOT NULL,
    data            jsonb NOT NULL DEFAULT '{}',
    status          text NOT NULL DEFAULT 'queued',
    fcm_message_id  text,
    apns_id         text,
    sent_at         timestamptz,
    error_message   text,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_push_log_tenant_time
    ON push_log(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_push_log_status
    ON push_log(status) WHERE status IN ('queued','failed');

-- mobile_sessions: per-device active-session tracker
CREATE TABLE IF NOT EXISTS mobile_sessions (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid NOT NULL,
    device_name     text,
    app_version     text,
    ip_address      text,
    user_agent      text,
    last_active_at  timestamptz NOT NULL DEFAULT now(),
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_mobile_sessions_user
    ON mobile_sessions(user_id, last_active_at DESC);
