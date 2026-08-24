# Feature Specification: Split page modules

**Feature Branch**: `001-split-page-modules`

**Created**: 2026-08-24

**Status**: Draft

**Input**: User description: "Modularity refactor — split large page files (Dashboard.tsx, ProfilePage.tsx) into smaller components, and split the 1528-line styles.css into per-page CSS files. No behavior changes."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Maintainer can locate a single component quickly (Priority: P1)

A developer opens the codebase looking for the Profile page's avatar upload logic. They need to find that file in seconds rather than scrolling through a 745-line page component.

**Why this priority**: This is the core motivation — readability of large page files is the original complaint. Without splitting, future polish work will keep making the files bigger and harder to navigate.

**Independent Test**: Open `web/src/components/profile/AvatarUpload.tsx` and confirm it exists, contains the `onPickAvatar` function from the current ProfilePage, and is exported. Verify ProfilePage.tsx no longer contains that function.

**Acceptance Scenarios**:

1. **Given** a developer wants to find avatar upload logic, **When** they navigate to `web/src/components/profile/AvatarUpload.tsx`, **Then** they find the function in < 100 lines of focused code.
2. **Given** the same developer, **When** they look at `web/src/pages/ProfilePage.tsx`, **Then** it imports `AvatarUpload` from `components/profile/` and no longer defines the function locally.

---

### User Story 2 - Maintainer can restyle one page without touching the whole stylesheet (Priority: P1)

A designer wants to tweak the Profile page's hero card padding. They edit `web/src/styles/profile.css` without touching `styles.css` or risking regressions on Dashboard.

**Why this priority**: Currently `styles.css` mixes `.dash-*` (dashboard) with `.prof-*` (profile) with `.sw-*` (settings/billing/auth). Changing one risks affecting the others. Splitting prevents accidental cross-page style leakage.

**Independent Test**: Importing `web/src/styles/profile.css` content alone should contain all `.prof-*` rules and zero `.dash-*` or `.sw-*` rules.

**Acceptance Scenarios**:

1. **Given** `web/src/styles/profile.css`, **When** grepped for `.prof-` selectors, **Then** every `.prof-*` rule in the current `styles.css` appears here.
2. **Given** the same file, **When** grepped for `.dash-` selectors, **Then** it returns no matches.
3. **Given** `web/src/styles/dashboard.css`, **When** grepped for `.dash-` selectors, **Then** every `.dash-*` rule from current `styles.css` appears here.

---

### User Story 3 - Profile page renders identically after refactor (Priority: P1)

A user opens the Profile page after the refactor. Every element looks identical to before — same hero card, same identity rows, same workspace stats, same form, same security panel, same tokens panel.

**Why this priority**: This is a structural refactor — any visual regression is a bug. Pixel-level parity is the goal.

**Independent Test**: Take a screenshot of Profile page before and after the refactor, diff with `compare-images`. Zero pixel differences (other than timestamps).

**Acceptance Scenarios**:

1. **Given** the current Profile page rendered with test data, **When** I capture a screenshot, **Then** I see: hero card with avatar + name + meta pills + email; Identity panel with 4 rows + copy buttons; Workspace panel with 4 stats; Display name form; Sign-in & access security panel with 4 rows.
2. **Given** the refactored Profile page rendered with the same test data, **When** I capture a screenshot, **Then** the same elements appear in the same positions with the same visual styling.
3. **Given** the diff between the two screenshots, **Then** no pixels differ (other than anti-aliasing variance and live-clock text).

---

### User Story 4 - Dashboard page renders identically after refactor (Priority: P1)

A user opens the Dashboard after the refactor. KPI cards, welcome header, error bar, Live Telemetry chart, Operations panel — all look identical.

**Why this priority**: Same as Story 3 — structural refactor must preserve visual state.

**Independent Test**: Same pixel-diff methodology as Story 3.

**Acceptance Scenarios**:

1. **Given** the current Dashboard rendered with live data, **When** I capture a screenshot, **Then** I see: topbar with greeting + search center + clock + refresh + avatar; welcome header with eyebrow + h2 + Live sync card; 4 metric cards; chart panel + Operations panel.
2. **Given** the refactored Dashboard, **Then** the same elements appear with identical styling.

---

### User Story 5 - All interactive behaviors still work (Priority: P2)

Every click handler, form submit, hover state, popover open/close, password copy, token create/revoke — same behavior as before the refactor.

**Why this priority**: Functionality matters as much as visuals. P2 because visual parity (Stories 3-4) already implies most behaviors; this catches edge cases.

**Independent Test**: Manually click through every interactive element and observe the behavior matches what was there before.

**Acceptance Scenarios**:

1. **Given** the refactored Profile page, **When** the user clicks "Copy" next to User ID, **Then** the button briefly shows "✓ Copied" and the clipboard contains the full UUID.
2. **Given** the refactored Profile page, **When** the user expands the API tokens panel, **Then** it loads existing tokens from `/api/v1/api-keys` and lets the user create + revoke.
3. **Given** the refactored Dashboard, **When** the user clicks the avatar or 📷 overlay, **Then** the file picker opens.

---

### User Story 6 - Build, type-check, and bundle size remain healthy (Priority: P2)

The refactor doesn't break the build pipeline or inflate the bundle.

**Why this priority**: Refactors that break builds get reverted. P2 because build issues are caught by CI.

**Independent Test**: Run `npm run type-check && npm run build` and verify exit 0; compare bundle size before/after.

**Acceptance Scenarios**:

1. **Given** the refactored source tree, **When** `npm run type-check` runs, **Then** exit code is 0 with no errors.
2. **Given** the refactored source tree, **When** `npm run build` runs, **Then** exit code is 0.
3. **Given** the post-refactor bundle, **When** its size is measured, **Then** it is within ±10KB of the pre-refactor bundle (current: 315KB JS + 72KB CSS).

---

### Edge Cases

- What happens if a component has the same name as an existing module? → Use the prefix `components/dashboard/`, `components/profile/`, etc., so names don't collide.
- What happens if a CSS rule in `styles.css` references multiple page-specific classes (e.g., `.dash-welcome` used on Profile)? → That rule belongs in `styles/dashboard.css` and is loaded on both pages via a shared `index.css`.
- What happens if the user has the app open during deploy? → Hard refresh clears cached JS chunks.
- What happens if a helper component (Stat, SecurityRow) is also used elsewhere later? → Keep them in `components/profile/` for now; promote to `components/shared/` when a second consumer appears (YAGNI).
- What happens to CSS `:root` design tokens when splitting files? → They stay in `styles/base.css` which all pages import; no duplication.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST split `web/src/pages/ProfilePage.tsx` into a page shell + 6 component files in `web/src/components/profile/`.
- **FR-002**: System MUST split `web/src/pages/Dashboard.tsx` into a page shell + 6 component files in `web/src/components/dashboard/`.
- **FR-003**: System MUST extract shared SVG icons used in MetricCard into `web/src/components/icons.tsx` for reuse.
- **FR-004**: System MUST split `web/src/styles.css` into per-page CSS files under `web/src/styles/`: `base.css`, `dashboard.css`, `profile.css`, `auth.css`, `shared.css`.
- **FR-005**: System MUST NOT change any user-visible behavior, styling, layout, or interaction as a result of this refactor.
- **FR-006**: System MUST preserve all existing design tokens (CSS variables) in `:root` without renaming or removing any.
- **FR-007**: System MUST keep `styles.css` importable as a single file (re-export from an index) so the build pipeline doesn't break.
- **FR-008**: System MUST update the Vite import paths so all pages import their own CSS file (e.g., `import '../styles/profile.css'`).
- **FR-009**: System MUST run `npm run type-check && npm run build` after each split step with exit 0.
- **FR-010**: System MUST capture before/after screenshots of Profile and Dashboard for visual diff verification.

### Key Entities *(include if feature involves data)*

- **Page component**: A top-level route component (e.g., `ProfilePage`). Owns data fetching, top-level state, and page composition. Delegates rendering to child components.
- **Section component**: A section of a page that has its own internal state OR is purely presentational (e.g., `HeroCard`, `IdentityPanel`, `MetricCard`).
- **Shared icon component**: An SVG icon defined once and exported as a named React component (e.g., `<ServerIcon />`, `<NetworkIcon />`).
- **Style module**: A CSS file containing a focused set of rules (`.dash-*`, `.prof-*`, `.sw-*`) imported by the page that needs them.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After refactor, no single page file exceeds 250 lines (current worst: `ProfilePage.tsx` at 745 lines, `Landing.tsx` at 495 lines, `Dashboard.tsx` at 403 lines).
- **SC-002**: After refactor, no single CSS file exceeds 400 lines (current worst: `styles.css` at 1528 lines).
- **SC-003**: Visual diff between pre-refactor and post-refactor Profile page screenshots shows zero pixel differences.
- **SC-004**: Visual diff between pre-refactor and post-refactor Dashboard screenshots shows zero pixel differences.
- **SC-005**: Total bundle size (JS + CSS) is within ±10KB of pre-refactor.
- **SC-006**: `npm run type-check` and `npm run build` both pass with exit 0.
- **SC-007**: A developer can locate any component file by name within 30 seconds (e.g., "where's the avatar upload?" → `web/src/components/profile/AvatarUpload.tsx`).

## Assumptions

- The user accepts that file structure will change and any future debugging must reference the new file names.
- No external API or contract changes; the refactor is internal-only.
- The existing CSS class names (`.dash-*`, `.prof-*`, `.sw-*`) are kept as-is — only the file location changes.
- Helper components that currently live at the bottom of page files (e.g., `Stat`, `SecurityRow` in ProfilePage) move to the `components/profile/` folder.
- The Vite build pipeline supports CSS imports from arbitrary paths without configuration changes.
- `Landing.tsx`, `Login.tsx`, `Signup.tsx`, `ForgotPassword.tsx`, `ResetPassword.tsx` are not in scope for this refactor (they are already small). Future changes can adopt the same pattern.
- The refactor preserves all 16 existing acceptance scenarios from the polish-pass spec (sidebar nav, KPI cards, error bar, profile hero, etc.).
