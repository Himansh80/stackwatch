# Tier 20 — UI Consistency Polish

**Speckit change ID:** `020-ui-consistency-polish`
**Status:** PLAN (3 of 5 artifacts)
**Depends on:** `spec.md` ✅ approved 2026-08-31

---

## 1. Document purpose

This plan translates the spec's component contracts + requirements into
**concrete file-by-file changes** in execution order. After approval,
`tasks.md` numbers each step with verification gates; `checklist.md`
provides per-feature acceptance.

---

## 2. Architecture overview

### 2.1 What changes

```
NEW files:
  web/src/components/shared/Button.tsx
  web/src/components/shared/Modal.tsx
  web/src/components/shared/DataTable.tsx
  web/src/components/shared/Input.tsx
  web/src/components/shared/Select.tsx
  web/src/components/shared/Textarea.tsx
  web/src/components/shared/SkeletonCard.tsx
  web/src/components/shared/BrandLogo.tsx
  web/src/pages/StyleGuide.tsx
  
DELETED files:
  web/src/components/AppSidebar.tsx (dead code per U2)
  web/src/components/Sidebar.tsx (old sidebar — verify first)

MODIFIED files:
  web/src/styles/components.css (+ add Button/Modal/DataTable/Input styles using tokens)
  web/src/styles/tokens.css (+ ensure all needed tokens exist)
  web/src/styles/DESIGN-SYSTEM.md (+ document new components)
  web/src/components/sidebar/Sidebar.tsx (use BrandLogo)
  web/src/components/sidebar/SidebarItem.tsx (verify against shared Button patterns)
  web/src/components/topbar/TopBar.tsx + sub-components (compose cleanly, use shared patterns)
  web/src/components/shared/StatusPill.tsx (verify contract, add if missing)
  web/src/components/shared/KpiCard.tsx (verify contract, add if missing)
  web/src/components/shared/EmptyState.tsx (verify contract, add if missing)
  web/src/components/AppShell.tsx (verify skip-link + main landmark)
  web/src/App.tsx (add /style-guide route)
  web/src/pages/*.tsx (27 files — see Phase D)
```

### 2.2 What stays

- Backend (no Go changes)
- `web/src/styles/tokens.css` token *values* (only add new tokens if absolutely needed)
- `web/src/components/icons.tsx` (30+ icons, sufficient)
- `web/src/components/proxmox/*` (out of scope, future Proxmox UI tier)
- Mobile (`mobile/`)
- Customer docs (`docs/`)
- Migrations (no DB changes)

### 2.3 Dependency graph

```
shared/Button ─┐
shared/Modal ──┤
shared/DataTable ┤
shared/Input ───────► pages/* (refactor) ──► verify
shared/Select ┤
shared/Textarea ┤
shared/SkeletonCard ┤
shared/BrandLogo ───► sidebar/Sidebar.tsx
              │
pages/StyleGuide ─► shows all new components
              │
audit (Phase A) ───► drives Phase D page order
```

---

## 3. Phase A — Audit (mandatory first)

**Goal:** Produce `docs/UI-AUDIT-REPORT.md` with exact file:line of every drift.

**Owner:** Dedicated subagent.

**Inputs:**
- `web/src/pages/*.tsx` (27 files)
- `web/src/components/**/*.tsx` (78 files, exclude `proxmox/`)
- `web/src/styles/*.css` (14 files)
- `web/src/styles/DESIGN-SYSTEM.md` (target contract)

**Tasks:**

1. Spawn subagent with goal: "Crawl every page + every shared component. For each element type (button, KPI card, status pill, modal, table, form input, empty state, loading state), catalog every usage. Output a table with: file path, line range, element type, current implementation, drift from design system, severity (critical/medium/low), proposed fix."

2. Subagent outputs `docs/UI-AUDIT-REPORT.md` (~2,000+ lines expected)

3. I review the audit, classify all "critical" drifts as MUST-FIX, "medium" as SHOULD-FIX, "low" as DEFER

4. Audit-driven per-page refactor order finalized in Phase D

**Gate:** Audit must be reviewed + classified before Phase B starts. No code changes until audit is approved.

**Effort:** 2-3 hours

---

## 4. Phase B — Shared components (7 new)

**Goal:** Build the shared components needed for page refactor. None of these exist yet.

**Order:** Component chosen first should be the one with the most existing inline equivalents (highest leverage).

### 4.1 Button.tsx [HIGHEST LEVERAGE]

**File:** `web/src/components/shared/Button.tsx` (~150 LOC)

**Dependencies:** `framer-motion` (already in use)

**Props:** From spec §3.1

**Implementation notes:**
- Use `motion.button` from framer-motion for the active scale
- Use `<style>` CSS classes (not styled-components — project uses plain CSS)
- Variants implemented as `className` joins: `btn btn-primary btn-md`
- Loading state: render `<Spinner size={16} />` instead of children
- Full width: `className="btn btn-full"` with `width: 100%`

**Tests:** `web/src/components/shared/Button.test.tsx` (~80 LOC)
- Renders 4 variants correctly
- Renders 3 sizes correctly
- Loading state shows spinner + disabled
- Click handler fires
- aria-disabled + aria-busy on loading

**Style location:** Add to `web/src/styles/components.css`

**Effort:** 1.5 hours

### 4.2 Modal.tsx

**File:** `web/src/components/shared/Modal.tsx` (~180 LOC)

**Dependencies:** `framer-motion`, `react` (useRef, useEffect)

**Props:** From spec §3.2

**Implementation notes:**
- Use AnimatePresence + motion.div for backdrop + panel
- Focus trap: track focusable elements, intercept Tab/Shift+Tab
- Restore focus: save opener element on open, return focus on close
- Body scroll lock: `document.body.style.overflow = 'hidden'` on mount, restore on unmount
- Escape key: global keydown listener (only when `closeOnEscape=true`)
- Backdrop click: onClick on backdrop div (only if `closeOnBackdrop=true`)

**Tests:** `web/src/components/shared/Modal.test.tsx` (~100 LOC)
- Opens/closes correctly
- Focus trap works (Tab cycles within)
- Escape closes (when enabled)
- Backdrop click closes (when enabled)
- Focus returns to opener on close
- Body scroll locked while open

**Style location:** Add to `web/src/styles/components.css`

**Effort:** 2 hours

### 4.3 DataTable.tsx

**File:** `web/src/components/shared/DataTable.tsx` (~250 LOC)

**Dependencies:** `framer-motion` (row hover), React generics

**Props:** From spec §3.3

**Implementation notes:**
- Generic `<T>` component
- Sort: track `sortBy` + `sortDir` state if uncontrolled, or read props if controlled
- Pagination: track `currentPage` + `pageSize` if uncontrolled
- Header: render `<thead>` with `<th>` elements; sortable headers wrap content in `<button>` for keyboard accessibility
- Body: render `<tbody>` with `<tr>` + `<td>` using `row.render(row)` for cell content
- Sticky header: `position: sticky; top: 0` with elevated background
- Empty: render `<EmptyState>` shared component (passed as prop OR default)
- Loading: render `<SkeletonRow>` shared component for N rows
- Pagination UI: prev/next buttons + "Showing X–Y of Z" text

**Tests:** `web/src/components/shared/DataTable.test.tsx` (~150 LOC)
- Renders columns + rows
- Sort: click header, fires onSort with correct key + dir
- Pagination: click next, fires onPageChange
- Empty: renders EmptyState
- Loading: renders skeletons
- Row click: fires onRowClick

**Style location:** Add to `web/src/styles/components.css`

**Effort:** 3 hours

### 4.4 Input.tsx + Select.tsx + Textarea.tsx [BATCHED]

**Files:**
- `web/src/components/shared/Input.tsx` (~120 LOC)
- `web/src/components/shared/Select.tsx` (~100 LOC)
- `web/src/components/shared/Textarea.tsx` (~80 LOC)

**Dependencies:** React.forwardRef for Input (for form libraries later)

**Props:** From spec §3.4

**Implementation notes:**
- Shared `BaseInputWrapper` sub-component for label + description + error layout
- Error state: red border + red description text + `aria-invalid` + `aria-describedby`
- Required indicator: `<span class="input-required" aria-hidden="true">*</span>` after label
- Use design tokens for all colors (no hardcoded)

**Tests:** Each gets a test file (~60 LOC each)
- Renders label + input + description
- Error state shows error message + aria-invalid
- Disabled state has aria-disabled
- Required shows asterisk
- Focus visible

**Style location:** Add to `web/src/styles/components.css`

**Effort:** 2 hours (batched, shared logic)

### 4.5 SkeletonCard.tsx [EXTEND]

**File:** `web/src/components/shared/SkeletonCard.tsx` (~50 LOC)

**Note:** SkeletonRow already exists; add SkeletonCard for card-shaped loading.

**Props:**
```typescript
interface SkeletonCardProps {
  width?: string | number;
  height?: string | number;
  variant?: 'kpi' | 'panel' | 'list';  // shapes the skeleton
}
```

**Implementation notes:**
- Pulse animation via CSS @keyframes
- KPI variant: 1 large block + 1 small block (simulating label + value)
- Panel variant: 1 large block (full card)
- List variant: 3 horizontal blocks stacked

**Effort:** 0.5 hours

### 4.6 BrandLogo.tsx

**File:** `web/src/components/shared/BrandLogo.tsx` (~40 LOC)

**Purpose:** Replace letter "S" brand mark in sidebar with proper SVG logo.

**Props:**
```typescript
interface BrandLogoProps {
  size?: number;  // default 32
  variant?: 'full' | 'mark';  // 'full' = mark + text, 'mark' = mark only
}
```

**Implementation notes:**
- Inline SVG (not external file)
- Geometric design (Datadog-style stacked rectangles, or simple "S" monogram)
- Uses `currentColor` so it inherits text color (cyan for brand)
- Animation: subtle hover (optional)

**Effort:** 0.5 hours

### Phase B total effort: ~10 hours

---

## 5. Phase C — StyleGuide page

**Goal:** Visual showcase for every new shared component, accessible at `/style-guide` (unlisted from nav).

**File:** `web/src/pages/StyleGuide.tsx` (~300 LOC)

**Sections:**
1. **Color tokens** — visual swatches of every `var(--color-*)` token
2. **Typography scale** — every `var(--text-*)` size rendered with sample text
3. **Spacing scale** — visual bars of every `var(--space-*)` value
4. **Buttons** — every variant × every size × every state (default/hover/active/disabled/loading)
5. **Modals** — one open modal example, one closed
6. **DataTable** — sample table with sortable headers, pagination
7. **Inputs** — every input variant (text, email, password, select, textarea) + error state
8. **StatusPill** — every status (up/stale/down/crit/warn/ok/unknown) × every size (sm/md/lg)
9. **KpiCard** — every accent × with/without sparkline × with/without delta
10. **EmptyState** — example empty state with icon + title + description + action
11. **Loading** — SkeletonRow + SkeletonCard examples

**Route registration:** `web/src/App.tsx`
```tsx
<Route path="/style-guide" element={isLoggedIn ? auth(StyleGuide, {}) : <Navigate to="/login" replace />} />
```

**NOT in sidebar nav** — only accessible via direct URL.

**Effort:** 2 hours

---

## 6. Phase D — Page-by-page refactor (27 pages)

**Goal:** Every page uses shared components + tokens. Zero hardcoded values. Zero emoji-as-icon. Consistent visual contract.

### 6.1 Order (dependency-driven)

Group pages by shared patterns:

**Group 1 — Foundation (no changes expected, just verify):**
- `Landing.tsx` — public marketing page
- `Login.tsx` — public auth
- `Signup.tsx` — public auth
- `ForgotPassword.tsx` — public auth
- `ResetPassword.tsx` — public auth

These pages use forms heavily → Input + Button + Select refactor applies. After Phase B, refactor these first.

**Group 2 — Workspace pages (5):**
- `Dashboard.tsx` — main authenticated page
- `ProfilePage.tsx` — user profile
- `BillingPage.tsx` — billing/subscription
- `SettingsPage.tsx` — app settings
- `HomelabPage.tsx` — homelab dashboard

These have KPI strips + status pills + buttons → most refactor work.

**Group 3 — Observability pages (8):**
- `ApmPage.tsx`
- `ApmServicePage.tsx`
- `LogsFullPage.tsx`
- `RumFullPage.tsx`
- `RumSessionPage.tsx`
- `SyntheticsPage.tsx`
- `SecurityPage.tsx`
- `CspmPage.tsx`

These have data tables + filters + status indicators.

**Group 4 — Infrastructure pages (3):**
- `CicdPage.tsx`
- `DatabasePage.tsx`
- `TrueNASWorkspace.tsx`

These have resource lists + status pills + buttons.

**Group 5 — Operations pages (4):**
- `IncidentsPage.tsx`
- `NotebookPage.tsx`
- `IntelligencePage.tsx`
- `SharedDashboardsPage.tsx`

These have incident/insight lists.

**Group 6 — Enterprise + Platform (2):**
- `EnterprisePage.tsx`
- `PlatformPage.tsx`

These have settings forms + plan info.

### 6.2 Per-page refactor pattern

For each page:

1. **Read the file** (~5 min)
2. **Identify drifts** vs spec (~5 min):
   - Inline buttons → replace with `<Button>`
   - Inline status indicators → replace with `<StatusPill>`
   - Inline KPI tiles → replace with `<KpiCard>`
   - Inline modals → replace with `<Modal>`
   - Inline tables → replace with `<DataTable>`
   - Inline form inputs → replace with `<Input>` / `<Select>` / `<Textarea>`
   - Hardcoded colors → replace with `var(--*)` tokens
   - Emoji-as-icon → replace with SVG from `icons.tsx`
   - Missing loading state → add `<SkeletonRow>` or `<SkeletonCard>`
   - Missing empty state → add `<EmptyState>` (if applicable)
3. **Apply refactor** (~15-30 min):
   - Edit the file
   - Commit per-page (so revert is per-page)
4. **Verify** (~5 min):
   - `npm run type-check` passes
   - `npm run build` passes
   - Visual screenshot via Playwright
   - Manual smoke test

### 6.3 Per-page effort estimate

| Group | Pages | Effort/page | Total |
|---|---|---|---|
| 1 — Foundation | 5 | 20 min | 1.7 hours |
| 2 — Workspace | 5 | 30 min | 2.5 hours |
| 3 — Observability | 8 | 25 min | 3.3 hours |
| 4 — Infrastructure | 3 | 25 min | 1.3 hours |
| 5 — Operations | 4 | 25 min | 1.7 hours |
| 6 — Enterprise + Platform | 2 | 20 min | 0.7 hours |

**Phase D total:** ~11 hours

### 6.4 Per-page commit convention

One commit per page:
```
refactor(tier20): Dashboard uses shared Button + StatusPill + KpiCard

- Replace 4 inline buttons with <Button>
- Replace 2 inline status dots with <StatusPill>
- Replace 4 KPI tiles with <KpiCard>
- Add skip-link + main landmark (already in AppShell)
- Add framer-motion AnimatePresence page transition

AC: AC-7, AC-8, AC-13, AC-14, AC-15, AC-17, AC-18
```

This makes revert per-page trivial.

---

## 7. Phase E — Dead code + branding

**Goal:** Delete dead code, fix brand mark.

### 7.1 Delete dead sidebar

**File:** `web/src/components/AppSidebar.tsx` (173 LOC)

**Verification before deletion:**
- `grep -rn "AppSidebar" web/src/` — only page comments mention it (no actual imports)
- Confirm AppShell uses `./sidebar/Sidebar` not `AppSidebar`

**Action:**
```bash
git rm web/src/components/AppSidebar.tsx
```

**Effort:** 5 min

### 7.2 Replace brand mark

**Files:**
- `web/src/components/sidebar/Sidebar.tsx` — replace `<span className="sb-brand-mark">S</span>` with `<BrandLogo size={32} />`

**Action:**
1. Import BrandLogo in Sidebar.tsx
2. Replace `<span className="sb-brand-mark" aria-hidden="true">S</span>` with `<BrandLogo size={32} />`
3. Update SidebarItem.tsx if it references brand mark anywhere

**Verification:** Visual — sidebar shows SVG logo, not "S"

**Effort:** 10 min

### 7.3 Topbar composition check

**Files:** `web/src/components/topbar/TopBar.tsx` + 8 sub-components

**Audit:** Verify all sub-components:
- Use shared Button for any buttons
- Use shared Input for any inputs (search bar)
- Use shared Modal for any modals (notifications dropdown)
- No hardcoded colors
- No emoji icons
- Consistent heights (40px md button height across sub-components)

**Likely findings:** TopBarSearch uses custom CSS, may need `<Input>` refactor.

**Effort:** 30 min

### 7.4 Update DESIGN-SYSTEM.md

**Add sections for new components:**
- Button (variants, sizes, states)
- Modal (focus trap, restore focus)
- DataTable (sort, pagination)
- Input/Select/Textarea (error states, validation)

**Effort:** 30 min

### Phase E total effort: ~1.5 hours

---

## 8. Phase F — Verification

**Goal:** Prove every AC passes, 4 back-to-back.

### 8.1 Automated verification

**Run order (sequential):**

```bash
# 1. Type check
cd web && npm run type-check

# 2. Build
npm run build

# 3. Grep checks (per FR/AC)
grep -rn "#[0-9a-fA-F]{3,6}" web/src/{components,pages}/  # AC-17: should be 0
grep -rn "[⌂$◉⚙◈▤◴≡◎◔◍◐⇄◧⚑◰⌬✦◊]" web/src/{components,pages}/  # AC-18: should be 0
grep -rn "style={{.*background" web/src/{components,pages}/  # AC-19: should be 0
find web/src/{components,pages} -name "*.tsx" -exec wc -l {} \; | awk '$1 > 400'  # AC-20: should be empty
test ! -f web/src/components/AppSidebar.tsx && echo "OK"  # AC-21
grep -rn "<StatusPill" web/src/pages/*.tsx | wc -l  # AC-7: should be ≥20 (excludes Landing/Login)
grep -rn "<KpiCard" web/src/pages/*.tsx | wc -l  # AC-8: should be ≥10
```

**Gate:** All grep checks must return expected values.

### 8.2 Visual verification (Playwright)

**Script:** `verify/ui-tier20-screenshots.py` (~100 LOC)

**Captures:**
- Every page at 1440×900 → `verify/ui-screenshots/desktop/{page}.png`
- Every page at 320×568 → `verify/ui-screenshots/mobile/{page}.png`
- Visual diff vs previous → `verify/ui-screenshots/diff/{page}.png`

**Pages to capture (27):**
dashboard, proxmox-vms, truenas, apm, logs, rum, synthetics, security, cspm, cicd, database, incidents, notebooks, shared, intelligence, enterprise, homelab, platform, profile, settings, billing, plus StyleGuide.

### 8.3 Accessibility verification

**Script:** `verify/ui-tier20-axe.py` (~80 LOC)

Uses `axe-core` (via Playwright or direct integration) to scan every page for accessibility violations.

**Targets:** 0 critical violations per page.

### 8.4 Lighthouse

**Script:** `verify/ui-tier20-lighthouse.py`

Run Lighthouse on /dashboard, /proxmox-vms, /apm, /logs, /settings.

**Targets:**
- Accessibility ≥90 per page
- Performance: LCP <1.5s, TTI <3s
- Bundle size delta <5%

### 8.5 Full-stack regression

**Reuse existing verifier:** `verify/tier0-stack-2026-08-17.py` (or similar) — verify all backend endpoints still work.

**Run 4 back-to-back times** with fresh JWT per run.

### 8.6 User manual test

**Checklist:**
- User opens every page in dashboard
- Every page looks identical-quality
- Every interactive element behaves identically
- Mobile (320px) is usable
- Accessibility scan passes

**Effort:** 2-3 hours

### Phase F total effort: ~5 hours

---

## 9. File-by-file changes summary

| Path | Change | Phase |
|---|---|---|
| `web/src/components/shared/Button.tsx` | CREATE | B |
| `web/src/components/shared/Button.test.tsx` | CREATE | B |
| `web/src/components/shared/Modal.tsx` | CREATE | B |
| `web/src/components/shared/Modal.test.tsx` | CREATE | B |
| `web/src/components/shared/DataTable.tsx` | CREATE | B |
| `web/src/components/shared/DataTable.test.tsx` | CREATE | B |
| `web/src/components/shared/Input.tsx` | CREATE | B |
| `web/src/components/shared/Input.test.tsx` | CREATE | B |
| `web/src/components/shared/Select.tsx` | CREATE | B |
| `web/src/components/shared/Select.test.tsx` | CREATE | B |
| `web/src/components/shared/Textarea.tsx` | CREATE | B |
| `web/src/components/shared/Textarea.test.tsx` | CREATE | B |
| `web/src/components/shared/SkeletonCard.tsx` | CREATE | B |
| `web/src/components/shared/BrandLogo.tsx` | CREATE | B |
| `web/src/components/shared/StatusPill.tsx` | VERIFY/EXTEND | B/D |
| `web/src/components/shared/KpiCard.tsx` | VERIFY/EXTEND | B/D |
| `web/src/components/shared/EmptyState.tsx` | VERIFY/EXTEND | B/D |
| `web/src/components/shared/SkeletonRow.tsx` | VERIFY | B |
| `web/src/pages/StyleGuide.tsx` | CREATE | C |
| `web/src/styles/components.css` | EXTEND | B |
| `web/src/styles/DESIGN-SYSTEM.md` | EXTEND | E |
| `web/src/components/sidebar/Sidebar.tsx` | MODIFY (BrandLogo) | E |
| `web/src/components/sidebar/SidebarItem.tsx` | VERIFY | E |
| `web/src/components/topbar/TopBar.tsx` + 8 sub | VERIFY/MODIFY | E |
| `web/src/components/AppShell.tsx` | VERIFY (skip-link + main) | D |
| `web/src/App.tsx` | MODIFY (+ /style-guide route) | C |
| `web/src/pages/*.tsx` (27 files) | MODIFY (refactor) | D |
| `web/src/components/AppSidebar.tsx` | DELETE | E |
| `docs/UI-AUDIT-REPORT.md` | CREATE (subagent output) | A |
| `verify/ui-tier20-screenshots.py` | CREATE | F |
| `verify/ui-tier20-axe.py` | CREATE | F |
| `verify/ui-tier20-lighthouse.py` | CREATE | F |

**Total files:** ~30 created, ~30 modified, 1 deleted

---

## 10. Migration strategy

### 10.1 No breaking changes

- All existing routes still work
- All existing functionality still works (4 back-to-back verifier proves)
- Old AppSidebar deleted only after audit confirms zero imports
- New shared components added alongside existing ones; pages migrated incrementally

### 10.2 Rollout phases

```
Phase A (audit)         → no user-visible changes
Phase B (shared comps)  → no user-visible changes (unused)
Phase C (StyleGuide)    → no user-visible changes (unlisted)
Phase D (page refactor) → user-visible per-page changes; per-page commits
Phase E (cleanup)       → brand mark changes (visible)
Phase F (verify)        → confirmation, no further changes
```

### 10.3 Per-page rollout

For each page in Phase D:
1. Read + identify drifts
2. Apply refactor in single commit
3. Verify build + tests + visual
4. **STOP if anything breaks** — don't proceed to next page
5. Move to next page only after current is verified

If user wants to review per-page: pause and ask before continuing.

### 10.4 Feature flag (optional)

If user wants extra safety, we could wrap StyleGuide + new components in `if (process.env.NODE_ENV !== 'production')` checks. **Decision: skip** — new components are inert until used; StyleGuide is unlisted.

---

## 11. Rollback plan

### 11.1 Per-file rollback

Every change is committed per-file (or per-page). Rollback is `git revert <commit-sha>` for the specific file.

### 11.2 Per-phase rollback

If Phase D breaks something:
- Stop immediately
- `git log --oneline -20` — find the last working commit
- `git revert` all Phase D commits since that point
- Working tree returns to pre-Phase-D state
- Re-plan Phase D with smaller scope

### 11.3 Full rollback

If entire Tier 20 is wrong direction:
- `git revert` all Tier 20 commits (proposal + spec + plan + tasks + checklist are docs, only build commits are code)
- Restore `web/src/components/AppSidebar.tsx` from git history
- Restore old brand mark
- Working tree returns to pre-Tier-20 state
- Estimated worst-case loss: 1 working day of code (proposal/spec/plan/tasks/checklist are documentation, not work)

### 11.4 Backup binaries

Per `stackwatch-build-deploy` skill discipline: build gentle, never break working production. If Tier 20 build breaks .115, immediately `cp /opt/stackwatch/bin/api-gateway-linux.bak-pre-tier20 /opt/stackwatch/bin/api-gateway-linux && systemctl restart stackwatch-api-gateway` (this is for backend changes — Tier 20 is frontend-only so doesn't apply, but principle stands).

---

## 12. Risk-driven deviations

If during execution we discover:

| Discovery | Deviation |
|---|---|
| Audit finds >100 critical drifts | Defer low + medium to Tier 21 or backlog |
| Per-page refactor breaks a flow | Stop, isolate the page, revert just that page |
| New shared components have API design issues | Revise spec §3 + propagate to plan |
| User changes design preferences mid-build | Pause, get sign-off, resume |
| Bundle size grows >5% | Tree-shake + lazy-load modals |
| Performance regression | Profile + optimize hot components |

---

## 13. Effort summary

| Phase | Effort |
|---|---|
| A — Audit (subagent) | 2-3 hours |
| B — Shared components (7) | ~10 hours |
| C — StyleGuide | 2 hours |
| D — Page refactor (27 pages) | ~11 hours |
| E — Dead code + branding | ~1.5 hours |
| F — Verification | ~5 hours |
| **Total** | **~31-33 hours (~5-6 working sessions)** |

---

## 14. Dependency graph (build order)

```
A (audit)
  ↓
B (shared components) ──→ C (StyleGuide)
  ↓                          ↓
D (page refactor, per-page) ←┘
  ↓
E (cleanup + branding)
  ↓
F (verify)
```

---

**Next:** Awaiting user review. After approval → write `tasks.md`
(numbered steps with verification gates per task) → `checklist.md`
(per-feature acceptance with ✅/❌).

**Last updated:** 2026-08-31