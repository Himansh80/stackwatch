# Spec: 015 — Apply design primitives to all remaining pages

## Functional requirements

For every page in the Phase A-D list:

### Status display → StatusPill
- Search for `dash-status`, `px-status`, `status-good`, `status-bad`, `status-neutral` classes
- Replace inline `<span>` status badges with `<StatusPill status={...} />`
- Map any custom status string to one of: `up`, `down`, `stale`, `unknown`, `ok`, `warn`, `crit`
- Use `size="sm"` in tables, `size="md"` in detail headers, `size="lg"` in empty states

### Time-series → TimeSeriesChart
- Search for inline `<svg>` line charts or table-based trend visualizations
- Replace with `<TimeSeriesChart values={...} color="..." />`
- Wire to existing fetch hooks (no new endpoints)
- Color by metric family: cyan=CPU, indigo=memory, green=disk, amber=network, red=error rate

### KPI cards → clickable KpiCard
- Search for non-clickable KPI cards (custom `<div className="...">` blocks)
- Replace with `<KpiCard ... onClick={...} />` if there's a sensible navigation
- Always include `delta` (last sync time) and `icon` (Lucide-style SVG)
- Add `trend` arrow if there's a known historical comparison

## Non-functional requirements

- **No new endpoints** — same backend, only visual changes
- **Module discipline** — every modified file ≤400 LOC
- **Tokens only** — colors/spacing/typography must come from tokens.css
- **Mobile responsive** — every page works on <768px viewport

## Acceptance

- All 16 pages render with the new primitives
- All routes still serve HTTP 200 from .115
- Visual consistency across the app
- `tsc --noEmit` exit 0
- `npm run build` exit 0

## Out of scope

- New pages / new routes / new endpoints
- Backend changes
- Mobile native app

## Test plan format

Per-phase (4 pages per phase):
1. Numbered test cases per page
2. Expected vs actual
3. Edge cases (empty state, error, mobile)
4. "What to tell me" outcome

Mega-test at end: full A-to-Z across all 16 pages — user's acceptance gate.
