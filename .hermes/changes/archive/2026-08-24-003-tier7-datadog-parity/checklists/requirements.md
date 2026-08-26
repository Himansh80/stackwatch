# Checklist: Tier 7 — Datadog Full Platform Parity (Phase 1 of 11)

**Feature**: 003-tier7-datadog-parity
**Status**: Draft

## Spec quality
- [ ] Each user story has clear "Why this priority" rationale
- [ ] Each user story has independent test path
- [ ] All acceptance scenarios are testable (Given/When/Then)
- [ ] Out-of-scope items explicitly listed (8 Tier 7 subtiers deferred to 004-007)
- [ ] Risks documented with mitigations
- [ ] Done criteria are measurable

## Plan quality
- [ ] 4 phases with clear boundaries
- [ ] Architecture decisions with rationale + trade-offs table
- [ ] Each phase has verification gate
- [ ] No phase skips verification
- [ ] File modifications enumerated per phase
- [ ] Bundle budget per phase (≤+30KB, ≤+60KB, ≤+80KB total)

## Task quality
- [ ] Tasks are 2-5 minutes each
- [ ] Each task has a clear "done" signal
- [ ] Final success criteria measurable
- [ ] Tasks ordered by dependency (Phase 1 → 2 → 3 → 4)

## Verification-first compliance (verification-first-completion skill)
- [ ] Every every phase ends with: type-check + lint + build exit 0
- [ ] Live changes verified end-to-end
- [ ] Bundle size checked at every phase
- [ ] Live verifier 4× back-to-back at end
- [ ] Evidence saved to `C:\Users\himan\AppData\Local\Temp\`

## Subagent-driven compliance (subagent-driven-development skill)
- [ ] Implementer subagent dispatched per phase
- [ ] Spec-compliance reviewer subagent dispatched before code-quality reviewer
- [ ] Code-quality reviewer subagent dispatched last
- [ ] Each subagent gets full context (no plan re-read)
- [ ] Review loops until APPROVED

## Modularity compliance (MODULARITY_RULES.md)
- [ ] All files under 400 LOC
- [ ] One concern per file
- [ ] Shared components in `components/shared/`
- [ ] Page-local components stay co-located
- [ ] No god-components

## Datadog parity
- [ ] APM trace model matches Datadog (services + traces + spans + deployments)
- [ ] Log Mgmt Full surface matches Datadog (monitors + archives + rehydrations + retention + patterns)
- [ ] RUM Full surface matches Datadog (web vitals + resources + interactions + long tasks + errors + sessions)

## Backend compliance
- [ ] All 30 routes registered under proper auth middleware
- [ ] All migrations idempotent (CREATE IF NOT EXISTS)
- [ ] All handlers have proper input validation (binding tags)
- [ ] All handlers honor tenant_id from JWT (no cross-tenant leaks)
- [ ] All handlers log audit events for create/update/delete

## Frontend compliance
- [ ] All new pages use existing motion variants (pageEnter, kpiStagger, etc.)
- [ ] All new pages honor prefers-reduced-motion
- [ ] All new shared components used in ≥2 places (or kept page-local)
- [ ] All new pages lazy-loaded via React.lazy()
- [ ] All new pages use existing design tokens (no hardcoded hex)

## Deploy compliance
- [ ] Build green before scp
- [ ] Scp api-gateway binary to .117 → run deploy script → restart ios-api-gateway
- [ ] Scp web bundle to .115 → update index.html → verify HTTP 200
- [ ] All 30 routes return 200 (or auth-gated 401)
- [ ] Live verifier 4× back-to-back PASS
- [ ] Evidence retained

## Archive compliance
- [ ] Move change folder to `.hermes/changes/archive/2026-08-24-003-tier7-datadog-parity/`
- [ ] Update journal.md
- [ ] Remove WIP / .bak files from working tree