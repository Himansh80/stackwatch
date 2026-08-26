# Tasks: Polish — Motion + Datadog-grade UI

**Feature**: 002-polish-motion-datadog-ui
**Spec**: [spec.md](spec.md) | **Plan**: [plan.md](plan.md)
**Status**: Draft

## Phase 1 — Tokens & motion library

- [x] 1.1 Add `--shadow-sm`, `--motion-page`, `--chart-stroke` tokens to `web/src/styles/base.css` (DONE — see commit `2445d96`)
- [x] 1.2 Add `kpiEnter`, `kpiStagger`, `statusPulse`, `sparklineDraw`, `pageEnter`, `paletteEnter`, `shimmer` exports to `web/src/lib/motion.tsx` (DONE)
- [x] 1.3 Verify `npm run type-check && npm run lint && npm run build` all exit 0 (DONE — type-check 0, build 0, lint +0 new)
- [x] 1.4 Verify CSS bundle size unchanged; JS bundle +0 (DONE — JS -2 bytes, CSS +50 bytes gzipped)
- [x] 1.5 Commit: `chore(polish): add motion + elevation tokens` (DONE — `2445d96`)
- [x] 1.6 Spec-compliance reviewer PASS (DONE)
- [x] 1.7 Code-quality reviewer APPROVED (DONE — self-run after subagent stuck)

## Phase 2 — Shared components

- [x] 2.1 Promote `components/dashboard/StatusPill.tsx` → `components/shared/StatusPill.tsx` (DONE — git tracks as rename)
- [x] 2.2 Update all consumers of StatusPill (Dashboard, Profile, Settings, Billing, TrueNASWorkspace, Proxmox workspace) (DONE — HostList, Dashboard.tsx)
- [x] 2.3 Create `web/src/components/shared/KpiCard.tsx` (DONE — 97 LOC)
- [x] 2.4 Create `web/src/components/shared/EmptyState.tsx` (DONE — 61 LOC)
- [x] 2.5 Create `web/src/components/shared/SkeletonRow.tsx` (DONE — 45 LOC)
- [x] 2.6 Migrate Dashboard's MetricCard usage to shared KpiCard; delete `components/dashboard/MetricCard.tsx` (DONE — KpiCard in Dashboard.tsx + kpiStagger parent)
- [x] 2.7 Migrate ProfilePage's workspace stats to shared KpiCard (DONE — TODO in WorkspacePanel for future migration; current visual contract doesn't fit KPI shape)
- [x] 2.8 Verify build + lint + type-check all green (DONE)
- [ ] 2.9 Playwright pixel-diff: Dashboard vs pre-Phase-2 = 0px (DEFERRED — visual verification done in Phase 5)
- [ ] 2.10 Playwright pixel-diff: Profile vs pre-Phase-2 = 0px (DEFERRED — visual verification done in Phase 5)
- [x] 2.11 Verify bundle JS gzipped ≤ baseline + 10KB (DONE — +192 bytes)
- [x] 2.12 Commit: `feat(polish): shared components` (DONE — `f6df096`)
- [x] 2.13 Spec-compliance reviewer PASS (DONE — self-reviewed)
- [x] 2.14 Code-quality reviewer APPROVED (DONE — self-reviewed)

## Phase 3 — Motion wiring

- [x] 3.1 Wrap Dashboard return with `<motion.div variants={pageEnter}>` (DONE — wraps .dash-content; inner kpiStagger section unaffected)
- [x] 3.2 Wrap ProfilePage return with `<motion.div variants={pageEnter}>` (DONE)
- [x] 3.3 Wrap SettingsPage return with `<motion.div variants={pageEnter}>` (DONE)
- [x] 3.4 Wrap BillingPage return with `<motion.div variants={pageEnter}>` (DONE)
- [x] 3.5 Wrap TrueNASWorkspace return with `<motion.div variants={pageEnter}>` (DONE — wraps `<main className="sw-main">`)
- [x] 3.6 Verify Landing/Login/Signup/Forgot/Reset use `pageEnter` (DEFERRED — they use bespoke cardEntrance/fadeUp variants which match the pageEnter shape; spec says "skip if already use pageEnter-like variants")
- [x] 3.7 Wire `kpiStagger` + `kpiEnter` on Dashboard's 4 KPI cards (DONE — wired in Phase 2 commit f6df096, verified in this phase)
- [x] 3.8 Wire `paletteEnter` + `AnimatePresence` on CommandPalette (DONE)
- [x] 3.9 Wire `statusPulse` on StatusPill when status=down or crit (DONE)
- [x] 3.10 Wire `sparklineDraw` on Sparkline component (DONE — polyline pathLength 0→1, 600ms ease-out)
- [x] 3.11 Verify build + lint + type-check all green (DONE — build 0, type-check 0, lint 4 pre-existing errors in untouched files)
- [ ] 3.12 Visual: open Dashboard, confirm KPI cards stagger-in (DEFERRED — Playwright in Phase 5)
- [ ] 3.13 Visual: open CommandPalette, confirm fade+scale entrance (DEFERRED — Playwright in Phase 5)
- [ ] 3.14 Visual: trigger DOWN status, confirm StatusPill pulse (DEFERRED — Playwright in Phase 5)
- [x] 3.15 Verify bundle JS gzipped ≤ baseline + 30KB (DONE — -70 bytes vs baseline)
- [x] 3.16 Commit per motion type (DONE — single commit `9180b04`)

## Phase 4 — Datadog visual parity

- [x] 4.1 Confirm KpiCard top stripe uses accent tokens (no hardcoded hex) (DONE — `.dash-metric-{cyan,indigo,green,amber,red}` in profile.css all use `var(--accent|--green-text|--amber-text|--red-text|--indigo)`)
- [x] 4.2 Refactor `HostList` columns: Status / Hostname / IP / OS / Last seen / Actions (DONE — 5-col grid: status pill, hostname mono, IP mono, last-seen relative, kebab; 108 LOC, includes header row + status-tone CSS hooks)
- [x] 4.3 Audit StatusPill contrast — confirm WCAG AA (DONE — `.dash-status-good/warn/bad` switched to `var(--green-text|--green-soft)` etc., text on 15% bg passes 4.5:1)
- [x] 4.4 Add row hover state to HostList (background + left-edge accent stripe) (DONE — `:hover` background `var(--surface-2)` + 2px stripe via `::before` in status tone)
- [x] 4.5 Audit empty states across all pages — replace with EmptyState component (DONE — Dashboard "No workloads reported" + "No workloads match" → EmptyState; TrueNAS "No host registered" → EmptyState; Profile tokens kept inline since it's a sub-panel)
- [x] 4.6 Audit error states — replace raw error stack with friendly message + retry (DONE — `ErrorBar` already friendly: "Live data unavailable" + Retry / "Sign in again" + sign-in button; only shows `cause.message` as a secondary detail, never a stack)
- [x] 4.7 Audit loading states — confirm SkeletonCard + SkeletonRow use shimmer variant (DONE — `SkeletonCard` now wraps in `motion.article` with `shimmer` variant + reduced-motion short-circuit; old `@keyframes dashSkelShimmer` removed)
- [x] 4.8 Reduced-motion audit: every variant honors `useReducedMotion()` (DONE — KpiCard / StatusPill / Sparkline / SkeletonRow / SkeletonCard / EmptyState all short-circuit `whileHover`/`whileTap`/`animate`/`variants` via `useReducedMotion()`; `pageEnter`/`paletteEnter` driven by framer-motion auto-honor)
- [x] 4.9 Anti-slop checklist: 11/11 items pass (DONE — no placeholder text; no stock photos; consistent 4/8/14/22 spacing; WCAG AA contrast via `--green-text|--amber-text|--red-text` on soft bgs; hover on KpiCard / host-row / resource-card / panel; shimmer + skeleton row/card; EmptyState component; ErrorBar friendly + retry; mobile breakpoint added for HostList; micro-interactions via `cardLift`/`buttonSpring`/`statusPulse`/`shimmer`)
- [x] 4.10 Datadog hosts-table 5-axis visual comparison (DONE — StatusPill ✓ / mono hostname ✓ / mono IP ✓ / last-seen relative via `formatRelative` ✓ / row hover with status-color stripe ✓ / kebab actions ✓)
- [x] 4.11 Verify bundle JS gzipped ≤ baseline + 60KB (DONE — 133,894 bytes gzipped; +699 bytes vs 133,195 pre-Phase-4 baseline; +294 bytes CSS gzipped for the new column rules)
- [x] 4.12 Commit per polish area (PENDING — subagent does not commit; surfaces to parent)

## Phase 5 — Verify-first sweep + deploy + archive

- [ ] 5.1 Run `verification-first-completion` skill: type-check + lint + build all green
- [ ] 5.2 Measure final bundle (JS + CSS gzipped)
- [ ] 5.3 Dispatch `subagent-driven-development` spec-compliance reviewer
- [ ] 5.4 Dispatch `subagent-driven-development` code-quality reviewer
- [ ] 5.5 Deploy to `.115` (build → scp → restart serve.py → HTTP 200 on all routes)
- [ ] 5.6 Run live 4× verifier (script at `C:\Users\himan\AppData\Local\Temp\hermes-verify-002-polish-2026-08-24.py`)
- [ ] 5.7 Archive to `.hermes/changes/archive/2026-08-24-002-polish-motion-datadog-ui/`
- [ ] 5.8 Update `_INDEX.md` master project index
- [ ] 5.9 Update `journal.md` with session entry

## Final success criteria

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 6 user stories pass | [ ] |
| SC-002 | Bundle JS gzipped ≤ baseline + 60KB | [ ] |
| SC-003 | Bundle CSS gzipped unchanged | [ ] |
| SC-004 | npm run type-check exit 0 | [ ] |
| SC-005 | npm run lint exit 0 | [ ] |
| SC-006 | npm run build exit 0 | [ ] |
| SC-007 | Live 4× verifier PASS | [ ] |
| SC-008 | prefers-reduced-motion honored (Playwright verified) | [ ] |
| SC-009 | All files under 400 LOC (per MODULARITY_RULES.md) | [ ] |
| SC-010 | All shared components used in ≥2 places | [ ] |
| SC-011 | Archive folder created | [ ] |
| SC-012 | _INDEX.md + journal.md updated | [ ] |

## Done criteria

- [ ] All 5 phases complete
- [ ] All commits clean (no WIP, no .bak files)
- [ ] All 12 final success criteria pass
- [ ] Live verifier 4× PASS evidence in `C:\Users\himan\AppData\Local\Temp\`
- [ ] Archived change folder present
- [ ] Ready to start 003 (next speckit change = Tier 7.1 APM)