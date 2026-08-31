# Tier 20 — UI Consistency Polish

**Speckit change ID:** `020-ui-consistency-polish`
## Status

**Status:** BUILDING — Phase A.1 (audit dispatched) + Phase B.1 (Button shipped).
**Started:** 2026-08-31

---

## What's in this folder

| File | Status | Lines | Purpose |
|---|---|---|---|
| `proposal.md` | ✅ done | 350 | Problem + scope + goals + risks + open questions |
| `spec.md` | ✅ done | ~450 | FR + NFR + user stories + acceptance criteria + scenarios + component contracts |
| `plan.md` | ✅ done | ~430 | Architecture + file-by-file changes + 6 phases (A–F) + migration + rollback |
| `tasks.md` | ✅ done | ~280 | Numbered steps (A.1 → F.7) with verification gates per task |
| `checklist.md` | ✅ done | ~430 | Per-feature acceptance (29 ACs + 27 page checklists + final sign-off) |
| `journal.md` | ☐ pending | (rolling) | Session notes as work progresses |

After all 5 planning artifacts approved → **build phase** (code → commit → deploy).
After build verified → **archive** (move whole folder to `.hermes/changes/archive/`).

---

## Quick summary (read this if you're lost)

**Problem:** Frontend was built tier-by-tier over 13 tiers. No consistency
audit was ever run. Initial sampling found 8 specific issues (U1–U8) —
emoji icons, dead code, oversized files, hardcoded colors, drift between
pages.

**Goal:** Make all 27 pages look like one app. Every button, KPI card,
status pill, modal, table, topbar — identical across pages.

**Reference:** Datadog + Stripe + Grafana Cloud aesthetic. Dark theme,
tabular-nums, KPI cards with colored stripes, framer-motion animations.

**Scope:** All files in `web/src/pages/` (27 .tsx) + `web/src/components/`
(78 files, except `proxmox/`) + `web/src/styles/` (14 .css). NO backend
changes. NO new design tokens.

**Effort:** ~18-22 hours (~3-4 working sessions).

**Success:** User opens every page in the dashboard, every page looks
identical-quality, every interactive element behaves identically, every
KPI card has the same shape, mobile (320px) is usable, accessibility
scan passes on every page.

---

## Current state (from project audit)

What's GOOD (don't touch):
- `web/src/styles/tokens.css` + `DESIGN-SYSTEM.md` — design system
- `web/src/components/shared/StatusPill.tsx` — used in 16/27 pages
- `web/src/components/shared/KpiCard.tsx` — used in 16/27 pages
- `web/src/components/icons.tsx` — 30+ SVG icons
- `web/src/components/AppShell.tsx` — framer-motion + mobile drawer + palette

What's BROKEN (must fix):
- U1: Old sidebar uses emoji icons
- U2: Old sidebar is dead code (delete)
- U3: Brand mark is letter "S" (replace with SVG logo)
- U4: 8 topbar sub-components need composition check
- U5: 31 `.bak-*` binaries + 20+ tier*-linux-build clutter
- U6: Working tree dirty
- U7: `.specify/memory/constitution.md` is blank template
- U8: `ProxmoxWorkspace.tsx` 668 LOC (over 400 cap)

---

## Acceptance criteria (full list in `proposal.md` §8)

G1–G11 measurable goals. AC-1 through AC-12 acceptance checks. User signs
off after manually testing every page.

---

## Next step

Awaiting user sign-off on all 5 planning artifacts. After approval →
**BUILD PHASE** (code → commit → verify).

---

**Owner:** StackWatch Engineering
**Last updated:** 2026-08-31