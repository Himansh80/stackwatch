# Checklist: Tier 7 Phase 2 — Synthetics Full + Security + CSPM

**Feature**: 004-tier7-phase2
**Status**: Draft

## Spec quality
- [ ] Each user story has clear "Why this priority" rationale
- [ ] Each user story has independent test path
- [ ] All acceptance scenarios are testable (Given/When/Then)
- [ ] Out-of-scope items explicitly listed
- [ ] Risks documented with mitigations
- [ ] Done criteria are measurable

## Plan quality
- [ ] 4 phases with clear boundaries
- [ ] Architecture decisions with rationale + trade-offs table
- [ ] Each phase has verification gate
- [ ] No phase skips verification
- [ ] File modifications enumerated per phase
- [ ] Bundle budget per phase

## Task quality
- [ ] Tasks are 2-5 minutes each
- [ ] Each task has a clear "done" signal
- [ ] Final success criteria measurable
- [ ] Tasks ordered by dependency

## Verification-first compliance
- [ ] Every phase ends with: type-check + lint + build exit 0
- [ ] Live changes verified end-to-end
- [ ] Bundle size checked at every phase
- [ ] Live verifier 4× back-to-back at end

## Subagent-driven compliance
- [ ] Implementer subagent dispatched per phase
- [ ] Spec-compliance reviewer subagent dispatched
- [ ] Code-quality reviewer subagent dispatched
- [ ] Review loops until APPROVED

## Modularity compliance
- [ ] All files under 400 LOC
- [ ] One concern per file
- [ ] Shared components in `components/shared/`
- [ ] Page-local components stay co-located
- [ ] No god-components

## Datadog parity
- [ ] Synthetics matches Datadog test types + SLA
- [ ] Security matches Datadog threats + compliance + SIEM
- [ ] CSPM matches Datadog CSPM resource + findings pattern

## Backend compliance
- [ ] All 19 routes registered under proper auth middleware
- [ ] All migrations idempotent
- [ ] All handlers have proper input validation
- [ ] All handlers honor tenant_id
- [ ] Background runner has per-tenant rate limit

## Frontend compliance
- [ ] All new pages use existing motion variants
- [ ] All new pages honor prefers-reduced-motion
- [ ] All new shared components used in ≥2 places
- [ ] All new pages lazy-loaded via React.lazy()
- [ ] All new pages use existing design tokens

## Deploy compliance
- [ ] Build green before scp
- [ ] Scp api-gateway binary + restart ios-api-gateway
- [ ] Scp web bundle + update index.html
- [ ] All 19 routes return 200/401
- [ ] Live verifier 4× back-to-back PASS

## Archive compliance
- [ ] Move change folder to `.hermes/changes/archive/2026-08-24-004-tier7-phase2/`
- [ ] Update journal.md
- [ ] Remove WIP / .bak files