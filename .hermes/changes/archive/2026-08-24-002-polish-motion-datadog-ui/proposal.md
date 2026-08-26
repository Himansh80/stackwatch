# Proposal: Polish — Motion + Datadog-grade UI

**Change folder**: `.hermes/changes/002-polish-motion-datadog-ui/`
**Spec folder**: `specs/002-polish-motion-datadog-ui/`
**Created**: 2026-08-24
**Status**: Draft → Ready for apply

## Why

The dashboard ships with 5 of 12 pages having **zero framer-motion** (Dashboard, Profile, Settings, Billing, TrueNASWorkspace). `framer-motion ^11.18.2` is installed and `lib/motion.tsx` has 11 motion exports — but only Landing/Login/Signup/Forgot/Reset use them. Meanwhile, Datadog's dashboard is the gold standard: KPI cards stagger in, status pills pulse, hosts tables have sparklines, command palette is a power-user feature.

We have the foundation. We need to apply it. Then Tier 7 (Datadog parity features) inherits this polish automatically.

## What changes

### Frontend only — no backend

| Change | Files |
|---|---|
| Add 4 design tokens (motion-page, elev-1/2, chart-stroke) | `web/src/styles/base.css` |
| Add 7 motion exports (kpiEnter, kpiStagger, statusPulse, sparklineDraw, pageEnter, paletteEnter, shimmer) | `web/src/lib/motion.tsx` |
| Promote StatusPill → shared | `web/src/components/shared/StatusPill.tsx` (was `dashboard/`) |
| New KpiCard | `web/src/components/shared/KpiCard.tsx` |
| New EmptyState | `web/src/components/shared/EmptyState.tsx` |
| New SkeletonRow | `web/src/components/shared/SkeletonRow.tsx` |
| Wire `pageEnter` on every page | 5 pages (Dashboard, Profile, Settings, Billing, TrueNASWorkspace) |
| Wire `kpiStagger` on Dashboard | `Dashboard.tsx` + KpiCard |
| Wire `paletteEnter` on CommandPalette | `CommandPalette.tsx` |
| Wire `statusPulse` on StatusPill | `shared/StatusPill.tsx` |
| Wire `sparklineDraw` on Sparkline | `web/src/components/dashboard/Sparkline.tsx` |
| Datadog parity pass on hosts table | `web/src/components/dashboard/HostList.tsx` |

### Bundle budget

- **Baseline**: 43KB gzipped JS (current, post 001-split-page-modules)
- **Target**: ≤ 103KB gzipped JS (+60KB ceiling)
- **CSS**: unchanged

### Migration

- **No database migration**
- **No API changes**
- **No env changes**
- **No new dependencies** (framer-motion already installed)

## Impact

| Area | Impact |
|---|---|
| Backend | None |
| Database | None |
| Frontend | 13 files modified, 3 new files |
| Build | +60KB JS gzipped max, +0 CSS |
| Tests | All existing Playwright + unit tests must still pass |
| Docs | `_INDEX.md` + `journal.md` updates after archive |
| Breaking changes | None — visual + motion only, all behavior preserved |

## Scope discipline

This change is **visual + motion only**. Any subagent that tries to add Tier 7 features (APM, full RUM, Security, etc.) or new backend endpoints is out of scope. Speckit-001 acceptance scenarios must still pass — no behavior changes.

## Done when

- [ ] All 6 user stories in spec.md pass acceptance criteria
- [ ] Phase 1-4 verifications green at every boundary
- [ ] Phase 5 verification-first sweep passes
- [ ] Subagent spec-compliance review PASS
- [ ] Subagent code-quality review APPROVED
- [ ] Live 4× verifier on `.115` PASS (4 × N checks = 100%)
- [ ] Bundle JS gzipped ≤ baseline + 60KB
- [ ] Bundle CSS gzipped unchanged
- [ ] `npm run type-check` exit 0
- [ ] `npm run lint` exit 0
- [ ] `npm run build` exit 0
- [ ] prefers-reduced-motion verified via Playwright emulate
- [ ] Archive folder created at `.hermes/changes/archive/2026-08-24-002-polish-motion-datadog-ui/`
- [ ] `_INDEX.md` + `journal.md` updated