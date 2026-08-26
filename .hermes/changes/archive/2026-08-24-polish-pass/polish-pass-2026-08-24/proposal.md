# Proposal: Dashboard + Profile polish pass (2026-08-24)

## Why
User reported three recurring complaints in sequence:
1. "the sidebar is not proper and redundant the profile and billing and settings page already in menu"
2. "polish everything ok"
3. "the profile page is not proper its messy no proper spacing fix this only"

The dashboard had grown visually inconsistent (mixed emoji icons on KPI cards, no color stripes, plain error bar that alarmed users on session expiry). The profile page was missing proper spacing hierarchy and the hero card had no icon affordances.

## What changed

### Sidebar (Dashboard, Settings, Billing pages)
- Removed redundant Profile + Settings links from sidebar nav — they're reachable via the topbar avatar menu
- Kept Overview + Billing on Dashboard sidebar, Overview + Settings + Billing on Settings/Billing sidebars
- Added left accent stripe + cyan-glow background to active nav item
- Hover state now lifts text/icon color, not just background

### KPI cards (Dashboard)
- Replaced emoji icons (◫ ◇ ▶ ♥) with inline SVG icons that match the card tone
- Each card now has a colored vertical stripe on the left (Datadog pattern)
- Bigger icon bubble (32px) using `--accent`/`--indigo`/etc. design tokens
- Bigger number (30px, tabular-nums, -1px tracking)
- Subtle hover lift (translateY -1px + brighter border)
- API health card turns green when status=ok instead of always amber

### Welcome header (Dashboard)
- Eyebrow line with pulsing cyan dot
- h2 uses design-token colors
- Live sync meta is now a small bordered card with a pulsing green dot
- Handles empty `userName` gracefully ("Welcome to StackWatch")

### Error bar (Dashboard)
- New softer `dash-error-info` variant (amber, not red) when error matches `/unauthor|sign.in|token|401/`
- Shows "Sign in" button that clears the token and routes to `/login`
- Icon circle prefix added for both variants

### Profile page
- Hero card padding 28px 32px → 32px 36px
- Email row now has a small circular `@` icon
- New `prof-hero-eyebrow` class with cyan-dot indicator
- Identity rows: 14px vertical padding (was 10px), 22px horizontal (was 20px)
- Workspace stats: 18px 22px padding (was 8px 20px), lighter background
- Display name form: 16px row gap (was 4px), bigger inputs with focus ring
- Form actions row now has a top border separator
- Field labels: uppercase, letter-spacing, `--text-soft` color

### Design system (foundation for all polish)
- Expanded `:root` token set: `--surface-2`, `--surface-3`, `--border-soft`, `--accent-soft`, `--border`, etc.
- Added status-text color tokens: `--green-text`, `--amber-text`, `--red-text`
- New radii: `--radius-sm` (6px), `--radius-lg` (14px)
- New shadows: `--shadow-md`, `--shadow-lg`
- Body now has `-webkit-font-smoothing: antialiased` for crisp text
- Selection color uses `--accent-soft`

## Impact
- Areas affected: `web/src/pages/Dashboard.tsx`, `web/src/pages/BillingPage.tsx`, `web/src/pages/SettingsPage.tsx`, `web/src/pages/ProfilePage.tsx`, `web/src/styles.css`
- Breaking changes: **No**. All changes are visual polish + sidebar nav reduction. Functional behavior is unchanged.
- Migration needed: **No**. Frontend-only changes; backend untouched.

## Commits
- `a533eee` feat(profile): avatar upload + inline API tokens
- `47029aa` polish: sidebar cleanup + KPI cards + welcome header
- `cc10269` polish(profile): fix spacing, hero card, identity rows
