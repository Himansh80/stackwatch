# Tasks: Tier 6 v4 — Real User Monitoring (M8)

## 1. Migration
- [ ] 1.1 migrations/011_rum_events.sql + apply

## 2. Backend handler
- [ ] 2.1 internal/handler/rum.go — 3 endpoints (events ingest, summary, list)
- [ ] 2.2 Routes in cmd/api-gateway/routes.go
- [ ] 2.3 Note: ingest endpoint may be unprotected (uses tenant_id query param)

## 3. JS snippet
- [ ] 3.1 web/public/rum.js — ~80 LOC, captures page load + errors + longtasks + fetch failures

## 4. Build + deploy
- [ ] 4.1 go build ./...
- [ ] 4.2 Restart api-gateway

## 5. Verifier
- [ ] 5.1 hermes-verify-tier6v4-full-2026-08-17.py
  - POST /rum/events with 5 events of different kinds
  - GET /rum/summary returns aggregates
  - GET /rum/events returns recent events
- [ ] 5.2 5 back-to-back PASS

## 6. Cross-check
- [ ] 6.1 T0+T1+T2+T3+T6+T6v2+T6v3 still clean

## Done criteria
- [ ] migration applied
- [ ] 3 endpoints registered
- [ ] JS snippet ships at web/public/rum.js
- [ ] verifier ≥6 checks, 5/5 PASS
- [ ] no regression
- [ ] commit landed
