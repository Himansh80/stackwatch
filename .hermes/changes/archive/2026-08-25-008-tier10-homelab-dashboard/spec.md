# Tier 10 Specification — Homelab Dashboard

## 1. Goals (measurable)

| Metric | Target |
|--------|--------|
| Tier 10 routes live on `.115` | ≥ 40 |
| Tier 10 DB tables on `.116` | 11 (or 10 if a shared table is reused) |
| All files < 400 LOC | Yes (every Go + TSX) |
| Per-user layout persistence | Yes (separate from tenant dashboards) |
| Service health poll cadence | 60s default, configurable 30s-300s |
| RSS poll cadence | 5 min default |
| Scheduler dispatch latency | < 5s past scheduled time |
| Tenant isolation on every query | Yes (RBAC `homelab:read` / `homelab:write`) |
| Cross-tier regressions | None (Tier 0-9 routes still pass) |

## 2. Sub-features

### H1 — Widget Framework

- 11 widget types, each a React component
- Grid layout via `react-grid-layout`
- Per-widget refresh cadence (default 60s)
- Drag-resize-reorder
- All widgets use existing motion exports (`pageEnter`, `kpiEnter`,
  `kpiStagger`)
- No new dependencies

### H2 — Service Status

**Tables:** `homelab_pinned_services`, `homelab_service_health`

**Routes (8):**
- `GET    /api/v1/homelab/services`         — list my pinned services
- `POST   /api/v1/homelab/services`         — pin a service (name, url, kind, icon)
- `PATCH  /api/v1/homelab/services/:id`     — update pin
- `DELETE /api/v1/homelab/services/:id`     — unpin
- `POST   /api/v1/homelab/services/:id/probe` — one-shot HTTP/TCP/ICMP probe
- `GET    /api/v1/homelab/services/:id/history` — recent health (last 100)
- `POST   /api/v1/homelab/services/probe-all`  — probe all (used by cron)
- `GET    /api/v1/homelab/services/aggregate` — summary (up/degraded/down counts)

**Widget:** `ServiceStatusWidget` — grid of pinned services with status pills.

### H3 — Personal Notes + Todos

**Tables:** `homelab_notes`, `homelab_todos`

**Routes (10):**
- Notes: list / create / get / update / delete / search
- Todos: list / create / get / update / delete / complete / search / due-soon

**Widgets:** `NotesWidget`, `TodosWidget`. Markdown rendering via `react-markdown`
(already a dep). Todos have priority (low/med/high/urgent) + due date + tags.

### H4 — Calendar

**Tables:** `homelab_calendars`, `homelab_events`

**Routes (5):**
- list calendars / add iCal URL / refresh / delete / upcoming events (next 30d)

**Background worker:** `internal/homelab/calendar.go` — every 30min fetches each
iCal URL, parses with `github.com/lukechampine/ical`, upserts events.

**Widget:** `CalendarWidget` — week view (Mon-Sun) with events as colored chips.

### H5 — Download Stats

**Tables:** `homelab_download_clients`, `homelab_download_snapshots`

**Routes (4):**
- list clients / add (Sonarr/Radarr/qBittorrent/SABnzbd) / delete
- GET current snapshot (queue size, speed, today's GB)

**Background worker:** `internal/homelab/downloads.go` — every 60s polls each
client's `/api/v3/queue` or equivalent.

**Widget:** `DownloadStatsWidget` — 3 KpiCards + sparkline.

### H6 — Media Server

**Tables:** `homelab_media_servers`, `homelab_now_playing`, `homelab_recent_additions`

**Routes (4):**
- list / add (Plex/Jellyfin/Emby) / delete / current state

**Background worker:** `internal/homelab/media.go` — every 60s polls
Plex `/library/recentlyAdded` + `/status/sessions`.

**Widget:** `MediaWidget` — now-playing + recent additions carousel.

### H7 — Search

**Tables:** `homelab_search_index` (materialized view or computed on-the-fly)

**Routes (2):**
- `GET /api/v1/homelab/search?q=...` — global search across notes, todos,
  services, integrations, alarms
- `GET /api/v1/homelab/search/suggestions?q=...` — autocomplete

**Implementation:** Single SQL UNION query across all searchable tables,
filtered by tenant_id + user_id. No Elasticsearch needed at this scale.

### H8 — Per-User Preferences

**Tables:** `homelab_user_prefs`

**Routes (3):**
- `GET    /api/v1/homelab/prefs` — get my prefs (layout, theme, refresh_interval, pinned_widgets)
- `PUT    /api/v1/homelab/prefs` — update prefs
- `DELETE /api/v1/homelab/prefs/layout` — reset to default layout

**Wid...**
