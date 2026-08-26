# Requirements: Dashboard + Profile polish

## Functional

### F1. Sidebar nav
- F1.1 The sidebar on `/dashboard` must show: Overview, Billing, Proxmox, TrueNAS, Terminal
- F1.2 The sidebar on `/settings` must show: Overview, Settings, Billing
- F1.3 The sidebar on `/billing` must show: Overview, Billing
- F1.4 The sidebar on `/profile` must continue to show: Profile, Billing, Settings (Profile page keeps its own nav for context)
- F1.5 Removing a sidebar item must not remove access — the same destination must be reachable via another route (avatar menu, topbar search, etc.)

### F2. Dashboard KPI cards
- F2.1 Each card must render with an SVG icon that visually matches the metric meaning
- F2.2 Each card must show a colored vertical stripe on the left matching its tone
- F2.3 The "API health" card tone must be green when `health.status === "ok"`, amber otherwise
- F2.4 The card value must use tabular-nums (so numbers don't visually jitter as they update)
- F2.5 The card must have a subtle hover state (translateY -1px + brighter border)

### F3. Welcome header
- F3.1 Eyebrow text + pulsing cyan dot must display on top
- F3.2 Main heading must adapt to whether `userName` is loaded (use "Welcome to StackWatch" fallback if not)
- F3.3 Live sync status indicator must pulse to show the connection is live

### F4. Error bar
- F4.1 When error message matches `/unauthor|sign.in|token|401/i`, render a softer amber bar with "Sign in" button (instead of loud red "Retry")
- F4.2 The "Sign in" button must clear the token (via `clearToken()`) and route to `/login`
- F4.3 Non-auth errors must continue to render the red error bar with "Retry" button
- F4.4 Both variants must show a circular icon prefix

### F5. Profile page hero
- F5.1 Hero card must render with 32px 36px padding
- F5.2 Email row must have a small circular `@` icon prefix
- F5.3 Eyebrow line "Your account" must appear with cyan dot

### F6. Profile page identity rows
- F6.1 Each row must have 14px vertical padding, 22px horizontal padding
- F6.2 Each row value must truncate with ellipsis if it overflows
- F6.3 Each row must have a copy button that flashes "✓ Copied" for ~1.6s after click

### F7. Profile page workspace stats
- F7.1 The 4-stat grid (Plan / Workspace / Status / Role) must have 18px 22px padding
- F7.2 Stats must use the design-token surface colors so they stand out from the panel background

### F8. Display name form
- F8.1 The form grid must use 16px row gap (was 4px)
- F8.2 Inputs must have a focus ring using `--accent-soft`
- F8.3 Form actions (Discard / Save) must have a top border separator above them

## Non-functional

### N1. Performance
- N1.1 The new SVG icons must not add more than 2KB to the CSS bundle
- N1.2 The `dash-welcome-meta-dot` pulse animation must use `transform` and `opacity` only (no layout thrash)

### N2. Accessibility
- N2.1 The new pulsing dots must include `aria-hidden="true"` so they're not announced
- N2.2 The Sign-in button must have visible focus styling matching the rest of the dashboard

### N3. Theme consistency
- N3.1 All new colors must use design tokens (`--accent`, `--green`, `--amber`, `--border-soft`, etc.) — no hex literals in components
- N3.2 Border radius must use `--radius`, `--radius-sm`, or `--radius-lg` — no magic numbers
