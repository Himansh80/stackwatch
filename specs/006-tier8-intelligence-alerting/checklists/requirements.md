# Checklist: Tier 8 — Intelligence & Alerting

**Feature**: 006-tier8-intelligence-alerting
**Status**: Draft

## Spec quality
- [ ] Each user story has clear "Why this priority" rationale
- [ ] Each user story has independent test path
- [ ] All acceptance scenarios are testable
- [ ] Out-of-scope items explicitly listed (Tiers 9-13)
- [ ] Risks documented with mitigations
- [ ] Done criteria are measurable (10 SCs)

## Plan quality
- [ ] 6 phases (Phase 0 split + 5 subtiers + final verify)
- [ ] Phase 0 marked as CRITICAL (blocks all subsequent phases)
- [ ] Each phase has verification gate
- [ ] No phase skips verification
- [ ] File modifications enumerated per phase

## Task quality
- [ ] Tasks are 2-5 minutes each
- [ ] Each task has a clear "done" signal
- [ ] Final success criteria measurable (10 SCs)
- [ ] Tasks ordered by dependency (Phase 0 → 1 → 2 → ... → 5 → 6)

## Verification-first compliance
- [ ] Every phase ends with: type-check + lint + build exit 0
- [ ] Live changes verified end-to-end
- [ ] Bundle size checked at every phase
- [ ] Live verifier 4× back-to-back at end

## Modularity compliance
- [ ] All Go files under 400 LOC
- [ ] One concern per file (handlers split by domain)
- [ ] Shared components in `components/shared/`
- [ ] Page-local components stay co-located
- [ ] No god-components
- [ ] routes_protected.go ≤ 350 LOC (after Phase 0 split)
- [ ] routes_intelligence.go ≤ 400 LOC (handles all 24 routes)

## Datadog style compliance
- [ ] AnomalyChart matches Datadog Watchdog sparkline pattern
- [ ] PredictionChart shows p10/p50/p90 band (Datadog Forecast pattern)
- [ ] CorrelationCard matches Datadog Correlations card pattern
- [ ] NoiseRuleEditor matches Datadog noise rule config UI
- [ ] RcaPanel matches Datadog RCA summary panel
- [ ] IntelligencePage is 4-tab layout (Anomalies / Predictions / Correlations / Noise)
- [ ] KPI strips with colored top stripes
- [ ] Status chips throughout
- [ ] All animations use existing motion exports

## Backend compliance
- [ ] All 24 routes registered under proper auth middleware
- [ ] All migrations idempotent (CREATE TABLE IF NOT EXISTS)
- [ ] All handlers have proper input validation (binding tags)
- [ ] All handlers honor tenant_id from JWT
- [ ] All files under 400 LOC
- [ ] No new dependencies

## Frontend compliance
- [ ] All new pages use existing motion variants (pageEnter, kpiEnter, kpiStagger)
- [ ] All new pages honor prefers-reduced-motion
- [ ] All new shared components used in ≥1 place
- [ ] All new pages use existing design tokens (no hardcoded hex)
- [ ] AppSidebar nav item added for Intelligence

## Deploy compliance
- [ ] Build green before scp
- [ ] Scp api-gateway binary + restart ios-api-gateway (use skill `stackwatch-build-deploy`)
- [ ] All 24 routes return 401 (verify each)
- [ ] Live verifier 4× back-to-back PASS

## Archive compliance
- [ ] Move change folder to `.hermes/changes/archive/2026-08-24-006-tier8-intelligence-alerting/`
- [ ] Update journal.md with "TIER 8 COMPLETE" marker
- [ ] Remove WIP / .bak files