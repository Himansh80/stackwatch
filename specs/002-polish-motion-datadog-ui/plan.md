# Implementation Plan: Polish — Motion + Datadog-grade UI

**Feature**: 002-polish-motion-datadog-ui
**Spec**: [spec.md](spec.md)
**Created**: 2026-08-24
**Status**: Ready for tasks

## Approach

Five-phase polish with verification at every phase boundary. Every phase ends with: build green, type-check green, lint green, bundle check, Playwright pixel-diff vs baseline, 4× live verifier on `.115`.

1. **Phase 1**: Extend `lib/motion.tsx` + `styles/base.css` with the new tokens. Foundation — no consumer changes yet. Verify build still works.
2. **Phase 2**: Promote `dashboard/StatusPill` → `shared/StatusPill`. Create `shared/KpiCard`, `shared/EmptyState`, `shared/SkeletonRow`. Migrate Dashboard, Profile, Settings, Billing to use shared components. Verify per-page pixel-diff.
3. **Phase 3**: Wire `pageEnter` on every route. Wire `kpiStagger` on Dashboard. Wire `paletteEnter` on CommandPalette. Wire `statusPulse` on StatusPill when status is critical/down.
4. **Phase 4**: Datadog visual parity — KPI top stripe, hosts table columns, status pill contrast, reduced-motion handling. Anti-slop pass.
5. **Phase 5**: Verify-first sweep with all skills and tools. Live 4× verifier on `.115`. Archive.

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Promote StatusPill to `components/shared/` | Used by 5+ pages; single source of truth | One rename; all imports change |
| New `shared/KpiCard` instead of per-page `MetricCard` | Pages consistency is the single biggest signal | One new component; Dashboard loses its bespoke MetricCard |
| Motion variants in `lib/motion.tsx` (no per-page variants) | Same source of truth, easy to update | Variant explosion stays out of pages |
| `pageEnter` at page level (not route) | Page owns its mount animation; route doesn't need framer-motion | Slightly more boilerplate per page |
| All new shared components use `useReducedMotion()` from `motion.tsx` | Honor user preference automatically | One import per component |
| Skip `AnimatePresence` on route changes | Vite + React Router already handle transitions; AnimatePresence adds complexity | No exit animation when navigating |
| Don't introduce a state-management library | Zustand/Jotai not yet justified; useState + URL state is enough | Same as today |

## Phase 1 — Tokens & motion library (foundation)

### 1.1 — Add tokens to `web/src/styles/base.css`

Add a new `:root` block (after existing tokens, before body):
- `--motion-page: 240ms cubic-bezier(0.16, 1, 0.3, 1);`
- `--elev-1: 0 1px 0 rgba(0,0,0,.4), 0 1px 2px rgba(0,0,0,.3);`
- `--elev-2: 0 4px 12px rgba(0,0,0,.45), 0 2px 4px rgba(0,0,0,.3);`
- `--chart-stroke: 1.5px;`

Verify `npm run build` exits 0 and CSS bundle size unchanged.

### 1.2 — Extend `web/src/lib/motion.tsx`

Add the 6 new exports from spec:
- `kpiEnter`, `kpiStagger`
- `statusPulse`
- `sparklineDraw`
- `pageEnter`
- `paletteEnter`
- `shimmer`

Plus a `useReducedMotion()` re-export (already exists).

Verify `npm run type-check` exit 0 and the file passes lint.

### Phase 1 verification

- `npm run type-check` exit 0
- `npm run lint` exit 0
- `npm run build` exit 0
- CSS bundle gzipped size unchanged
- JS bundle gzipped size +0 (no new usage yet)
- Commit: `chore(polish): add motion + elevation tokens (no consumer changes)`

## Phase 2 — Shared components

### 2.1 — Promote StatusPill

- Move `web/src/components/dashboard/StatusPill.tsx` → `web/src/components/shared/StatusPill.tsx`
- Update all consumers (Dashboard, Profile, Settings, Billing, TrueNASWorkspace, Proxmox workspace)
- Verify build green

### 2.2 — Create `web/src/components/shared/KpiCard.tsx`

Props: `label: string`, `value: string | number`, `delta?: string`, `icon?: ReactNode`, `status?: 'up' | 'stale' | 'down' | 'crit' | 'neutral'`, `accent?: 'cyan' | 'green' | 'amber' | 'red' | 'indigo' | 'violet'`.

Visual contract:
- Border-radius: `--radius`
- Background: `--surface`
- Border: 1px `--border`
- Padding: 24px
- Top edge: 2px stripe in `accent` color
- Hover: `cardLift.whileHover` (already in motion.tsx)
- Body: label uppercase 11px / value 36px mono / delta 12px

Wrap with `motion.div` using `kpiEnter`.

### 2.3 — Create `web/src/components/shared/EmptyState.tsx`

Props: `illustration: ReactNode`, `headline: string`, `subhead?: string`, `cta?: { label: string, onClick: () => void }`, `icon?: ReactNode`.

Visual contract:
- Centered, max-width 480px
- Illustration 96×96 with `--accent-soft` background
- Headline 24px / 600
- Subhead 14px / `--muted`
- CTA: primary button style (existing `buttonSpring`)

### 2.4 — Create `web/src/components/shared/SkeletonRow.tsx`

Props: `columns: number`, `rows?: number`.

Visual contract:
- Rows with shimmer animation (1.4s loop)
- Uses `shimmer` variant from motion.tsx
- `--surface-2` background, `--surface-3` highlight

### 2.5 — Migrate Dashboard to KpiCard

- Delete `web/src/components/dashboard/MetricCard.tsx`
- Update `Dashboard.tsx` to import `KpiCard` from `shared/`
- Verify Dashboard still renders identically (pixel-diff)

### 2.6 — Migrate ProfilePage

- Update ProfilePage to use shared components where appropriate (KpiCard for workspace stats)
- Verify ProfilePage still renders identically (pixel-diff)

### Phase 2 verification

- `npm run type-check` exit 0
- `npm run lint` exit 0
- `npm run build` exit 0
- Playwright pixel-diff: Dashboard, Profile both 0px vs pre-Phase-2 baseline
- Bundle JS gzipped ≤ baseline + 10KB (KpiCard + StatusPill dedup saves more than new cost)
- Commit per file or per migration

## Phase 3 — Motion wiring

### 3.1 — `pageEnter` on every page

For each of: Dashboard, Profile, Settings, Billing, TrueNASWorkspace, Landing, Login, Signup, ForgotPassword, ResetPassword:
- Wrap top-level return with `<motion.div initial="hidden" animate="show" variants={pageEnter}>`
- Landing/Login/Signup/Forgot/Reset already have motion — verify they use the same `pageEnter` shape (replace bespoke variants)

### 3.2 — `kpiStagger` on Dashboard's KPI strip

- The 4 KPI cards use `motion.div` with parent `variants={kpiStagger}` and child `variants={kpiEnter}`
- Verify only animates on mount (not on data refresh)

### 3.3 — `paletteEnter` on CommandPalette

- Add AnimatePresence wrapping the palette panel
- `variants={paletteEnter}` with `initial="hidden"`, `animate="show"`, `exit="exit"`
- Backdrop: opacity fade in/out

### 3.4 — `statusPulse` on StatusPill when status=down or crit

- Conditional `animate` prop on the dot
- Verify pulse respects `prefers-reduced-motion`

### 3.5 — `sparklineDraw` on Sparkline component

- Apply `pathLength` animation on mount
- Verify chart still updates correctly on data refresh (no re-animation)

### Phase 3 verification

- `npm run type-check` exit 0
- `npm run lint` exit 0
- `npm run build` exit 0
- Visual: open Dashboard, confirm KPI cards stagger-in within 320ms
- Visual: open CommandPalette (⌘K), confirm fade+scale entrance
- Visual: trigger a DOWN status (kill an agent), confirm StatusPill dot pulses
- Bundle JS gzipped ≤ baseline + 30KB
- Commit per motion type

## Phase 4 — Datadog visual parity

### 4.1 — KPI top stripe color tokens

Confirm KpiCard's top stripe uses accent color tokens, not hardcoded hex.

### 4.2 — Hosts table columns

Update `web/src/components/dashboard/HostList.tsx` to columns:
1. Status pill (UP/STALE/DOWN)
2. Hostname (mono font)
3. IP (mono)
4. OS pill
5. Sparkline (24×48)
6. Last seen (relative time)
7. Actions (kebab `⋮`)

Verify on Dashboard.

### 4.3 — Status pill contrast

Audit: `--green-text` on `--green-soft`, `--amber-text` on `--amber-soft`, `--red-text` on `--red-soft`. Confirm WCAG AA (4.5:1).

### 4.4 — Row hover state

`web/src/components/dashboard/HostList.tsx`: row hover → background `--surface-2`, left-edge 2px accent stripe in status color, transition 150ms.

### 4.5 — Empty state illustrations

Audit all empty states across Dashboard, Profile, Settings, Billing, TrueNASWorkspace. Replace any "No data" text with `EmptyState` component using Lucide-style SVG illustrations.

### 4.6 — Error states

Audit all error states. Replace raw error stack with friendly message + retry button.

### 4.7 — Loading states

Confirm `SkeletonCard` (Dashboard) and `SkeletonRow` (new) use the `shimmer` variant from motion.tsx.

### 4.8 — Reduced motion audit

Verify every motion variant honors `prefers-reduced-motion`. Use Playwright `emulateMedia({ reducedMotion: 'reduce' })` and confirm zero animations.

### Phase 4 verification

- All Phase 3 verifications still pass
- Anti-slop checklist: 11/11 items pass
  - [ ] No placeholder text
  - [ ] No stock photos
  - [ ] Consistent spacing (4/8/12/16/24/32/48/64/96 scale)
  - [ ] Proper hierarchy (eye goes to most important first)
  - [ ] WCAG AA contrast (4.5:1 body, 3:1 large)
  - [ ] Functional hover states
  - [ ] Loading states with shimmer
  - [ ] Empty states with EmptyState component
  - [ ] Error states with friendly message + retry
  - [ ] Mobile responsive (320px + 1920px)
  - [ ] Micro-interactions (button spring, status pulse, page enter)
- Visual diff vs Datadog: 5-axis comparison on hosts table (status pill / font / sparkline / hover / last-seen)
- Bundle JS gzipped ≤ baseline + 60KB
- Commit per polish area

## Phase 5 — Verify-first sweep + deploy + archive

### 5.1 — Pre-deploy verifier (use verification-first-completion skill)

Run `npm run type-check && npm run lint && npm run build`. All exit 0.

### 5.2 — Bundle audit

Measure final bundle:
- `dist/assets/index-*.js` (gzipped)
- `dist/assets/index-*.css` (gzipped)

Compare to baseline from spec §"Acceptance scenarios".

### 5.3 — Subagent spec-compliance review (use subagent-driven-development skill)

Dispatch a fresh subagent to verify:
- All 6 user stories from spec.md pass
- All Phase 1-4 done criteria met
- No scope creep (no new backend endpoints, no Tier 7 features)

### 5.4 — Subagent code-quality review

Dispatch a fresh subagent to verify:
- All files under 400 LOC (per MODULARITY_RULES.md)
- No god-components
- All shared components used in ≥2 places (otherwise move back to page-local)
- Type-check + lint + build all exit 0

### 5.5 — Deploy to `.115`

Build, then scp to `.115`, restart serve.py, verify HTTP 200 on all routes.

### 5.6 — Live 4× verifier

Run a verifier script 4 back-to-back times. Each run must pass 100%.

Verifier checks:
- `GET /` returns 200
- All 12 routes return 200
- Login flow works (200 + token)
- Dashboard KPI cards stagger-in (Playwright capture)
- Command palette opens with motion (Playwright capture)
- StatusPulse triggers when status=down (seed data)
- Reduced motion: zero animations (Playwright emulate)
- Bundle size in budget

Save the verifier script to `C:\Users\himan\AppData\Local\Temp\hermes-verify-002-polish-2026-08-24.py`.

### 5.7 — Archive

Move to `.hermes/changes/archive/2026-08-24-002-polish-motion-datadog-ui/`.

Update `_INDEX.md` (master project index) with this change entry.

Update `.hermes/changes/` parent listing.

### Phase 5 verification

- All previous phase verifications still pass
- 4× back-to-back verifier PASS (4 × N checks = 100%)
- Archived change folder present
- `_INDEX.md` updated
- `journal.md` entry written

## Risks + mitigations

| Risk | Mitigation |
|------|------------|
| Bundle bloat from framer-motion in every component | Tree-shake via named imports; lazy-load framer-motion in shared components only; verify budget at every phase |
| Reduced-motion users get jarring experience | Centralize `useReducedMotion` in `motion.tsx`; every variant honors it; Playwright emulate verifies |
| Visual regressions break existing flows | Pixel-diff before/after every page change; speckit 001-split-page-modules acceptance scenarios must still pass |
| Animation fights with live data updates | KPI cards animate only on mount; data refresh doesn't re-animate |
| Subagent spec-creep (Tier 7 features added) | Strict scope-creep check in spec-compliance review; subagent prompt explicitly forbids new endpoints |

## Done criteria

- [ ] All 6 user stories pass acceptance scenarios
- [ ] Phase 1-4 verification green at each boundary
- [ ] Phase 5 verify-first sweep passes (verification-first-completion skill)
- [ ] Phase 5 subagent spec-compliance review PASS
- [ ] Phase 5 subagent code-quality review APPROVED
- [ ] Live 4× verifier on `.115` PASS (4 × N checks = 100%)
- [ ] Bundle JS gzipped ≤ baseline + 60KB
- [ ] Bundle CSS gzipped unchanged
- [ ] `npm run type-check` exit 0
- [ ] `npm run lint` exit 0
- [ ] `npm run build` exit 0
- [ ] prefers-reduced-motion verified via Playwright emulate
- [ ] Archive folder created
- [ ] `_INDEX.md` + `journal.md` updated