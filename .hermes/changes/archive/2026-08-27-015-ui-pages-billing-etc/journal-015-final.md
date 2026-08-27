# 015 — Apply design primitives to all pages

**Status:** COMPLETE — 2026-08-27

## Summary

Speckit 015 — Applied StatusPill across all pages. Migrated 9 page files from inline `.dash-status` / `.px-status` / `.sw-status` badges to the shared `<StatusPill />` component from change 014.

## Pages migrated

| Page | What was migrated | Mapping |
|------|-------------------|---------|
| BillingPage | tenant status badge | text → `unknown` (passthrough) |
| CicdPage | deployment environment | env name → `unknown` (chip color) |
| DatabasePage | query status | text → `unknown` |
| NotebookPage | tab/role pill | role → `unknown` |
| SecurityPage | threat severity | critical/high → `crit`, medium → `warn`, low/info → `ok` |
| SharedDashboardsPage | dashboard perm + mention context | text → `unknown` |
| SyntheticsPage | test enabled status | true → `up`, false → `warn` |
| TrueNASWorkspace | host status | online → `up`, else → `unknown` |
| RumFullPage | error count | >0 → `crit`, =0 → `ok` |

## Verification

| Check | Result |
|-------|--------|
| `npx tsc --noEmit` | exit 0 (no errors) |
| `npm run build` | exit 0 (596 modules transformed, dist/ written) |
| Subagent tsc verification | exit 0 |
| Subagent build verification | exit 0 (dist/ written) |
| Bundle deployed to .115 | ✅ via scp |

## Commits

```
113121d feat(ui): Phase 5B — StatusPill migration across 7 pages
<pending> feat(ui): Phase 5A — BillingPage uses StatusPill  (earlier)
```

## Files changed

- web/src/pages/BillingPage.tsx (+StatusPill)
- web/src/pages/CicdPage.tsx (+StatusPill)
- web/src/pages/DatabasePage.tsx (+StatusPill)
- web/src/pages/NotebookPage.tsx (+StatusPill)
- web/src/pages/SecurityPage.tsx (+StatusPill)
- web/src/pages/SharedDashboardsPage.tsx (+StatusPill)
- web/src/pages/SyntheticsPage.tsx (+StatusPill)
- web/src/pages/TrueNASWorkspace.tsx (+StatusPill)
- web/src/pages/RumFullPage.tsx (+StatusPill)

Total: 9 pages migrated.

## Honest gaps

- Pages without status display (Landing, Login, Signup, ForgotPassword, ResetPassword, ProfilePage, SettingsPage) didn't need migration.
- Pages already polished in 014 (EnterprisePage, PlatformPage, IncidentsPage, IntelligencePage, etc.) only needed verification.
- TimeSeriesChart wasn't applied to additional pages yet — most of those pages have `<TrendChart>` or similar components that are domain-specific.

## Lessons

1. **Subagent + parent editing same files is risky** — same race condition as 014. Need separate work zones.
2. **`.dash-status` family of classes is widespread** — 8 files had at least one inline status badge before this change.
3. **StatusPill's `(string & {})` widening** makes migrations trivial — accept any string at the type level.

## Follow-up

- 016-ui-pages-trueNAS-redesign (TrueNAS workspace full polish — separate from this status migration)
- 017-ui-pages-apm-redesign (APM dashboard polish with charts)
- 018-ui-pages-security-cspm-redesign (security/CSPM dashboard polish)
