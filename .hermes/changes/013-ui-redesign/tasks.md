# Tasks: 013 — UI/UX Redesign (System + Shell)

## Phase 0 — Design System

- [ ] 0.1 Read existing CSS to understand current color/spacing patterns (audit, not migration)
- [ ] 0.2 Create `web/src/styles/tokens.css` — all CSS variables
- [ ] 0.3 Rewrite `web/src/styles/base.css` — reset, typography, scrollbars, focus, dark theme defaults
- [ ] 0.4 Create `web/src/styles/DESIGN-SYSTEM.md` — human-readable design language doc
- [ ] 0.5 Import tokens.css + base.css in `web/src/index.html` (or main.tsx)
- [ ] 0.6 **GATE**: user reviews DESIGN-SYSTEM.md + screenshots sample components using tokens

**Done criteria:**
- All 4 sub-tasks complete
- `tsc --noEmit` exit 0
- 1 sample component (e.g., a styled `<div>`) demonstrates 5+ tokens

## Phase 1 — Shell

- [ ] 1.1 Create `nav-config.ts` — typed nav structure (4 sections, 21 items)
- [ ] 1.2 Create `components/sidebar/SidebarItem.tsx`
- [ ] 1.3 Create `components/sidebar/SidebarSection.tsx`
- [ ] 1.4 Create `components/sidebar/Sidebar.tsx` (orchestrator)
- [ ] 1.5 Create `components/sidebar/SidebarFooter.tsx`
- [ ] 1.6 Create `components/topbar/TopBarLogo.tsx`
- [ ] 1.7 Create `components/topbar/TopBarBreadcrumb.tsx`
- [ ] 1.8 Create `components/topbar/TopBarSearch.tsx` (command palette trigger)
- [ ] 1.9 Create `components/topbar/TopBarTime.tsx` (wraps existing TimeWidget)
- [ ] 1.10 Create `components/topbar/TopBarWeather.tsx` (wraps existing weather widget)
- [ ] 1.11 Create `components/topbar/TopBarRefresh.tsx`
- [ ] 1.12 Create `components/topbar/TopBarNotifications.tsx`
- [ ] 1.13 Create `components/topbar/TopBarProfile.tsx` (wraps existing ProfileMenu)
- [ ] 1.14 Create `components/topbar/TopBar.tsx` (orchestrator)
- [ ] 1.15 Create `components/AppShell.tsx`
- [ ] 1.16 Modify `App.tsx` — wrap routes in `<AppShell>`
- [ ] 1.17 Add CSS for shell in `web/src/styles/base.css` (or new `web/src/styles/shell.css`)
- [ ] 1.18 Add mobile drawer logic (hamburger toggle, slide-in, backdrop)
- [ ] 1.19 Add skip-link to main content
- [ ] 1.20 Add framer-motion page transitions
- [ ] 1.21 Delete `components/AppSidebar.tsx` (replaced by sidebar/)
- [ ] 1.22 **GATE**: user visually reviews all 21 routes through new shell, reports any regressions

**Done criteria:**
- All 21 sub-tasks complete
- Every existing route renders inside new shell without errors
- Sidebar shows correct active state per route
- Topbar shows all 7 zones
- Mobile (<768px) shows collapsed shell with hamburger
- Page transitions animate
- `tsc --noEmit` exit 0
- Lighthouse a11y ≥90 on Overview

## Phase 2 — Cleanup

- [ ] 2.1 `git rm -rf web/src/components/shared/handler/` — remove 145 .go files
- [ ] 2.2 Verify no imports reference `shared/handler`
- [ ] 2.3 Run `tsc --noEmit` to confirm nothing breaks
- [ ] 2.4 Commit: "chore(frontend): remove 145 backend .go files from shared/handler"
- [ ] 2.5 **GATE**: user reviews commit, approves final state

**Done criteria:**
- `ls web/src/components/shared/handler/` → not found
- `grep -r "shared/handler" web/src/` → no matches
- tsc clean

## Final acceptance

- [ ] All Phase 0 + Phase 1 + Phase 2 tasks complete
- [ ] Per-phase test plans reviewed + accepted by user
- [ ] Final mega-test: navigate through all 21 routes, no regressions, shell looks like Datadog
- [ ] Archive change folder to `.hermes/changes/archive/2026-08-26-013-ui-redesign/`
- [ ] Update `journal-ui-redesign.md` with final summary
- [ ] Update skill `stackwatch-build-deploy` with any new gotchas

## Out of scope (deferred to follow-on changes)

- 014-ui-page-overview (Dashboard page redesign)
- 015-ui-page-proxmox (Proxmox section redesign, 15 pages)
- 016-ui-page-truenas (TrueNAS redesign)
- 017-ui-page-observability (APM, Logs, RUM, Synthetics redesign)
- 018-ui-page-operations (Incidents, Notebooks, etc.)
- 019-ui-page-billing-profile-settings (Settings section)
- 020-ui-page-landing (Landing page redesign)
- 021-ui-page-auth (Login, Signup, Forgot, Reset)
- 022-light-mode (theme switcher + light tokens)

Each will be its own speckit change with its own proposal/spec/plan/tasks/test-plan.