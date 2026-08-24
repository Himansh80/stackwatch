-- Tier 7 Phase 3 — Notebook (D11)
--
-- Two tables back the Datadog Notebook surface:
--   - notebooks:    one row per collaborative markdown runbook / notebook.
--                   title is required (free-text short label), content
--                   holds the markdown body (rendered client-side as a
--                   textarea + simple markdown preview; no rich editor
--                   per spec). author_id is nullable so the row survives
--                   a deleted user. last_edited_at is bumped on every
--                   PUT and used as the default sort key for the list
--                   endpoint.
--   - notebook_collaborators: per-notebook ACL. Role values are enforced
--                             at the API edge ('viewer' | 'editor',
--                             default 'viewer'). The pair (notebook_id,
--                             user_id) is UNIQUE so adding the same
--                             user twice is a no-op (handled by ON
--                             CONFLICT). Cascade on delete cleans up
--                             collaborators when a notebook goes away.
--
-- Both tables tenant-scoped; every query honors tenant_id from JWT.
-- The collaborators table is the only one that doesn't carry tenant_id
-- directly — we resolve tenant ownership through the parent notebook's
-- tenant_id so cross-tenant collab smuggling is impossible.
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.
-- pgcrypto is required for gen_random_uuid(); keep the ensure line
-- even though most installs already have it.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- notebooks: one row per collaborative markdown runbook.
CREATE TABLE IF NOT EXISTS notebooks (
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id       uuid NOT NULL,
  title           text NOT NULL,
  content         text NOT NULL DEFAULT '',
  author_id       uuid,
  last_edited_at  timestamptz NOT NULL DEFAULT now(),
  created_at      timestamptz NOT NULL DEFAULT now()
);
-- Default sort key for the list endpoint is "most recently edited
-- first per tenant". The composite covers the common ?role=… filter
-- because we join through collaborators on user_id.
CREATE INDEX IF NOT EXISTS notebooks_tenant_edited_idx
  ON notebooks (tenant_id, last_edited_at DESC);

-- notebook_collaborators: per-notebook ACL. No tenant_id here —
-- tenant ownership is enforced via JOIN to notebooks in every query.
CREATE TABLE IF NOT EXISTS notebook_collaborators (
  notebook_id  uuid NOT NULL REFERENCES notebooks(id) ON DELETE CASCADE,
  user_id      uuid NOT NULL,
  role         text NOT NULL DEFAULT 'viewer',
  added_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (notebook_id, user_id)
);
-- Reverse lookup: "which notebooks am I a collaborator on?" — the
-- Shared-with-me view in the UI does this JOIN every render.
CREATE INDEX IF NOT EXISTS notebook_collaborators_user_idx
  ON notebook_collaborators (user_id);
