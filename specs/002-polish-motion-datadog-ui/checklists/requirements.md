# Checklist: Polish — Motion + Datadog-grade UI

**Feature**: 002-polish-motion-datadog-ui
**Status**: Draft

## Spec quality

- [ ] Each user story has clear "Why this priority" rationale
- [ ] Each user story has independent test path
- [ ] All acceptance scenarios are testable (Given/When/Then)
- [ ] Out-of-scope items explicitly listed (Tier 7 features deferred to 003+)
- [ ] Risks documented with mitigations
- [ ] Done criteria are measurable

## Plan quality

- [ ] 5 phases with clear boundaries
- [ ] Architecture decisions with rationale + trade-offs table
- [ ] Each phase has verification gate
- [ ] No phase skips verification
- [ ] File modifications enumerated per phase
- [ ] Bundle budget per phase (≤baseline, +10KB, +30KB, +60KB)

## Task quality

- [ ] Tasks are 2-5 minutes each
- [ ] Each task has a clear "done" signal
- [ ] Final success criteria measurable
- [ ] Tasks ordered by dependency (can't do Phase 3 before Phase 1)

## Verification-first compliance (verification-first-completion skill)

- [ ] Every phase ends with: type-check + lint + build exit 0
- [ ] Visual changes verified with Playwright pixel-diff
- [ ] Bundle size checked at every phase
- [ ] Live verifier 4× back-to-back at end
- [ ] Evidence saved to `C:\Users\himan\AppData\Local\Temp\`

## Subagent-driven compliance (subagent-driven-development skill)

- [ ] Implementer subagent dispatched per task group (not per file)
- [ ] Spec-compliance reviewer subagent dispatched before code-quality reviewer
- [ ] Code-quality reviewer subagent dispatched last
- [ ] Each subagent gets full context (no plan re-read)
- [ ] Review loops until APPROVED

## Anti-slop compliance (agent-design-intelligence skill)

- [ ] No placeholder text
- [ ] No stock photos
- [ ] Spacing on the 4/8/12/16/24/32/48/64/96 scale
- [ ] Type on the 12/14/16/18/20/24/30/36/48/60/72 scale
- [ ] Color tokens (no hardcoded hex in components)
- [ ] WCAG AA contrast (4.5:1 body, 3:1 large)
- [ ] Hover states on all interactive elements
- [ ] Loading states with shimmer animation
- [ ] Empty states with EmptyState component (not raw "No data")
- [ ] Error states with friendly message + retry
- [ ] Mobile responsive (320px + 1920px)

## Framer-motion coverage

- [ ] All 12 pages use motion
- [ ] Reduced-motion honored via `useReducedMotion()`
- [ ] No per-page bespoke variants (use lib/motion.tsx)
- [ ] AnimatePresence only where list reordering matters (CommandPalette)

## Modularity compliance (MODULARITY_RULES.md)

- [ ] All files under 400 LOC
- [ ] One concern per file
- [ ] Shared components in `components/shared/`
- [ ] Page-local components stay in `components/<page>/` until reused
- [ ] No god-components

## Datadog visual parity

- [ ] KPI strip matches Datadog pattern (rounded card + top stripe + hover lift)
- [ ] Hosts table matches Datadog columns (status / hostname / IP / OS / sparkline / last-seen / actions)
- [ ] Status pill colors match Datadog convention (green/amber/red with text variants)
- [ ] Command palette matches Linear ⌘K quality
- [ ] Empty states designed (not raw "No data")

## Deploy compliance

- [ ] Build green before scp
- [ ] Scp to .115 with checksum verify
- [ ] serve.py restart successful
- [ ] All routes return HTTP 200
- [ ] Live verifier 4× back-to-back PASS
- [ ] Evidence retained

## Archive compliance

- [ ] Move change folder to `.hermes/changes/archive/2026-08-24-002-polish-motion-datadog-ui/`
- [ ] Update `_INDEX.md`
- [ ] Update `journal.md`
- [ ] Remove WIP / .bak files from working tree