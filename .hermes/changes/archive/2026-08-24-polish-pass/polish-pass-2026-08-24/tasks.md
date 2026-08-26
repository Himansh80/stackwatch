# Tasks: Dashboard + Profile polish

## Done

- [x] Expand `:root` design tokens (surface ramp, accent-soft, status-text, radii, shadows)
- [x] Add `--radius-sm`, `--radius-lg` and use them across components
- [x] Body font-smoothing antialiased
- [x] Sidebar nav: remove Profile + Settings from Dashboard sidebar
- [x] Sidebar nav: remove Profile + Settings from Billing/Settings sidebars (only the relevant page active)
- [x] Sidebar nav: add active accent stripe + cyan glow background
- [x] Sidebar nav: hover state lifts text/icon color
- [x] KPI cards: SVG icons replacing emoji
- [x] KPI cards: colored vertical stripe via `currentColor`
- [x] KPI cards: API health tone changes with status
- [x] KPI cards: bigger icon bubble (32px), tabular-nums value
- [x] KPI cards: hover lift (translateY -1px)
- [x] Welcome header: eyebrow with pulsing cyan dot
- [x] Welcome header: meta card with pulsing green dot
- [x] Welcome header: fallback for empty userName
- [x] Error bar: new `dash-error-info` softer amber variant
- [x] Error bar: auth-error regex detection
- [x] Error bar: Sign-in button that clears token and routes to /login
- [x] Error bar: icon circle prefix for both variants
- [x] Profile hero: padding 32px 36px
- [x] Profile hero: prof-hero-eyebrow class with cyan dot
- [x] Profile hero: prof-hero-email-icon (@ circle prefix)
- [x] Profile identity rows: 14px 22px padding, 16px gap
- [x] Profile identity rows: minmax(0, 1fr) for value column
- [x] Profile workspace stats: 18px 22px padding, lighter background
- [x] Profile display name form: 16px row gap, bigger inputs, focus ring
- [x] Profile form actions: top border separator

## Done criteria

- [x] All tasks complete
- [x] No regressions in 4 verification scenarios (S1, S7, S9, S14)
- [x] Bundle size under budget (CSS 72.4KB < 75KB, JS 315.1KB < 320KB)
- [x] Committed to main with clear messages
- [x] Deployed to .115 with static_server.py running
