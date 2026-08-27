# Proposal: 015 — Apply design primitives to all remaining pages

## Why

Change 014 shipped the new shared primitives — `StatusPill`, `TimeSeriesChart`, extended `KpiCard` (with `trend` + `onClick`) — and used them on Overview + Proxmox. The other 15 pages still render with their old per-page status badges, ad-hoc charts, and non-clickable KPI cards.

This change applies the design system primitives to all remaining pages so the entire app feels like one cohesive Datadog-style product instead of two well-designed pages next to thirteen legacy ones.

## What changes

For every page in `web/src/pages/`, swap any of the following for the shared primitive:

| Pattern | Replace with |
|---------|--------------|
| `<span className="dash-status ...">running</span>` | `<StatusPill status="up" />` |
| `<div className="px-status px-status-green">running</div>` | `<StatusPill status="up" />` |
| Inline `<svg>` line chart | `<TimeSeriesChart values={...} />` |
| Non-clickable KPI card | `<KpiCard onClick={...} />` |
| Hardcoded status colors (`#10b981` etc.) | Token class (`--green` etc.) |

## Pages (in order of traffic)

| # | Page | Current status | Phase |
|---|------|----------------|-------|
| 1 | BillingPage | mostly polished | Phase A |
| 2 | HomelabPage | tab dispatcher | Phase A |
| 3 | ApmPage + ApmServicePage | list + detail | Phase B |
| 4 | LogsFullPage | tab dispatcher | Phase B |
| 5 | RumFullPage + RumSessionPage | list + session | Phase B |
| 6 | SyntheticsPage | status + checks | Phase C |
| 7 | SecurityPage | 4-tab surface | Phase C |
| 8 | CspmPage | compliance | Phase C |
| 9 | CicdPage | pipelines | Phase C |
| 10 | DatabasePage | query mon | Phase D |
| 11 | IntelligencePage | 5-tab unified | Phase D |
| 12 | IncidentsPage | 4-tab | Phase D |
| 13 | NotebookPage | 3-tab | Phase D |
| 14 | SharedDashboardsPage | collab | Phase D |
| 15 | EnterprisePage | 6-tab | Phase D |
| 16 | PlatformPage | 6-tab | Phase D |

## Scope guardrails

- **No new routes, no new endpoints, no new components.** Apply existing primitives only.
- **Phase boundary per group** (A-D, 4 pages each). Per-phase test plan at end of each phase.
- **Mega-test A-to-Z at end** — user runs manual test across all 16 pages.
- **Module discipline** — never exceed 400 LOC per file. If a page grows, split.
- **Tokens only** — no new colors, no new spacing, no new typography. Use what's in tokens.css.

## Out of scope

- New pages
- New endpoints
- Backend changes
- Mobile app
- Documentation updates (those ship with the next docs sweep)

## Impact

| Area | Impact |
|------|--------|
| Files modified | 16 pages + maybe 4-5 components |
| Breaking | No — visual refresh only |
| Risk | Drift in visual consistency if primitives are applied inconsistently |
| Mitigation | Per-phase test plan + screenshot evidence + commit per page |

## Workflow

- Speckit: proposal → spec → tasks → checklist → build → archive
- Phase gates: A (4 pages) → B (6 pages) → C (6 pages) → D (4 pages)
- Per-phase test plan: A-to-Z for each group, user runs manual test
- Mega-test at end across all 16 pages
- Reuse primitives from 014 (StatusPill, TimeSeriesChart, KpiCard)

## Acceptance

- All 16 pages use StatusPill for status display (no more inline status spans)
- All 16 pages use TimeSeriesChart for time-series (no more inline SVG)
- All 16 pages have clickable KPI cards (where KPIs exist)
- Verified live at .115
- `tsc --noEmit` exit 0
- `npm run build` exit 0
- Per-phase test plan delivered to user
- Mega-test A-to-Z at the end

## Rollback

- Per-page commits (one commit per page = easy to revert)
- Single mega-commit at end if user wants full rollback
