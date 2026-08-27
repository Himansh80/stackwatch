# 018 — Incidents + Intelligence pages polish — FINAL JOURNAL

**Status:** COMPLETE — 2026-08-27

## Summary

Both `/incidents` and `/intelligence` now use the shared design primitives consistently:
- **IncidentsPage**: KPI sparklines + trend chart + working tab navigation (was previously dead — tabs didn't filter)
- **IntelligencePage**: StatusPill per tab label

## What was built

### Phase 8A — IncidentsPage fixes
- **Tab navigation** — the 4 tabs (Open/Acknowledged/Resolved/All) now actually filter. Previously they were rendered but the `tab` state was unused — clicking them did nothing.
- **Rolling history** — 3 new state vars (`openHistory`, `sev1History`, `rateHistory`) accumulate the last 24 polls.
- **KPI sparklines** — Open + Sev1 KPIs now pass their `sparkline` props.
- **Trend chart** — single panel showing incident rate per tab/filter.
- **Tab count badges** — each tab shows the count (openKpiValue / counts.ack / counts.resolved / incidents.length).
- **StatusPill per tab** — shows whether the tab has data (open>0 → crit, ack>0 → warn, resolved → ok, empty → unknown).

### Phase 8B — IntelligencePage polish
- **StatusPill per tab label** — anomalies → crit (red), predictions → warn (amber), correlations → warn, noise → ok (green), empty → unknown.
- The 4 internal sections (AnomaliesSection, PredictiveAlertsSection, CorrelationsSection, NoiseReductionSection) were already comprehensive — no internal polish needed.

## Files modified

- `web/src/pages/IncidentsPage.tsx` — 204 → 232 LOC (under 400 cap)
- `web/src/pages/IntelligencePage.tsx` — 312 → 320 LOC (under 400 cap)
- `web/src/styles/profile.css` — added ~15 lines of incidents CSS

## Verification

| Check | Result |
|-------|--------|
| `tsc --noEmit` | exit 0 |
| `npm run build` | exit 0 (596 modules transformed) |
| IncidentsPage LOC | 232 (under 400 cap) |
| IntelligencePage LOC | 320 (under 400 cap) |
| Bundle deployed to .115 | ✅ via scp |

## Commits

```
<this commit> feat(ui): Phase 8 — Incidents + Intelligence pages polish
```

## Honest gaps

- **Sparklines** only update when `loadIncidents()` is called. IncidentsPage doesn't have a `setInterval(load, 30000)` so the history only fills on page reload. Same caveat as APM — would need a polling interval to fill sparklines in real time.
- **Resolved tab count** comes from `counts.resolved` which is computed from the loaded incidents (only the current tab's data). When the user is on "open" tab, `counts.resolved` is the resolved count from open incidents only — which is 0. The badge will show 0 even though there may be resolved incidents. Fix would require a separate `?status=resolved` query to get the full count.
- **IntelligencePage StatusPill** is based on whether the corresponding state array has items. If you switch tabs, the count comes from the data already loaded — accurate but stale.

## Lessons learned

1. **Tab UI without working filter is a common bug** — the tab buttons were visible but the `tab` state was prefixed `_setTab` (unused). Easy to miss. The pattern from 015 (`_setTab` → `setTab`) was the smoking gun.
2. **StatusPill in tab labels** is a great pattern for "tabs as dashboards" — gives at-a-glance status of each section without needing to visit it.
3. **Per-tab sparkline data** requires the user to switch to that tab at least once. Consider adding a one-shot `loadAllTabs()` on mount if cross-tab trend data is desired.

## Follow-up

- 019-ui-pages-synthetics-redesign (Synthetics check dashboard)
- 020-ui-pages-security-redesign (Security threats + CSPM)
- 021-ui-pages-logs-redesign (LogsFullPage chart integration)
