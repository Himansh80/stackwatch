-- Tier 8 — Intelligence & Alerting (006-tier8-intelligence-alerting)
-- Phase 4: Alert Deduplication + Noise Reduction (Tier 8.4).
--
-- Adds two tables for the noise-reduction surface:
--
--   1. alert_noise_rules — operator-authored suppression rules. Each rule
--      matches alerts by `fingerprint_pattern` (a glob-ish LIKE pattern on
--      the metric_name fingerprint, e.g. `*cpu*`, `disk.fill.*`,
--      `host.node-04.*`). When matched, the rule suppresses alerts in that
--      channel set for `suppression_window_seconds`.
--
--   2. snooze_log — per-alert manual snooze actions. Every snooze is a
--      row with an `expires_at`; the noise-reduction engine treats a
--      snoozed alert as suppressed until the row expires. A history
--      query returns the most recent N rows for an audit trail.
--
-- Naming: NO `intelligence_` prefix here — these tables are noise-only
-- (no overlap with the anomaly / predictive / correlation surfaces), and
-- the spec is clear that names should be plain. The intelligence_ prefix
-- is reserved for tables that back the existing Tier 8 streaming
-- surfaces; noise rules and snooze history are an orthogonal concern.
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS) so re-applying
-- this migration is a no-op. No data migration is needed — the tables
-- start empty; Phase 4 surfaces rules + snooze history alongside the
-- existing anomaly / predictive / correlation streams.
--
-- ---------------------------------------------------------------------------
-- alert_noise_rules — one row per (tenant_id, fingerprint_pattern) rule.
--
-- `fingerprint_pattern` is a glob pattern applied to the alert's
-- metric_name / fingerprint at evaluation time. We use SQL `LIKE`
-- semantics with `%` wildcards; the handler converts any leading /
-- trailing `*` from the UI to `%` before the query.
--
-- `suppression_window_seconds` is the rolling window in which
-- duplicate matches are suppressed (e.g. 3600 = "suppress for the
-- next hour").
--
-- `channels` is the subset of notification channels the rule applies
-- to: {'email', 'slack', 'pagerduty', 'webhook'}. Empty array means
-- "all channels".
--
-- `enabled` lets operators temporarily disable a rule without
-- deleting it (deleted rules lose audit history).
--
-- The (tenant_id, fingerprint_pattern) index makes "list rules by
-- pattern" a single seek; the (tenant_id, enabled) index makes
-- "list enabled rules for evaluation" cheap.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS alert_noise_rules (
    id                          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                        text NOT NULL,
    fingerprint_pattern         text NOT NULL,
    suppression_window_seconds  integer NOT NULL DEFAULT 3600,
    channels                    text[] NOT NULL DEFAULT '{}',
    enabled                     boolean NOT NULL DEFAULT true,
    created_at                  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alert_noise_rules_pattern
    ON alert_noise_rules(tenant_id, fingerprint_pattern);

CREATE INDEX IF NOT EXISTS idx_alert_noise_rules_enabled
    ON alert_noise_rules(tenant_id, enabled);

-- ---------------------------------------------------------------------------
-- snooze_log — one row per manual snooze action.
--
-- Created by POST /api/v1/noise/snooze. `user_id` is stamped from the
-- JWT (never the body) so a user can't snooze on behalf of someone
-- else. `expires_at = now() + duration_seconds` at insert time —
-- the noise-reduction engine queries `expires_at > now()` to decide
-- whether a snooze is still active.
--
-- `reason` is an optional free-text note from the operator ("known
-- noisy metric", "in maintenance window", etc.) that the history
-- endpoint surfaces for audit.
--
-- (tenant_id, alert_id) makes "is this alert snoozed?" a single
-- seek; (tenant_id, expires_at) supports "list active snoozes"
-- and "list recently expired snoozes" queries.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS snooze_log (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    alert_id          uuid NOT NULL,
    user_id           uuid NOT NULL,
    duration_seconds  integer NOT NULL,
    expires_at        timestamptz NOT NULL,
    reason            text,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_snooze_log_alert
    ON snooze_log(tenant_id, alert_id);

CREATE INDEX IF NOT EXISTS idx_snooze_log_expiry
    ON snooze_log(tenant_id, expires_at);
