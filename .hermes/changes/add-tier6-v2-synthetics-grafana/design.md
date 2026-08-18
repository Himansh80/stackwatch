# Design: Tier 6 v2 — Synthetics + Grafana Datasource

## Approach

### M7 Synthetics

The `synthetics_checks` table already exists (migration 008). Add:

**Migration 009:** `synthetics_results` table (one row per check run)
```sql
CREATE TABLE synthetics_results (
    id BIGSERIAL PRIMARY KEY,
    check_id UUID NOT NULL REFERENCES synthetics_checks(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status TEXT NOT NULL,           -- 'success' | 'failure' | 'timeout'
    response_ms INT,                -- HTTP RTT or TCP connect time
    status_code INT,                -- HTTP status (NULL for TCP/ICMP)
    error TEXT,                     -- error message if failure
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sr_check_ts ON synthetics_results (check_id, ts DESC);
CREATE INDEX idx_sr_tenant_ts ON synthetics_results (tenant_id, ts DESC);
```

**Package: `internal/synthetics`**
- `runner.go` — `Run(check)` returns a result for HTTP/TCP/ICMP
  - HTTP: `http.Client.Do(req)` with timeout, capture status + RTT
  - TCP: `net.DialTimeout("tcp", host:port, timeout)`, capture RTT
  - ICMP: `exec.Command("ping", "-c", "1", "-W", "1", host)` (best-effort)
- `scheduler.go` — ticker that wakes every 30s, scans checks where
  `now - last_run_at >= interval_sec`, runs due checks, persists results

**Endpoints (`internal/handler/synthetics.go`)**
- `POST /api/v1/synthetics` — create check (body: name, kind, target, interval_sec, timeout_ms)
- `GET /api/v1/synthetics` — list checks for tenant
- `GET /api/v1/synthetics/:id` — get one check
- `PATCH /api/v1/synthetics/:id` — update (name, interval_sec, enabled, etc.)
- `DELETE /api/v1/synthetics/:id` — delete (cascades to results)
- `POST /api/v1/synthetics/:id/run` — run now, return latest result
- `GET /api/v1/synthetics/:id/results?since=24h` — result history

**Scheduler start**: in `cmd/api-gateway/main.go` after `buildRouter`,
spawn a goroutine `go synthetics.RunScheduler(ctx, pool, 30*time.Second)`.

### M4 Grafana Datasource

Grafana's Prometheus datasource speaks a tiny JSON protocol:
- `GET /api/v1/grafana/search?query=<prefix>` → list of metric names matching prefix
- `POST /api/v1/grafana/query` (body: `{query, from, to, step}`) → same as `/prom/query`

**Endpoints** (`internal/handler/grafana.go`)
- Reuse PromQL parsing via `internal/promql` package
- `/api/v1/grafana/search` returns `[]string` of metric names
- `/api/v1/grafana/query` returns Prometheus matrix response

Search implementation: SELECT DISTINCT metric_name FROM metric_points
WHERE tenant_id = $1 AND metric_name LIKE $2 LIMIT 100.

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Scheduler in-process (goroutine in api-gateway) | One binary to deploy | api-gateway restart loses scheduler state (DB has it) |
| ICMP via exec.Command("ping") | Simple, no Go ICMP lib | Requires CAP_NET_RAW for unprivileged pings (root OK on .115) |
| Synthetics_results separate from metric_points | Cleaner schema | One more table; we DON'T insert into metric_points for v2 (could later) |
| Grafana adapter reuses /prom query | Zero new code, just renames endpoint | Need to add /grafana/search which uses different SQL |
| No auth on Grafana endpoints for v2 | Grafana uses basic auth in its datasource config — we use JWT | Document: set Authorization: Bearer <api_key> in Grafana config |

## Risks
- **R1: ping requires CAP_NET_RAW** — root has it. If running as non-root in future, ICMP fails silently. Mitigation: detect + fall back to TCP port probe.
- **R2: Scheduler goroutine on api-gateway restart** — picks up where it left off via `last_run_at` check. Mitigation: idempotent — re-running a check that just ran is fine.
- **R3: HTTP checks leak goroutines** if many checks are due at once. Mitigation: bounded concurrency (semaphore of 10).
