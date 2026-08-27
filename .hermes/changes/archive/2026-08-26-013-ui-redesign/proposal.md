# Proposal: 013 — UI/UX Redesign (System + Shell)

## Why

StackWatch's current UI is functional but visually inconsistent:
- No formal design system — colors, spacing, type are scattered across `styles/*.css`
- 145 `.go` files accidentally live in `web/src/components/shared/handler/` (backend pollution)
- Topbar / sidebar / pages built ad-hoc over 14 tiers — no shared tokens, no shared components
- User feedback: "the ui/ux is not proper accurate and perfect"

This redesign establishes a real design system so all30+ pages can be rebuilt consistently in follow-on phases.

## What changes

**Phase 0 — Design System** (this phase)
- Create `web/src/styles/tokens.css` — CSS variables for color, spacing, type, elevation, motion
- Create `web/src/styles/base.css` overhaul — reset, typography, scrollbars, focus rings, dark-only theme
- Document the design language in `web/src/styles/DESIGN-SYSTEM.md`

**Phase 1 — Shell** (this phase)
- Rebuild `web/src/components/AppShell.tsx` — topbar + sidebar + content area as a single shared layout
- New `web/src/components/topbar/*` — logo, breadcrumb, search/command, time widget, refresh, notifications, profile menu
- New `web/src/components/sidebar/*` — sectioned nav with active state, grouped by domain (Workspace / Infrastructure / Observability / Operations)
- `App.tsx` updated to wrap all routes in `<AppShell>`
- Animations via framer-motion (page enter, sidebar slide, topbar fade)

**Cleanup** (this phase)
- `git rm -rf web/src/components/shared/handler/` — remove 145 .go files from frontend tree

## Scope guardrails (user-confirmed 2026-08-26)

- **Phase 0 + Phase 1 ONLY.** No page redesigns in this change. Pages will use the new shell but keep their existing internal layout (verified to render correctly through the shell).
- Follow-up changes (`014-ui-page-overview`, `015-ui-page-proxmox`, ...) will redesign pages one at a time.

## Out of scope (this change)

- No new routes
- No backend changes
- No data fetching changes
- No per-page redesigns (each page keeps its current renderer)

## Impact

| Area | Impact |
|------|--------|
| Files added | 8-12 new (tokens.css, DESIGN-SYSTEM.md, AppShell.tsx, topbar/*, sidebar/*) |
| Files removed | 145 (.go in shared/handler/) |
| Files modified | App.tsx, all page imports (route inside AppShell wrapper) |
| Breaking | No — pages render through new shell but their own JSX unchanged |
| Risk | Sidebar / topbar visual regression if tokens are off. Mitigated by per-phase test plan. |

## Workflow

- **Speckit methodology** — full proposal/spec/plan/tasks/checklist, archive after complete
- **Design intelligence** — `agent-design-intelligence` skill as the aesthetic bible (110K★ taste-skill source)
- **Popular web designs** — `popular-web-designs` skill for reference (Stripe, Linear, Vercel patterns)
- **Ruflo** — installed for future page-redesign phases (not used this phase; one design system to keep coherent)
- **framer-motion** — every transition is animated (universal standard)
- **Per-phase test plan** — at end of Phase 0 and Phase 1
- **Modular file structure** — every new file ≤400 LOC

## Acceptance

- Design tokens visible in DevTools (e.g., `var(--color-surface)`) and used by ≥10 components
- Sidebar matches Datadog aesthetic: dark, sectioned nav, active item has accent stripe
- Topbar has all 7 zones: logo, breadcrumb, search, time, weather, refresh, notifications, profile
- `npm run type-check` exit 0
- `npx tsc --noEmit` exit 0
- 100% of existing routes still render through the new shell (no broken pages)
- Lighthouse accessibility score ≥90 on Overview page

## Rollback

- Phase 1 changes live in `web/src/components/AppShell.tsx` + new `topbar/`, `sidebar/` dirs
- Reverting means removing AppShell wrapper in App.tsx + restoring old AppSidebar/TopBar imports
- All changes committed as a single commit per phase (easy to revert one phase at a time)