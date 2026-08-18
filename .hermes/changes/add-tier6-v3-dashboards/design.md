# Design: Tier 6 v3 — Dashboard Builder (M10)

## Approach

The simplest possible dashboard persistence:
- One table: `dashboards` with `layout` JSONB
- Layout is an array of panel definitions
- Each panel: `{"id": uuid, "type": "timeseries"|"stat"|"table", "title": "...", "query": "...", "grid": {"x":0,"y":0,"w":6,"h":4}}`
- One endpoint evaluates all panels against the current PromQL parser

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Single `dashboards` table with JSONB layout | Simple, no panel table | Can't query across panels (fine for v3) |
| `POST /dashboards/:id/eval` runs all panel queries on demand | No caching complexity | Server load per dashboard view (acceptable for v3) |
| No version history / no sharing | v3 stays small | Future: add `version`, `created_by`, `is_shared` |
| No validation of panel type or query at write time | Lets frontend iterate freely | Bad queries → 400 on eval, not write |

## Implementation details

### Migration `010_dashboards.sql`
```sql
CREATE TABLE dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    layout JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_dashboards_tenant ON dashboards(tenant_id, updated_at DESC);
CREATE UNIQUE INDEX idx_dashboards_default_per_tenant ON dashboards(tenant_id) WHERE is_default;
```

### Handlers (`internal/handler/dashboards.go`)

- `POST   /api/v1/dashboards`              — create (body: name, description, layout)
- `GET    /api/v1/dashboards`              — list for tenant
- `GET    /api/v1/dashboards/:id`          — get one
- `PATCH  /api/v1/dashboards/:id`          — update name/description/layout
- `DELETE /api/v1/dashboards/:id`          — delete
- `POST   /api/v1/dashboards/:id/eval`     — evaluate all panels, return data
- `POST   /api/v1/dashboards/:id/default`   — mark as default
- `DELETE /api/v1/dashboards/:id/default`   — unmark

### Eval endpoint behavior

Request body:
```json
{ "from": <unix_ms>, "to": <unix_ms>, "step": 30000 }
```
Default: last 1h, 30s step.

For each panel in layout:
1. Parse `panel.query` with internal/promql (skip if invalid)
2. Run against metric_points table with the standard PromQL-lite SQL
3. Return `{panel_id, type, title, values, error}` per panel
4. Frontend renders time-series / stat / table from `values`

## Risks
- **R1: Eval endpoint heavy load** (one panel = one DB query). Mitigation: limit panels to 20 per dashboard, time-bound to 10s.
- **R2: JSONB layout can be anything** — bad data on read. Mitigation: validate panel shape at eval time (id, type, query present).
- **R3: Concurrent updates lose data**. Mitigation: optimistic locking via `updated_at` check in PATCH.
