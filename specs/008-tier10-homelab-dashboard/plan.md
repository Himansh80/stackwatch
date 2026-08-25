# Tier 10 Plan — Homelab Dashboard

## Phase 0 — Routes split (mandatory before Phase 1)

Why: `routes_enterprise.go` is now at 278 LOC; adding 40+ more Tier 10
routes would push it over 400. Extract `routes_homelab.go` (mountHomelabRoutes)
following the Tier 9 Phase 0 pattern.

**Outcome:** `routes_homelab.go` placeholder created, `routes_protected.go`
untouched, `routes_enterprise.go` unchanged.

**Files:** 1 new, 1 modified (cmd/api-gateway/routes.go to call mountHomelabRoutes).

## Phase 1 — Widget Framework + Page shell (H1)

Why: every other sub-feature mounts inside the grid. Build the shell first.

**Files (estimated):**
- `internal/handler/handlers_homelab_layout.go` (~150 LOC) — 3 routes:
  GET/PUT /homelab/layout, GET /homelab/layout/defaults
- `internal/handler/handlers_homelab_prefs.go` (~150 LOC) — 3 routes:
  GET/PUT /homelab/prefs, DELETE /homelab/prefs/layout
- `migrations/041_homelab.sql` (initial 2 tables): homelab_user_layouts, homelab_user_prefs
- `web/src/pages/HomelabPage.tsx` (~150 LOC) — page shell with tab strip
- `web/src/components/homelab/HomelabGrid.tsx` (~150 LOC) — react-grid-layout wrapper
- `web/src/components/homelab/HomelabKpiStrip.tsx` (~150 LOC) — top KPI strip
- `cmd/api-gateway/routes_homelab.go` — mountHomelabRoutes stub

**Tables:** 2 (`homelab_user_layouts`, `homelab_user_prefs`)
**Routes:** 6

## Phase 2 — Service Status (H2)

**Files:**
- `internal/handler/handlers_homelab_services_types.go` (~150 LOC)
- `internal/handler/handlers_homelab_services.go` (~200 LOC, 5 routes)
- `internal/handler/handlers_homelab_services_probe.go` (~200 LOC, 3 routes incl. probe + history)
- `internal/homelab/servicehealth.go` (~250 LOC, ticker worker)
- `web/src/components/homelab/widgets/ServiceStatusWidget.tsx` (~180 LOC)
- Migration adds: `homelab_pinned_services`, `homelab_service_health`

**Tables:** +2 (4 total)
**Routes:** +8 (14 total)

## Phase 3 — Notes + Todos (H3)

**Files:**
- `internal/handler/handlers_homelab_notes.go` (~200 LOC, 6 routes)
- `internal/handler/handlers_homelab_todos.go` (~250 LOC, 8 routes)
- `web/src/components/homelab/widgets/NotesWidget.tsx` (~200 LOC)
- `web/src/components/homelab/widgets/TodosWidget.tsx` (~250 LOC)
- Migration adds: `homelab_notes`, `homelab_todos`

**Tables:** +2 (6 total)
**Routes:** +10 (24 total)

## Phase 4 — Calendar (H4)

**Files:**
- `internal/handler/handlers_homelab_calendars.go` (~200 LOC, 5 routes)
- `internal/homelab/calendar.go` (~250 LOC, iCal fetcher + parser)
- `web/src/components/homelab/widgets/CalendarWidget.tsx` (~200 LOC)
- Add dependency: `github.com/lukechampine/ical`
- Migration adds: `homelab_calendars`, `homelab_events`

**Tables:** +2 (8 total)
**Routes:** +5 (29 total)

## Phase 5 — Download Stats (H5)

**Files:**
- `internal/handler/handlers_homelab_downloads.go` (~200 LOC, 4 routes)
- `internal/homelab/downloads.go` (~250 LOC, Sonarr/Radarr/qBittorrent/SABnzbd clients)
- `web/src/components/homelab/widgets/DownloadStatsWidget.tsx` (~150 LOC)
- Migration adds: `homelab_download_clients`, `homelab_download_snapshots`

**Tables:** +2 (10 total)
**Routes:** +4 (33 total)

## Phase 6 — Media Server (H6)

**Files:**
- `internal/handler/handlers_homelab_media.go` (~200 LOC, 4 routes)
- `internal/homelab/media.go` (~250 LOC, Plex/Jellyfin/Emby clients)
- `web/src/components/homelab/widgets/MediaWidget.tsx` (~200 LOC)
- Migration adds: `homelab_media_servers`, `homelab_now_playing`, `homelab_recent_additions`

**Tables:** +3 (13 total)
**Routes:** +4 (37 total)

## Phase 7 — Search (H7)

**Files:**
- `internal/handler/handlers_homelab_search.go` (~200 LOC, 3 routes)
- `web/src/components/homelab/widgets/GlobalSearch.tsx` (~150 LOC)
- No migration (search reads existing tables)

**Tables:** +0 (13 total)
**Routes:** +3 (40 total)

## Phase 8 — RSS (H9)

**Files:**
- `internal/handler/handlers_homelab_rss.go` (~200 LOC, 5 routes)
- `internal/homelab/rss.go` (~200 LOC, RSS poller)
- `web/src/components/homelab/widgets/RssWidget.tsx` (~180 LOC)
- Add dependency: `github.com/mmcdole/gofeed`
- Migration adds: `homelab_rss_feeds`, `homelab_rss_items`

**Tables:** +2 (15 total)
**Routes:** +5 (45 total)

## Phase 9 — Task Scheduler (H10)

**Files:**
- `internal/handler/handlers_homelab_scheduler.go` (~250 LOC, 5 routes)
- `internal/homelab/scheduler.go` (~300 LOC, cron-style scheduler)
- `web/src/components/homelab/widgets/SchedulerWidget.tsx` (~200 LOC)
- Migration adds: `homelab_scheduler_jobs`, `homelab_scheduler_runs`

**Tables:** +2 (17 total)
**Routes:** +5 (50 total)

## Phase 10 — Finalize

- AppSidebar entry: "Homelab"
- Update existing tests
- Notebook with TIER 10 COMPLETE marker
- Archive `.hermes/changes/008-tier10-homelab-dashboard/`
- Commit `docs(tier10): TIER 10 COMPLETE marker + archive + journal`

## Total (estimated)

- **17 DB tables**
- **50 routes**
- **50+ files** (all <400 LOC)
- **11 widgets** + 1 grid + 1 KPI strip + 1 page shell
- **5 background workers** (servicehealth, calendar, downloads, media, rss, scheduler)
- **2 new deps**: `ical`, `gofeed`

## Risks

1. **Scheduler (H10) security**: user-schedulable cron is SSRF/RCE risk.
   Mitigation: action allowlist (HTTP probe / webhook / email only — no shell).
2. **iCal parsing**: use vetted `lukechampine/ical`; never hand-roll.
3. **Layout persistence must not regress Tier 7 dashboards**: separate table.
4. **Service pin count cap**: 50 per user.
5. **All polling workers must respect tenant_id** — no cross-tenant leakage.

## Verifier

`hermes-verify-tier10-final.py` — checks every route + every widget renders
+ every worker started without panic.
