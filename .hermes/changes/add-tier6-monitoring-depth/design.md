# Design: Tier 6 — Monitoring Depth (scoped v1)

## Scope (v1)

We have 10 sub-features in the plan. Shipping all 10 is multi-session
work. **Tier 6 v1** ships the three with the highest impact-to-LOC
ratio:

| Sub | What | Endpoints | LOC |
|-----|------|-----------|-----|
| **M3 Prometheus compat** | remote_write + PromQL query | `/api/v1/prom/write`, `/api/v1/prom/query` | ~200 |
| **M6 Loki push** | JSON ingest compatible with Promtail/Vector | `/api/v1/loki/push` | ~80 |
| **M2 ML anomaly** | Z-score + EWMA per metric, query anomalies | `/api/v1/anomaly/list`, `/api/v1/anomaly/detect` | ~150 |

**Deferred to v2** (each = separate session):
- M1 per-second metrics + DB partitioning
- M4 Grafana datasource
- M5 distributed tracing (OTLP)
- M7 synthetics
- M8 RUM
- M9 network deep inspection
- M10 dashboard builder

## Approach (all three)

Each sub-feature is **stateless** except for the query layer. Write
paths (Prom remote_write, Loki push) are pure ingest — store data,
no joins. Query paths (PromQL, anomaly) read from existing
`metric_points` table. No new sidecar binary needed for v1; we
can extend the api-gateway.

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Add to api-gateway, no new sidecar | Existing handler pattern; same auth + tenant scoping | Larger binary (still under 25MB) |
| Reuse `metric_points` table for Prom storage | Already has (tenant_id, server_id, metric_name, ts, value) | No native Prom labels (we add them later if needed) |
| PromQL subset (not full) | Only support: `metric_name`, `avg()`, `max()`, `min()`, `rate()`, basic `=` filters | No labels, regex, joins, subqueries — good enough for v1 |
| Anomaly: rolling Z-score (window=20) | Standard; works on streaming data | Needs ~20 samples before first score |
| Loki: store in `metric_points` with `metric_name = "log"` and `value = 1` | One table, one ingest path | No free-text search; v1 only counts logs per host per minute |

## Implementation details

### M3 Prometheus compat (PromQL subset)

**`POST /api/v1/prom/write`** — accepts Prometheus remote_write protobuf
OR JSON (we expose both for ease). For v1 we accept JSON only:
```
{
  "timeseries": [
    {
      "labels": {"metric": "cpu_pct", "host_id": "...", "tenant_id": "..."},
      "samples": [[<unix_ms>, "<value_str>"]]
    }
  ]
}
```
For each sample: INSERT INTO metric_points (tenant_id, server_id,
metric_name, value, ts) where metric_name = labels["metric"].

**`POST /api/v1/prom/query`** — accepts a PromQL-lite query:
```
{
  "query": "avg(cpu_pct)",
  "from": <unix_ms>,
  "to": <unix_ms>,
  "step": 30000,
  "host_id": "..."  // optional
}
```
Parse the query with a tiny hand-rolled parser supporting:
- Bare metric: `cpu_pct` → SELECT avg(value) ... GROUP BY bucket(ts, step)
- avg/min/max/sum/count metric
- `metric{host_id="..."}` filter
- time range from/to with step bucket

Return Prometheus-compatible JSON: `{status: "success", data: {resultType: "matrix", result: [...]}}`

### M6 Loki push

**`POST /api/v1/loki/push`** — accepts Loki-compatible JSON:
```
{
  "streams": [
    {
      "labels": "{job=\"syslog\"}",
      "entries": [{"ts": <ns>, "line": "..."}]
    }
  ]
}
```
For each entry: INSERT INTO metric_points with metric_name="log",
value=1, plus a new column for the message (we add `message TEXT`
to metric_points via migration). Aggregation: SELECT count(*)
GROUP BY server_id, bucket(ts, 60s).

### M2 ML anomaly

**`internal/ml/anomaly.go`** — streaming z-score per (tenant, server,
metric):
```
type Detector struct {
    Window  int            // 20 samples
    EwmaAlpha float64       // 0.3
    Values  []float64       // rolling window
    Mean    float64
    Stddev  float64
}
func (d *Detector) Update(v float64) (zscore float64, isAnomaly bool)
```

**`GET /api/v1/anomaly/list?host_id=X&metric=cpu_pct&since=1h`** —
returns recent anomalies (z-score > 3).

**`POST /api/v1/anomaly/detect`** — synchronous detection: given the
last 20 metric values, return current z-score + anomaly flag. Useful
for testing without waiting for streaming.

## Migration

```sql
-- migrations/008_monitoring_depth.sql
ALTER TABLE metric_points ADD COLUMN message TEXT;
CREATE INDEX IF NOT EXISTS idx_metric_points_message ON metric_points (tenant_id, metric_name, ts) WHERE metric_name = 'log';
```

(Partitioning + new tables deferred to v2.)

## Risks
- **R1: PromQL subset parser is fragile**. Mitigation: hand-rolled parser with strict syntax validation (reject unknown functions with 400).
- **R2: Anomaly Z-score assumes normal distribution**. Mitigation: EWMA + absolute deviation fallback for highly-skewed metrics.
- **R3: Loki push is just counters, not full-text search**. Mitigation: document as known limitation.
- **R4: prometheus remote_write protobuf** is complex. v1 accepts JSON only; protobuf in v2 if needed.
