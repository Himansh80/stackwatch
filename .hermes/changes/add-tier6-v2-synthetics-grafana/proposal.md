# Proposal: Tier 6 v2 — Synthetics + Grafana Datasource

## Why
Tier 6 v1 shipped Prom/Loki/Anomaly. Two missing pieces round out the
"monitoring depth" story:
- **M7 Synthetics**: HTTP/TCP/ICMP uptime checks on a schedule. Without
  these, StackWatch can only see what's broken inside a server, not
  whether your public endpoint is even reachable.
- **M4 Grafana datasource**: lets existing Grafana users point at our
  PromQL endpoint. Without this, our Prom support is invisible.

## What changes
- New endpoints under /api/v1/synthetics/* (CRUD + run-now)
- New /api/v1/grafana/query + /api/v1/grafana/search (Prometheus
  datasource protocol adapter so Grafana can connect)
- Migration 009: synthetics_results table (per-run history)
- Background scheduler: ticker runs every 30s, kicks off due checks
- Frontend: 1 page (synthetics list + detail), Grafana config docs

## Impact
- Areas affected: api-gateway, scheduler (in-process), new migration
- Breaking changes: **none** — purely additive
- Migration: 009_synthetics_results.sql
- New package: internal/synthetics (runner + scheduler)
