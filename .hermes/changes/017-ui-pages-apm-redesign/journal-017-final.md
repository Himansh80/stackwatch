# 017 — APM pages polish — FINAL JOURNAL

**Status:** COMPLETE — 2026-08-27

## Summary

APM dashboard (`/apm`) and service detail (`/apm/service?id=...`) now follow the Datadog design system:
- 4 KPI cards with rolling 24-poll sparklines (Services, Traces/min, Errors/min, P95 global)
- 3-up TimeSeriesChart trend row below KPIs (Traces/min / Errors/min / P95 latency)
- ApmServicePage has back-link, service status StatusPill (Healthy/Watch/Degraded), and 3-card health KPI grid

## What was built

### Phase 7A — ApmPage upgrades
- Added 3 rolling history state vars: `tracesHistory`, `errorsHistory`, `p95History` (max 24 entries)
- Each load() call appends the latest sample; old samples roll off
- KPI cards now receive `sparkline={...}` prop → shows the trend
- New `apm-trend-grid` section with 3 TimeSeriesChart panels (renders only when ≥2 samples)
- Status-aware: empty state ("Waiting for data…", "No errors yet", "Waiting for latency data")

### Phase 7B — ApmServicePage upgrades
- Back-link at top: `← Back to services`
- Service status StatusPill computed from `error_rate`:
  - ≥5% → `crit` → "Degraded"
  - ≥1% → `warn` → "Watch"
  - <1% → `ok` → "Healthy"
- 3-card health KPI grid:
  - **Requests / sec** (cyan) with language·framework subtitle
  - **Error rate** % (green or red based on threshold) with env subtitle
  - **P95 latency** ms (amber) with slow/healthy subtitle
- Meta pills simplified to just language/framework/env (the metrics moved to KPI cards)

## Files modified

- `web/src/pages/ApmPage.tsx` — 333 LOC (under 400 cap)
- `web/src/pages/ApmServicePage.tsx` — 184 LOC (under 400 cap)
- `web/src/styles/profile.css` — added ~30 lines of APM CSS

## Verification

| Check | Result |
|-------|--------|
| `tsc --noEmit` | exit 0 |
| `npm run build` | exit 0 (596 modules transformed) |
| ApmPage LOC | 333 (under 400 cap) |
| ApmServicePage LOC | 184 (under 400 cap) |
| Bundle deployed to .115 | ✅ via scp |

## Commits

```
<this commit> feat(ui): Phase 7 — APM pages polish
```

## Honest gaps

- **Sparkline data** is per-polling-cycle (every time `load()` runs). APM page doesn't have a polling interval — sparklines only fill after the user reloads the page. Add `setInterval(load, 30000)` in a follow-up session if real-time sparklines are wanted.
- **Trend chart panels** also only show data once enough polls accumulate (≥2 samples).
- **Error rate color logic** uses static thresholds (5%/1%) — could be made configurable per-service in a future change.
- **No export button** — APM dashboard doesn't have CSV/PDF export like some enterprise tools.

## Lessons learned

1. **Sparklines + trend charts use the same data structure** (number[]) — `TimeSeriesChart` and `KpiCard` sparkline are fully compatible. Build once, reuse everywhere.
2. **KPI cards should ALWAYS have a sparkline when the metric has temporal context** — even if it's just the last 5 data points. The visual feedback is much richer than a static number.
3. **Service status from error_rate** is a sensible default but should become a per-service policy eventually.

## Follow-up

- 018-ui-pages-incidents-redesign (Incidents + Notifications + Intelligence)
- 019-ui-pages-synthetics-redesign (Synthetics check dashboard)
- 020-ui-pages-security-redesign (Security threats + CSPM)
