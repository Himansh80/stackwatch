-- Tier 7 Phase 2 — Security (D6)
--
-- Five tables for the security surface (threats + compliance + SIEM + audit exports).
-- All tenant-scoped. Phase 2 ships read-only endpoints (list/dashboard); writes
-- arrive in a future change when ingest pipelines land.
--
-- - security_threats:        one row per detected threat (severity + source_ip + actor)
-- - compliance_rules:        one row per rule definition, UNIQUE on (tenant, framework, rule_id)
-- - compliance_results:      one row per evaluation outcome, FK CASCADE from compliance_rules
-- - siem_events:             one row per security event streamed from any source
-- - audit_log_exports:       one row per requested export job
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.
-- pgcrypto is required for gen_random_uuid() on older Postgres; keep the
-- ensure-installed line for safety even though most installs already have it.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- security_threats: a threat detection (login anomaly, suspicious IP, etc.).
CREATE TABLE IF NOT EXISTS security_threats (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  severity      text NOT NULL,                       -- 'low' | 'medium' | 'high' | 'critical'
  threat_type   text NOT NULL,                       -- 'failed_login' | 'brute_force' | 'anomalous_ip' | 'data_exfil' | etc.
  source_ip     text,
  user_id       uuid,                                -- nullable: some threats aren't tied to a user
  description   text NOT NULL,
  detected_at   timestamptz DEFAULT now(),
  resolved_at   timestamptz,
  metadata      jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS security_threats_tenant_detected_idx
  ON security_threats (tenant_id, detected_at DESC);

-- compliance_rules: a rule definition. Seeded below with PCI/SOC2/GDPR defaults.
CREATE TABLE IF NOT EXISTS compliance_rules (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  framework     text NOT NULL,                       -- 'pci' | 'soc2' | 'gdpr' | 'hipaa'
  rule_id       text NOT NULL,
  description   text NOT NULL,
  severity      text NOT NULL DEFAULT 'medium',
  remediation   text,
  UNIQUE (tenant_id, framework, rule_id)
);

-- compliance_results: most-recent evaluation outcome for a (rule, resource) pair.
-- Old results stay around so the audit trail is preserved.
CREATE TABLE IF NOT EXISTS compliance_results (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  rule_id       uuid NOT NULL REFERENCES compliance_rules(id) ON DELETE CASCADE,
  resource_id   text,
  status        text NOT NULL,                       -- 'pass' | 'fail' | 'unknown'
  evidence      text,
  evaluated_at  timestamptz DEFAULT now()
);
CREATE INDEX IF NOT EXISTS compliance_results_tenant_rule_idx
  ON compliance_results (tenant_id, rule_id);

-- siem_events: a single event streamed from any source (auth, network, app).
CREATE TABLE IF NOT EXISTS siem_events (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  event_type    text NOT NULL,                       -- 'auth_success' | 'auth_failure' | 'network_anomaly' | etc.
  severity      text NOT NULL,                       -- 'low' | 'medium' | 'high' | 'critical'
  message       text NOT NULL,
  source        text,                                -- e.g. 'auth-svc', 'firewall', 'app:web'
  occurred_at   timestamptz DEFAULT now(),
  metadata      jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX IF NOT EXISTS siem_events_tenant_occurred_idx
  ON siem_events (tenant_id, occurred_at DESC);

-- audit_log_exports: a request to export the legacy audit_log table to JSON/CSV/syslog.
-- The export worker (future change) polls rows in status='queued'.
CREATE TABLE IF NOT EXISTS audit_log_exports (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  format        text NOT NULL,                       -- 'json' | 'csv' | 'syslog'
  destination   text NOT NULL,                       -- path or URL
  start_time    timestamptz NOT NULL,
  end_time      timestamptz NOT NULL,
  status        text NOT NULL DEFAULT 'queued',      -- 'queued' | 'running' | 'completed' | 'failed'
  created_at    timestamptz DEFAULT now()
);

-- Seed default compliance rules. ON CONFLICT DO NOTHING makes this safe to
-- re-run after a manual rule edit (the spec only inserts the seed set; users
-- can add their own rules via a future write endpoint and we must not
-- overwrite them).
INSERT INTO compliance_rules (tenant_id, framework, rule_id, description, severity, remediation)
SELECT t.id, r.framework, r.rule_id, r.description, r.severity, r.remediation
FROM tenants t
CROSS JOIN (VALUES
  -- PCI DSS baseline rules
  ('pci',  'PCI-1.1',  'Encrypt cardholder data at rest and in transit',                                    'high',     'Enable TLS for all cardholder data flows; encrypt storage with AES-256 or stronger.'),
  ('pci',  'PCI-2.1',  'No default passwords on system components',                                          'critical', 'Rotate every vendor default credential; enforce per-host uniqueness.'),
  ('pci',  'PCI-6.5',  'Protect against common application vulnerabilities (XSS, SQLi, etc.)',              'high',     'Enable WAF; enforce input validation on every endpoint; run quarterly pen tests.'),
  -- SOC 2 baseline rules
  ('soc2', 'SOC2-CC1.1','Logical access controls enforce least privilege',                                  'medium',   'Enforce RBAC; quarterly access reviews; revoke on role change.'),
  ('soc2', 'SOC2-CC7.1','Detect and respond to security events via monitoring',                             'medium',   'Forward logs to SIEM; alert on auth failures, privilege changes, exfil attempts.'),
  -- GDPR baseline rule
  ('gdpr', 'GDPR-25',  'Data protection by design and by default',                                           'high',     'Minimize PII collection; encrypt at rest; document lawful basis per processing activity.')
) AS r(framework, rule_id, description, severity, remediation)
ON CONFLICT (tenant_id, framework, rule_id) DO NOTHING;