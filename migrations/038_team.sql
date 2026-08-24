-- Tier 7 Phase 3 — Team Collaboration (D12)
--
-- Three tables back the D12 collaboration surface:
--   - dashboard_shares: per-user share grants on a dashboard. The
--                        (dashboard_id, user_id) pair is UNIQUE so
--                        re-sharing the same user is idempotent (the
--                        UI calls onPermissionChange → ON CONFLICT
--                        flips the column to its new value). The
--                        dashboard_id FK CASCADEs so deleting a
--                        dashboard cleans its shares; user deletion
--                        intentionally does NOT cascade — sharing a
--                        dashboard with a deleted user is treated as
--                        "ghost share" and surfaced in the UI as a
--                        revoked-but-retained audit row.
--   - timeline_comments: free-text comments attached to an incident
--                        timeline. incident_id is nullable so comments
--                        can outlive an incident (used as a long-term
--                        audit row) — when incident_id IS set it
--                        CASCADEs. Every query joins through
--                        tenant_id so a comment from tenant A cannot
--                        leak into tenant B even if both reference the
--                        same incident_id (e.g. in a federated/shared
--                        setup).
--   - mentions: per-user "you've been @-mentioned" inbox row. The
--               table is tenant-scoped so a single global "my
--               mentions" query can hit a covering index. The
--               (mentioned_user_id, read_at IS NULL) index powers
--               the unread-badge KPI on the Shared dashboards page
--               and the sidebar bell.
--
-- All three tables:
--   - Use CREATE TABLE IF NOT EXISTS so re-running is safe.
--   - Use gen_random_uuid() — pgcrypto must be installed.
--   - Tenant-scoped wherever a natural place exists.
--   - Have created_at defaulted to now() so list endpoints can
--     ORDER BY without an explicit timestamp.
--
-- This migration is independent of 038_notebook.sql; both can be
-- applied in any order because they touch disjoint table names.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---- dashboard_shares ---------------------------------------------
-- Per-(dashboard, user) share grant. Default permission 'view';
-- 'edit' lets the recipient save dashboard state. UNIQUE on
-- (dashboard_id, user_id) so duplicate POSTs are no-ops via ON
-- CONFLICT. Index on user_id alone supports the "dashboards shared
-- with me" join without a full scan.
CREATE TABLE IF NOT EXISTS dashboard_shares (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  dashboard_id uuid NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
  user_id      uuid NOT NULL,
  permission   text NOT NULL DEFAULT 'view',
  created_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (dashboard_id, user_id)
);
CREATE INDEX IF NOT EXISTS dashboard_shares_user_idx
  ON dashboard_shares (user_id);

-- ---- timeline_comments --------------------------------------------
-- Comments on an incident timeline. incident_id is nullable so
-- comments can survive an incident deletion (audit-trail mode).
-- FK CASCADE on incident_id, but tenant_id is the primary filter —
-- the composite (tenant_id, incident_id, created_at DESC) index
-- covers the common "give me the timeline for THIS incident" read.
CREATE TABLE IF NOT EXISTS timeline_comments (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL,
  incident_id uuid REFERENCES incidents(id) ON DELETE CASCADE,
  user_id     uuid NOT NULL,
  body        text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS timeline_comments_tenant_incident_idx
  ON timeline_comments (tenant_id, incident_id, created_at DESC);

-- ---- mentions ----------------------------------------------------
-- Per-mention inbox row. context_type tells the UI which surface to
-- deep-link to ('incident' | 'notebook' | 'dashboard' | 'comment');
-- context_id is the row id in that surface's table. The partial
-- index on (tenant_id, mentioned_user_id, read_at IS NULL) powers
-- the "unread count" badge query without scanning read rows.
CREATE TABLE IF NOT EXISTS mentions (
  id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id          uuid NOT NULL,
  mentioned_user_id  uuid NOT NULL,
  mentioning_user_id uuid NOT NULL,
  context_type       text NOT NULL,
  context_id         uuid NOT NULL,
  read_at            timestamptz,
  created_at         timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS mentions_user_unread_idx
  ON mentions (tenant_id, mentioned_user_id, (read_at IS NULL));
