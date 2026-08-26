# Feature Specification: Tier 7 Phase 2 — Synthetics Full (D5) + Security (D6) + CSPM (D7)

**Feature Branch**: `004-tier7-phase2`
**Created**: 2026-08-24
**Status**: Draft
**Source change folder**: `.hermes/changes/004-tier7-phase2/`
**Master plan reference**: `MASTER_BUILD_PLAN.md` §"Tier 7 — 7.4 Synthetics Full / 7.5 Security / 7.6 CSPM"

**Input**: Ship the next 3 Tier 7 subtiers end-to-end.

## Goal

Bring StackWatch closer to Datadog parity with 3 of the remaining 8 Tier 7 subtiers:

1. **7.4 Synthetics Full (D5)** — Multi-step browser tests, API tests, ICMP tests, CI configs, SLA, SLA results.
2. **7.5 Security (D6)** — Threat detection, audit trails, compliance rules, SIEM, audit log exports.
3. **7.6 CSPM (D7)** — Cloud Security Posture Management — scan cloud resources, find misconfigurations.

## Out of scope (future speckit changes)

- 7.7 CI/CD Visibility (D8)
- 7.8 DB Monitoring (D9)
- 7.9 Service Management (D10)
- 7.10 Notebook (D11)
- 7.11 Team/Collab (D12)

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Synthetics (Priority: P1)

A user opens StackWatch → Synthetics. They see a list of tests with status (passing/failing/paused). Each test has SLA (target uptime %, response time budget). They create a new multi-step browser test that:
1. Navigates to login page
2. Fills in credentials
3. Clicks submit
4. Waits for redirect
5. Asserts dashboard text appears

They run the test on-demand. Failures alert via existing channels.

**Why P1**: Synthetics = proactive monitoring. Without it, StackWatch only detects problems after users see them.

**Acceptance scenarios**:
1. Given a user has at least one test, **when** they open Synthetics, **then** the test list renders with status pill + SLA pill.
2. Given a user creates a multi-step browser test, **when** they trigger run-now, **then** the test executes and returns pass/fail within SLA budget.

### User Story 2 — Security (Priority: P1)

A user opens StackWatch → Security. They see:
- Threat feed (failed logins, suspicious IPs, anomalies)
- Compliance dashboard (PCI / SOC 2 / GDPR rules with pass/fail)
- SIEM events list
- Audit log with filterable search

**Acceptance scenarios**:
1. Given a tenant has audit log entries, **when** they open Security, **then** threats + compliance + audit events render in 3 tabs.

### User Story 3 — CSPM (Priority: P2)

A user opens StackWatch → CSPM. They see cloud resources (Proxmox VMs, TrueNAS datasets, etc.) and any misconfigurations (open ports, weak credentials, missing encryption).

**Acceptance scenarios**:
1. Given a tenant has registered Proxmox/TrueNAS hosts, **when** they open CSPM, **then** resources + findings render with severity badges.

## Database additions (migrations)

### Migration 033: Synthetics tables
```sql
CREATE TABLE synthetics_tests (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  name text NOT NULL,
  type text NOT NULL,  -- 'http' | 'tcp' | 'icmp' | 'browser' | 'multi_step'
  url text NOT NULL,
  method text DEFAULT 'GET',
  body text,
  headers jsonb DEFAULT '{}'::jsonb,
  assertions jsonb DEFAULT '[]'::jsonb,
  interval_seconds int NOT NULL DEFAULT 300,
  timeout_ms int NOT NULL DEFAULT 30000,
  locations text[] DEFAULT '{}',
  sla_uptime_pct decimal DEFAULT 99.9,
  sla_response_ms int DEFAULT 1000,
  enabled boolean DEFAULT true,
  created_at timestamptz DEFAULT now()
);
CREATE INDEX ON synthetics_tests (tenant_id, enabled);

CREATE TABLE synthetics_results (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  test_id uuid NOT NULL REFERENCES synthetics_tests(id) ON DELETE CASCADE,
  started_at timestamptz NOT NULL,
  duration_ms int NOT NULL,
  status text NOT NULL,  -- 'pass' | 'fail' | 'timeout' | 'error'
  response_code int,
  error_message text,
  assertions_passed int DEFAULT 0,
  assertions_failed int DEFAULT 0,
  response_body_excerpt text
);
CREATE INDEX ON synthetics_results (tenant_id, test_id, started_at DESC);

CREATE TABLE synthetics_locations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  name text NOT NULL,
  region text NOT NULL,
  enabled boolean DEFAULT true,
  UNIQUE (tenant_id, name)
);
```

### Migration 034: Security tables
```sql
CREATE TABLE security_threats (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  severity text NOT NULL,  -- 'low' | 'medium' | 'high' | 'critical'
  threat_type text NOT NULL,
  source_ip text,
  user_id uuid,
  description text NOT NULL,
  detected_at timestamptz DEFAULT now(),
  resolved_at timestamptz,
  metadata jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX ON security_threats (tenant_id, detected_at DESC);

CREATE TABLE compliance_rules (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  framework text NOT NULL,  -- 'pci' | 'soc2' | 'gdpr' | 'hipaa'
  rule_id text NOT NULL,
  description text NOT NULL,
  severity text NOT NULL DEFAULT 'medium',
  remediation text,
  UNIQUE (tenant_id, framework, rule_id)
);

CREATE TABLE compliance_results (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  rule_id uuid NOT NULL REFERENCES compliance_rules(id) ON DELETE CASCADE,
  resource_id text,
  status text NOT NULL,  -- 'pass' | 'fail' | 'unknown'
  evidence text,
  evaluated_at timestamptz DEFAULT now()
);

CREATE TABLE siem_events (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  event_type text NOT NULL,
  severity text NOT NULL,
  message text NOT NULL,
  source text,
  occurred_at timestamptz DEFAULT now(),
  metadata jsonb DEFAULT '{}'::jsonb
);
CREATE INDEX ON siem_events (tenant_id, occurred_at DESC);

CREATE TABLE audit_log_exports (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  format text NOT NULL,  -- 'json' | 'csv' | 'syslog'
  destination text NOT NULL,
  start_time timestamptz NOT NULL,
  end_time timestamptz NOT NULL,
  status text NOT NULL DEFAULT 'queued',
  created_at timestamptz DEFAULT now()
);
```

### Migration 035: CSPM tables
```sql
CREATE TABLE cspm_resources (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  provider text NOT NULL,  -- 'proxmox' | 'truenas' | 'aws' | 'gcp' | 'azure'
  resource_type text NOT NULL,  -- 'vm' | 'lxc' | 'dataset' | 'bucket' | 'volume'
  resource_id text NOT NULL,
  name text NOT NULL,
  region text,
  config jsonb DEFAULT '{}'::jsonb,
  last_scanned_at timestamptz DEFAULT now(),
  UNIQUE (tenant_id, provider, resource_type, resource_id)
);

CREATE TABLE cspm_findings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id uuid NOT NULL,
  resource_id text NOT NULL,
  severity text NOT NULL,
  finding_type text NOT NULL,  -- 'open_port' | 'weak_credential' | 'unencrypted_volume' | etc.
  description text NOT NULL,
  remediation text,
  detected_at timestamptz DEFAULT now(),
  resolved_at timestamptz,
  UNIQUE (tenant_id, resource_id, finding_type)
);
CREATE INDEX ON cspm_findings (tenant_id, severity, detected_at DESC);
```

## Backend API additions

### Synthetics (handlers_synthetics.go)
- `GET /api/v1/synthetics/locations` — list test locations
- `POST /api/v1/synthetics/locations` — create location
- `GET /api/v1/synthetics/tests-full` — list tests
- `POST /api/v1/synthetics/tests-full` — create test
- `GET /api/v1/synthetics/tests-full/:test_id` — test detail
- `PUT /api/v1/synthetics/tests-full/:test_id` — update test
- `DELETE /api/v1/synthetics/tests-full/:test_id` — delete test
- `POST /api/v1/synthetics/tests-full/:test_id/run` — run now
- `GET /api/v1/synthetics/tests-full/:test_id/sla` — SLA report (uptime %, p95 response time, breach count)
- `GET /api/v1/synthetics/tests-full/:test_id/results` — recent results
- `POST /api/v1/synthetics/ci-configs` — CI integration (GitHub Actions etc.)
- `GET /api/v1/synthetics/ci-configs` — list CI configs
- `POST /api/v1/synthetics/webhook` — webhook receiver for external systems

### Security (handlers_security.go)
- `GET /api/v1/security/threats` — list threats
- `GET /api/v1/security/audit-trails` — list audit events (alias to existing audit_log)
- `GET /api/v1/security/compliance` — compliance dashboard (PCI/SOC2/GDPR rules + status)
- `GET /api/v1/security/siem` — SIEM event stream

### CSPM (handlers_cspm.go)
- `GET /api/v1/cspm/resources` — list cloud resources
- `GET /api/v1/cspm/findings` — list misconfigurations

## Files this change modifies (estimated)

### Backend (~1100 LOC across 4 handler files)
- `migrations/033_synthetics.sql` (NEW)
- `migrations/034_security.sql` (NEW)
- `migrations/035_cspm.sql` (NEW)
- `internal/handler/handlers_synthetics.go` (NEW, ~350 LOC, 13 routes)
- `internal/handler/handlers_security.go` (NEW, ~250 LOC, 4 routes)
- `internal/handler/handlers_cspm.go` (NEW, ~200 LOC, 2 routes)
- `internal/handler/handlers_synthetics_runner.go` (NEW, ~200 LOC, background test executor)
- `cmd/api-gateway/routes.go` (+19 routes)

### Frontend (~700 LOC across 4 components + 3 pages)
- `web/src/components/shared/SlaBadge.tsx` (NEW, ~60 LOC)
- `web/src/components/shared/ThreatCard.tsx` (NEW, ~80 LOC)
- `web/src/components/shared/ComplianceBar.tsx` (NEW, ~100 LOC)
- `web/src/components/shared/CspmSeverityBadge.tsx` (NEW, ~50 LOC)
- `web/src/pages/SyntheticsPage.tsx` (NEW, ~200 LOC)
- `web/src/pages/SecurityPage.tsx` (NEW, ~150 LOC)
- `web/src/pages/CspmPage.tsx` (NEW, ~120 LOC)
- `web/src/components/AppSidebar.tsx` (+3 nav items)

## Risks

| Risk | Mitigation |
|------|------------|
| Background test runner overloading DB | Rate limit per tenant; max 100 concurrent tests globally |
| Multi-step browser tests need headless Chrome | Stub it for Phase 2; full implementation in future change |
| CSPM scanning is expensive | Manual trigger only; cache results 1h |
| SIEM event volume | Filter by severity >= medium by default |

## Done criteria

- [ ] All 3 user stories pass acceptance scenarios
- [ ] 19 routes registered + verified live (401 unauth, 200/201 auth)
- [ ] 3 migrations applied to prod
- [ ] All files under 400 LOC
- [ ] Bundle JS gzipped ≤ (current baseline + 60KB)
- [ ] Archive folder created
- [ ] journal.md updated
- [ ] Ready to start 005 (Synthetics Full + remaining Tier 7)