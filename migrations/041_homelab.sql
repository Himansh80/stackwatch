-- Tier 10 — Homelab Dashboard (008-tier10-homelab-dashboard)
--
-- Adds the first two tables of the homelab layout + per-user
-- preferences surface (Phase 1, H1 — Widget Framework). Future
-- phases of this change add their tables to this same file:
--
--   Phase 1 (this migration):
--     homelab_user_layouts — per-user grid layout JSON
--     homelab_user_prefs   — per-user theme/refresh/pinned-widgets
--
--   Phase 2 (H2 — Service Status):
--     homelab_pinned_services + homelab_service_health
--
--   Phase 3 (H3 — Notes + Todos):
--     homelab_notes + homelab_todos
--
--   Phase 4 (H4 — Calendar) [this update]:
--     homelab_calendars + homelab_events
--
--   Phase 5 (H5 — Download Stats):
--     homelab_download_clients + homelab_download_snapshots
--
--   Phase 6 (H6 — Media Server):
--     homelab_media_servers + homelab_now_playing + homelab_recent_additions
--
--   Phase 8 (H9 — RSS):
--     homelab_rss_feeds + homelab_rss_items
--
--   Phase 9 (H10 — Scheduler):
--     homelab_scheduler_jobs + homelab_scheduler_runs
--
-- All tables are idempotent (CREATE TABLE IF NOT EXISTS / CREATE
-- INDEX IF NOT EXISTS) so re-applying this file is a no-op. No
-- data migration is needed — the tables start empty (Phase 1) and
-- Phase 4 adds only new tables (no destructive ALTERs).
--
-- Why homelab_* (NOT tier10_* or h1_*): future phases add their
-- tables to this same migration under the same homelab_ prefix so
-- a single migration backs the whole tier, matching the Tier 9
-- pattern (migrations/040_enterprise.sql) where one file backs all
-- 6 phases.
--
-- Per-user (not per-tenant) design:
--   Both tables are UNIQUE on (tenant_id, user_id) so layouts and
--   preferences are uniquely scoped PER USER. A user changing their
--   layout in tenant A does not affect another user in the same
--   tenant — this is critical for multi-user homelabs (US-8 in the
--   speckit proposal: "my homelab doesn't change under me when my
--   co-founder rearranges theirs").
--
--   ON DELETE CASCADE on tenant_id + user_id keeps the per-user
--   invariants intact when a tenant or user is removed: their
--   homelab preferences vanish with them, no orphan rows.
--
-- Why JSONB for layout (not normalized widget positions):
--   The layout is a sparse, user-driven array of widget descriptors
--   ({i, x, y, w, h, type, config}). It is read + replaced as a
--   whole (PUT replaces the entire JSONB blob). Normalizing this
--   would be a huge waste — there is no aggregate query, no JOIN,
--   no index lookup on individual widget positions. JSONB keeps
--   the schema flexible for future widget types without ALTER TABLE
--   every time we ship a new widget.
--
--   pinned_widgets uses text[] instead of JSONB because it's a
--   simple, fixed-shape list of widget-type strings ('notes',
--   'todos', 'services', ...) — text[] supports fast "is X pinned"
--   lookups via the && / @> operators if we ever need them.

-- ---------------------------------------------------------------------------
-- homelab_user_layouts — one row per (tenant, user) holding the grid layout.
--
-- `layout` is a JSONB array of widget descriptors. The shape of each
-- descriptor (used by the frontend HomelabGrid component) is:
--
--   {
--     "i":      "kpi-services",     // stable React key
--     "x":      0,                  // column
--     "y":      0,                  // row
--     "w":      3,                  // width (in 12-col grid)
--     "h":      1,                  // height (in row units)
--     "type":   "kpi-services",     // widget type id
--     "config": { "color": "cyan" }  // widget-specific overrides
--   }
--
-- PUT /api/v1/homelab/layout replaces the entire `layout` array —
-- UPSERT semantics on (tenant_id, user_id). If no row exists, one
-- is INSERTed. If a row exists, layout + updated_at are updated.
--
-- GET /api/v1/homelab/layout returns the row's layout. If the user
-- has never saved a layout (no row), the handler returns the
-- default layout (KPI strip + Services widget + Notes widget)
-- instead of 404 — first-run UX must not feel broken.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_user_layouts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    layout      JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

-- Supports the "show me all users with custom layouts in this tenant"
-- admin query (not exposed yet, but cheap to add the index now).
CREATE INDEX IF NOT EXISTS idx_homelab_user_layouts_user
    ON homelab_user_layouts(user_id);
CREATE INDEX IF NOT EXISTS idx_homelab_user_layouts_tenant
    ON homelab_user_layouts(tenant_id);

-- ---------------------------------------------------------------------------
-- homelab_user_prefs — one row per (tenant, user) holding preferences.
--
-- Fields:
--   theme           'light' | 'dark' | 'auto'   (validated handler-side)
--   refresh_seconds 30..300                     (validated handler-side)
--   pinned_widgets  text[] of widget-type ids   ('notes','todos',...)
--   default_landing 'homelab' | 'dashboard' | ... (any dashboard route)
--
-- UPSERT on (tenant_id, user_id) — first GET auto-creates the row
-- with platform defaults (theme='auto', refresh_seconds=60,
-- pinned_widgets='{}', default_landing='homelab'). PUT updates only
-- the fields the caller sends (PATCH semantics; unmentioned fields
-- are left alone).
--
-- DELETE /api/v1/homelab/prefs/layout does NOT delete the prefs row —
-- it only resets the layout (handler deletes from homelab_user_layouts
-- and the next GET on that path returns the default layout). Theme /
-- refresh / pinned-widgets / default-landing preferences survive
-- a layout reset.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_user_prefs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id           UUID NOT NULL,
    theme             TEXT NOT NULL DEFAULT 'auto',
    refresh_seconds   INTEGER NOT NULL DEFAULT 60,
    pinned_widgets    TEXT[] NOT NULL DEFAULT '{}',
    default_landing   TEXT NOT NULL DEFAULT 'homelab',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

-- Same as above — index by user_id supports "list users with custom
-- prefs in tenant X" admin queries that don't exist yet but might.
CREATE INDEX IF NOT EXISTS idx_homelab_user_prefs_user
    ON homelab_user_prefs(user_id);
CREATE INDEX IF NOT EXISTS idx_homelab_user_prefs_tenant
    ON homelab_user_prefs(tenant_id);

-- ---------------------------------------------------------------------------
-- homelab_pinned_services — Phase 2 (H2 — Service Status).
--
-- Per-user pinboard of services the user wants one-click access to
-- (Proxmox, Jellyfin, Pi-hole, PiKVM, NAS dashboard, ...). Each
-- pin is a (name, url, kind, icon, enabled) tuple:
--
--   name    — display label shown on the dashboard tile
--   url     — full URL (http/https/tcp/icmp depending on kind)
--   kind    — 'http' | 'https' | 'tcp' | 'icmp' (worker uses this
--             to pick the probe implementation)
--   icon    — emoji or short text ('🛜', '🎬', '🖥️'); optional
--   enabled — when false, the background worker skips this pin but
--             the row is preserved so the user can re-enable it
--
-- Why (tenant_id, user_id) UNIQUE on name: per the speckit proposal
-- US-8 ("my homelab doesn't change under me when my co-founder
-- rearranges theirs"), two users in the same tenant can each pin a
-- service called "Proxmox" without colliding. The UNIQUE constraint
-- is scoped to (tenant, user, name) so the same user can't double-pin
-- "Proxmox" but a sibling user can.
--
-- Per-user pin count cap is enforced handler-side at 50 (see
-- maxPinnedServicesPerUser in handlers_homelab_services.go) — matches
-- the speckit proposal §"Risks" item 4 ("per-user pin count cap (50)").
-- We don't use a CHECK constraint because we want the cap to evolve
-- without ALTER TABLE.
--
-- Why ON DELETE CASCADE on tenant_id: removing a tenant should remove
-- all their homelab state, including pins. user_id does NOT cascade to
-- a user table because the homelab is intentionally decoupled from
-- Tier 0 user identities (a deleted user in Tier 0 leaves their
-- homelab state intact for audit / restore — Tier 0 cleanup is its
-- own concern).
CREATE TABLE IF NOT EXISTS homelab_pinned_services (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL,
    kind        TEXT NOT NULL DEFAULT 'http',
    icon        TEXT,
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id, name)
);

-- Supports the worker's "list all enabled pins across all users in
-- this tenant" tick query. Partial index — only enabled rows are
-- scanned on each 60s tick, so disabled pins never bloat the index.
CREATE INDEX IF NOT EXISTS idx_homelab_pinned_services_user
    ON homelab_pinned_services(user_id);
CREATE INDEX IF NOT EXISTS idx_homelab_pinned_services_enabled
    ON homelab_pinned_services(tenant_id, enabled) WHERE enabled = true;

-- ---------------------------------------------------------------------------
-- homelab_service_health — Phase 2 (H2 — Service Status).
--
-- Append-only time-series of probe results. Every probe (one-shot
-- via POST /services/:id/probe, batch via POST /services/probe-all,
-- and the 60s background tick) inserts one row here.
--
-- Columns:
--   status        'up' | 'degraded' | 'down' | 'unknown'
--                 (mapped from probe results: HTTP <400=up, 4xx=degraded,
--                 5xx=down, conn refused/timeout=down, icmp-no-reply=down)
--   latency_ms    round-trip time of the probe in milliseconds; nullable
--                 because some probe types (e.g. icmp-no-reply) don't
--                 have a meaningful RTT
--   status_code   HTTP status code returned (only for http/https probes);
--                 nullable for tcp/icmp
--   error_message human-readable failure reason; empty string on success,
--                 populated with the conn error / timeout message on failure
--   checked_at    server timestamp when the probe result was recorded;
--                 default NOW() so the worker can batch-INSERT without
--                 supplying a timestamp
--
-- Cleanup: the worker prunes rows older than 30 days on every tick so
-- the table stays bounded (a user pinning 50 services at 60s cadence
-- = 50 * 1440 = 72,000 rows/day → 2.16M rows/30d; pruning once a day
-- holds it at ~2.16M steady-state).
CREATE TABLE IF NOT EXISTS homelab_service_health (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    service_id      UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'unknown',
    latency_ms      INTEGER,
    status_code     INTEGER,
    error_message   TEXT,
    checked_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The hottest read pattern is "latest health per service_id for
-- user X" — DESC index on (service_id, checked_at) makes the
-- aggregate endpoint O(1) per service. The (tenant_id) index
-- supports the worker's tenant-wide prune query (DELETE WHERE
-- tenant_id = $1 AND checked_at < now() - interval '30 days').
CREATE INDEX IF NOT EXISTS idx_homelab_service_health_service_time
    ON homelab_service_health(service_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_homelab_service_health_tenant
    ON homelab_service_health(tenant_id);
CREATE INDEX IF NOT EXISTS idx_homelab_service_health_user_service
    ON homelab_service_health(user_id, service_id, checked_at DESC);

-- ---------------------------------------------------------------------------
-- homelab_notes — Phase 3 (H3 — Personal Notes + Todos).
--
-- Markdown notes that the user keeps for their homelab. Notes are
-- UNIQUE on (tenant_id, user_id) so layouts and preferences are
-- uniquely scoped PER USER. A user changing their notes in tenant A
-- does not affect another user in the same tenant — this is critical
-- for multi-user homelabs (US-8 in the speckit proposal: "my homelab
-- doesn't change under me when my co-founder rearranges theirs").
--
-- Per-user (not per-tenant) design:
--   The notes table is UNIQUE on (tenant_id, user_id) so notes are
--   uniquely scoped PER USER. ON DELETE CASCADE on tenant_id keeps the
--   per-user invariants intact when a tenant is removed.
--
-- Why text[] for tags:
--   tags uses text[] instead of JSONB because it's a simple,
--   fixed-shape list of tag strings ('homelab', 'urgent', 'todo',
--   ...). text[] supports fast "is X tagged" lookups via the && /
--   @> operators when filtering by tag in GET /homelab/notes.
--
-- Fields:
--   title    — display label shown on the dashboard tile
--   body     — Markdown body (rendered on click in the UI; the
--              backend stores raw text and lets the frontend render
--              it however it wants)
--   tags     — text[] of tag strings
--   pinned   — when true, the note floats to the top of the list
--   updated_at — server timestamp; the UI uses it for relative
--                timestamps ("2 hours ago")
--
-- GET /api/v1/homelab/notes returns the caller's notes with optional
-- filters: ?pinned=true&tag=X&limit=50. PATCH replaces any subset of
-- the fields. DELETE removes the note.
---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_notes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    tags        TEXT[] NOT NULL DEFAULT '{}',
    pinned      BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Supports the "list my notes" query — cheap O(N) scan where N is
-- the user's notes (typically <100). user_id alone is enough; the
-- tenant_id is filtered alongside it in the WHERE clause.
CREATE INDEX IF NOT EXISTS idx_homelab_notes_user
    ON homelab_notes(user_id);

-- Supports the "filter by tag" query — GIN index on the text[]
-- column lets Postgres use the && / @> operators for fast "tagged
-- with X" lookups.
CREATE INDEX IF NOT EXISTS idx_homelab_notes_tags
    ON homelab_notes USING GIN(tags);

-- ---------------------------------------------------------------------------
-- homelab_todos — Phase 3 (H3 — Personal Notes + Todos).
--
-- Todos the user wants to track for their homelab (buy new SSD for
-- NAS, schedule maintenance window, etc.). Per-user (NOT per-tenant)
-- — same US-8 reasoning as notes.
--
-- Fields:
--   title       — display label
--   description — optional long-form description (Markdown allowed
--                 but not required)
--   priority    — 'low' | 'medium' | 'high' | 'urgent'
--                 (validated handler-side via allowedTodoPriorities
--                 whitelist; no CHECK constraint so the enum can
--                 evolve without ALTER TABLE)
--   due_date    — optional timestamptz (when the todo is due). NULL
--                 means no deadline.
--   completed_at — nullable timestamptz. NULL = not done; non-null =
--                 done at that timestamp. Powers the "show me my
--                 completed todos" filter (?completed=true|false).
--   tags        — text[] of tag strings (same as notes)
--
-- Indexes:
--   (user_id, due_date) — powers the "todos due soon" KPI strip
--                         (GET /homelab/todos/due-soon filters
--                         WHERE due_date BETWEEN now AND now+7d).
--   (user_id, completed_at) — powers the "filter by completed"
--                             filter. The handler filters by both
--                             user_id and completed_at in one query.
---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_todos (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    title       TEXT NOT NULL,
    description TEXT,
    priority    TEXT NOT NULL DEFAULT 'medium',
    due_date    TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    tags        TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Powers the "due soon" endpoint (next 7 days) — DESC index so the
-- handler can ORDER BY due_date ASC LIMIT 50 cheaply.
CREATE INDEX IF NOT EXISTS idx_homelab_todos_user_due
    ON homelab_todos(user_id, due_date);

-- Powers the "completed vs active" filter — the handler queries
-- WHERE completed_at IS NULL (active) or WHERE completed_at IS NOT
-- NULL (completed), ordered by completed_at DESC.
CREATE INDEX IF NOT EXISTS idx_homelab_todos_completed
    ON homelab_todos(user_id, completed_at);
-- ---------------------------------------------------------------------------
-- homelab_calendars — Phase 4 (H4 — Calendar).
--
-- Per-user iCal subscriptions. CalendarWorker (internal/homelab/calendar.go)
-- fetches every enabled row's ical_url every 30min, parses with
-- github.com/lukechampine/ical, upserts VEVENTs into homelab_events.
--
-- Fields: name (display label), ical_url (http(s) feed), color (CSS hex
-- validated against allowedCalendarColors), enabled (skip when false),
-- last_synced_at + last_sync_status + last_sync_error (worker bookkeeping;
-- status is free-form so future 'rate_limited'/'auth_required' values
-- don't need an ALTER TABLE).
--
-- Per-user (NOT per-tenant) per speckit proposal §US-8. No UNIQUE on
-- (tenant_id, user_id, name): duplicate names are benign and the
-- enabled flag is the soft-disable mechanism.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_calendars (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL,
    name             TEXT NOT NULL,
    ical_url         TEXT NOT NULL,
    color            TEXT NOT NULL DEFAULT '#3b82f6',
    enabled          BOOLEAN NOT NULL DEFAULT true,
    last_synced_at   TIMESTAMPTZ,
    last_sync_status TEXT,
    last_sync_error  TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Worker's "list all enabled calendars across all tenants" tick.
-- Partial index — only enabled rows are scanned; disabled calendars
-- never bloat the index.
CREATE INDEX IF NOT EXISTS idx_homelab_calendars_user
    ON homelab_calendars(user_id);
CREATE INDEX IF NOT EXISTS idx_homelab_calendars_enabled
    ON homelab_calendars(tenant_id, enabled) WHERE enabled = true;

-- ---------------------------------------------------------------------------
-- homelab_events — Phase 4 (H4 — Calendar).
--
-- Cached events parsed from each calendar's iCal feed. One row per
-- VEVENT; UNIQUE (calendar_id, uid) is the dedup key (iCal UID is
-- publisher-stable across syncs). ON DELETE CASCADE on calendar_id
-- so removing a calendar wipes its events in one statement.
--
-- Fields: uid + summary + description + location + starts_at + ends_at
-- + all_day + raw_ical (kept for debug + future features). No rrule
-- column — most events the publisher expands already; can be added
-- later without breaking existing data.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    user_id      UUID NOT NULL,
    calendar_id  UUID NOT NULL REFERENCES homelab_calendars(id) ON DELETE CASCADE,
    uid          TEXT NOT NULL,
    summary      TEXT NOT NULL,
    description  TEXT,
    location     TEXT,
    starts_at    TIMESTAMPTZ NOT NULL,
    ends_at      TIMESTAMPTZ,
    all_day      BOOLEAN NOT NULL DEFAULT false,
    raw_ical     TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (calendar_id, uid)
);

-- Hot path: "show me my events in [from, to]" — week view + upcoming
-- list endpoint. (user_id, starts_at) covers the common query
-- without a JOIN; (calendar_id, starts_at) covers the worker's
-- "events for this calendar since last sync" query.
CREATE INDEX IF NOT EXISTS idx_homelab_events_user_time
    ON homelab_events(user_id, starts_at);
CREATE INDEX IF NOT EXISTS idx_homelab_events_calendar_time
    ON homelab_events(calendar_id, starts_at);

-- ---------------------------------------------------------------------------
-- homelab_download_clients — Phase 5 (H5 — Download Stats).
--
-- Per-user registry of download clients the user wants one-click
-- stats for. Covers the *arr stack (Sonarr/Radarr/Lidarr/Readarr),
-- torrent (qBittorrent) and Usenet (SABnzbd) — see
-- allowedDownloadClientKinds in handlers_homelab_downloads_types.go
-- for the full list.
--
-- Fields:
--   name         display label (e.g. "Home Sonarr"); unique per
--                (tenant, user) so a user can't double-pin the
--                same name; a sibling user can
--   kind         'sonarr' | 'radarr' | 'qbittorrent' | 'sabnzbd'
--                | 'lidarr' | 'readarr' (worker dispatches by kind)
--   base_url     full http(s) root URL of the client API
--   api_key      bearer / X-Api-Key value; nullable for qBittorrent
--                when the user relies on username+password instead
--   username     qBittorrent login (nullable for *arr + SABnzbd)
--   password     qBittorrent login (nullable for *arr + SABnzbd)
--   enabled      when false, the background worker skips this client
--                but the row is preserved so the user can re-enable
--                without re-entering credentials
--   last_polled_at       server timestamp of the most recent poll
--   last_poll_status     'success' | 'error' | 'unreachable' (free-form
--                        so future 'auth_required' / 'rate_limited'
--                        values don't need an ALTER TABLE)
--   last_poll_error      human-readable error string (trimmed to 200
--                        chars by the worker); empty on success
--
-- Per-user (NOT per-tenant) per speckit proposal §US-8 — two users
-- in the same tenant can each pin their own "Home Sonarr".
--
-- credentials are stored in plaintext — acceptable for a homelab
-- self-hosted tool where the rows are already gated behind
-- tenant_id + user_id; a future hardening pass can wrap these
-- columns with pgcrypto (the *_enc bytea variants) but adds a
-- round-trip for every poll and is out of scope for Tier 10.
---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_download_clients (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id            UUID NOT NULL,
    name               TEXT NOT NULL,
    kind               TEXT NOT NULL,
    base_url           TEXT NOT NULL,
    api_key            TEXT,
    username           TEXT,
    password           TEXT,
    enabled            BOOLEAN NOT NULL DEFAULT true,
    last_polled_at     TIMESTAMPTZ,
    last_poll_status   TEXT,
    last_poll_error    TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id, name)
);

-- Powers the worker's "list all enabled clients across all
-- tenants" tick query. Partial index — only enabled rows are
-- scanned on each 60s tick, so disabled clients never bloat the
-- index.
CREATE INDEX IF NOT EXISTS idx_homelab_download_clients_user
    ON homelab_download_clients(user_id);
CREATE INDEX IF NOT EXISTS idx_homelab_download_clients_enabled
    ON homelab_download_clients(tenant_id, enabled) WHERE enabled = true;

-- ---------------------------------------------------------------------------
-- homelab_download_snapshots — Phase 5 (H5 — Download Stats).
--
-- Append-only time-series of polled state. Every poll (60s tick
-- from DownloadsWorker, plus the immediate fire-and-forget poll
-- fired by POST /downloads/clients after-create) inserts one row
-- here. Dashboard reads the latest snapshot per client to render
-- the KPI strip + per-client stat rows.
--
-- Columns:
--   queue_count           total items in the download queue
--   queue_size_bytes      sum of bytes remaining (queue + active)
--   download_speed_bytes_per_sec  current downstream throughput
--   upload_speed_bytes_per_sec    current upstream throughput
--   today_downloaded_bytes sum of bytes downloaded in the rolling
--                           24h window (or "today" per the client API)
--   today_uploaded_bytes   sum of bytes uploaded in the same window
--   raw_payload           jsonb — full response body from the client
--                         API. Kept for debug + future features
--                         (per-torrent breakdown, history charts).
--                         Sized at the worker's 1 MiB cap so a
--                         pathological payload can't bloat the row.
--   polled_at             server timestamp of the poll
--
-- FK ON DELETE CASCADE on client_id — removing a client wipes its
-- snapshot history in one statement.
--
-- Cleanup: future phases may add retention (e.g. keep 7d); for now
-- the table grows unboundedly but a power user with 6 clients at
-- 60s cadence = 6 * 1440 = 8640 rows/day = ~3.15M rows/year, well
-- within Postgres' comfort zone. A retention sweep can be added
-- to the existing DownloadsWorker tick without schema changes.
---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS homelab_download_snapshots (
    id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                     UUID NOT NULL,
    user_id                       UUID NOT NULL,
    client_id                     UUID NOT NULL REFERENCES homelab_download_clients(id) ON DELETE CASCADE,
    queue_count                   INTEGER NOT NULL DEFAULT 0,
    queue_size_bytes              BIGINT NOT NULL DEFAULT 0,
    download_speed_bytes_per_sec  BIGINT NOT NULL DEFAULT 0,
    upload_speed_bytes_per_sec    BIGINT NOT NULL DEFAULT 0,
    today_downloaded_bytes        BIGINT NOT NULL DEFAULT 0,
    today_uploaded_bytes          BIGINT NOT NULL DEFAULT 0,
    raw_payload                   JSONB,
    polled_at                     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hottest read pattern is "latest snapshot per client" — DESC
-- index on (client_id, polled_at) makes the dashboard's per-client
-- stat row O(1). (user_id, polled_at) supports a future
-- "all clients at time T" tenant-wide query.
CREATE INDEX IF NOT EXISTS idx_homelab_download_snapshots_client_time
    ON homelab_download_snapshots(client_id, polled_at DESC);
CREATE INDEX IF NOT EXISTS idx_homelab_download_snapshots_user_time
    ON homelab_download_snapshots(user_id, polled_at DESC);
