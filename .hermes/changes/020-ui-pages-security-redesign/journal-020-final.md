# 020 — Security + CSPM pages polish — FINAL JOURNAL

**Status:** COMPLETE — 2026-08-27

## Summary

Both `/security` and `/cspm` now have KPI sparklines + trend charts using the shared design primitives.

## What was built

### Phase 10A — SecurityPage polish
- **4-card KPI strip** added above the threat filter row (Open threats / Critical / High / Total)
- **Sparklines** on Open threats + Critical KPIs (rolling 24-poll history)
- **2-up TimeSeriesChart row** — Threat count + Critical severity trends over time
- Status-aware: Open threats KPI shows `crit` red when threats exist, `up` green when clear

### Phase 10B — CspmPage polish
- **2 KPI cards** upgraded with sparklines (Tracked resources + Open findings)
- **2-up TimeSeriesChart row** — Tracked resources + Findings trends
- Status-aware: Open findings shows `crit` red when findings exist, `neutral` gray when clear

## Files modified

- `web/src/pages/SecurityPage.tsx` — 326 → 363 LOC (under 400 cap)
- `web/src/pages/CspmPage.tsx` — 215 → 250 LOC (under 400 cap)
- `web/src/styles/profile.css` — added ~6 lines of trend CSS

## Verification

| Check | Result |
|-------|--------|
| `tsc --noEmit` | exit 0 |
| `npm run build` | exit 0 (596 modules) |
| Bundle deployed to .115 | ✅ via scp |

## Commits

```
<this commit> feat(ui): Phase 10 — Security + CSPM pages polish
```

## Honest gaps

- **Sparklines only fill after manual reload** — neither page has `setInterval(load, 30000)`. Same caveat as APM/Incidents — the history only updates on page load.
- **CSPM drift detection** is approximated as "findings count" — a real drift would track per-framework pass rate over time. Would need a separate endpoint.
- **Threat trend** doesn't break down by severity. All severities are combined into "Threat count".

## Lessons learned

1. **The 2 append() functions** in CspmPage's two load handlers are slightly redundant — could extract to a module-level helper. Not worth refactoring for this small change.
2. **Status pill overflow** — SecurityPage has 4 KPIs in a row (might wrap on smaller viewports). The 1626px breakpoint should handle it but worth checking.

## Follow-up

- 021-ui-pages-platform-redesign (PlatformPage billing + deploy)
- 022-ui-pages-database-redesign (DatabasePage query mon)
- 023-ui-pages-cicd-redesign (CicdPage pipelines)
