# Tier 20 — UI Consistency Polish

**Speckit change ID:** `020-ui-consistency-polish`
**Status:** SPEC (2 of 5 artifacts)
**Depends on:** `proposal.md` ✅ approved 2026-08-31

---

## 1. Document purpose

This spec expands the proposal's goals + requirements into **concrete,
testable contracts** for every shared component, every page, and every
cross-cutting concern. After this spec is approved, `plan.md` describes
file-by-file changes; `tasks.md` numbers each step; `checklist.md`
provides the per-feature acceptance.

---

## 2. Glossary

| Term | Definition |
|---|---|
| **Shared component** | A React component in `web/src/components/shared/` that is used by ≥2 pages. Adding a new one requires a StyleGuide showcase. |
| **Visual contract** | The fixed look + behavior of a component across all usages (sizes, colors, states, animations). |
| **State** | One of: default, hover, focus-visible, active, disabled, loading, error. Every interactive element must define all applicable states. |
| **Variant** | A distinct visual style of a component (e.g. Button.primary vs Button.ghost). |
| **Token** | A CSS custom property from `web/src/styles/tokens.css` (e.g. `var(--color-primary)`). Components MUST use tokens, never hardcoded values. |
| **Empty state** | A page-level or component-level state where there is no data to show. MUST include guidance ("Add your first server", not "No data"). |
| **Loading state** | A page-level or component-level state where data is being fetched. MUST show a skeleton or spinner, never blank space. |
| **Tier 20 Scope** | The union of all pages (27) + all components (78, except `proxmox/`) + all CSS (14 files). |

---

## 3. Component specifications

Every component below has a contract that ALL existing usages MUST match
and ALL new usages MUST follow. Components marked **[NEW]** must be
created; components marked **[EXTEND]** need additional variants or
props; components marked **[VERIFY]** should already exist but need
contract verification.

### 3.1 Button **[NEW — high priority]**

**Purpose:** Clickable element that triggers an action.

**Location:** `web/src/components/shared/Button.tsx`

**Props interface:**
```typescript
interface ButtonProps {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger';  // default: 'primary'
  size?: 'sm' | 'md' | 'lg';                              // default: 'md'
  icon?: ReactNode;                                        // leading icon
  iconRight?: ReactNode;                                   // trailing icon
  loading?: boolean;                                       // shows spinner, disables click
  disabled?: boolean;
  fullWidth?: boolean;
  onClick?: (e: MouseEvent<HTMLButtonElement>) => void;
  type?: 'button' | 'submit' | 'reset';                   // default: 'button'
  children: ReactNode;
  'aria-label'?: string;
}
```

**Visual contract:**
- Height: sm=32px, md=40px, lg=48px
- Padding: sm=`var(--space-2) var(--space-3)`, md=`var(--space-2) var(--space-4)`, lg=`var(--space-3) var(--space-6)`
- Border radius: `var(--radius-md)` (6px)
- Font: 500 weight, `var(--text-base)` (14px), `var(--leading-tight)`, letter-spacing `var(--tracking-wide)` (0.04em)
- Background: primary=`var(--color-primary)`, secondary=`var(--color-surface-elevated)`, ghost=`transparent`, danger=`var(--color-error)`
- Text color: primary=`var(--color-text-inverse)`, secondary=`var(--color-text)`, ghost=`var(--color-text)`, danger=`var(--color-text)`
- Border: primary/secondary/danger=none, ghost=`1px solid var(--color-border-strong)`
- Hover transition: `background var(--duration-base) var(--ease-out)` (150ms)
- Hover: primary→`var(--color-primary-hover)`, secondary→`var(--color-surface-hover)`, ghost→`var(--color-surface-hover)`, danger→`var(--color-error-hover)`
- Active scale: `transform: scale(0.97)` via framer-motion
- Focus-visible: 2px outline `var(--color-border-focus)` with 2px offset
- Disabled: opacity 0.5, cursor not-allowed, no hover effect
- Loading: replace children with `<Spinner size={size === 'sm' ? 12 : 16} />` + disabled

**Accessibility contract:**
- Renders a real `<button>` element (not a `<div>`)
- `aria-disabled` when loading
- `aria-busy` when loading
- Loading state announced via `aria-live="polite"` if used in async flow

**Usage:**
```tsx
<Button variant="primary" onClick={handleSave}>Save changes</Button>
<Button variant="ghost" icon={<TrashIcon />} onClick={handleDelete}>Delete</Button>
<Button variant="danger" loading={isDeleting}>Delete server</Button>
```

### 3.2 Modal **[NEW — high priority]**

**Purpose:** Overlay dialog for focused interactions (forms, confirmations,
view details).

**Location:** `web/src/components/shared/Modal.tsx`

**Props interface:**
```typescript
interface ModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  description?: string;
  size?: 'sm' | 'md' | 'lg';  // default 'md' (480px wide)
  children: ReactNode;
  footer?: ReactNode;          // typically Cancel + Save buttons
  closeOnEscape?: boolean;     // default true
  closeOnBackdrop?: boolean;   // default true
  initialFocusRef?: RefObject<HTMLElement>; // focus management
}
```

**Visual contract:**
- Backdrop: `rgba(0, 0, 0, 0.6)` with `backdrop-filter: blur(4px)`
- Panel: `var(--color-surface-elevated)`, `var(--radius-2xl)`, shadow L3
- Width: sm=320px, md=480px, lg=720px
- Padding: 32px (var(--space-8))
- Title: 18px, 600 weight, `var(--color-text)`
- Description: 14px, `var(--color-text-muted)`, 12px below title
- Footer: right-aligned, 16px gap, divider line above (`1px solid var(--color-border)`)
- Animation: enter=`opacity 0→1, scale 0.95→1, y 8→0` 200ms ease-out; exit=reverse 150ms
- z-index: 1000 (above everything)

**Accessibility contract:**
- Focus trap: when opened, focus moves to first focusable element (or `initialFocusRef` if provided); Tab cycles within modal; Shift+Tab cycles backward
- Restore focus: when closed, focus returns to element that opened the modal
- Escape key closes (if `closeOnEscape`)
- `role="dialog"`, `aria-modal="true"`, `aria-labelledby` points to title id
- Body scroll locked while open (`document.body.style.overflow = 'hidden'`)
- Background is `inert` (or `aria-hidden="true"` on siblings) for screen readers

**Usage:**
```tsx
<Modal
  open={isOpen}
  onClose={() => setIsOpen(false)}
  title="Delete server"
  description="This cannot be undone."
  footer={
    <>
      <Button variant="ghost" onClick={() => setIsOpen(false)}>Cancel</Button>
      <Button variant="danger" onClick={handleDelete}>Delete</Button>
    </>
  }
>
  Are you sure you want to delete {server.name}? All metrics will be lost.
</Modal>
```

### 3.3 DataTable **[NEW — high priority]**

**Purpose:** Sortable, paginated table for displaying structured data.

**Location:** `web/src/components/shared/DataTable.tsx`

**Props interface:**
```typescript
interface Column<T> {
  key: keyof T | string;
  header: string;
  render?: (row: T) => ReactNode;
  sortable?: boolean;
  width?: string | number;
  align?: 'left' | 'center' | 'right';
}

interface DataTableProps<T> {
  columns: Column<T>[];
  rows: T[];
  loading?: boolean;
  emptyState?: ReactNode;
  pageSize?: number;             // default 25
  currentPage?: number;
  onPageChange?: (page: number) => void;
  sortBy?: string;
  sortDir?: 'asc' | 'desc';
  onSort?: (key: string) => void;
  onRowClick?: (row: T) => void;
  rowKey: (row: T) => string;
  stickyHeader?: boolean;
}
```

**Visual contract:**
- Container: `var(--color-surface)` background, `var(--radius-lg)` border-radius, 1px border
- Header row: `var(--color-surface-elevated)` background, 11px uppercase `var(--tracking-widest)`, `var(--color-text-muted)`, 40px height
- Body rows: 14px, `var(--color-text)`, 48px height, `font-variant-numeric: tabular-nums` on numeric cells
- Row hover: `var(--color-surface-hover)` background
- Row click (if `onRowClick`): cursor pointer
- Sortable column header: arrow icon (▲ ascending, ▼ descending, ↕ unsorted)
- Pagination: bottom-right, "Showing 1–25 of 100", prev/next buttons
- Loading: skeleton rows (`<SkeletonRow>` shared component)
- Empty state: `<EmptyState>` shared component if provided
- `position: sticky; top: 0` on header if `stickyHeader`

**Accessibility contract:**
- Real `<table>` semantics: `<thead>`, `<tbody>`, `<tr>`, `<th>`, `<td>`
- Sortable headers are `<button>` inside `<th>` with `aria-sort="ascending|descending|none"`
- Row click: real `<button>` inside `<td>` or whole row has `role="button"` + `tabIndex={0}` + Enter/Space handler
- Pagination: `<nav aria-label="Pagination">`, prev/next `<button>` with `aria-label="Previous page"`

**Usage:**
```tsx
<DataTable
  columns={[
    { key: 'name', header: 'Server', sortable: true },
    { key: 'status', header: 'Status', render: (r) => <StatusPill status={r.status} /> },
    { key: 'cpu', header: 'CPU', align: 'right', render: (r) => `${r.cpu}%` },
  ]}
  rows={servers}
  rowKey={(s) => s.id}
  onRowClick={(s) => navigate(`/servers/${s.id}`)}
  stickyHeader
/>
```

### 3.4 Input **[NEW]**, Select **[NEW]**, Textarea **[NEW]**

**Purpose:** Form input controls with consistent styling, validation
display, and accessibility.

**Location:** `web/src/components/shared/{Input,Select,Textarea}.tsx`

**Shared props:**
```typescript
interface BaseInputProps {
  label?: string;
  description?: string;       // hint text below label
  error?: string;             // error message (overrides description styling)
  required?: boolean;
  disabled?: boolean;
  size?: 'sm' | 'md';         // default 'md' (40px height)
  fullWidth?: boolean;
  'aria-label'?: string;
  'aria-describedby'?: string;
}
```

**Visual contract:**
- Container: `display: flex; flex-direction: column; gap: var(--space-1)`
- Label: 14px, 500 weight, `var(--color-text)`, 4px below container top
- Required indicator: `*` after label, `var(--color-error)`
- Input: 40px height (sm=32px), `var(--color-surface)` bg, 1px border `var(--color-border-strong)`, focus→`var(--color-border-focus)` 2px
- Padding: `var(--space-2) var(--space-3)`
- Font: 14px, `var(--color-text)`
- Placeholder: `var(--color-text-disabled)`
- Description: 12px, `var(--color-text-muted)`, 4px above input
- Error state: border `var(--color-error)`, description text replaced with error message in `var(--color-error)`, `aria-invalid="true"`, `aria-describedby` points to error id
- Disabled: opacity 0.5, cursor not-allowed, `aria-disabled="true"`
- Focus ring: 2px `var(--color-border-focus)` with 2px offset, via `:focus-visible`

**Accessibility contract:**
- `<label htmlFor>` properly associated with input id
- Error messages linked via `aria-describedby`
- Required: `aria-required="true"` + visible asterisk
- Type-specific: Input has `type` prop (text, email, password, etc.); Textarea has `rows` prop

**Usage:**
```tsx
<Input
  label="Server name"
  description="A friendly name shown in dashboards"
  value={name}
  onChange={setName}
  required
  error={errors.name}
/>
```

### 3.5 StatusPill **[VERIFY — exists]**

**Location:** `web/src/components/shared/StatusPill.tsx` (already exists)

**Contract from existing source (verified):**
- Props: `status`, `label?`, `size?` (sm/md/lg), `iconOnly?`, `ariaLabel?`
- Visual: 8px/10px/12px dot + 11px/12px/14px label, uppercase, letter-spaced
- Status colors: up/ok=green, stale/warn=amber, down/crit=red, unknown=gray
- `role="status"`, `aria-label` defaults to status text

**Acceptance:** Already exists; VERIFY visual contract is documented and
USED in 27/27 pages. Audit will confirm.

### 3.6 KpiCard **[VERIFY — exists]**

**Location:** `web/src/components/shared/KpiCard.tsx` (already exists)

**Contract from existing source (verified):**
- Props: `label`, `value`, `delta?`, `icon?`, `status?`, `accent?`, `sparkline?`, `trend?`, `onClick?`
- Accent options: cyan/indigo/green/amber/red/violet
- Status → accent mapping: up=cyan, neutral=cyan, stale=amber, down=red, crit=red
- Visual: 3px left stripe in accent color, hover lift via `cardLift` motion
- Animation: `kpiEnter` variants, respects `prefers-reduced-motion`
- Optional sparkline in matching tone

**Acceptance:** Already exists; VERIFY visual contract is documented
and USED in 27/27 pages with metrics.

### 3.7 EmptyState **[VERIFY — exists]**

**Location:** `web/src/components/shared/EmptyState.tsx` (already exists)

**Required contract (from proposal §6 FR-7):**
- Props: `icon?` (Lucide-style SVG), `title`, `description`, `action?` (Button or Link)
- Visual: 64px icon in muted color, 24px gap, centered, max-width 400px
- Title: 18px, 600 weight
- Description: 14px, muted, 8px below title
- Action: 16px below description

**Acceptance:** VERIFY current contract matches; update if missing
props.

### 3.8 SkeletonRow + SkeletonCard **[EXTEND]**

**Locations:** `web/src/components/shared/SkeletonRow.tsx` (exists); add `SkeletonCard.tsx`

**Required contract:**
- SkeletonRow: 1 row × N columns, 32px height, `var(--color-surface-elevated)` blocks with 8px gap
- SkeletonCard: 240×120px box with rounded corners, simulating a card layout (label block + value block + delta block)
- Animation: subtle pulse via CSS `@keyframes` 1500ms ease-in-out infinite, color `var(--color-surface)` ↔ `var(--color-surface-hover)`

**Acceptance:** SkeletonRow already exists; ADD SkeletonCard and use in pages.

---

## 4. Functional Requirements (expanded)

Each FR below is concrete, testable, and verifiable via grep, visual
inspection, or automated test.

| ID | Requirement | Verification method |
|---|---|---|
| **FR-1** | Every status indicator (server, alert, etc.) MUST use `<StatusPill>` | `grep "<StatusPill" web/src/pages/*.tsx` ≥1 per page |
| **FR-2** | Every KPI tile MUST use `<KpiCard>` | `grep "<KpiCard" web/src/pages/*.tsx` ≥1 per page (except Landing/Login/Signup which have no metrics) |
| **FR-3** | Every button MUST use `<Button>` (no inline `style={{ background: ... }}`) | `grep "style={{.*background"` returns 0 in `web/src/{components,pages}/` |
| **FR-4** | Every modal MUST use `<Modal>` | `grep "position: fixed"` returns 0 in pages |
| **FR-5** | Every table MUST use `<DataTable>` | `grep "<table"` returns 0 in pages (only in shared components) |
| **FR-6** | Every form input MUST use `<Input>` / `<Select>` / `<Textarea>` | `grep "<input" web/src/pages/*.tsx` returns 0 except in shared components |
| **FR-7** | Every empty state MUST use `<EmptyState>` with title + description + optional action | manual review |
| **FR-8** | Every loading state MUST use `<SkeletonRow>` or `<SkeletonCard>` or `<Spinner>` | manual review |
| **FR-9** | Every page MUST be wrapped in `<AnimatePresence mode="wait">` with `initial/animate/exit` (200ms ease-out) | `grep "AnimatePresence" web/src/components/AppShell.tsx` + per-page route transition |
| **FR-10** | Every interactive element MUST have hover + focus-visible + active states | `grep "whileHover" + manual `+ Lighthouse accessibility audit |
| **FR-11** | Every page MUST have `<a href="#main" className="skip-link">Skip to main content</a>` | `grep "skip-link" web/src/components/AppShell.tsx` (single source in AppShell, covers all pages) |
| **FR-12** | Every page MUST be navigable via keyboard (Tab order matches visual order) | manual keyboard test per page |
| **FR-13** | Every color MUST come from `var(--*)` token | `grep -E "#[0-9a-fA-F]{3,6}" web/src/{components,pages}/` returns 0 |
| **FR-14** | Every page MUST have `<main id="main">` landmark | `grep "<main" web/src/components/AppShell.tsx` covers all pages via wrapper |
| **FR-15** | Every page MUST show a `<title>` (via document.title set in useEffect or `<Helmet>`) | manual review + browser tab inspection |
| **FR-16** | The brand mark in the sidebar MUST be an SVG logo (not a letter "S") | `web/src/components/sidebar/Sidebar.tsx` shows `<BrandLogo>` component |
| **FR-17** | Every list/table cell that displays a timestamp MUST use the user's timezone via `new Date(iso).toLocaleString()` (not raw ISO string) | `grep ".toISOString().slice" returns 0 in pages` |
| **FR-18** | The dead-code sidebar `AppSidebar.tsx` MUST be deleted | `test -f web/src/components/AppSidebar.tsx` fails |

---

## 5. Non-Functional Requirements (expanded)

| ID | Requirement | Verification method |
|---|---|---|
| **NFR-1** | No source file under `web/src/{components,pages}/` exceeds 400 LOC | `find web/src/{components,pages} -name "*.tsx" -exec wc -l {} \; \| awk '$1 > 400'` returns empty |
| **NFR-2** | Lighthouse accessibility score ≥90 on `/dashboard`, `/proxmox-vms`, `/apm`, `/logs`, `/settings` | Lighthouse CLI |
| **NFR-3** | Bundle size increase after Tier 20 <5% | `du -sh web/dist/` before vs after |
| **NFR-4** | All existing functionality MUST keep working — no regression in click-through paths | 4 back-to-back verifier runs |
| **NFR-5** | Visual diff after refactor: each page screenshot vs previous ≤1% pixel delta | Playwright visual regression |
| **NFR-6** | `npm run build` MUST complete without warnings | CI build |
| **NFR-7** | `npm run type-check` (TypeScript strict-mode) MUST compile with zero errors | CI build |
| **NFR-8** | First Contentful Paint <1.5s on /dashboard | Lighthouse |
| **NFR-9** | Time to Interactive <3s on /dashboard | Lighthouse |
| **NFR-10** | Zero `console.error` or `console.warn` on any page load | Browser DevTools |

---

## 6. User stories (expanded with concrete scenarios)

| ID | As a | I want | So that | Concrete scenario |
|---|---|---|---|---|
| **US-01** | Operator | Every status pill (up/stale/down/crit/warn/ok) looks identical across every page | I can scan multiple pages without re-learning visual language | Open /dashboard → see "UP" pill. Open /proxmox-vms → see "UP" pill. They have same dot size, same label size, same color, same padding. |
| **US-02** | Operator | Every KPI tile (label, value, delta, sparkline) has the same shape, padding, stripe color logic everywhere | My eyes don't have to re-calibrate per page | Dashboard shows 4 KPI cards. Profile shows 3 KPI cards. Both use identical height, padding, stripe position, sparkline position. |
| **US-03** | Operator | Every button has identical height, padding, hover state | Click targets feel predictable | "Save" button on every form is 40px tall with same hover color. |
| **US-04** | Operator | Every page has a consistent empty state with action guidance | I know what to do next when a page is empty | /alerts with 0 alerts shows "No alerts" + "Create your first alert rule" button. |
| **US-05** | Operator | Every page transition uses 200ms ease-out | The app feels cohesive, not jarring | Click sidebar link → page fades in over 200ms. |
| **US-06** | Operator on mobile | Sidebar collapses to drawer, topbar stays, KPI strip wraps to 1-col | I can use the dashboard on a phone | At 320px: sidebar hidden, hamburger visible, clicking opens drawer. KPI strip is 1 column. |
| **US-07** | Operator with screen reader | Skip-link + ARIA labels on every interactive element | I can navigate without a mouse | Tab once → "Skip to main content" link. Tab again → first interactive element with aria-label. |
| **US-08** | Operator in low light | All text meets WCAG AA (4.5:1 body) | I can read it | Lighthouse: 0 contrast issues. |
| **US-09** | Operator comparing data | Every table is sortable + paginated consistently | I can find data quickly | /servers has sortable columns. Click "CPU" header → sorts. Click again → reverse sort. Pagination at bottom. |
| **US-10** | New operator onboarding | Every interactive element has visible focus state | I can use keyboard from day 1 | Tab to "Add server" button → blue focus ring visible. Enter → opens modal. |

---

## 7. Acceptance criteria (expanded, testable)

Each AC is a concrete test step.

### 7.1 Component contracts

- [ ] **AC-1**: `<Button>` exists at `web/src/components/shared/Button.tsx` with 4 variants + 3 sizes + loading state, exported as default
- [ ] **AC-2**: `<Modal>` exists at `web/src/components/shared/Modal.tsx` with focus trap + framer-motion enter/exit + escape close
- [ ] **AC-3**: `<DataTable>` exists at `web/src/components/shared/DataTable.tsx` with sortable columns + pagination + loading/empty states
- [ ] **AC-4**: `<Input>`, `<Select>`, `<Textarea>` exist with error states + required indicator
- [ ] **AC-5**: `<StatusPill>` + `<KpiCard>` + `<EmptyState>` + `<SkeletonRow>` + `<SkeletonCard>` all documented in their source files
- [ ] **AC-6**: Each new shared component has a showcase on `web/src/pages/StyleGuide.tsx` (unlisted from nav, behind `/style-guide` route)

### 7.2 Page-level consistency

- [ ] **AC-7**: All 27 pages use `<StatusPill>` where applicable (grep ≥1 match per page)
- [ ] **AC-8**: All 27 pages with metrics use `<KpiCard>` (grep ≥1 match per applicable page)
- [ ] **AC-9**: All 27 pages wrap content in `<main id="main">` (via AppShell wrapper)
- [ ] **AC-10**: All 27 pages have a document title (visible in browser tab)
- [ ] **AC-11**: All 27 pages have skip-link (via AppShell)
- [ ] **AC-12**: All 27 pages have page transition (via AnimatePresence in AppShell)

### 7.3 Visual consistency

- [ ] **AC-13**: All KPI cards across all pages have identical height, padding, stripe position (visual diff ≤1%)
- [ ] **AC-14**: All status pills across all pages have identical dot size, label size, padding
- [ ] **AC-15**: All buttons across all pages have identical height (40px md, 32px sm, 48px lg)
- [ ] **AC-16**: All modals across all pages have identical backdrop, padding, footer position

### 7.4 Code hygiene

- [ ] **AC-17**: Zero hardcoded hex colors in `web/src/{components,pages}/` (grep clean)
- [ ] **AC-18**: Zero emoji-as-icon in `web/src/{components,pages}/` (grep clean)
- [ ] **AC-19**: Zero `style={{ ... }}` inline overrides on Button/Modal/Card components (grep clean)
- [ ] **AC-20**: Zero file >400 LOC (find + wc check)
- [ ] **AC-21**: `web/src/components/AppSidebar.tsx` deleted (file does not exist)
- [ ] **AC-22**: Brand mark is SVG logo (not letter "S")

### 7.5 Build + runtime

- [ ] **AC-23**: `npm run build` completes with zero warnings
- [ ] **AC-24**: `npm run type-check` zero errors (strict mode)
- [ ] **AC-25**: Lighthouse accessibility ≥90 on 5 representative pages
- [ ] **AC-26**: Bundle size delta <5%
- [ ] **AC-27**: Zero console errors/warnings on any page load
- [ ] **AC-28**: 4 back-to-back full-stack verifier runs all PASS
- [ ] **AC-29**: User (himan) signs off after manually testing every page

---

## 8. Scenarios (concrete user flows)

### Scenario 1: New operator onboarding

> Alice just joined the team. She opens StackWatch for the first time.

1. She lands on https://stackwatch.smarthomelab.fun/app.html → redirects to /login
2. Sees Login page with consistent dark theme, primary button
3. Logs in → /dashboard
4. KPI strip at top shows 4 cards (servers up, alerts open, etc.)
5. Click "Add server" → modal opens with focus trap
6. Press Escape → modal closes, focus returns to button
7. Navigate to /proxmox-vms via sidebar
8. Sees Proxmox UI page with same topbar/sidebar as dashboard
9. KPI strip at top, same shape as dashboard

**Pass criteria:** No visual jarring between pages. Every interactive
element behaves identically.

### Scenario 2: Cross-page comparison

> Bob needs to compare server metrics on two pages.

1. Open /dashboard → sees KPI strip with "CPU Average: 42%"
2. Open /servers → see per-server CPU list
3. Click server row → /servers/{id}
4. Sees per-server metrics with KPI cards

**Pass criteria:** KPI cards identical across all 3 pages. Click
behavior identical (hover, focus, transition).

### Scenario 3: Mobile usage

> Carol checks alerts on her phone.

1. Open StackWatch at 320px width
2. Sidebar collapses, hamburger visible
3. KPI strip wraps to 1 column
4. Tables become horizontally scrollable (or collapse to cards)
5. Modals fit screen with proper padding

**Pass criteria:** No horizontal scroll on main content, no
overlapping elements, all interactive elements tappable.

### Scenario 4: Keyboard navigation

> Dave uses keyboard only (no mouse).

1. Tab into page → "Skip to main content" link
2. Enter → jumps to main content
3. Tab through KPI cards (focus rings visible)
4. Tab to sidebar items → focus rings visible, Enter navigates
5. Tab to first button → focus ring, Enter activates
6. Tab through modal → focus trap (cannot escape without Escape)
7. Escape → modal closes, focus returns to opener

**Pass criteria:** All interactive elements reachable via Tab, focus
visible, Enter/Space activate, Escape closes modals.

---

## 9. Data model

**No backend data model changes.** This tier is frontend-only refactor.

If audit reveals new shared components need to persist state (e.g. table
column preferences, layout choices), use `localStorage` only — no DB.

---

## 10. Out of Scope

(Inherited from proposal §3, restated for spec clarity)

- ❌ No new feature work (this tier is polish-only)
- ❌ No backend changes
- ❌ No new design tokens
- ❌ No redesign of layout / color palette / typography
- ❌ No tier-specific customization
- ❌ No icon additions
- ❌ No accessibility library rewrites
- ❌ `web/src/components/proxmox/` (covered by future Proxmox-UI tier)
- ❌ Backend (no Go changes)
- ❌ Tests (separate test infrastructure tier)
- ❌ Mobile app (`mobile/` — separate React Native stack)

---

## 11. Dependencies

(Inherited from proposal §10, expanded)

| Depends on | Status | Notes |
|---|---|---|
| `web/src/styles/tokens.css` | ✅ exists | Source of truth for tokens |
| `web/src/styles/DESIGN-SYSTEM.md` | ✅ exists | Philosophy + anti-slop rules |
| `web/src/components/icons.tsx` | ✅ exists | 30+ SVG icons |
| `web/src/components/shared/StatusPill.tsx` | ✅ exists | Verified contract above (§3.5) |
| `web/src/components/shared/KpiCard.tsx` | ✅ exists | Verified contract above (§3.6) |
| `web/src/components/shared/EmptyState.tsx` | ✅ exists | VERIFY contract |
| `web/src/components/shared/SkeletonRow.tsx` | ✅ exists | VERIFY + add SkeletonCard |
| `<Button>` shared | ❌ MISSING | Create |
| `<Modal>` shared | ❌ MISSING | Create |
| `<DataTable>` shared | ❌ MISSING | Create |
| `<Input>` shared | ❌ MISSING | Create |
| `<Select>` shared | ❌ MISSING | Create |
| `<Textarea>` shared | ❌ MISSING | Create |
| `framer-motion` | ✅ installed | AnimatePresence patterns in use |
| Playwright | ❓ Check | May need install for verification |

---

## 12. Open questions (inherited from proposal §12)

1. Per-page vs per-component order: per-component for shared, per-page for composition (default)
2. Button variants: 4 (primary/secondary/ghost/danger) enough? (default: 4)
3. Modal API: framer-motion vs CSS-only? (default: framer-motion)
4. DataTable scope: pagination + sort + filter all in one? (default: all in one to start)
5. Style-guide page: ship in same PR? (default: same PR, behind `/style-guide`, unlisted)
6. Old sidebar deletion: immediate? (default: immediate)

User feedback invited on any of these.

---

## 13. Risks (inherited from proposal §11, expanded)

| ID | Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| **R-1** | Refactor breaks existing functionality | Medium | High | 4 back-to-back verifier runs after every page; visual diff ≤1% |
| **R-2** | New shared components introduce bugs | Medium | Medium | StyleGuide showcase + Playwright matrix before page migration |
| **R-3** | User disagrees with design choices | Low | High | Per-page sign-off before next page |
| **R-4** | Audit finds 100+ drifts → scope creep | High | Medium | Severity gate: fix only critical+medium; defer low to backlog |
| **R-5** | Bundle size grows | Low | Low | Tree-shake unused variants; lazy-load modals |
| **R-6** | Mobile regression on a page | Medium | High | Playwright at 320×568 is mandatory gate |
| **R-7** | Color drift invisible to grep | Medium | Low | Visual diff catches; Lighthouse flags contrast |
| **R-8** | User abandons Tier 20 mid-way | Medium | High | Per-page commits; revert any single page if if user dissatisfied |
| **R-9** | Performance regression from new components | Low | Medium | Lighthouse + per-component render time budget |
| **R-10** | New TypeScript errors | Low | Low | `npm run type-check` mandatory gate |

---

## 14. Glossary (component-specific)

| Term | Definition |
|---|---|
| **Focus trap** | Modal prevents Tab from leaving; cycles within |
| **Restore focus** | When modal closes, focus returns to opener |
| **Sticky header** | Header row stays visible while body scrolls |
| **Empty state** | UI for "no data yet" with action guidance |
| **Loading state** | UI for "fetching data" with skeleton or spinner |
| **Empty + loading pattern** | Show skeleton while loading; show EmptyState if loading completes with 0 rows |
| **Skip link** | Hidden link that appears on Tab; jumps to main content |
| **AnimatePresence** | framer-motion component that animates children on mount/unmount |

---

**Next:** Awaiting user review. After approval → write `plan.md`
(file-by-file changes) → `tasks.md` (numbered steps) → `checklist.md`
(per-feature acceptance).

**Last updated:** 2026-08-31