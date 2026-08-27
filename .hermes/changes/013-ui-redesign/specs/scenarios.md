# Scenarios: 013 — UI/UX Redesign (System + Shell)

## S1 — Design tokens resolve correctly

**Given** any component imports the design system
**When** a user opens the app in any modern browser
**Then** every CSS variable (`var(--color-surface)` etc.) resolves to a real color value
**And** the page renders with the dark theme

**Verification:** open DevTools, inspect any element, see the computed color values match the tokens.

## S2 — Topbar shows all 7 zones

**Given** user is logged in and on the Overview page
**When** the page loads
**Then** the topbar shows: logo, breadcrumb (Overview), command palette trigger, time widget, weather widget, refresh button, notifications bell, profile avatar (8 elements total — breadcrumb is required, command palette is the search trigger,)

**Verification:** screenshot inspection.

## S3 — Sidebar shows all 21 nav items in 4 sections

**Given** user is logged in
**When** the sidebar renders
**Then** 4 sections are visible: Workspace (5), Infrastructure (2), Observability (9), Operations (5)
**And** the current route has an accent stripe on the left and elevated background

**Verification:** navigate to each route, confirm active state moves.

## S4 — Page transitions animate

**Given** user is on the Overview page
**When** user clicks "Proxmox" in the sidebar
**Then** Overview fades + slides up, Proxmox fades + slides in
**And** the transition takes ~200ms

**Verification:** visual inspection during navigation.

## S5 — Mobile shell collapses correctly

**Given** viewport width is <768px
**When** user loads the page
**Then** sidebar is hidden by default, topbar shows only logo + hamburger + avatar
**When** user taps hamburger
**Then** sidebar slides in from left, backdrop appears
**When** user taps a nav item
**Then** sidebar closes, page navigates

**Verification:** resize browser to mobile width.

## S6 — All existing pages still render

**Given** the new AppShell wraps all routes in App.tsx
**When** user navigates to any of 21 routes
**Then** the page renders inside the shell without errors
**And** the page's existing renderer is unchanged

**Verification:** click through every nav item, confirm no console errors and content loads.

## S7 — `.go` files removed from frontend tree

**Given** the cleanup task
**When** `git rm -rf web/src/components/shared/handler/` runs
**Then** `ls web/src/components/shared/handler/` returns "No such file or directory"
**And** `grep -r "shared/handler" web/src/` returns no matches
**And** `tsc --noEmit` still exits 0

**Verification:** shell commands.

## S8 — Keyboard navigation works

**Given** user is on any page
**When** user presses Tab repeatedly
**Then** focus moves through interactive elements in logical order: sidebar items, topbar elements, page content

**Verification:** keyboard test.

## S9 — Refresh button animates

**Given** user is on Overview
**When** user clicks the refresh button
**Then** button shows rotate animation (~500ms)
**And** page data refreshes

**Verification:** visual + click test.

## S10 — Accessibility baseline

**Given** any page
**When** Lighthouse accessibility audit runs
**Then** score is ≥90

**Verification:** `npx lighthouse http://localhost:5180/overview --only-categories=accessibility` (or browser DevTools Lighthouse tab).