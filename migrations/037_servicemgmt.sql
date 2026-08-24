-- Tier 7 Phase 3 — Service Management (D10)
--
-- Four tables back the Datadog incident-management surface:
--   - incidents:    a single sev1/sev2/sev3/sev4 event. Lifecycle:
--                   open → acknowledged (commander assigned) → resolved.
--                   resolved_at is set on resolve; commander_id is set on
--                   acknowledge (from the JWT's user_id).
--   - war_rooms:    per-incident comms channel (Slack-style). participants
--                   is jsonb so we can stash user ids + roles without a
--                   separate table for Phase 3.
--   - postmortems:  blameless write-up attached to a resolved incident.
--                   content is plain text for now (markdown render happens
--                   client-side); published_at is null until author marks
--                   it ready for the team.
--   - tasks:        action items, optionally attached to an incident.
--                   incident_id is nullable so tasks can also stand alone
--                   (on-call todos with no incident yet).
--
-- All tables tenant-scoped; every query honors tenant_id from JWT.
-- FKs cascade so deleting an incident cleans up its war rooms,
-- postmortems, and tasks.
--
-- Idempotent: every CREATE uses IF NOT EXISTS so re-running is safe.
-- pgcrypto is required for gen_random_uuid(); keep the ensure line
-- even though most installs already have it.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- incidents: one row per declared incident.
-- Severity values (enforced at API edge):
--   'sev1' | 'sev2' | 'sev3' | 'sev4'
-- Status values (enforced at API edge):
--   'open' | 'acknowledged' | 'resolved'
CREATE TABLE IF NOT EXISTS incidents (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id     uuid NOT NULL,
  title         text NOT NULL,
  description   text NOT NULL DEFAULT '',
  severity      text NOT NULL DEFAULT 'sev3',
  status        text NOT NULL DEFAULT 'open',
  commander_id  uuid,
  started_at    timestamptz NOT NULL DEFAULT now(),
  resolved_at   timestamptz,
  postmortem_id uuid
);
CREATE INDEX IF NOT EXISTS incidents_tenant_status_started_idx
  ON incidents (tenant_id, status, started_at DESC);
CREATE INDEX IF NOT EXISTS incidents_tenant_severity_idx
  ON incidents (tenant_id, severity);

-- war_rooms: one row per incident (we keep history by soft-creating a
-- new row only if a second channel is opened; the simple model is
-- "one war room per incident" which matches Datadog's default).
CREATE TABLE IF NOT EXISTS war_rooms (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL,
  incident_id  uuid NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  channel_url  text NOT NULL DEFAULT '',
  participants jsonb NOT NULL DEFAULT '[]'::jsonb,
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS war_rooms_tenant_incident_idx
  ON war_rooms (tenant_id, incident_id);

-- postmortems: blameless write-up attached to one incident.
CREATE TABLE IF NOT EXISTS postmortems (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id    uuid NOT NULL,
  incident_id  uuid NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  content      text NOT NULL,
  author_id    uuid,
  published_at timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS postmortems_tenant_incident_idx
  ON postmortems (tenant_id, incident_id);

-- tasks: action items, optionally tied to an incident.
-- Status values (enforced at API edge):
--   'todo' | 'in_progress' | 'done'
CREATE TABLE IF NOT EXISTS tasks (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id   uuid NOT NULL,
  incident_id uuid REFERENCES incidents(id) ON DELETE CASCADE,
  title       text NOT NULL,
  assignee_id uuid,
  status      text NOT NULL DEFAULT 'todo',
  due_at      timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS tasks_tenant_status_idx
  ON tasks (tenant_id, status);
CREATE INDEX IF NOT EXISTS tasks_tenant_incident_idx
  ON tasks (tenant_id, incident_id);
