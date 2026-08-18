# Proposal: Tier 6 — Monitoring Depth (Netdata + Prom + Grafana + Loki)

## Why
StackWatch today stores per-server metrics every 30s. To compete with
Netdata/Prometheus/Grafana/Loki, operators need:
- Per-second metrics (not 30s) for fast incident response
- Prometheus remote_write compatibility (so existing Prom exporters work)
- PromQL queries against stored metrics
- Loki-compatible log API (so existing Promtail/Vector setups work)
- ML anomaly detection (Z-score per metric per host)
- Distributed tracing (OTLP)
- Synthetic checks (HTTP/TCP/ICMP on a schedule)
- RUM for customer sites
- Dashboard builder (drag-and-drop widgets)

## What changes
- New `cmd/analytics/main.go` binary (port :8090) — analytics sidecar
  that owns: PromQL, anomaly, synthetics, Loki, OTLP receiver
- New `pkg/ml/anomaly.go` — z-score + EWMA + k-means clustering
- New endpoints under `/api/v1/analytics/*` (PromQL, anomaly CRUD,
  synthetics CRUD, OTLP, Loki push)
- New RUM ingest endpoints under `/api/v1/rum/*`
- New dashboard builder persistence + endpoints under `/api/v1/dashboards/*`
- New migration `migrations/008_monitoring_depth.sql` (metric_points
  partition by time, traces, logs, rum_events, synthetics tables)
- Frontend: dashboard builder, anomaly viz, trace flame graph

## Impact
- Areas affected: api-gateway (proxy to analytics sidecar), new
  cmd/analytics sidecar, agent (per-second collector), frontend
- Breaking changes: **none** — Tier 6 is purely additive
- Migration needed: **yes** — partitioning + 4 new tables
- Server impact: agent emits every 1s instead of 30s → ~30x more
  metrics data. Mitigation: per-host downsampling (keep 1s for 1h,
  30s for 30d, 5m for 1y).
