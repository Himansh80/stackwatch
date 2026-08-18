# Tasks: Tier 6 v2 — Synthetics + Grafana

## 1. Migration
- [ ] 1.1 migrations/009_synthetics_results.sql
- [ ] 1.2 Apply to .116

## 2. Synthetics runner
- [ ] 2.1 internal/synthetics/runner.go — Run(check) for HTTP/TCP/ICMP
- [ ] 2.2 internal/synthetics/runner_test.go — 5+ unit tests

## 3. Synthetics scheduler
- [ ] 3.1 internal/synthetics/scheduler.go — ticker every 30s
- [ ] 3.2 Bounded concurrency (semaphore)

## 4. Synthetics handlers
- [ ] 4.1 internal/handler/synthetics.go — CRUD + run-now + results
- [ ] 4.2 Routes in routes.go

## 5. Start scheduler on api-gateway boot
- [ ] 5.1 cmd/api-gateway/main.go: spawn scheduler goroutine

## 6. Grafana datasource
- [ ] 6.1 internal/handler/grafana.go — /grafana/search + /grafana/query
- [ ] 6.2 Routes in routes.go

## 7. Build + deploy
- [ ] 7.1 go build ./...
- [ ] 7.2 Cross-compile + scp + restart

## 8. Verifier
- [ ] 8.1 hermes-verify-tier6v2-full-2026-08-17.py
  - synthetics: create check, list, run-now, results
  - grafana: search returns list, query returns matrix
  - cross-tier regression: T0-T6 still pass
- [ ] 8.2 5 back-to-back PASS

## Done criteria
- [ ] migration applied
- [ ] All handlers + routes registered
- [ ] 5+ unit tests for runner
- [ ] verifier: ≥10 checks, 5/5 back-to-back PASS
- [ ] T0-T6 cross-check still clean
- [ ] Commit landed
