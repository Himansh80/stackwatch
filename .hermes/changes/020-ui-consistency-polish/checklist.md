# Tier 20 — UI Consistency Polish

**Speckit change ID:** `020-ui-consistency-polish`
**Status:** CHECKLIST (5 of 5 artifacts)
**Depends on:** `tasks.md` ✅ approved 2026-08-31

---

## 1. Document purpose

Per-feature acceptance checklist. Run through this AFTER tasks are complete.
Mark each item ✅ (pass) or ❌ (fail). For every ❌, file a remediation task.

The "Verification method" column tells you HOW to check. For most, it's a
grep command or a manual browser step.

---

## 2. Component contracts (AC-1 through AC-6)

### AC-1: Button component exists

- [ ] File `web/src/components/shared/Button.tsx` exists
- [ ] Exports default function `Button`
- [ ] Props interface includes: variant, size, icon, iconRight, loading, disabled, fullWidth, onClick, type, children, aria-label
- [ ] 4 variants: primary, secondary, ghost, danger
- [ ] 3 sizes: sm, md, lg
- [ ] Loading state renders Spinner + disabled
- [ ] framer-motion `whileTap` for active scale
- [ ] All colors use `var(--*)` tokens
- **Verification:** Read file + open StyleGuide → Buttons section

### AC-2: Modal component exists

- [ ] File `web/src/components/shared/Modal.tsx` exists
- [ ] Exports default function `Modal`
- [ ] Props: open, onClose, title, description, size, children, footer, closeOnEscape, closeOnBackdrop, initialFocusRef
- [ ] Focus trap implemented (Tab/Shift+Tab cycle within)
- [ ] Focus restores to opener on close
- [ ] Body scroll lock while open
- [ ] Escape key closes (when closeOnEscape)
- [ ] Backdrop click closes (when closeOnBackdrop)
- [ ] `role="dialog"`, `aria-modal="true"`, `aria-labelledby`
- **Verification:** Read file + open StyleGuide → Modals section + manual focus test

### AC-3: DataTable component exists

- [ ] File `web/src/components/shared/DataTable.tsx` exists
- [ ] Exports generic `<T>` component
- [ ] Props: columns, rows, loading, emptyState, pageSize, currentPage, onPageChange, sortBy, sortDir, onSort, onRowClick, rowKey, stickyHeader
- [ ] Renders real `<table>` semantics
- [ ] Sortable headers are `<button>` inside `<th>`
- [ ] Empty state renders EmptyState (passed as prop OR default)
- [ ] Loading renders SkeletonRow
- [ ] Sticky header option works
- [ ] Pagination UI shows "Showing X–Y of Z"
- **Verification:** Read file + open StyleGuide → DataTable section + manual sort/pagination test

### AC-4: Input + Select + Textarea components exist

- [ ] File `web/src/components/shared/Input.tsx` exists
- [ ] File `web/src/components/shared/Select.tsx` exists
- [ ] File `web/src/components/shared/Textarea.tsx` exists
- [ ] All export default functions
- [ ] Shared props: label, description, error, required, disabled, size, fullWidth, aria-label, aria-describedby
- [ ] Error state: red border + red error text + `aria-invalid="true"`
- [ ] Required indicator (asterisk) in label
- [ ] Disabled state: `aria-disabled="true"` + opacity
- [ ] Focus ring on `:focus-visible`
- **Verification:** Read files + open StyleGuide → Inputs section

### AC-5: Existing shared components verified

- [ ] `StatusPill.tsx` matches spec §3.5 contract
- [ ] `KpiCard.tsx` matches spec §3.6 contract
- [ ] `EmptyState.tsx` has props: icon, title, description, action
- [ ] `SkeletonRow.tsx` exists + matches spec §3.8
- [ ] `SkeletonCard.tsx` exists (NEW) with kpi/panel/list variants
- **Verification:** Read each file + compare to spec

### AC-6: StyleGuide page exists with showcases

- [ ] File `web/src/pages/StyleGuide.tsx` exists
- [ ] Accessible at `/style-guide` (auth required)
- [ ] NOT in sidebar nav
- [ ] Shows: color tokens, typography, spacing, buttons, modals, DataTable, inputs, StatusPill, KpiCard, EmptyState, SkeletonRow + SkeletonCard
- **Verification:** Navigate to /style-guide in browser + scroll through

---

## 3. Page-level consistency (AC-7 through AC-12)

### AC-7: All pages use StatusPill

- [ ] `grep -c "<StatusPill" web/src/pages/Dashboard.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/ProfilePage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/BillingPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/SettingsPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/HomelabPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/ApmPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/LogsFullPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/RumFullPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/SyntheticsPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/SecurityPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/CspmPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/CicdPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/DatabasePage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/IncidentsPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/NotebookPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/IntelligencePage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/EnterprisePage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/PlatformPage.tsx` ≥ 1
- [ ] `grep -c "<StatusPill" web/src/pages/TrueNASWorkspace.tsx` ≥ 1

### AC-8: All pages with metrics use KpiCard

- [ ] `grep -c "<KpiCard" web/src/pages/Dashboard.tsx` ≥ 4
- [ ] `grep -c "<KpiCard" web/src/pages/ProfilePage.tsx` ≥ 1
- [ ] `grep -c "<KpiCard" web/src/pages/BillingPage.tsx` ≥ 1
- [ ] `grep -c "<KpiCard" web/src/pages/HomelabPage.tsx` ≥ 1
- [ ] `grep -c "<KpiCard" web/src/pages/ApmPage.tsx` ≥ 1
- [ ] `grep -c "<KpiCard" web/src/pages/LogsFullPage.tsx` ≥ 1
- [ ] `grep -c "<KpiCard" web/src/pages/PlatformPage.tsx` ≥ 1
- [ ] `grep -c "<KpiCard" web/src/pages/IntelligencePage.tsx` ≥ 1
- [ ] `grep -c "<KpiCard" web/src/pages/EnterprisePage.tsx` ≥ 1

### AC-9: All pages wrapped in main landmark

- [ ] `web/src/components/AppShell.tsx` renders `<main id="main">` (covers all 27 pages via wrapper)

### AC-10: All pages have document title

- [ ] Document title set per route (via useEffect in pages or Helmet wrapper)

### AC-11: All pages have skip-link

- [ ] `web/src/components/AppShell.tsx` renders `<a href="#main" className="skip-link">` (covers all 27 pages)

### AC-12: All pages have page transition

- [ ] `web/src/components/AppShell.tsx` uses AnimatePresence (covers all 27 pages)

---

## 4. Visual consistency (AC-13 through AC-16)

### AC-13: KPI cards identical across pages

- [ ] All KPI cards have height 120px
- [ ] All KPI cards have padding `var(--space-6)`
- [ ] All KPI cards have 3px left stripe in accent color
- [ ] All KPI cards use `font-variant-numeric: tabular-nums` on values
- [ ] Visual diff across pages ≤1% (Playwright)
- **Verification:** Open Dashboard, Profile, Billing, Homelab side by side; compare KPI strips

### AC-14: Status pills identical across pages

- [ ] All status pills have dot size 10px (md)
- [ ] All status pills have label size 12px (md), uppercase, letter-spaced
- [ ] Status colors: up/ok=green, stale/warn=amber, down/crit=red, unknown=gray
- **Verification:** Open Dashboard, Apm, Logs side by side; compare status pills

### AC-15: Buttons identical across pages

- [ ] All md buttons 40px tall
- [ ] All md buttons padding `var(--space-2) var(--space-4)`
- [ ] All primary buttons bg `var(--color-primary)`
- [ ] All primary buttons hover bg `var(--color-primary-hover)`
- **Verification:** Open Login, Settings, Billing; compare "Save" / "Submit" / "Confirm" buttons

### AC-16: Modals identical across pages

- [ ] All modals backdrop `rgba(0, 0, 0, 0.6)` + `backdrop-filter: blur(4px)`
- [ ] All modals panel `var(--color-surface-elevated)`, `var(--radius-2xl)`
- [ ] All modals padding `var(--space-8)` (32px)
- [ ] All modals footer right-aligned with divider above
- **Verification:** Open modal on multiple pages; compare visually

---

## 5. Code hygiene (AC-17 through AC-22)

### AC-17: Zero hardcoded hex in components/pages

- [ ] `grep -rE "#[0-9a-fA-F]{3,6}" web/src/{components,pages}/` returns 0 matches
- **Verification:** Run grep; expect zero output

### AC-18: Zero emoji-as-icon in components/pages

- [ ] `grep -rE "[⌂$◉⚙◈▤◴≡◎◔◍◐⇄◧⚑◰⌬✦◊]" web/src/{components,pages}/` returns 0 matches
- **Verification:** Run grep; expect zero output

### AC-19: Zero inline background styles on shared components

- [ ] `grep -rE "style=\{\{[^}]*background" web/src/{components,pages}/` returns 0 matches
- **Verification:** Run grep; expect zero output

### AC-20: Zero files >400 LOC

- [ ] `find web/src/{components,pages} -name "*.tsx" -exec wc -l {} \; | awk '$1 > 400'` returns empty
- **Verification:** Run find; expect no output

### AC-21: AppSidebar.tsx deleted

- [ ] `test -f web/src/components/AppSidebar.tsx` returns non-zero (file does not exist)
- [ ] `git grep AppSidebar web/src/` returns only `web/src/pages/*.tsx` comments (not imports)
- **Verification:** Run commands; verify

### AC-22: Brand mark is SVG (not letter "S")

- [ ] `grep "sb-brand-mark" web/src/components/sidebar/Sidebar.tsx` returns 0 matches (old span removed)
- [ ] `grep "<BrandLogo" web/src/components/sidebar/Sidebar.tsx` returns ≥1 match (new component)
- [ ] Visual: sidebar shows SVG logo, not letter
- **Verification:** Open dashboard; inspect sidebar

---

## 6. Build + runtime (AC-23 through AC-29)

### AC-23: npm run build completes zero warnings

- [ ] Run `cd web && npm run build`; expect no warnings
- **Verification:** Run build; check stderr

### AC-24: npm run type-check zero errors

- [ ] Run `cd web && npm run type-check`; expect zero errors
- **Verification:** Run type-check; check exit code

### AC-25: Lighthouse accessibility ≥90 on 5 pages

- [ ] /dashboard ≥ 90
- [ ] /proxmox-vms ≥ 90
- [ ] /apm ≥ 90
- [ ] /logs ≥ 90
- [ ] /settings ≥ 90
- **Verification:** Run `lighthouse <url>` per page

### AC-26: Bundle size delta <5%

- [ ] `du -sh web/dist/` after Tier 20 vs before Tier 20
- [ ] Delta <5%
- **Verification:** Compare sizes

### AC-27: Zero console errors/warnings on page load

- [ ] Open /dashboard in browser DevTools console; 0 errors, 0 warnings
- [ ] Repeat for /proxmox-vms, /apm, /logs, /settings
- **Verification:** Manual browser check

### AC-28: 4 back-to-back full-stack verifier runs all PASS

- [ ] Run 1: PASS
- [ ] Run 2: PASS
- [ ] Run 3: PASS
- [ ] Run 4: PASS
- **Verification:** Execute verifier 4 times; all must pass

### AC-29: User signs off

- [ ] User manually tested every page
- [ ] User confirms every page looks identical-quality
- [ ] User confirms every interactive element behaves identically
- [ ] User signs off in writing (comment on commit OR ack in chat)
- **Verification:** User ack

---

## 7. Page-by-page checklist

For each page below, verify the 5 acceptance criteria. If any fails, file remediation task.

### Dashboard.tsx

- [ ] Uses <StatusPill> for server status indicators
- [ ] Uses <KpiCard> for KPI strip (≥4 cards)
- [ ] Uses <Button> for all buttons (no inline buttons)
- [ ] Has loading state with <SkeletonCard>
- [ ] Has empty state with <EmptyState> (if applicable)

### ProfilePage.tsx

- [ ] Uses <StatusPill> for account status
- [ ] Uses <KpiCard> for profile stats
- [ ] Uses <Button> for actions
- [ ] Uses <Input> / <Select> for forms
- [ ] Mobile responsive at 320px

### BillingPage.tsx

- [ ] Uses <StatusPill> for plan status
- [ ] Uses <KpiCard> for plan metrics
- [ ] Uses <Button> for plan actions
- [ ] No hardcoded colors
- [ ] Tablet responsive at 768px

### SettingsPage.tsx

- [ ] Uses <Input> / <Select> / <Textarea> for all forms
- [ ] Uses <Button> for save actions
- [ ] Has loading state with <SkeletonRow>
- [ ] Has error states on inputs
- [ ] Lighthouse accessibility ≥90

### HomelabPage.tsx

- [ ] Uses <StatusPill> for service health
- [ ] Uses <KpiCard> for homelab stats
- [ ] Uses <DataTable> for service lists
- [ ] Uses <Button> for actions
- [ ] Mobile responsive at 320px

### Landing.tsx

- [ ] Uses <Button> for CTAs (no inline buttons)
- [ ] Uses <Input> / <Select> in forms (if any)
- [ ] No hardcoded colors
- [ ] Public page (no auth required)
- [ ] Lighthouse performance ≥90

### Login.tsx + Signup.tsx + ForgotPassword.tsx + ResetPassword.tsx

- [ ] Uses <Input> with email validation
- [ ] Uses <Input type="password"> for passwords
- [ ] Uses <Button> for submit
- [ ] Has error states on inputs
- [ ] Mobile responsive at 320px

### ApmPage.tsx + ApmServicePage.tsx

- [ ] Uses <StatusPill> for service status
- [ ] Uses <KpiCard> for APM metrics
- [ ] Uses <DataTable> for trace lists
- [ ] Has loading state with <SkeletonCard>
- [ ] Has empty state for no traces

### LogsFullPage.tsx

- [ ] Uses <StatusPill> for log levels (if applicable)
- [ ] Uses <Input> for search
- [ ] Uses <DataTable> for log entries
- [ ] Has loading state with <SkeletonRow>
- [ ] Has empty state for no logs

### RumFullPage.tsx + RumSessionPage.tsx

- [ ] Uses <StatusPill> for session states
- [ ] Uses <KpiCard> for RUM metrics
- [ ] Uses <DataTable> for session lists
- [ ] Has loading state
- [ ] Has empty state

### SyntheticsPage.tsx

- [ ] Uses <StatusPill> for test results (pass/fail)
- [ ] Uses <KpiCard> for synthetic metrics
- [ ] Uses <DataTable> for test list
- [ ] Uses <Button> for actions (run now, edit, delete)
- [ ] Uses <Modal> for edit dialogs

### SecurityPage.tsx

- [ ] Uses <StatusPill> for severity
- [ ] Uses <KpiCard> for security metrics
- [ ] Uses <DataTable> for threat list
- [ ] Has loading state
- [ ] Has empty state

### CspmPage.tsx

- [ ] Uses <StatusPill> for compliance status
- [ ] Uses <DataTable> for findings list
- [ ] Has loading state
- [ ] Has empty state
- [ ] Mobile responsive

### CicdPage.tsx

- [ ] Uses <StatusPill> for pipeline status
- [ ] Uses <DataTable> for pipeline list
- [ ] Has loading state
- [ ] Has empty state
- [ ] Mobile responsive

### DatabasePage.tsx

- [ ] Uses <StatusPill> for query state
- [ ] Uses <DataTable> for slow queries
- [ ] Has loading state
- [ ] Has empty state
- [ ] Mobile responsive

### IncidentsPage.tsx

- [ ] Uses <StatusPill> for incident state (open/ack/resolved)
- [ ] Uses <DataTable> for incidents list
- [ ] Uses <Button> for ack/resolve actions
- [ ] Uses <Modal> for incident detail
- [ ] Has empty state

### NotebookPage.tsx

- [ ] Uses <Button> for actions
- [ ] Uses <Input> / <Textarea> for notebook content
- [ ] Has loading state
- [ ] Has empty state
- [ ] Mobile responsive

### IntelligencePage.tsx

- [ ] Uses <StatusPill> for anomaly state
- [ ] Uses <KpiCard> for intelligence metrics
- [ ] Uses <DataTable> for anomaly list
- [ ] Has loading state
- [ ] Has empty state

### EnterprisePage.tsx

- [ ] Uses <StatusPill> for SSO/SCIM status
- [ ] Uses <KpiCard> for enterprise metrics
- [ ] Uses <Input> / <Select> for forms
- [ ] Uses <Button> for actions
- [ ] Has loading state

### PlatformPage.tsx

- [ ] Uses <StatusPill> for platform status
- [ ] Uses <KpiCard> for platform metrics
- [ ] Uses <DataTable> for usage/billing
- [ ] Uses <Button> for actions
- [ ] Has loading state

### TrueNASWorkspace.tsx

- [ ] Uses <StatusPill> for disk/pool status
- [ ] Uses <KpiCard> for storage metrics
- [ ] Uses <DataTable> for dataset/disk lists
- [ ] Uses <Button> for actions
- [ ] Has loading state

### SharedDashboardsPage.tsx

- [ ] Uses <Button> for actions
- [ ] Uses <DataTable> for dashboard list
- [ ] Has loading state
- [ ] Has empty state
- [ ] Mobile responsive

---

## 8. Final sign-off

After all above ✅, Tier 20 is complete.

- [ ] All 29 ACs pass
- [ ] All 27 page checklists complete
- [ ] User (himan) has manually tested every page
- [ ] All 4 back-to-back verifier runs pass
- [ ] Move folder to `.hermes/changes/archive/2026-XX-XX-020-ui-consistency-polish/`

---

**Last updated:** 2026-08-31