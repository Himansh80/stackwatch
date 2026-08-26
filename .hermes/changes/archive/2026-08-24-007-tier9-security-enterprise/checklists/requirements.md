# Checklist: Tier 9 — Security & Enterprise

**Feature**: 007-tier9-security-enterprise
**Status**: Draft

## Spec quality
- [ ] Each user story has clear "Why this priority" rationale
- [ ] Each user story has independent test path
- [ ] All acceptance scenarios are testable
- [ ] Out-of-scope items explicitly listed (Tiers 10-13)
- [ ] Risks documented with mitigations
- [ ] Done criteria are measurable (10 SCs)

## Plan quality
- [ ] 7 phases (Phase 0 split + 6 subtiers + final verify)
- [ ] Phase 0 marked as CRITICAL (blocks all subsequent phases)
- [ ] Each phase has verification gate
- [ ] No phase skips verification
- [ ] File modifications enumerated per phase

## Task quality
- [ ] Tasks are 2-5 minutes each
- [ ] Each task has a clear "done" signal
- [ ] Final success criteria measurable (10 SCs)
- [ ] Tasks ordered by dependency (Phase 0 → 1 → 2 → ... → 6 → 7)

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
- [ ] routes_protected.go ≤ 410 LOC (after Phase 0 split)
- [ ] routes_enterprise.go ≤ 400 LOC (handles all 32 routes)

## Security compliance (strict)
- [ ] SSO provider configs encrypted at rest (use existing `internal/crypto` package)
- [ ] SCIM tokens hashed with bcrypt (cost ≥ 10) — plaintext returned ONCE on creation
- [ ] SAML signatures validated with x509 + audience + NotOnOrAfter
- [ ] OIDC JWTs validated with JWKS from discovery_url
- [ ] State parameter signed with HMAC + expiry ≤ 10 minutes
- [ ] RBAC permission checks happen on EVERY protected route (not just enterprise routes)
- [ ] Audit log archive downloads require valid JWT + tenant match
- [ ] Compliance reports require audit-read permission
- [ ] Org settings changes require super-admin

## Backend compliance
- [ ] All 32 routes registered under proper auth middleware
- [ ] All migrations idempotent (CREATE TABLE IF NOT EXISTS or ALTER TABLE ... IF NOT EXISTS)
- [ ] All handlers have proper input validation (binding tags)
- [ ] All handlers honor tenant_id from JWT
- [ ] All files under 400 LOC
- [ ] No new dependencies (use existing crypto + oauth2 packages if available)

## Frontend compliance
- [ ] All new pages use existing motion variants (pageEnter, kpiEnter, kpiStagger)
- [ ] All new pages honor prefers-reduced-motion
- [ ] All new shared components used in ≥1 place
- [ ] All new pages use existing design tokens (no hardcoded hex)
- [ ] AppSidebar nav item added for Enterprise

## Deploy compliance
- [ ] Build green before scp
- [ ] Scp api-gateway binary + restart ios-api-gateway (use skill `stackwatch-build-deploy`)
- [ ] All 32 routes return 401 (verify each)
- [ ] Live verifier 4× back-to-back PASS

## Archive compliance
- [ ] Move change folder to `.hermes/changes/archive/2026-08-24-007-tier9-security-enterprise/`
- [ ] Update journal.md with "TIER 9 COMPLETE" marker
- [ ] Remove WIP / .bak files