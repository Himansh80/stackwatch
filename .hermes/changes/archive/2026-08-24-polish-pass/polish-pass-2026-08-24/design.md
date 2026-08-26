# Design: Dashboard + Profile polish

## Approach

Three-tier refactor-light polish:

### Tier 1: Design tokens (foundation)
Expand `:root` in `styles.css` with a full color/radius/shadow scale. Every subsequent change references these tokens, never raw hex.

### Tier 2: Component-level polish
- Replace inline emoji icons with inline SVG components
- Add colored accent stripes on KPI cards
- Promote the welcome header to a proper card-style meta block

### Tier 3: Spacing + typography pass
- Standardize padding to 32px / 22px / 18px / 14px / 10px ladder
- Replace hardcoded `color:#7387a5` with `var(--muted)`
- Add `font-variant-numeric: tabular-nums` to all numeric displays

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Use inline SVGs (not icon library) | Zero new dependency, fully styleable via `currentColor` | More JSX in components |
| Add `--surface-2`, `--surface-3` tokens | Multi-step surface ramp enables better card hierarchy | Slightly more tokens to maintain |
| Remove Profile/Settings from sidebar | They were duplicate entries with the avatar menu | Users have to use avatar menu to reach them |
| Use `min-width: 0` on flex children | Prevents grid blowout when long IDs don't wrap | Subtle behavior change vs default |
| Auth-error regex is case-insensitive | Matches "Unauthorized", "Token expired", "401" etc. | False positives if any error contains "sign" |

## Implementation details

### File changes
- `web/src/styles.css` — token expansion, sidebar nav, KPI cards, welcome header, error bar, profile hero/identity/workspace/form polish
- `web/src/pages/Dashboard.tsx` — error-bar JSX with auth-error branch, MetricCard with SVG icons, welcome-meta restructure
- `web/src/pages/BillingPage.tsx` — sidebar Profile/Settings removal
- `web/src/pages/SettingsPage.tsx` — sidebar Profile removal
- `web/src/pages/ProfilePage.tsx` — prof-hero-eyebrow + email-icon + workspace stats spacing

### Patterns used
- Inline SVG with `stroke="currentColor"` so each icon takes the card's tone color
- CSS `currentColor` for stripe backgrounds on `.dash-metric-*`
- `@keyframes sw-live-pulse` for the welcome-meta dot pulse
- IIFE in JSX (`{error && (() => { ... })()}`) to express the conditional error-bar branches without extracting a component
- `font-variant-numeric: tabular-nums` on every numeric value to prevent layout jitter on updates

### Known limitations
- The error-bar IIFE is inline JSX; a future refactor should extract it as `<ErrorBar error={...} />`
- KPI card SVG icons are duplicated in JSX; should be moved to `components/icons.tsx` in the modularity refactor
- Profile page still has helper components (`Stat`, `SecurityRow`) inline at the bottom of the file
- `styles.css` is now ~1500 lines; should be split into `base.css`, `dashboard.css`, `profile.css`, etc.

## Risks

| Risk | Mitigation |
|------|------------|
| Removing sidebar items confuses users | Avatar menu clearly shows "Profile" + "Settings" links |
| Auth-error regex false positive | Conservative pattern; only fires when error message contains auth keywords |
| New SVGs cause bundle bloat | Each SVG < 200 bytes; well under budget |
| Large padding changes break narrow viewports | Added `@media (max-width: 1100px)` overrides where needed |
