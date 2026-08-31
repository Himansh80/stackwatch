# Tier 20 — UI Consistency Polish

**Speckit change ID:** `020-ui-consistency-polish`
**Status:** TASKS (4 of 5 artifacts)
**Depends on:** `plan.md` ✅ approved 2026-08-31

---

## 1. Document purpose

Numbered steps for the engineer (me) to execute. Each task has:
- **Estimated effort**
- **Files touched**
- **Verification gate** — what must be true before moving to next task
- **Cross-references** — links to spec section + plan phase + acceptance criterion

After tasks.md is approved → `checklist.md` provides per-feature acceptance.

---

## 2. Task list (numbered, gated)

### Phase A — Audit (~2-3 hours)

#### Task A.1 — Spawn audit subagent (~30 min)

- **Files touched:** none (subagent creates `docs/UI-AUDIT-REPORT.md`)
- **Goal:** Catalog every drift across 27 pages + 78 components
- **Subagent prompt:**
  ```
  Crawl every .tsx file under web/src/pages/ (27 files) and web/src/components/
  (78 files, EXCLUDE web/src/components/proxmox/). For each file, catalog every
  usage of: button, KPI card, status pill, modal, table, form input, empty
  state, loading state. Output a Markdown table with: file path, line range,
  element type, current implementation, drift from web/src/styles/DESIGN-SYSTEM.md,
  severity (critical/medium/low), proposed fix. Save to docs/UI-AUDIT-REPORT.md.
  Target ~2,000 lines.
  ```
- **Verification gate:**
  - [ ] `docs/UI-AUDIT-REPORT.md` exists, ≥1,500 lines
  - [ ] Every page mentioned with element count
  - [ ] Every drift has severity + proposed fix
- **References:** Spec §4 FR-1 to FR-18, Plan §3

#### Task A.2 — Review audit + classify drifts (~30 min)

- **Files touched:** `docs/UI-AUDIT-REPORT.md` (add summary section)
- **Goal:** Classify every drift as MUST-FIX / SHOULD-FIX / DEFER
- **Activity:** Read audit, append "## Classification" section with three lists
- **Verification gate:**
  - [ ] MUST-FIX count ≤50 (else trigger scope reduction in plan §12)
  - [ ] All MUST-FIX items assigned to a page in Phase D order
- **References:** Plan §12 risk-driven deviations

#### Task A.3 — Commit audit (~5 min)

- **Files touched:** `docs/UI-AUDIT-REPORT.md`
- **Commit:** `docs(tier20): UI audit report + drift classification`
- **Verification gate:** Git working tree clean

---

### Phase B — Shared components (~10 hours)

#### Task B.1 — Create Button component (~1.5 hrs)

- **Files touched:**
  - `web/src/components/shared/Button.tsx` (NEW, ~150 LOC)
  - `web/src/components/shared/Button.test.tsx` (NEW, ~80 LOC)
  - `web/src/styles/components.css` (add `.btn-*` rules)
- **References:** Spec §3.1, Plan §4.1
- **Implementation:**
  - 4 variants (primary/secondary/ghost/danger)
  - 3 sizes (sm/md/lg)
  - loading state with `<Spinner>`
  - framer-motion `motion.button` for active scale
  - All colors via `var(--*)` tokens
- **Tests:**
  - Renders 4 variants × 3 sizes correctly
  - Loading state shows spinner + disabled
  - Click handler fires
  - aria-disabled + aria-busy on loading
- **Verification gate:**
  - [ ] `Button.tsx` exported as default
  - [ ] `npm test -- Button` passes (8+ test cases)
  - [ ] `npm run type-check` passes
  - [ ] Manual: button has correct hover state in browser

#### Task B.2 — Create Modal component (~2 hrs)

- **Files touched:**
  - `web/src/components/shared/Modal.tsx` (NEW, ~180 LOC)
  - `web/src/components/shared/Modal.test.tsx` (NEW, ~100 LOC)
  - `web/src/styles/components.css` (add `.modal-*` rules)
- **References:** Spec §3.2, Plan §4.2
- **Implementation:**
  - AnimatePresence + motion.div for backdrop + panel
  - Focus trap (intercept Tab/Shift+Tab)
  - Restore focus on close
  - Body scroll lock via `document.body.style.overflow`
  - Escape key + backdrop click close (configurable)
  - `role="dialog"` + `aria-modal="true"` + `aria-labelledby`
- **Tests:** 6+ test cases (open/close/focus trap/escape/backdrop/restore focus)
- **Verification gate:**
  - [ ] `Modal.tsx` exported as default
  - [ ] `npm test -- Modal` passes
  - [ ] Manual: focus trap works in browser

#### Task B.3 — Create DataTable component (~3 hrs)

- **Files touched:**
  - `web/src/components/shared/DataTable.tsx` (NEW, ~250 LOC)
  - `web/src/components/shared/DataTable.test.tsx` (NEW, ~150 LOC)
  - `web/src/styles/components.css` (add `.table-*` rules)
- **References:** Spec §3.3, Plan §4.3
- **Implementation:**
  - Generic `<T>` component
  - Sort + pagination (controlled or uncontrolled)
  - Sticky header option
  - Empty state via `<EmptyState>` prop
  - Loading state via `<SkeletonRow>`
  - ARIA: real `<table>` semantics, sortable headers as `<button>`
- **Tests:** 6+ test cases (render, sort, pagination, empty, loading, row click)
- **Verification gate:**
  - [ ] `DataTable.tsx` exported as default
  - [ ] `npm test -- DataTable` passes
  - [ ] Manual: sticky header works at scroll

#### Task B.4 — Create Input + Select + Textarea (~2 hrs, batched)

- **Files touched:**
  - `web/src/components/shared/Input.tsx` (NEW, ~120 LOC)
  - `web/src/components/shared/Select.tsx` (NEW, ~100 LOC)
  - `web/src/components/shared/Textarea.tsx` (NEW, ~80 LOC)
  - 3 × `.test.tsx` (~60 LOC each)
  - `web/src/styles/components.css` (add `.input-*` rules)
- **References:** Spec §3.4, Plan §4.4
- **Implementation:**
  - Shared `BaseInputWrapper` sub-component
  - Error state: red border + red error text + aria-invalid
  - Required indicator
  - React.forwardRef for Input
- **Tests:** Each input gets 5+ test cases (render, error, disabled, required, focus)
- **Verification gate:**
  - [ ] All 3 components exported as default
  - [ ] `npm test -- Input Select Textarea` passes
  - [ ] Manual: error state shows red border in browser

#### Task B.5 — Create SkeletonCard (~30 min)

- **Files touched:**
  - `web/src/components/shared/SkeletonCard.tsx` (NEW, ~50 LOC)
  - `web/src/styles/components.css` (add `.skeleton-card` rules)
- **References:** Spec §3.8, Plan §4.5
- **Implementation:** 3 variants (kpi/panel/list), pulse animation via CSS @keyframes
- **Verification gate:**
  - [ ] `SkeletonCard.tsx` exported as default
  - [ ] Manual: pulse animation visible

#### Task B.6 — Create BrandLogo (~30 min)

- **Files touched:**
  - `web/src/components/shared/BrandLogo.tsx` (NEW, ~40 LOC)
- **References:** Spec §16, Plan §4.6
- **Implementation:** Inline SVG with `currentColor`, 2 variants (full/mark)
- **Verification gate:**
  - [ ] `BrandLogo.tsx` exported as default
  - [ ] Manual: SVG renders in browser

#### Task B.7 — Verify existing shared components (~30 min)

- **Files touched:** none (read-only)
- **Activity:** Read `StatusPill.tsx`, `KpiCard.tsx`, `EmptyState.tsx`, `SkeletonRow.tsx`. Confirm each matches spec contract (§3.5, §3.6, §3.7, §3.8). If drift, add to Phase E.
- **Verification gate:**
  - [ ] Each component's contract matches spec
  - [ ] Any drift documented for Phase E

#### Task B.8 — Commit Phase B (~10 min)

- **Commit:** `feat(tier20): shared components (Button, Modal, DataTable, Input, Select, Textarea, SkeletonCard, BrandLogo)`
- **Verification gate:** `npm run type-check` passes; `npm run build` passes; all tests pass

---

### Phase C — StyleGuide (~2 hours)

#### Task C.1 — Create StyleGuide page (~1.5 hrs)

- **Files touched:**
  - `web/src/pages/StyleGuide.tsx` (NEW, ~300 LOC)
- **References:** Plan §5
- **Implementation:**
  - 11 sections: tokens, typography, spacing, buttons, modals, tables, inputs, status pills, KPI cards, empty states, loading states
  - Each section shows visual examples of every variant/state
- **Verification gate:**
  - [ ] All components render in StyleGuide
  - [ ] No console errors on /style-guide

#### Task C.2 — Add /style-guide route (~5 min)

- **Files touched:** `web/src/App.tsx`
- **Commit:** `feat(tier20): StyleGuide page + route`
- **Verification gate:**
  - [ ] `/style-guide` navigates correctly (auth required)
  - [ ] NOT in sidebar nav (unlisted)

---

### Phase D — Page refactor (~11 hours)

#### Task D.0 — Group 1: Foundation pages (~1.7 hrs)

- **Pages:** Landing.tsx, Login.tsx, Signup.tsx, ForgotPassword.tsx, ResetPassword.tsx
- **Per-page pattern:**
  1. Read file
  2. Identify drifts vs spec
  3. Apply refactor (replace inline buttons, form inputs, status pills with shared)
  4. Verify build + visual
  5. Commit per-page
- **Per-page commit:** `refactor(tier20): <PageName> uses shared components`
- **Files:** 5 page files
- **Verification gate:**
  - [ ] All 5 pages use `<Button>` for primary CTA
  - [ ] All 5 pages use `<Input>` / `<Select>` for forms
  - [ ] All 5 pages pass `npm run type-check`

#### Task D.1 — Group 2: Workspace pages (~2.5 hrs)

- **Pages:** Dashboard.tsx, ProfilePage.tsx, BillingPage.tsx, SettingsPage.tsx, HomelabPage.tsx
- **Per-page:** Same pattern as D.0
- **Verification gate:**
  - [ ] All 5 pages use `<KpiCard>` for KPI strip
  - [ ] All 5 pages use `<StatusPill>` for status indicators
  - [ ] All 5 pages pass type-check + build

#### Task D.2 — Group 3: Observability pages (~3.3 hrs)

- **Pages:** ApmPage.tsx, ApmServicePage.tsx, LogsFullPage.tsx, RumFullPage.tsx, RumSessionPage.tsx, SyntheticsPage.tsx, SecurityPage.tsx, CspmPage.tsx
- **Per-page:** Same pattern
- **Verification gate:**
  - [ ] All 8 pages use `<DataTable>` for data display
  - [ ] All 8 pages use `<StatusPill>` for severity indicators

#### Task D.3 — Group 4: Infrastructure pages (~1.3 hrs)

- **Pages:** CicdPage.tsx, DatabasePage.tsx, TrueNASWorkspace.tsx
- **Verification gate:** All 3 pages consistent

#### Task D.4 — Group 5: Operations pages (~1.7 hrs)

- **Pages:** IncidentsPage.tsx, NotebookPage.tsx, IntelligencePage.tsx, SharedDashboardsPage.tsx
- **Verification gate:** All 4 pages consistent

#### Task D.5 — Group 6: Enterprise + Platform pages (~0.7 hrs)

- **Pages:** EnterprisePage.tsx, PlatformPage.tsx
- **Verification gate:** All 2 pages consistent

#### Task D.6 — Commit Phase D (~10 min)

- **Verification gate:**
  - [ ] All 27 pages refactored
  - [ ] `npm run type-check` passes
  - [ ] `npm run build` passes
  - [ ] No regression in click-through paths

---

### Phase E — Dead code + branding (~1.5 hours)

#### Task E.1 — Delete AppSidebar.tsx (~5 min)

- **Files touched:** `git rm web/src/components/AppSidebar.tsx`
- **Pre-check:** `grep -rn "AppSidebar" web/src/` confirms only page comments mention it
- **Commit:** `chore(tier20): delete dead sidebar (replaced by sidebar/Sidebar)`
- **Verification gate:**
  - [ ] File deleted
  - [ ] `git grep AppSidebar` returns only SPECIFIC references (not imports)
  - [ ] App still works (no import error)

#### Task E.2 — Replace brand mark (~10 min)

- **Files touched:**
  - `web/src/components/sidebar/Sidebar.tsx` (import BrandLogo, replace span)
- **Commit:** `refactor(tier20): sidebar uses BrandLogo (SVG instead of letter S)`
- **Verification gate:**
  - [ ] Sidebar shows SVG logo
  - [ ] No visual regression

#### Task E.3 — Topbar composition check (~30 min)

- **Files touched:** `web/src/components/topbar/TopBar.tsx` + 8 sub-components
- **Activity:** Read each sub-component; identify any inline button/input; replace with shared
- **Verification gate:**
  - [ ] All topbar sub-components use shared Button/Input
  - [ ] No hardcoded colors in topbar
  - [ ] No emoji icons

#### Task E.4 — Update DESIGN-SYSTEM.md (~30 min)

- **Files touched:** `web/src/styles/DESIGN-SYSTEM.md`
- **Activity:** Add sections documenting Button, Modal, DataTable, Input/Select/Textarea
- **Commit:** `docs(tier20): DESIGN-SYSTEM.md updated with new components`
- **Verification gate:**
  - [ ] Each new component has documentation
  - [ ] Anti-anti-slop checklist updated

---

### Phase F — Verification (~5 hours)

#### Task F.1 — Type check + build (~15 min)

- **Activity:** Run `npm run type-check && npm run build` in web/
- **Verification gate:** Zero errors, zero warnings

#### Task F.2 — Grep checks (per FR/AC) (~30 min)

- **Activity:** Run all grep commands from plan §8.1
- **Verification gate:**
  - [ ] AC-17: zero hardcoded hex
  - [ ] AC-18: zero emoji-as-icon
  - [ ] AC-19: zero inline background styles
  - [ ] AC-20: zero files >400 LOC
  - [ ] AC-21: AppSidebar.tsx deleted
  - [ ] AC-22: Brand mark is SVG (not letter S)

#### Task F.3 — Playwright screenshots (~1.5 hrs)

- **Files touched:**
  - `verify/ui-tier20-screenshots.py` (NEW)
- **Activity:** Capture every page at 1440×900 + 320×568; save to verify/ui-screenshots/
- **Verification gate:**
  - [ ] All 27 pages captured at desktop + mobile
  - [ ] StyleGuide captured
  - [ ] Visual diff ≤1% per page

#### Task F.4 — axe-core accessibility scan (~1 hr)

- **Files touched:**
  - `verify/ui-tier20-axe.py` (NEW)
- **Activity:** Scan every page with axe-core; collect violations
- **Verification gate:**
  - [ ] 0 critical violations per page
  - [ ] 0 serious violations on home pages

#### Task F.5 — Lighthouse (~30 min)

- **Files touched:**
  - `verify/ui-tier20-lighthouse.py` (NEW)
- **Activity:** Run Lighthouse on /dashboard, /proxmox-vms, /apm, /logs, /settings
- **Verification gate:**
  - [ ] Accessibility ≥90 per page
  - [ ] LCP <1.5s
  - [ ] TTI <3s
  - [ ] Bundle size delta <5%

#### Task F.6 — Full-stack regression (4 back-to-back) (~1 hr)

- **Files touched:** none
- **Activity:** Run existing full-stack verifier 4 times
- **Verification gate:**
  - [ ] All 4 runs PASS
  - [ ] No endpoint regression

#### Task F.7 — User manual test (~30 min)

- **Activity:** User opens every page in browser; signs checklist
- **Verification gate:** User signs off

---

## 3. Verification gates summary

| Gate | When | Pass criteria |
|---|---|---|
| G1 | After Task A.2 | Audit report reviewed + classified |
| G2 | After Task B.8 | All shared components built + tested |
| G3 | After Task C.2 | StyleGuide accessible at /style-guide |
| G4 | After Task D.6 | All 27 pages refactored + builds clean |
| G5 | After Task E.4 | Dead code deleted + brand fixed + docs updated |
| G6 | After Task F.2 | All grep checks pass |
| G7 | After Task F.3 | Playwright screenshots captured |
| G8 | After Task F.4 | axe-core clean |
| G9 | After Task F.5 | Lighthouse thresholds met |
| G10 | After Task F.6 | 4 back-to-back full-stack passes |
| G11 | After Task F.7 | User signs off |

---

## 4. Total effort

| Phase | Effort |
|---|---|
| A — Audit | 2-3 hours |
| B — Shared components | ~10 hours |
| C — StyleGuide | 2 hours |
| D — Page refactor | ~11 hours |
| E — Cleanup + branding | ~1.5 hours |
| F — Verification | ~5 hours |
| **Total** | **~31-33 hours** |

---

**Next:** Awaiting user review. After approval → write `checklist.md`
(per-feature acceptance ✅/❌).

**Last updated:** 2026-08-31