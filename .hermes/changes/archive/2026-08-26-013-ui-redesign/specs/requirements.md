# Requirements: 013 — UI/UX Redesign (System + Shell)

## Functional requirements

### F1 — Design tokens (CSS custom properties)

- **F1.1** All colors, spacing, typography, elevation, motion durations, easing curves are defined as CSS custom properties in `web/src/styles/tokens.css`
- **F1.2** Color palette is dark-theme-first (no light mode this phase); accents are Tailwind-inspired but extended (primary=cyan, success=green, warning=amber, error=red)
- **F1.3** Spacing scale: 4 / 8 / 12 / 16 / 24 / 32 / 48 / 64 / 96 / 128 (px) — no other values used in component CSS
- **F1.4** Type scale: 11 / 12 / 14 / 16 / 18 / 20 / 24 / 30 / 36 / 48 / 60 (px) — body=14px (was16px, increase density)
- **F1.5** Font stack: system-ui sans for body, JetBrains Mono for tabular numbers (KPI values)
- **F1.6** Border radius scale: 4 / 6 / 8 / 12 / 16 — small/interactive/cards/panels/modals
- **F1.7** Elevation shadows: 4 levels (flat / subtle / moderate / high) defined as `--shadow-{0..3}`
- **F1.8** Motion: durations 100 / 150 / 200 / 300 / 500 (ms), easings `--ease-out / --ease-in / --ease-in-out / --ease-spring`

### F2 — Topbar

- **F2.1** 7 zones from left to right:
  1. Logo + brand ("StackWatch")
  2. Breadcrumb (current section, e.g., "Proxmox / router / VM 100")
  3. Command palette trigger (Cmd+K)
  4. Time widget (date + time, 12h/24h respected)
  5. Weather widget (city + temp + condition, IP-based)
  6. Refresh button (with rotate animation on click)
  7. Notifications bell + Profile avatar menu
- **F2.2** Height: 56px (matches current; reduced from ~72px if it was bigger)
- **F2.3** Sticky to top, blurs on scroll (`backdrop-filter: blur(8px)`)
- **F2.4** All zones have proper hover / focus / active states
- **F2.5** Mobile (<768px): collapses to logo + hamburger + avatar only; full topbar moves to sidebar drawer

### F3 — Sidebar

- **F3.1** Fixed left, width 240px on desktop
- **F3.2** Sections (top to bottom):
  - **Workspace**: Overview, Billing, Homelab, Profile, Settings
  - **Infrastructure**: Proxmox, TrueNAS
  - **Observability**: APM, Logs, RUM, Synthetics, Security, CSPM, CI/CD, Database, Intelligence
  - **Operations**: Incidents, Notebooks, Collaboration, Enterprise, Platform
- **F3.3** Each nav item has icon + label + active stripe (3px left border in accent color)
- **F3.4** Active route uses different background (`--color-surface-elevated`) + accent text
- **F3.5** Hover state: subtle background change + accent text color
- **F3.6** Section headers are uppercase, 11px, letter-spaced, muted color
- **F3.7** Footer: status pill (e.g., "Operational · 1 host online") + version + sign out
- **F3.8** Mobile: becomes slide-in drawer from left, closes on backdrop click or nav selection

### F4 — AppShell (shared layout)

- **F4.1** All routes render inside `<AppShell>` which provides topbar + sidebar + content area
- **F4.2** Content area is `<main>` with proper ARIA roles, scrollable, padding 24px
- **F4.3** Page transitions use framer-motion `AnimatePresence` with fade + 8px slide-up
- **F4.4** Focus management: skip-link to main content (a11y)
- **F4.5** Keyboard nav: Tab cycles focusable elements in logical order; Escape closes any open modals/menus

### F5 — Cleanup

- **F5.1** Remove `web/src/components/shared/handler/` (145 .go files)
- **F5.2** Verify no JS/TS imports reference that directory
- **F5.3** `.gitignore` already excludes backend .go files; no change needed

## Non-functional requirements

- **N1 Performance**: Topbar renders in <16ms; sidebar renders in <16ms (60fps)
- **N2 Accessibility**: Lighthouse a11y score ≥90 on Overview page; all interactive elements have aria-labels
- **N3 Bundle size**: Adding tokens.css + AppShell adds <8KB gzipped
- **N4 Browser support**: Modern Chrome / Edge / Firefox / Safari (last 2 versions); no IE
- **N5 Dark mode**: This phase ships dark-only. Light mode deferred.

## Out of scope (this change)

- Light theme
- Per-page redesign (each page keeps current renderer; just renders inside new shell)
- Backend changes
- New routes
- Data fetching changes
- Mobile-optimized layouts beyond the responsive shell (per-page mobile polish is later)

## Edge cases

- **E1** User navigates to a route that requires login but session expired: redirect to /login, login page renders WITHOUT the shell
- **E2** User opens sidebar drawer on mobile, then resizes to desktop: drawer closes, sidebar shows inline
- **E3** Time widget shows wrong timezone: respect `user.timezone` from `/api/v1/auth/profile` if set, else browser locale
- **E4** Refresh button pressed while data is loading: button is disabled, shows spinner
- **E5** Notifications bell shows 0: just shows bell icon, no badge