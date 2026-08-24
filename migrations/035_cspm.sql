-- Tier 7 Phase 2 — CSPM (D7)
--
-- Two tables for the Cloud Security Posture Management surface.
-- All tenant-scoped. Phase 3 ships read-only endpoints (list resources / list
-- findings); actual scanning + remediation ship in a future change.
--
-- - cspm_resources: one row per cloud resource tracked by CSPM
--                   (Proxmox VM/LXC, TrueNAS dataset, AWS/GCP/Azure bucket/volume)
-- - cspm_findings:  one row per misconfiguration detected during a scan
--
-- Notes:
--   * resource_id is a TEXT identifier, NOT a UUID — it points at the
--     native identifier on whatever provider's API (Proxmox VMID, AWS ARN,
--     ZFS dataset path, etc). NO foreign key constraint.
--   * Both tables are tenant-scoped; every read honors tenant_id from JWT.
--   * Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.
--   * pgcrypto is required for gen_random_uuid() on older Postgres; keep the
--     ensure-installed line for safety even though most installs already have it.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- cspm_resources: a cloud resource enrolled in posture management.
CREATE TABLE IF NOT EXISTS cspm_resources (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       uuid NOT NULL,
  provider        text NOT NULL,                       -- 'proxmox' | 'truenas' | 'aws' | 'gcp' | 'azure'
  resource_type   text NOT NULL,                       -- 'vm' | 'lxc' | 'dataset' | 'bucket' | 'volume'
  resource_id     text NOT NULL,                       -- native identifier on the provider
  name            text NOT NULL,
  region          text,
  config          jsonb DEFAULT '{}'::jsonb,
  last_scanned_at timestamptz DEFAULT now(),
  UNIQUE (tenant_id, provider, resource_type, resource_id)
);
CREATE INDEX IF NOT EXISTS cspm_resources_tenant_provider_idx
  ON cspm_resources (tenant_id, provider);

-- cspm_findings: one row per misconfiguration detected during a scan.
-- Severity taxonomy mirrors security_threats so the same UI tone
-- mapping works on both surfaces.
CREATE TABLE IF NOT EXISTS cspm_findings (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  resource_id   text NOT NULL,                         -- matches cspm_resources.resource_id (no FK — text identifier)
  severity      text NOT NULL,                         -- 'critical' | 'high' | 'medium' | 'low' | 'info'
  finding_type  text NOT NULL,                         -- 'open_port' | 'weak_credential' | 'unencrypted_volume' | etc.
  description   text NOT NULL,
  remediation   text,
  detected_at   timestamptz DEFAULT now(),
  resolved_at   timestamptz,
  UNIQUE (tenant_id, resource_id, finding_type)
);
CREATE INDEX IF NOT EXISTS cspm_findings_tenant_severity_idx
  ON cspm_findings (tenant_id, severity, detected_at DESC);