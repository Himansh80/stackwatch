# Feature Specification: Polish — Motion + Datadog-grade UI

**Feature Branch**: `002-polish-motion-datadog-ui`
**Created**: 2026-08-24
**Status**: Draft
**Source change folder**: `.hermes/changes/002-polish-motion-datadog-ui/`

**Input**: User instruction "make UI/UX better, better animations, use framer motion, my touch but identical to Datadog and other platforms". All features/tools/exact parity with Datadog + Grafana + Better Stack + Stripe + Linear.

## Goal

Bring the existing 12 dashboard pages to ** Datadog/Grafana visual quality ** with ** tasteful framer-motion everywhere** , without changing behavior. Tier 7+ features inherit this polish automatically because the components live in `web/src/components/shared/` and `web/src/lib/motion.tsx`.

## Scope (this change)

Polish the **currently-shipped** stack of pages and shared components. No new backend endpoints. No Tier 7 features (those are 003+ changes).

### In scope

1. **Motion gap**: 5 of 12 pages have ZERO framer-motion (Dashboard, ProfilePage, SettingsPage, BillingPage, TrueNASWorkspace). Landing/Login/Signup/Forgot/Reset already use motion. Close the gap.
2. **Visual polish**: anti-slop pass on every page — placeholder text, stock photos, contrast, hover states, loading/empty/error states, micro-interactions.
3. **Datadog visual parity**: apply Datadog conventions for KPI strip, status pills, hosts table, time-series charts, command palette. Reference the screenshots stored in `screenshots/datadog-ref/` if present, otherwise use the design tokens in `web/src/styles/base.css` plus `agent-design-intelligence` skill heuristics.
4. **Shared motion library**: extend `web/src/lib/motion.tsx` with Datadog-grade variants (kpiEnter, statusPulse, tableRowStagger, chartReveal) so all pages stay in sync.
5. **Bundle stays ≤ baseline + 60KB gzipped** — motion is JS overhead; budget matters.
6. **All existing functionality preserved** — speckit spec `001-split-page-modules` acceptance scenarios must still pass.

### Out of scope (will be separate speckit changes)

- Tier 7 Datadog parity features (APM, full RUM, Security, CSPM, CI/CD Vis, DB Monitoring, Service Mgmt, Notebook, Team Collab)
- New pages (those ship with Tier 7)
- Backend API changes
- Mobile / RN
- Auth flow changes

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Page feels alive (Priority: P1)

A user opens Dashboard. KPI cards fade-in with a 50ms stagger, the live pulse dot breathes, sparkline charts draw on mount, the topbar search button is keyboard-accessible with a focus ring. No element pops in instantly.

**Why P1**: First impression. Datadog's dashboard is the gold standard — every micro-interaction is tuned. We need the same here.

**Independent test**: Open Dashboard after fresh login; record a 2-second screen capture; verify (a) cards animate in not pop, (b) the live dot is breathing, (c) `prefers-reduced-motion: reduce` zeroes all transitions.

**Acceptance scenarios**:

1. **Given** a user on Dashboard, **when** the page mounts, **then** the 4 KPI cards enter with `kpiEnter` variant (opacity 0→1, y 8→0, 320ms ease-out, 50ms stagger).
2. **Given** the same user, **when** they look at the live-sync pill, **then** the dot breathes (1.8s loop) — same animation as Landing's `LiveDot`.
3. **Given** the user has `prefers-reduced-motion: reduce` set, **when** they load any page, **then** all motion has duration 0 and the page renders instantly with no animation.

### User Story 2 — Datadog-grade KPI strip (Priority: P1)

The KPI cards on Dashboard, the host-count strip on Profile, the metric tiles on Settings/Billing/TrueNAS workspace — all follow the same visual contract: rounded card, 1px border, hover lift (`translateY(-1px)` + shadow L2), top-edge accent stripe color-coded by metric type (cyan=primary, green=ok, amber=warn, red=crit).

**Why P1**: Consistency across all surfaces is the single biggest visual quality signal.

**Independent test**: Render Dashboard + Profile + Settings in Playwright side-by-side; assert all KPI cards share 4 design tokens (radius, border-color, hover-lift, top-stripe).

**Acceptance scenarios**:

1. **Given** any KPI card on any page, **when** inspected, **then** it uses `--radius`, `--border`, `cardLift.whileHover`, and a top-stripe whose color is bound to a metric status via `--green` / `--amber` / `--red` / `--accent`.
2. **Given** the same KPI card, **when** hovered, **then** it lifts to `translateY(-1px)` with shadow L2 within 150ms.
3. **Given** the same KPI card, **when** it represents a "critical" metric, **then** the top stripe is `--red` not `--green`.

### User Story 3 — Hosts table = Datadog hosts table (Priority: P2)

The host list (Dashboard `HostList`, TrueNASWorkspace, Proxmox workspace — wherever it exists) shows: status pill (UP/STALE/DOWN), hostname (mono font), IP, OS pill, sparkline mini-chart per row, last-seen relative timestamp, row hover highlight.

**Why P2**: Hosts table is the most-used Datadog surface — operators glance at it hundreds of times a day. Polishing this is high-leverage.

**Independent test**: Compare `HostList` rendering to Datadog's Infrastructure → Hosts list. Assert visual parity on 5 axes (status pill, font, sparkline column, row hover, last-seen format).

**Acceptance scenarios**:

1. **Given** the hosts table on Dashboard, **when** rendered, **then** each row has columns: Status (pills), Hostname (mono `JetBrains Mono`), IP (mono), OS (pill), Sparkline (24px wide, 48px tall, no axis), Last seen (relative "12s ago" format), Actions (kebab `⋮`).
2. **Given** the same row, **when** the host is DOWN, **then** the status pill is `--red-text` with `--red-soft` background and a pulsing 1.5s dot.
4. **Given** the same row, **when** hovered, **then** background changes to `--surface-2` and a left-edge accent stripe appears in the status color.

### User Story 4 — Command palette = Spotlight/Linear-quality (Priority: P2)

The existing `CommandPalette.tsx` (139 LOC) gets framer-motion entrance/exit, keyboard navigation highlights, and group dividers — matching Linear's ⌘K.

**Why P2**: Power-user feature; if it's polished it becomes a daily driver.

**Acceptance scenarios**:

1. **Given** the user presses ⌘K, **when** the palette opens, **then** it fades + scales from 0.98 to 1.0 over 180ms with backdrop blur.
2. **Given** the user types in the palette, **when** results filter, **then** each result row enters with `fadeUp` and exits with `fadeOut`, with AnimatePresence handling the reorder.

### User Story 5 — Empty / Loading / Error states look designed (Priority: P2)

Every page that fetches data has explicit Empty, Loading, Error states with illustrations (Lucide-style inline SVG, not stock photos), one CTA, and consistent spacing.

**Why P2**: Datadog's empty states are designed — most "AI slop" dashboards have plain "No data" text.

**Acceptance scenarios**:

1. **Given** Dashboard with 0 servers registered, **when** it renders, **then** the host list area shows an empty state with: illustration, headline "No servers yet", subhead "Install the agent on your first server", primary CTA button "Get install command" (which calls the existing endpoint).
2. **Given** the page is loading, **when** it renders, **then** skeleton cards use the existing `SkeletonCard` component with a shimmer animation (1.4s loop).
3. **Given** the page errors (e.g. 500), **when** it renders, **then** a friendly error bar with retry button appears — no raw error stack exposed.

### User Story 6 — Build + bundle + type-check stay healthy (Priority: P1)

The polish pass must NOT inflate the bundle beyond budget.

**Acceptance scenarios**:

1. **Given** all polish work merged, **when** `npm run build` runs, **then** it exits 0.
2. **Given** the same, **when** bundle is measured, **then** JS gzipped is ≤ (baseline + 60KB), CSS gzipped is unchanged.
3. **Given** the same, **when** `npm run type-check` runs, **then** it exits 0.
4. **Given** the same, **when** `npm run lint` runs, **then** it exits 0.

## Design system additions (proposed)

### Tokens added to `web/src/styles/base.css`

```css
/* Page transition — fade + lift between routes */
--motion-page: 240ms cubic-bezier(0.16, 1, 0.3, 1);

/* Status pill colors (already exist as `--green`, `--amber`, `--red`) */
--status-up: var(--green);
--status-stale: var(--amber);
--status-down: var(--red);

/* Card elevation tokens (already in base.css — confirm) */
--elev-1: 0 1px 0 rgba(0,0,0,.4), 0 1px 2px rgba(0,0,0,.3);
--elev-2: 0 4px 12px rgba(0,0,0,.45), 0 2px 4px rgba(0,0,0,.3);

/* Sparkline + chart line widths */
--chart-stroke: 1.5px;
```

### New motion exports in `web/src/lib/motion.tsx`

```ts
// KPI strip entrance — Datadog-style stagger
export const kpiEnter: Variants = {
  hidden: { opacity: 0, y: 8 },
  show:   { opacity: 1, y: 0, transition: { duration: 0.32, ease: EASE_OUT } },
};
export const kpiStagger: Variants = stagger(0.05, 0.06);

// Status pulse — for "DOWN" / "CRIT" pills
export const statusPulse = {
  animate: { opacity: [1, 0.55, 1] },
  transition: { duration: 1.5, repeat: Infinity, ease: 'easeInOut' },
};

// Sparkline draw — stroke-dashoffset animation on mount
export const sparklineDraw: Variants = {
  hidden: { pathLength: 0, opacity: 0 },
  show:   { pathLength: 1, opacity: 1, transition: { duration: 0.6, ease: EASE_OUT } },
};

// Page enter — used by every route
export const pageEnter: Variants = {
  hidden: { opacity: 0, y: 4 },
  show:   { opacity: 1, y: 0, transition: { duration: 0.24, ease: EASE_OUT } },
};

// Command palette
export const paletteEnter: Variants = {
  hidden: { opacity: 0, scale: 0.98 },
  show:   { opacity: 1, scale: 1, transition: { duration: 0.18, ease: EASE_OUT } },
  exit:   { opacity: 0, scale: 0.98, transition: { duration: 0.12 } },
};

// Skeleton shimmer
export const shimmer: Variants = {
  animate: { backgroundPosition: ['0% 50%', '100% 50%', '0% 50%'] },
  transition: { duration: 1.4, repeat: Infinity, ease: 'linear' },
};
```

### Component additions

- `web/src/components/shared/KpiCard.tsx` — shared KPI tile, takes `label`, `value`, `delta`, `icon`, `status` (up/down/stale/crit). All pages migrate to this.
- `web/src/components/shared/StatusPill.tsx` — promoted from `components/dashboard/StatusPill.tsx` to `shared/`. Used by Dashboard, Profile, Settings, Billing, TrueNASWorkspace, Proxmox workspace.
- `web/src/components/shared/EmptyState.tsx` — takes `illustration`, `headline`, `subhead`, `cta`. Used by all list pages.
- `web/src/components/shared/SkeletonRow.tsx` — for hosts table loading.

## Files this change modifies (estimated)

- `web/src/lib/motion.tsx` — add 6 new exports
- `web/src/styles/base.css` — add 1 token block (~12 lines)
- `web/src/pages/Dashboard.tsx` — wrap with pageEnter, kpiStagger; migrate MetricCard to shared/KpiCard
- `web/src/pages/ProfilePage.tsx` — wrap with pageEnter; add HeroCard motion; fadeUp on stat panels
- `web/src/pages/SettingsPage.tsx` — wrap with pageEnter; stagger sections
- `web/src/pages/BillingPage.tsx` — wrap with pageEnter; animate plan card
- `web/src/pages/TrueNASWorkspace.tsx` — wrap with pageEnter; add StatusPill motion
- `web/src/pages/Landing.tsx` — keep existing motion, refine to match new tokens (no logic change)
- `web/src/components/CommandPalette.tsx` — add paletteEnter/Exit; AnimatePresence on results
- `web/src/components/AppSidebar.tsx` — add stagger on nav items
- `web/src/components/TimeWidget.tsx` — fade-in on tick
- `web/src/components/dashboard/StatusPill.tsx` — promote to shared/
- New: `web/src/components/shared/KpiCard.tsx`
- New: `web/src/components/shared/EmptyState.tsx`
- New: `web/src/components/shared/SkeletonRow.tsx`

**Total**: ~13 files modified, 3 new files, no new dependencies (already installed).

## Risks

| Risk | Mitigation |
|------|------------|
| Bundle bloat from framer-motion in every component | Use shared `motion.div` wrapper; lazy-import framer-motion in shared components only; tree-shake via named imports |
| Reduced-motion users get jarring experience | Centralize `useReducedMotion` in `motion.tsx`; every variant honors it via `useReducedMotion()` short-circuit |
| Visual regressions break existing flows | Playwright pixel-diff before/after every page; speckit 001-split-page-modules acceptance scenarios must still pass |
| Animation fights with the live data updates | Use `AnimatePresence mode="popLayout"` for list items; KPI cards don't re-animate on data refresh (mount-only) |

## Done criteria

- [ ] All 6 user stories pass acceptance criteria
- [ ] Playwright pixel-diff Dashboard vs previous build = 0px (other than intentional motion)
- [ ] Playwright pixel-diff Profile vs previous build = 0px
- [ ] Bundle JS gzipped ≤ baseline + 60KB
- [ ] Bundle CSS gzipped unchanged
- [ ] `npm run type-check` exit 0
- [ ] `npm run build` exit 0
- [ ] `npm run lint` exit 0
- [ ] prefers-reduced-motion: zero animations verified via Playwright emulate
- [ ] Deployed to .115, verifier 4× back-to-back PASS
- [ ] Archive to `.hermes/changes/archive/2026-08-24-002-polish-motion-datadog-ui/`