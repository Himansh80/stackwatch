# Proposal: Tier 6 v3 — Dashboard Builder (M10)

## Why
Tier 6 v1+v2 shipped metrics + logs + synthetics + anomaly detection.
But there's no way to save custom dashboards — every time you reload,
your queries are gone. Power users want persistent dashboards with
named panels.

## What changes
- New `dashboards` table (id, tenant_id, name, layout JSONB, timestamps)
- New endpoints under /api/v1/dashboards/* (CRUD + evaluate)
- Frontend: 1 page (list + view) — drag-drop UI deferred to v3.5
- Use existing promql package to evaluate panels server-side

## Impact
- Areas affected: api-gateway (new handlers), new migration
- Breaking changes: **none**
- Migration: 010_dashboards.sql
- New package: internal/dashboards (handler + maybe a tiny panel registry)
