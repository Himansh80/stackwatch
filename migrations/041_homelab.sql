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
--   Phase 4 (H4 — Calendar):
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
-- data migration is needed for Phase 1 — the tables start empty.
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