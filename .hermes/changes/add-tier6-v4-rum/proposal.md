# Proposal: Tier 6 v4 — Real User Monitoring (M8)

## Why
RUM captures browser-side performance metrics from real users:
page load time, JS errors, slow network requests, etc. Even without
external customers, this is useful for monitoring our OWN dashboard's
performance. It also lays the groundwork for M5 distributed tracing
(same instrumentation pattern).

## What changes
- New `rum_events` table (id, tenant_id, ts, kind, value, url, attrs JSONB)
- Endpoints: POST /api/v1/rum/events (ingest), GET /api/v1/rum/summary
- New JS snippet at web/public/rum.js (browser-side collector, ~80 LOC)
- Document how to add the snippet to any HTML page

## Impact
- Areas affected: api-gateway, web/public
- Breaking changes: **none**
- Migration: 011_rum_events.sql
