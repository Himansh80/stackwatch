# Implementation Plan: Tier 7 Phase 2 — Synthetics Full + Security + CSPM

**Feature**: 004-tier7-phase2
**Spec**: [spec.md](spec.md)
**Created**: 2026-08-24
**Status**: Ready for tasks

## Approach

Three-phase build with verification at every boundary. Each subtier (Synthetics, Security, CSPM) ships end-to-end (migration + handlers + shared components + pages) in its own phase.

1. **Phase 1 — Synthetics Full (D5)**: Migration 033 → synthetics_tests/synthetics_results/synthetics_locations → handlers_synthetics.go + handlers_synthetics_runner.go → 1 shared component (SlaBadge) + 1 page (SyntheticsPage)
2. **Phase 2 — Security (D6)**: Migration 034 → security_threats/compliance_rules/compliance_results/siem_events/audit_log_exports → handlers_security.go → 2 shared components (ThreatCard, ComplianceBar) + 1 page (SecurityPage)
3. **Phase 3 — CSPM (D7)**: Migration 035 → cspm_resources/cspm_findings → handlers_cspm.go → 1 shared component (CspmSeverityBadge) + 1 page (CspmPage)

Each phase is independently deployable. Phase 1 (Synthetics) is independent; Phase 2 (Security) reuses existing audit_log; Phase 3 (CSPM) is independent.

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Background runner as separate file | Synthetics test execution is heavy; isolate it | One more file to maintain |
| Stub browser-test runner | Full headless Chrome is too heavy for v1 | Multi-step tests have limited assertions |
| Compliance rules seeded via migration | Reusable defaults for PCI/SOC2/GDPR | Less customizable |
| CSPM scanned on-demand | Cloud scanning is expensive | Not real-time |

## Phase 1 — Synthetics Full (D5)

### 1.1 — Migration 033
- 3 tables: synthetics_tests, synthetics_results, synthetics_locations
- All idempotent (`CREATE TABLE IF NOT EXISTS`), FKs with `ON DELETE CASCADE`
- Apply to prod DB on `.116`

### 1.2 — Handlers
- `handlers_synthetics.go` (~350 LOC, 13 routes): CRUD tests + run-now + SLA + results + CI configs + webhook
- `handlers_synthetics_runner.go` (~200 LOC): background goroutine that polls `synthetics_tests` every 30s, executes enabled tests (HTTP via Go net/http, TCP via net.Dial, ICMP via external ping command), inserts results, evaluates SLA

### 1.3 — Routes
- Register 13 synthetics routes under `protected` group
- Add background runner `go runner.RunSynthetics(ctx, pool)` in `main.go`

### 1.4 — Shared components
- `SlaBadge.tsx` (~60 LOC): green/amber/red pill based on uptime % vs target

### 1.5 — Pages
- `SyntheticsPage.tsx` (~200 LOC): KPI strip (passing/failing/paused counts) + tests table + run-now button

### 1.6 — Sidebar nav
- Add "Synthetics" link

## Phase 2 — Security (D6)

### 2.1 — Migration 034
- 5 tables: security_threats, compliance_rules, compliance_results, siem_events, audit_log_exports
- Seed `compliance_rules` with default PCI/SOC2/GDPR rules

### 2.2 — Handlers
- `handlers_security.go` (~250 LOC, 4 routes): threats + audit + compliance + SIEM
- Reuse existing `audit_log` table for audit-trail (no migration)

### 2.3 — Routes
- Register 4 security routes

### 2.4 — Shared components
- `ThreatCard.tsx` (~80 LOC): severity pill + threat type + source IP + description
- `ComplianceBar.tsx` (~100 LOC): stacked bar showing %pass/%fail per framework

### 2.5 — Pages
- `SecurityPage.tsx` (~150 LOC): 3 tabs (Threats / Compliance / SIEM)

### 2.6 — Sidebar nav
- Add "Security" link

## Phase 3 — CSPM (D7)

### 3.1 — Migration 035
- 2 tables: cspm_resources, cspm_findings
- Backfill initial resources from existing Proxmox hosts

### 3.2 — Handlers
- `handlers_cspm.go` (~200 LOC, 2 routes): resources + findings
- `scanResource()` helper that takes a resource, evaluates rules, returns findings

### 3.3 — Routes
- Register 2 CSPM routes

### 3.4 — Shared components
- `CspmSeverityBadge.tsx` (~50 LOC): critical/high/medium/low pill

### 3.5 — Pages
- `CspmPage.tsx` (~120 LOC): resources table + findings table with severity filter

### 3.6 — Sidebar nav
- Add "CSPM" link

## Phase 4 — Verify-first sweep + deploy + archive

Same as before:
- Final build green
- Subagent reviews
- Deploy to `.115`
- Live 4× verifier
- Archive folder created
- journal.md updated

## Risks + mitigations

| Risk | Mitigation |
|------|------------|
| Background test runner overloading DB | Rate limit per tenant; max 100 concurrent globally |
| Multi-step browser tests need headless Chrome | Stub for v1 |
| CSPM scanning is expensive | Manual trigger only; cache 1h |
| SIEM event volume | Filter severity >= medium by default |

## Done criteria

- [ ] All 3 user stories pass
- [ ] Phase 1-3 verification gates green
- [ ] Phase 4 verify-first sweep passes
- [ ] Live 4× verifier PASS
- [ ] Bundle JS gzipped ≤ baseline + 60KB
- [ ] All files under 400 LOC
- [ ] All motion variants use useReducedMotion()
- [ ] Migrations applied to prod
- [ ] All 19 routes registered + verified
- [ ] Archive folder created
- [ ] journal.md updated