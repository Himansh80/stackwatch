# Tier 10 Tasks — Homelab Dashboard (Homarr Parity)

## Phase 0 — Routes split (foundation)

- [ ] 0.1 Create `cmd/api-gateway/routes_homelab.go` with stub `mountHomelabRoutes(protected *gin.RouterGroup, pool *db.Pool)`
- [ ] 0.2 Modify `cmd/api-gateway/routes.go` to call `mountHomelabRoutes` after `mountEnterpriseRoutes`
- [ ] 0.3 Verify `go build ./cmd/api-gateway` exits 0
- [ ] 0.4 Commit: `feat(tier10): routes split (Phase 0)`

## Phase 1 — Widget Framework + Page shell (H1)

- [ ] 1.1 Migration `041_homelab.sql`: tables `homelab_user_layouts`, `homelab_user_prefs`
- [ ] 1.2 Apply migration to prod
- [ ] 1.3 `internal/handler/handlers_homelab_layout.go` (~150 LOC)
  - GET /api/v1/homelab/layout — get my layout
  - PUT /api/v1/homelab/layout — update layout JSON
  - GET /api/v1/homelab/layout/defaults — return default grid
- [ ] 1.4 `internal/handler/handlers_homelab_prefs.go` (~150 LOC)
  - GET /api/v1/homelab/prefs — get my prefs
  - PUT /api/v1/homelab/prefs — update prefs
  - DELETE /api/v1/homelab/prefs/layout — reset to defaults
- [ ] 1.5 Register 6 routes in `mountHomelabRoutes`
- [ ] 1.6 `web/src/pages/HomelabPage.tsx` (~150 LOC) — page shell with tab strip
- [ ] 1.7 `web/src/components/homelab/HomelabGrid.tsx` (~150 LOC) — react-grid-layout wrapper
- [ ] 1.8 `web/src/components/homelab/HomelabKpiStrip.tsx` (~150 LOC) — top KPI strip
- [ ] 1.9 Add Homelab link to AppSidebar
- [ ] 1.10 Verify: `go build` / `go vet` / `npm run build` all green
- [ ] 1.11 Deploy to .115 and verify all 6 routes return 401 without auth
- [ ] 1.12 Commit: `feat(tier10): Widget Framework + Page shell (Phase 1)`

## Phase 2 — Service Status (H2)

- [ ] 2.1 Migration adds `homelab_pinned_services`, `homelab_service_health`
- [ ] 2.2 Apply migration to prod
- [ ] 2.3 `internal/handler/handlers_homelab_services_types.go` (~150 LOC)
- [ ] 2.4 `internal/handler/handlers_homelab_services.go` (~200 LOC, 5 routes):
  - GET /homelab/services
  - POST /homelab/services
  - PATCH /homelab/services/:id
  - DELETE /homelab/services/:id
  - GET /homelab/services/aggregate
- [ ] 2.5 `internal/handler/handlers_homelab_services_probe.go` (~200 LOC, 3 routes):
  - POST /homelab/services/:id/probe (one-shot)
  - GET /homelab/services/:id/history (last 100)
  - POST /homelab/services/probe-all
- [ ] 2.6 `internal/homelab/servicehealth.go` (~250 LOC) — background worker
- [ ] 2.7 `web/src/components/homelab/widgets/ServiceStatusWidget.tsx` (~180 LOC)
- [ ] 2.8 Register 8 routes in `mountHomelabRoutes`
- [ ] 2.9 Verify gates green + deploy + live verify
- [ ] 2.10 Commit: `feat(tier10): Service Status (Phase 2)`

## Phase 3 — Notes + Todos (H3)

- [ ] 3.1 Migration adds `homelab_notes`, `homelab_todos`
- [ ] 3.2 Apply migration to prod
- [ ] 3.3 `internal/handler/handlers_homelab_notes.go` (~200 LOC, 6 routes)
- [ ] 3.4 `internal/handler/handlers_homelab_todos.go` (~250 LOC, 8 routes)
- [ ] 3.5 `web/src/components/homelab/widgets/NotesWidget.tsx` (~200 LOC) — Markdown
- [ ] 3.6 `web/src/components/homelab/widgets/TodosWidget.tsx` (~250 LOC)
- [ ] 3.7 Register 14 routes in `mountHomelabRoutes`
- [ ] 3.8 Verify gates green + deploy + live verify
- [ ] 3.9 Commit: `feat(tier10): Notes + Todos (Phase 3)`

## Phase 4 — Calendar (H4)

- [ ] 4.1 Migration adds `homelab_calendars`, `homelab_events`
- [ ] 4.2 Apply migration to prod
- [ ] 4.3 Add `github.com/lukechampine/ical` to go.mod
- [ ] 4.4 `internal/handler/handlers_homelab_calendars.go` (~200 LOC, 5 routes)
- [ ] 4.5 `internal/homelab/calendar.go` (~250 LOC) — iCal fetcher
- [ ] 4.6 `web/src/components/homelab/widgets/CalendarWidget.tsx` (~200 LOC)
- [ ] 4.7 Register 5 routes
- [ ] 4.8 Verify + deploy + commit

## Phase 5 — Download Stats (H5)

- [ ] 5.1 Migration adds `homelab_download_clients`, `homelab_download_snapshots`
- [ ] 5.2 Apply migration to prod
- [ ] 5.3 `internal/handler/handlers_homelab_downloads.go` (~200 LOC, 4 routes)
- [ ] 5.4 `internal/homelab/downloads.go` (~250 LOC) — Sonarr/Radarr/qBittorrent/SABnzbd clients
- [ ] 5.5 `web/src/components/homelab/widgets/DownloadStatsWidget.tsx` (~150 LOC)
- [ ] 5.6 Register 4 routes
- [ ] 5.7 Verify + deploy + commit

## Phase 6 — Media Server (H6)

- [ ] 6.1 Migration adds `homelab_media_servers`, `homelab_now_playing`, `homelab_recent_additions`
- [ ] 6.2 Apply migration to prod
- [ ] 6.3 `internal/handler/handlers_homelab_media.go` (~200 LOC, 4 routes)
- [ ] 6.4 `internal/homelab/media.go` (~250 LOC) — Plex/Jellyfin/Emby clients
- [ ] 6.5 `web/src/components/homelab/widgets/MediaWidget.tsx` (~200 LOC)
- [ ] 6.6 Register 4 routes
- [ ] 6.7 Verify + deploy + commit

## Phase 7 — Search (H7)

- [ ] 7.1 `internal/handler/handlers_homelab_search.go` (~200 LOC, 3 routes)
- [ ] 7.2 `web/src/components/homelab/widgets/GlobalSearch.tsx` (~150 LOC)
- [ ] 7.3 Register 3 routes
- [ ] 7.4 Verify + deploy + commit

## Phase 8 — RSS (H9)

- [ ] 8.1 Migration adds `homelab_rss_feeds`, `homelab_rss_items`
- [ ] 8.2 Apply migration to prod
- [ ] 8.3 Add `github.com/mmcdole/gofeed` to go.mod
- [ ] 8.4 `internal/handler/handlers_homelab_rss.go` (~200 LOC, 5 routes)
- [ ] 8.5 `internal/homelab/rss.go` (~200 LOC) — RSS poller
- [ ] 8.6 `web/src/components/homelab/widgets/RssWidget.tsx` (~180 LOC)
- [ ] 8.7 Register 5 routes
- [ ] 8.8 Verify + deploy + commit

## Phase 9 — Task Scheduler (H10)

- [ ] 9.1 Migration adds `homelab_scheduler_jobs`, `homelab_scheduler_runs`
- [ ] 9.2 Apply migration to prod
- [ ] 9.3 `internal/handler/handlers_homelab_scheduler.go` (~250 LOC, 5 routes)
- [ ] 9.4 `internal/homelab/scheduler.go` (~300 LOC) — cron-style dispatcher
- [ ] 9.5 `web/src/components/homelab/widgets/SchedulerWidget.tsx` (~200 LOC)
- [ ] 9.6 Register 5 routes
- [ ] 9.7 Verify + deploy + commit

## Phase 10 — Finalize (TIER 10 COMPLETE)

- [ ] 10.1 All 50 routes verified live with real auth
- [ ] 10.2 Cross-tier regression check (Tier 0-9 still pass)
- [ ] 10.3 AppSidebar entry confirmed
- [ ] 10.4 `journal-tier10-final.md` written
- [ ] 10.5 Archive `.hermes/changes/008-tier10-homelab-dashboard/`
- [ ] 10.6 Final commit: `docs(tier10): TIER 10 COMPLETE marker + archive + journal`

## Out-of-scope (Tier 11+)

- Push-button deploy (PL1) — Tier 11
- Multi-region HA (PL6) — Tier 11
- Mobile widgets — Tier 13
