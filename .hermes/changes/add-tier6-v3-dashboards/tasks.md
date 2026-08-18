# Tasks: Tier 6 v3 — Dashboard Builder

## 1. Migration
- [ ] 1.1 migrations/010_dashboards.sql + apply to .116

## 2. Handler
- [ ] 2.1 internal/handler/dashboards.go — CRUD (5 endpoints) + eval + default (2 endpoints)
- [ ] 2.2 Eval endpoint: loop panels, run PromQL, return per-panel data
- [ ] 2.3 Routes in cmd/api-gateway/routes.go

## 3. Build + deploy
- [ ] 3.1 go build ./...
- [ ] 3.2 Cross-compile + restart

## 4. Verifier
- [ ] 4.1 hermes-verify-tier6v3-full-2026-08-17.py
  - Create dashboard with 2 panels
  - List/get/update
  - Eval: returns data for both panels
  - Mark default, list (default first)
  - Delete, verify gone
- [ ] 4.2 5 back-to-back PASS

## 5. Cross-check
- [ ] 5.1 T0+T1+T2+T3+T6+T6v2 still clean

## Done criteria
- [ ] migration applied
- [ ] 8 endpoints registered
- [ ] verifier ≥10 checks, 5/5 PASS
- [ ] no regression in other tiers
- [ ] commit landed
