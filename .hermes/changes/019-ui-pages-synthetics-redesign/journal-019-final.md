# 019 — Synthetics + Logs + RUM pages polish — FINAL JOURNAL

**Status:** COMPLETE — 2026-08-27

## Summary

All three observability surfaces (`/synthetics`, `/logs`, `/rum`) now use the shared design primitives consistently.

## What was built

### Phase 9A — SyntheticsPage polish
- **Sparklines on KPI cards** (Passing + Failing)
- **Trend chart row** below KPI strip with 2 TimeSeriesCharts (Passing / Failing)
- **Fixed misindented StatusPill JSX** in test row (was rendering with weird leading whitespace)
- Pass/fail logic: tests with `sla_uptime_pct >= 99.5` count as passing (approximation — real pass rate would require `/synthetics/results/:test_id` per test)

### Phase 9B — LogsFullPage overhaul
- **Fixed dead tab state** — `_setTab` was unused, so clicking tabs did nothing. Now `setTab` is wired.
- **Added tab navigation UI** — the page had 5 tabs in code but no way to switch between them. Now rendered as buttons with StatusPill per tab.
- **Added KPI strip** — 4-card grid (Errors / Warnings / Info / Debug) computed client-side from the loaded logs array.
- StatusPill per tab indicates whether that section has data (green if populated, gray if empty).

### Phase 9C — RumFullPage verification
- Already shipped in Phase 5 — session-level StatusPill per row (×{errors} / 0).
- Verified no additional changes needed.

## Files modified

- `web/src/pages/SyntheticsPage.tsx` — 345 → 366 LOC (under 400 cap)
- `web/src/pages/LogsFullPage.tsx` — 143 → 175 LOC (under 400 cap)
- `web/src/pages/RumFullPage.tsx` — unchanged (verified)
- `web/src/styles/profile.css` — added ~10 lines

## Verification

| Check | Result |
|-------|--------|
| `tsc --noEmit` | exit 0 |
| `npm run build` | exit 0 (596 modules) |
| Bundle deployed to .115 | ✅ via scp |

## Commits

```
<this commit> feat(ui): Phase 9 — Synthetics + Logs + RUM pages polish
```

## Honest gaps

- **Pass rate approximation**: Real pass rate requires `/synthetics/results/:test_id` per test. Until that endpoint is exposed, the KPI uses `sla_uptime_pct >= 99.5` as a proxy.
- **LogsFullPage tab loading is lazy**: Each tab needs to be visited at least once to load its data. The StatusPill stays gray (unknown) until data loads.
- **Logs KPI strip**: Shows 0s when no search has been performed. Initial render shows empty state.
- **RumFullPage**: No new work — already shipped in Phase 5.

## Lessons learned

1. **LogsFullPage had NO tab UI** — the code referenced 5 different tabs but never rendered the UI to switch between them. The phase-2 `_setTab` underscore-prefix was hiding the bug. Users could only ever see the Search tab.
2. **Misindented JSX is a code smell** — the StatusPill in SyntheticsPage had 3 leading levels of indentation that hinted at a copy-paste bug. Worth fixing for readability even when the JSX still parses.
3. **Pass-rate approximation** is fine for a KPI card. Adding a real-time results query per test would slow down the page. The 99.5% threshold gives a reasonable signal.

## Follow-up

- 020-ui-pages-security-redesign (Security threats + CSPM)
- 021-ui-pages-platform-redesign (PlatformPage billing + deploy)
