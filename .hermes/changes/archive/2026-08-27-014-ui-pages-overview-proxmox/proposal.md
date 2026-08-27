# Proposal: 014 — UI Page Redesigns (Overview + Proxmox)

## Why

Change 013 shipped the new AppShell (topbar + sidebar + design tokens). The
shell is in place, but pages still render with their old, pre-shell layouts —
they inherit the new topbar/sidebar chrome but their *content* still looks
like the old dashboard. The user said:

  > "the ui/ux is not proper accurate and perfect"
  > "the dashboard like the datadog dashboard board"

Phase 2 of the UI roadmap (per change 013's "follow-on" note) is per-page
redesigns. This change redesigns the two highest-traffic pages — Overview
and Proxmox workspace — so they match the shell's Datadog aesthetic.

## What changes

**Overview page (`/overview`)**
- KPI strip → 4-up Datadog-style cards (Servers up / Hosts online / Alerts firing / Active incidents) — each card has a colored top stripe, big numeric value, sublabel, mini sparkline
- Resource trend section → 3 time-series charts (CPU / Memory / Disk) with hover crosshairs, time-range pills (1h / 6h / 24h / 7d), gradient fills
- Fleet status grid → clickable server tiles sized by CPU%, colored by status
- Recent events → vertical timeline with severity dots + relTime
- Empty states → polished illustrations + CTA

**Proxmox workspace (`/proxmox`) and host detail (`/proxmox-hosts/:id`)**
- Workspace: sectioned layout (Hosts / VMs / LXC / Storage / Network / Tasks) — each section has its own KPI strip + table
- Host detail: tabbed (Summary / VMs / LXC / Storage / Network / Firewall / Tasks)
- VMs table: Datadog-style with per-host sparklines, sortable columns, status pills
- LXC table: same pattern as VMs
- All actions via floating action menu (no inline button sprawl)

**Shared page primitives** (this phase also adds reusable components so we don't rebuild them per page)
- `<KpiCard />` — already exists at `components/shared/KpiCard.tsx` — extend with sparkline slot
- `<TimeSeriesChart />` — NEW — SVG-based, hover crosshair, gradient fill
- `<StatusPill />` — NEW — colored dot + label, sized sm/md/lg
- `<DataTable />` — NEW — sortable, status-aware, responsive (table → cards on mobile)
- `<EmptyState />` — already exists — extend with illustration slot

## Scope guardrails (user-confirmed this turn)

- **Overview + Proxmox only.** Other pages (Billing, Homelab, APM, Logs, etc.)
  keep their current renderers and ship in 015+ — one page per speckit.
- **No new routes.** No new API endpoints. Same backend, better UI.
- **No backend changes.** Pure frontend (web/src/** + web/src/styles/**).
- **Module discipline.** Every new file ≤400 LOC. Every component used in
  ≥2 places (no one-off widgets).
- **Same data sources.** `/api/v1/fleet/summary`, `/api/v1/proxmox/*` — same
  endpoints, better rendering.

## Out of scope (this change)

- Other pages (Billing, APM, Logs, etc.) → 015-ui-pages-billing-etc.
- New features
- Mobile native app (Tier 13 already shipped)
- Backend changes

## Impact

| Area | Impact |
|------|--------|
| Files added | 4-6 new (page redesigns + 3-4 shared primitives) |
| Files modified | `pages/OverviewPage.tsx`, `pages/ProxmoxPage.tsx`, `pages/ProxmoxHostDetailPage.tsx` |
| Breaking | No |
| Risk | Visual regression on the most-trafficked pages. Mitigated by per-phase test plan with screenshot evidence |

## Workflow

- Speckit methodology — proposal + spec + plan + tasks + checklist, archive
- framer-motion for transitions
- ui-ux-pro-max / agent-design-intelligence skills for design patterns
- popular-web-designs (Stripe / Linear / Vercel) for reference
- Per-phase test plan (after Phase A — overview; Phase B — proxmox)
- Module discipline (≤400 LOC per file)

## Acceptance

- Overview page renders Datadog-style KPI strip + 3 charts + fleet grid + timeline
- Proxmox workspace shows sectioned layout with Hosts / VMs / LXC / Storage / Network / Tasks
- Proxmox host detail tabbed with 7 tabs
- All 3 new shared primitives used in ≥2 places
- `npx tsc --noEmit` exit 0
- `npm run build` exit 0
- Live verified at https://stackwatch.smarthomelab.fun/overview and /proxmox
- Lighthouse a11y ≥90

## Rollback

- Revert the single commit "feat(ui): 014 page redesigns — overview + proxmox"
- Old renderers stay in git history
