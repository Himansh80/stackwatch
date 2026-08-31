# Tier 20 — UI Consistency Polish

**Speckit change ID:** `020-ui-consistency-polish`
**Status:** Proposal (awaiting user review)
**Author:** StackWatch Engineering
**Created:** 2026-08-31
**Target merge:** after user sign-off

---

## 1. Background

The StackWatch frontend was built tier-by-tier over 13 tiers (T1 → T14) plus
recent polish phases (013–021). UI patterns were established as needed but
**no single consistency audit has ever been run**. Initial sampling
(this session, 2026-08-31) found:

- **Dead-code sidebar** (`AppSidebar.tsx`, 173 LOC) using emoji icons in
  violation of `DESIGN-SYSTEM.md §10 Anti-slop` ("Emoji as primary UI
  language"). New sidebar (`sidebar/Sidebar.tsx`) is mounted by AppShell
  but the old file is still on disk.
- **8 topbar sub-components** (`TopBarLogo`, `TopBarGreeting`,
  `TopBarProfile`, `TopBarRefresh`, `TopBarSearch`, `TopBarTime`,
  `TopBarWeather`, `TopBarBreadcrumb`, `TopBarNotifications`) — split
  for size but composition has not been visually verified.
- **Token-compliance drift**: 233+ handler files + 78 components + 24
  pages — sample reads (`web/src/components/AppShell.tsx`,
  `web/src/components/sidebar/Sidebar.tsx`, `web/src/components/shared/KpiCard.tsx`)
  show mix of `var(--*)` tokens AND hardcoded hex values.
- **StatusPill / KpiCard shared components exist** (good — 16/27 pages
  import each) but the unused 11 likely have inline equivalents.

**The user requirement is explicit** (2026-08-31, point 13): "if the ui is
not broken, similar across each page each element, each button must be
similar looking if that element is used across the platform then it must
be behave similarly and also looks same the size and color and also the
kpi cards and other elements too."

This proposal addresses that requirement.

---

## 2. Goals (measurable)

| ID | Goal | Target | Verification |
|---|---|---|---|
| G1 | Every page uses `StatusPill` from `shared/` for status indicators | 27/27 pages | grep `<StatusPill` count == 27 |
| G2 | Every page uses `KpiCard` from `shared/` for KPI tiles | 27/27 pages | grep `<KpiCard` count == 27 |
| G3 | Zero hardcoded hex colors in `web/src/components/` or `web/src/pages/` | 0 occurrences | grep `'#[0-9a-f]{3,6}'` returns 0 |
| G4 | Zero emoji characters used as primary UI iconography | 0 in components | grep `[⌂$◉⚙◈▤◴≡◎◔◍◐⇄◧⚑◰⌬✦◊]` returns 0 in components/ |
| G5 | Every page has skip-link + main landmark | 27/27 pages | manual + axe-core scan |
| G6 | Every page is mobile-responsive at 320px and 1920px | 27/27 pages | Playwright @ 320×568 + 1920×1080 |
| G7 | Every interactive element has hover + focus + active state | 100% | grep `whileHover` coverage + manual |
| G8 | Every KPI value uses `font-variant-numeric: tabular-nums` | 100% | grep + visual |
| G9 | Every empty state uses `<EmptyState>` from shared/ | 27/27 pages | grep |
| G10 | Every loading state uses `<Skeleton>` from shared/ (or spinner pattern) | 27/27 pages | grep |
| G11 | Every page has framer-motion page-transition | 27/27 pages | visual + grep `AnimatePresence` |

---

## 3. Non-goals (explicit)

- ❌ No new feature work (this tier is polish-only).
- ❌ No backend changes.
- ❌ No new design tokens (use existing `tokens.css`).
- ❌ No redesign of layout, color palette, or typography.
- ❌ No tier-specific customization (e.g. Proxmox UI keeps its own chrome via `ProxmoxShell`).
- ❌ No icon additions (use existing `icons.tsx` 30-icon library).
- ❌ No accessibility library rewrites (use semantic HTML + ARIA only).

---

## 4. Scope

### In scope
- All files under `web/src/pages/` (27 .tsx files)
- All files under `web/src/components/` except `proxmox/` (78 → 78 files)
- All files under `web/src/styles/` (14 .css files + DESIGN-SYSTEM.md)
- Adding new shared components if audit reveals a missing pattern

### Out of scope (separate tier)
- `web/src/components/proxmox/` (covered by Tier 14 + future Proxmox-UI tier)
- Backend (no Go changes)
- Tests (test infrastructure is separate; visual regressions covered by
  Playwright in CI later)
- Mobile app (`mobile/` — separate React Native stack)

---

## 5. User stories

| ID | As a | I want | So that |
|---|---|---|---|
| US-01 | Operator using StackWatch | Every status pill (up/stale/down/crit/warn/ok) looks identical across every page | I can scan multiple pages without re-learning visual language |
| US-02 | Operator | Every KPI tile (label / value / delta / sparkline) is the same shape, padding, stripe color logic everywhere | My eyes don't have to re-calibrate per page |
| US-03 | Operator | Every button (primary / secondary / ghost / danger) has identical height, padding, hover | Click targets feel predictable |
| US-04 | Operator | Every page has a consistent empty state with action guidance | I know what to do next when a page is empty |
| US-05 | Operator | Every page transition uses 200ms ease-out | The app feels cohesive, not jarring |
| US-06 | Operator on mobile | Sidebar collapses to drawer, topbar stays, KPI strip wraps to 1-col | I can use the dashboard on a phone |
| US-07 | Operator with screen reader | Skip-link + ARIA labels on every interactive element | I can navigate without a mouse |
| US-08 | Operator in low light | All text meets WCAG AA (4.5:1 body) | I can read it |

---

## 6. Functional requirements

| ID | Requirement |
|---|---|
| FR-1 | `<StatusPill>` MUST be used for every status indicator (server up/stale/down, alert severity, alert state, etc.) |
| FR-2 | `<KpiCard>` MUST be used for every KPI tile; inline `<div class="metric">` is forbidden |
| FR-3 | Every button MUST use one of: `<Button variant="primary|secondary|ghost|danger">` (shared component to be added if not present) OR `class="btn-primary"` etc. — no inline `style={{ background: ... }}` |
| FR-4 | Every modal MUST use `<Modal>` from shared/ (create if missing) — no inline `position: fixed` overlays |
| FR-5 | Every table MUST use `<DataTable>` from shared/ — no inline `<table>` markup in pages |
| FR-6 | Every form input MUST use `<Input>`, `<Select>`, `<Textarea>` from shared/ |
| FR-7 | Every empty state MUST use `<EmptyState>` with a `title` + `description` + optional `action` |
| FR-8 | Every loading state MUST use `<Skeleton variant="card|row|block">` or `Spinner` from shared/ |
| FR-9 | Every page transition MUST be wrapped in `<AnimatePresence>` with `initial/animate/exit` (200ms ease-out) |
| FR-10 | Every interactive element MUST have `hover` + `focus-visible` + `active` states via CSS or framer-motion |
| FR-11 | Every page MUST have `<a href="#main" className="skip-link">Skip to main content</a>` |
| FR-12 | Every page MUST be navigable via keyboard (Tab order matches visual order) |
| FR-13 | Every color MUST come from `var(--*)` token — zero hardcoded hex outside `tokens.css` |

---

## 7. Non-functional requirements

| ID | Requirement |
|---|---|
| NFR-1 | No file >400 LOC after refactor |
| NFR-2 | Lighthouse accessibility score ≥90 on every page |
| NFR-3 | Bundle size increase <5% (current dist ~XXX KB) |
| NFR-4 | All existing functionality MUST keep working — no regression in click-through paths |
| NFR-5 | Every changed page MUST pass visual regression: screenshot diff vs previous ≤1% pixel delta |
| NFR-6 | Build (npm run build) MUST complete without warnings |
| NFR-7 | TypeScript strict-mode MUST compile with zero errors |

---

## 8. Acceptance criteria (verification gates)

- [ ] **AC-1**: Visual audit report (`docs/UI-AUDIT-REPORT.md`) lists every drift found
  with `file:line` and proposed fix
- [ ] **AC-2**: Every page imports `StatusPill` AND uses it ≥1 time (except pages
  with no status to display, e.g. Landing)
- [ ] **AC-3**: Every page with metrics imports `KpiCard` AND uses it ≥1 time
- [ ] **AC-4**: Zero hardcoded hex in `web/src/{components,pages}/` (grep clean)
- [ ] **AC-5**: Zero emoji-as-icon in `web/src/{components,pages}/` (grep clean)
- [ ] **AC-6**: Playwright screenshot of every page at 1440×900 saved to
  `verify/ui-screenshots/` and matches design reference within 1% pixel delta
- [ ] **AC-7**: Playwright screenshot of every page at 320×568 shows
  usable layout (no horizontal scroll, no overlapping elements)
- [ ] **AC-8**: axe-core accessibility scan returns 0 critical violations on every page
- [ ] **AC-9**: Lighthouse accessibility ≥90 on /dashboard, /proxmox-vms, /apm, /logs, /alerts, /settings
- [ ] **AC-10**: Bundle size delta <5%
- [ ] **AC-11**: 4 back-to-back full-stack verifier runs all PASS
- [ ] **AC-12**: User (himan) signs off after manually testing every page

---

## 9. Approach (high level)

### 9.1 Audit-first phase (mandatory before any change)

Spawn one **dedicated subagent** to:
1. Read every page (27) and every shared component (~34 files)
2. For each: catalog every status pill / KPI card / button / modal / table / form input
3. Output a `docs/UI-AUDIT-REPORT.md` with:
   - Per-file table of: element, current implementation, drift from design system
   - Severity rating (critical/medium/low) per drift
   - Proposed fix (specific code change or refactor)
   - Estimated effort

**Gate**: Audit must be complete and reviewed before any code changes.

### 9.2 Shared-component completion

Audit will likely reveal **missing shared components**. Add them in order:
1. `<Button>` (if missing) — variants: primary, secondary, ghost, danger
2. `<Modal>` (if missing) — with framer-motion enter/exit
3. `<DataTable>` (if missing) — sortable, paginated, with loading/empty states
4. `<Input>`, `<Select>`, `<Textarea>` (if missing) — typed bindings, error states

**Gate**: Each new component MUST have a 1-page showcase in
`web/src/pages/StyleGuide.tsx` for visual verification.

### 9.3 Page-by-page refactor

For each of 27 pages (in dependency order: shared → layout → pages):
1. Replace inline status pills with `<StatusPill>`
2. Replace inline KPI tiles with `<KpiCard>`
3. Replace inline buttons with `<Button>`
4. Replace inline modals with `<Modal>`
5. Replace inline tables with `<DataTable>`
6. Replace inline form inputs with `<Input>` / `<Select>`
7. Add framer-motion `<AnimatePresence>` page transition
8. Add skip-link + main landmark if missing
9. Remove hardcoded hex values → tokens
10. Verify mobile responsive (CSS only)

**Per-page verification**: After refactor, take Playwright screenshot,
compare to design reference, run axe-core scan. All three must pass
before marking page done.

### 9.4 Dead-code cleanup

Delete:
- `web/src/components/AppSidebar.tsx` (old sidebar, dead code)
- Any inline components duplicated in `shared/`
- Any `style={{...}}` blocks in components (move to CSS classes using tokens)

### 9.5 Verification

- Run all 4 back-to-back verifier scripts
- Run Playwright matrix (27 pages × 2 viewports = 54 screenshots)
- Run axe-core on every page
- User manual testing pass

---

## 10. Dependencies

| Depends on | Status | Notes |
|---|---|---|
| `web/src/styles/tokens.css` | ✅ exists | Single source of truth for tokens |
| `web/src/styles/DESIGN-SYSTEM.md` | ✅ exists | Philosophy + anti-slop rules |
| `web/src/components/icons.tsx` | ✅ exists | 30+ SVG icons |
| `web/src/components/shared/StatusPill.tsx` | ✅ exists | Used in 16/27 pages |
| `web/src/components/shared/KpiCard.tsx` | ✅ exists | Used in 16/27 pages |
| `<Button>`, `<Modal>`, `<DataTable>` shared | ❓ TBD | Audit will reveal |
| `framer-motion` | ✅ installed | AnimatePresence patterns in use |
| Playwright | ❓ Check | May need install for verification |

---

## 11. Risks

| ID | Risk | Mitigation |
|---|---|---|
| R-1 | Refactor breaks existing functionality | 4 back-to-back verifier runs after every page; visual diff ≤1% |
| R-2 | New shared components introduce bugs | Style-guide showcase + Playwright matrix before page migration |
| R-3 | User disagrees with design choices (e.g. button height) | Per-page sign-off before next page |
| R-4 | Audit finds 100+ drifts → scope creep | Severity gate: fix only critical+medium; defer low to backlog |
| R-5 | Bundle size grows | Tree-shake unused variants; lazy-load modals |
| R-6 | Mobile regression on a page | Playwright at 320×568 is mandatory gate |
| R-7 | Color drift invisible to grep (CSS variables in JS) | Visual diff catches these; Lighthouse flags contrast |

---

## 12. Open questions

1. **Per-page vs per-component order**: Should I refactor all `<StatusPill>` drift
   across all pages first, then `<KpiCard>`, etc.? Or one page at a time? — Default: per-component for shared, per-page for component composition.
2. **Button variants**: Need confirmation on `primary / secondary / ghost / danger`
   — enough? Or also `link / success / warning`? — Default: 4 variants, defer others.
3. **Modal API**: framer-motion vs CSS-only? — Default: framer-motion (matches shell).
4. **DataTable scope**: pagination + sort + filter — all in one, or modular? — Default: all in one to start, refactor later if hot.
5. **Style-guide page**: ship in same PR or separate? — Default: same PR, behind `/style-guide` route, unlisted from nav.
6. **Old sidebar deletion**: immediate or staged? — Default: immediate (it's dead code, only referenced in page comments).

---

## 13. Rollback plan

Every refactor is per-file with `git revert`-safe boundaries:
- Each shared component addition: 1 commit, revertable
- Each page refactor: 1 commit, revertable
- Old `AppSidebar.tsx` deletion: 1 commit (restoring is `git revert`)

If user signs off "Tier 20 is wrong direction":
- `git revert` all commits in this speckit change
- Working tree returns to pre-Tier-20 state
- Estimated worst-case loss: 1 day of work

---

## 14. Effort estimate

| Phase | Effort |
|---|---|
| 9.1 Audit (subagent) | 2-3 hours |
| 9.2 Shared components (≤4) | 2-4 hours |
| 9.3 Page refactor (27 pages × ~30 min) | 12-15 hours |
| 9.4 Dead-code cleanup | 30 min |
| 9.5 Verification | 2-3 hours |
| **Total** | **~18-22 hours** (3-4 working sessions) |

---

## 15. Success definition

After Tier 20 ships:
- [ ] User opens every page in `https://stackwatch.smarthomelab.fun/app.html`
- [ ] Every page looks identical-quality (Datadog-tier)
- [ ] Every interactive element behaves identically across pages
- [ ] Every KPI card has the same shape, padding, stripe
- [ ] Every status pill has the same dot+label rendering
- [ ] Every button has the same height + hover state
- [ ] Mobile (320px) is usable everywhere
- [ ] Accessibility scan passes on every page
- [ ] User says: "this looks like one app, not 13 apps stitched together"

---

**Next**: Awaiting user review. If approved → write `spec.md` (FR + NFR +
scenarios expanded) → `plan.md` (file-by-file changes) → `tasks.md`
(numbered steps with verification gates) → `checklist.md` (per-feature
acceptance).

**Last updated:** 2026-08-31