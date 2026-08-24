# Checklist: Tier 7 Phase 3 — Final 5 subtiers

**Feature**: 005-tier7-phase3
**Status**: Draft

## Spec quality
- [ ] Each user story has clear "Why this priority" rationale
- [ ] Each user story has independent test path
- [ ] All acceptance scenarios are testable
- [ ] Out-of-scope items explicitly listed
- [ ] Risks documented with mitigations
- [ ] Done criteria are measurable

## Plan quality
- [ ] 6 phases (Phase 0 routes split + 5 subtiers + final verify)
- [ ] Phase 0 marked as CRITICAL (blocks all subsequent phases)
- [ ] Each phase has verification gate
- [ ] No phase skips verification
- [ ] File modifications enumerated per phase

## Task quality
- [ ] Tasks are 2-5 minutes each
- [ ] Each task has a clear "done" signal
- [ ] Final success criteria measurable
- [ ] Tasks ordered by dependency (Phase 0 → 1 → 2 → ...)

## Verification-first compliance
- [ ] Every phase ends with: type-check + lint + build exit 0
- [ ] Live changes verified end-to-end
- [ ] Bundle size checked at every phase
- [ ] Live verifier 4× back-to-back at end

## Modularity compliance
- [ ] All files under 400 LOC
- [ ] One concern per file
- [ ] Shared components in `components/shared/`
- [ ] Page-local components stay co-located
- [ ] No god-components
- [ ] routes.go ≤ 200 LOC (after Phase 0 split)
- [ ] routes_protected.go ≤ 400 LOC

## Datadog parity (Tier 7 completion)
- [ ] All 11 Tier 7 subtiers shipped after this change
- [ ] CI/CD matches Datadog pipelines
- [ ] DB Mon matches Datadog DBM
- [ ] Service Mgmt matches Datadog incidents
- [ ] Notebook matches Datadog notebooks
- [ ] Team/Collab matches Datadog shared dashboards

## Backend compliance
- [ ] All 30 routes registered under proper auth middleware
- [ ] All migrations idempotent
- [ ] All handlers have proper input validation
- [ ] All handlers honor tenant_id
- [ ] routes.go + routes_protected.go both under 400 LOC

## Frontend compliance
- [ ] All new pages use existing motion variants
- [ ] All new pages honor prefers-reduced-motion
- [ ] All new shared components used in ≥2 places
- [ ] All new pages use existing design tokens
- [ ] AppSidebar nav items added for all 5 new pages

## Deploy compliance
- [ ] Build green before scp
- [ ] Scp api-gateway binary + restart ios-api-gateway
- [ ] Scp web bundle + update index.html
- [ ] All 30 routes return 200/401
- [ ] Live verifier 4× back-to-back PASS

## Archive compliance
- [ ] Move change folder to `.hermes/changes/archive/2026-08-24-005-tier7-phase3/`
- [ ] Update journal.md with "Tier 7 COMPLETE" marker
- [ ] Remove WIP / .bak files