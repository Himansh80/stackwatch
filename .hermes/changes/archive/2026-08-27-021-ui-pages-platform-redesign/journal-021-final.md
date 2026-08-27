# 021 — Platform + Database + CI/CD pages polish — FINAL JOURNAL

**Status:** COMPLETE — 2026-08-27

## Summary

All three remaining platform/operations surfaces (`/platform`, `/database`, `/cicd`) now use the shared design primitives consistently.

## What was built

### Phase 11A — PlatformPage polish
- **StatusPill per tab button** — active tab shows `up` (green), inactive shows `unknown` (gray)
- Subtitle stays as-is alongside the pill

### Phase 11B — DatabasePage polish
- **Rolling history state** (slowHistory, queryHistory)
- **KPI sparklines** on the "Slow (≥1s avg)" KPI card
- **2-up TimeSeriesChart row** — Slow queries + Query volume (estimated)

### Phase 11C — CicdPage polish
- **Rolling history state** (pipelineHistory, failureHistory)
- **KPI sparklines** on the "Pipelines today" KPI card
- **2-up TimeSeriesChart row** — Pipelines + Failures

## Files modified

- `web/src/pages/PlatformPage.tsx` — 147 → 156 LOC (under 400 cap)
- `web/src/pages/DatabasePage.tsx` — 288 → 313 LOC (under 400 cap)
- `web/src/pages/CicdPage.tsx` — 235 → 247 LOC (under 400 cap)
- `web/src/styles/profile.css` — added ~6 lines of trend CSS

## Verification

| Check | Result |
|-------|--------|
| `tsc --noEmit` | exit 0 |
| `npm run build` | exit 0 (596 modules) |
| Bundle deployed to .115 | ✅ via scp |

## Commits

```
<this commit> feat(ui): Phase 11 — Platform + Database + CI/CD polish
```

## Honest gaps

- **Query volume estimation** — DatabasePage uses `slow_queries_count * 7` as a proxy for total query volume. Real volume would require a separate counter endpoint.
- **No status pills on tab buttons when content is loaded** — PlatformPage status pills only show "active" vs "inactive" tab. Could be enhanced to show data presence (green=has content, gray=empty) per tab in a future change.
- **Sparklines only update on page reload** — same caveat as all other pages in this series. Need polling interval for real-time updates.

## Lessons learned

1. **StatusPill on tabs is a great navigation aid** — adds a tiny color dot that signals tab state (active/inactive, loaded/empty) without taking much space.
2. **Pattern is consistent across all 21+ pages** — KPI + sparklines + trend chart + StatusPill everywhere. This is the consistent Datadog aesthetic the user asked for.
3. **3 more pages added in this speckit** — cumulatively we've now polished ~16+ pages with the same primitives (014-021).

## Follow-up

- 022-ui-pages-incidents-intelligence-final-polish (final sweep)
- 023-ui-pages-mobile-or-docs (mobile app or docs site)
