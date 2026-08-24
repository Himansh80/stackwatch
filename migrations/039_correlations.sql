-- Tier 8 — Intelligence & Alerting (006-tier8-intelligence-alerting)
-- Phase 3: Alert Correlation + RCA (Tier 8.3).
--
-- Adds three tables for the alert-correlation + root-cause-analysis
-- surface:
--
--   1. alert_correlations    — groups of related alerts (root + members)
--   2. rca_hints             — root-cause-analysis suggestions per alert
--   3. correlation_feedback  — user feedback on correlations (for learning)
--
-- Naming: same `intelligence_` prefix as Phase 1+2 — keeps the
-- intelligence surface namespaced from Tier 6's streaming anomaly
-- tables. The Phase 1 anomaly tables + Phase 2 predictive tables
-- stay untouched.
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS) so re-applying
-- this migration is a no-op. No data migration is needed — the tables
-- start empty; Phase 3 surfaces correlation + RCA on the existing
-- anomaly_events / predictive_alerts streams.
--
-- ---------------------------------------------------------------------------
-- alert_correlations — one row per (tenant, correlation_id) group.
--
-- `correlation_id` is the GROUP identifier (multiple member_alert_ids
-- rows can share the same correlation_id). `root_alert_id` is the
-- alert the correlator identified as the most likely trigger of the
-- group. `member_alert_ids` is the rest of the related alerts.
--
-- similarity_score is a 0..1 measure of how tightly clustered the
-- members are (1 = identical fingerprint / same metric + server + 5-min
-- window; 0 = nothing in common).
--
-- auto_detected = true when the background correlator produced the row;
-- false when an operator created it via POST /correlations/manual
-- (useful so the UI can tag "manual" vs "auto" badges).
--
-- The (tenant_id, correlation_id) index makes the "list groups" query
-- a single seek; (root_alert_id) makes "what group is this alert in?"
-- a fast lookup. The arrays of UUIDs are queried via ANY() in the
-- group-detail handler.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS alert_correlations (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    correlation_id      uuid NOT NULL,
    root_alert_id       uuid NOT NULL,
    member_alert_ids    uuid[] NOT NULL DEFAULT '{}',
    similarity_score    double precision NOT NULL DEFAULT 0,
    auto_detected       boolean NOT NULL DEFAULT true,
    created_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_alert_correlations_group
    ON alert_correlations(tenant_id, correlation_id);

CREATE INDEX IF NOT EXISTS idx_alert_correlations_root
    ON alert_correlations(root_alert_id);

CREATE INDEX IF NOT EXISTS idx_alert_correlations_member
    ON alert_correlations USING GIN (member_alert_ids);

-- ---------------------------------------------------------------------------
-- rca_hints — one row per (tenant, alert_id) suggestion.
--
-- `likely_root` is a short human-readable string ("resource_saturation",
-- "deployment_regression", "network_partition", "disk_pressure",
-- etc.) that the UI renders as a tag. confidence is 0..1.
--
-- `reasoning` is a longer free-text explanation ("3 critical alerts
-- fired within 5 min on srv-prod-04 — likely resource saturation")
-- that the UI shows in a quote-block on the CorrelationCard.
--
-- `similar_past_incidents` is a JSONB array of past incident
-- summaries: [{id, summary, similarity, occurred_at}]. Default
-- empty so the handler can build the array incrementally without
-- INSERT-vs-UPDATE branching.
--
-- (tenant_id, alert_id) index makes "RCA hints for this alert" a
-- single seek.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS rca_hints (
    id                       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    alert_id                 uuid NOT NULL,
    likely_root              text NOT NULL,
    confidence               double precision NOT NULL DEFAULT 0,
    reasoning                text NOT NULL,
    similar_past_incidents   jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at               timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rca_hints_alert
    ON rca_hints(tenant_id, alert_id);

-- ---------------------------------------------------------------------------
-- correlation_feedback — one row per user-judgement on a correlation.
--
-- Captured by POST /correlations/feedback. user_id comes from the
-- JWT (never the body). `useful` is true (helpful) / false (wrong /
-- misleading). `note` is an optional free-text reason. This is the
-- raw material for a future "learn-from-feedback" model improvement
-- pass (Phase 5+).
--
-- (tenant_id, correlation_id) index makes "show feedback history for
-- this correlation" a single seek; the primary key is just an
-- internal id because the same user may submit feedback multiple
-- times (we want to keep history, not enforce uniqueness).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS correlation_feedback (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    correlation_id  uuid NOT NULL,
    user_id         uuid NOT NULL,
    useful          boolean NOT NULL,
    note            text,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_correlation_feedback_lookup
    ON correlation_feedback(tenant_id, correlation_id);