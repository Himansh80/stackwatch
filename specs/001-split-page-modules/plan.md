# Implementation Plan: Split page modules

**Feature**: split-page-modules
**Spec**: [spec.md](spec.md)
**Created**: 2026-08-24
**Status**: Ready for tasks

## Approach

Three-phase refactor with verification at each boundary:

1. **Phase 1**: Extract SVG icons + Page sub-components from Dashboard.tsx
2. **Phase 2**: Extract Page sub-components from ProfilePage.tsx
3. **Phase 3**: Split styles.css into per-page modules
4. **Phase 4**: Visual verification + bundle size check

Each phase ends with: build green, type-check green, screenshot diff = 0px, commit.

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Per-page component folders (`components/dashboard/`, `components/profile/`) | Co-location with their consumer; clear ownership | Slightly deeper imports |
| Keep helper components local (`Stat`, `SecurityRow` in `components/profile/`) | Only Profile uses them today; YAGNI until a 2nd consumer | Promote to `components/shared/` when reused |
| Extract shared SVG icons to `components/icons.tsx` | 4 icons used only in MetricCard, but pattern enables future reuse | Single export surface per icon |
| CSS files in `web/src/styles/` (not `components/dashboard/styles.css`) | Co-locate by FEATURE not by component; Profile and Dashboard share `topbar` + `sidebar` rules | Files aren't auto-co-located with components |
| `styles.css` becomes `styles/index.css` re-export | Existing Vite entry doesn't break | Single import per page |
| One commit per phase | Each phase produces a green, screenshot-verified state | 4 commits instead of 1 |
| Use Playwright `compare-images` for visual diff | Catches subtle layout regressions humans miss | Adds Playwright dependency (already installed) |

## Implementation details

### Phase 1: Dashboard extraction (6 files)

**1.1 — Extract shared icons**

Create `web/src/components/icons.tsx`:
```ts
export function ServerIcon() { return <svg>...</svg> }
export function NetworkIcon() { return <svg>...</svg> }
export function PlayIcon() { return <svg>...</svg> }
export function HeartIcon() { return <svg>...</svg> }
```

Modify `web/src/pages/Dashboard.tsx`:
- Remove inline `iconPaths` object
- Import icons from `components/icons.tsx`
- Change `MetricCard` props to use `<ServerIcon />` instead of `icon="hosts"` string lookup

**1.2 — Extract MetricCard**

Create `web/src/components/dashboard/MetricCard.tsx` containing the `MetricCard` function.
Update Dashboard.tsx to import it.

**1.3 — Extract SkeletonCard**

Create `web/src/components/dashboard/SkeletonCard.tsx`.
Update Dashboard.tsx.

**1.4 — Extract TrendChart**

Create `web/src/components/dashboard/TrendChart.tsx`. Move the function and the inline SVG empty state.

**1.5 — Extract HostList (Operations panel content)**

Create `web/src/components/dashboard/HostList.tsx` with the host row rendering.

**1.6 — Extract ErrorBar + WelcomeHeader**

Create `web/src/components/dashboard/ErrorBar.tsx` (the auth-error IIFE branch).
Create `web/src/components/dashboard/WelcomeHeader.tsx`.
Update Dashboard.tsx to be a thin shell: load data, compose components.

**Phase 1 verification:**
- `npm run type-check` exit 0
- `npm run build` exit 0
- Dashboard.tsx < 200 lines
- Screenshot diff = 0px
- Commit: `refactor(dashboard): split Dashboard.tsx into 6 component files`

### Phase 2: ProfilePage extraction (6 files)

**2.1 — Extract AvatarUpload (with file helpers)**

Create `web/src/components/profile/AvatarUpload.tsx`. Includes the `fileToResizedDataUrl` helper (used only here).

**2.2 — Extract HeroCard**

Create `web/src/components/profile/HeroCard.tsx`.

**2.3 — Extract IdentityPanel**

Create `web/src/components/profile/IdentityPanel.tsx` containing the ID rows list + copy logic.

**2.4 — Extract WorkspacePanel + Stat**

Create `web/src/components/profile/WorkspacePanel.tsx` and move `Stat` into it.

**2.5 — Extract NameFormPanel**

Create `web/src/components/profile/NameFormPanel.tsx`.

**2.6 — Extract SecurityPanel + SecurityRow + TokensPanel**

Create `web/src/components/profile/SecurityPanel.tsx` (security list).
Move `SecurityRow` into it.
Create `web/src/components/profile/TokensPanel.tsx` (the inline API tokens panel).

Update ProfilePage.tsx to be a thin shell.

**Phase 2 verification:**
- All checks above
- ProfilePage.tsx < 250 lines (target: ~200)
- Commit: `refactor(profile): split ProfilePage.tsx into 6 component files`

### Phase 3: Split styles.css (5 files)

**3.1 — Extract base tokens + buttons**

Create `web/src/styles/base.css` with:
- `:root` token block
- `*`, `body`, `html` rules
- `.sw-button`, `.sw-button-*`
- `.sw-input`, `.sw-form-*` (shared across all auth/profile/settings)

**3.2 — Extract shared chrome**

Create `web/src/styles/shared.css` with:
- `.dash-app` (the flex root)
- `.dash-sidebar`, `.dash-brand`, `.dash-nav-*`
- `.dash-topbar`, `.dash-greeting`, `.dash-greeting-live`, `.dash-icon-button`, `.dash-live-dot`
- `.dash-topbar-search`, `.dash-topbar-center`, `.dash-topbar-clock`
- `.dash-content`, `.dash-welcome`, `.dash-welcome-meta`, `.dash-welcome-meta-*`
- `.dash-error`, `.dash-error-icon`, `.dash-error-info`
- `.tw-wrapper`, `.dash-topbar-clock-*`, `.dash-topbar-clock-clickable`
- TimeWidget styles (`.tw-*`)

**3.3 — Extract dashboard styles**

Create `web/src/styles/dashboard.css` with all `.dash-metric*`, `.dash-grid-main`, `.dash-panel*`, `.dash-chart-*`, `.dash-host-*`, `.dash-status-*`, `.dash-panel-empty`, `.dash-banner-ok`, `.dash-resource-*`.

**3.4 — Extract profile styles**

Create `web/src/styles/profile.css` with all `.prof-*` rules.

**3.5 — Extract auth styles**

Create `web/src/styles/auth.css` with `.sw-topbar`, `.sw-side*`, `.sw-page*`, `.sw-stat*`, `.sw-layout*`, `.sw-form-grid`, `.sw-input`, `.sw-alert`, `.sw-danger-zone` etc.

**3.6 — Replace styles.css with index re-export**

Create `web/src/styles/index.css` that imports all 5 files in order.

Replace `import '../styles.css'` everywhere with `import '../styles'`.

**Phase 3 verification:**
- All checks above
- styles/index.css < 50 lines (just imports)
- Each individual CSS file < 400 lines
- Dashboard.css + profile.css + auth.css + shared.css + base.css sum to ~1528 (same total)
- Commit: `refactor(styles): split styles.css into per-page modules`

### Phase 4: Final verification

- Take before screenshot of Profile (already done)
- Take before screenshot of Dashboard (already done)
- Apply all 3 phases
- Take after screenshots, diff with Playwright `compare-images`
- Bundle size delta within ±10KB
- All acceptance scenarios S1–S16 from polish-pass spec still pass
- Final commit: `chore(refactor): verify split-page-modules complete`

## File-by-file diff map

### Created (13 new files)

```
web/src/components/icons.tsx
web/src/components/dashboard/MetricCard.tsx
web/src/components/dashboard/SkeletonCard.tsx
web/src/components/dashboard/TrendChart.tsx
web/src/components/dashboard/HostList.tsx
web/src/components/dashboard/WelcomeHeader.tsx
web/src/components/dashboard/ErrorBar.tsx
web/src/components/profile/AvatarUpload.tsx
web/src/components/profile/HeroCard.tsx
web/src/components/profile/IdentityPanel.tsx
web/src/components/profile/WorkspacePanel.tsx
web/src/components/profile/NameFormPanel.tsx
web/src/components/profile/SecurityPanel.tsx
web/src/components/profile/TokensPanel.tsx
web/src/styles/base.css
web/src/styles/dashboard.css
web/src/styles/profile.css
web/src/styles/auth.css
web/src/styles/shared.css
web/src/styles/index.css
```

### Modified

```
web/src/pages/Dashboard.tsx        (403 → ~200 lines)
web/src/pages/ProfilePage.tsx      (745 → ~250 lines)
web/src/pages/SettingsPage.tsx      (import path update)
web/src/pages/BillingPage.tsx       (import path update)
web/src/pages/Login.tsx             (import path update)
web/src/pages/Signup.tsx            (import path update)
web/src/pages/ForgotPassword.tsx    (import path update)
web/src/pages/ResetPassword.tsx     (import path update)
web/src/pages/Landing.tsx           (import path update)
web/src/main.tsx (or App.tsx)       (import path update)
```

### Deleted

```
web/src/styles.css      (1528 lines → split into 5 files)
```

(Replaced by `web/src/styles/index.css` which imports the 5 new files.)

## Risks

| Risk | Mitigation |
|------|------------|
| Visual regression in any page | Pixel-diff before/after with Playwright `compare-images` |
| Build breaks from missing CSS imports | Each phase builds + type-checks before next phase |
| Bundle size growth | CSS is loaded synchronously per page import; minifier handles dedup |
| Lost code during extraction | Each phase: read entire file → extract → verify count of unique lines |
| Helper components (Stat, SecurityRow) only used in Profile — premature extraction? | Keep them in `components/profile/`; promote to `components/shared/` on 2nd consumer |
| Diff noise from long UUIDs / timestamps in diff output | Use `git diff --stat` for human-readable summary |

## Verification strategy

1. **Per phase**:
   - `npm run type-check` exit 0
   - `npm run build` exit 0
   - Capture screenshot of Profile + Dashboard
   - Compare with Playwright `compare-images` to pre-refactor screenshot
   - One git commit per phase

2. **Final**:
   - All 16 acceptance scenarios from polish-pass spec still pass (manual smoke test)
   - Bundle size within ±10KB
   - No file > 250 lines for page components, > 400 for CSS
   - All components findable by name within 30 seconds

## Open questions

None — all defaults documented in spec Assumptions. User approved the spec at message #1.
