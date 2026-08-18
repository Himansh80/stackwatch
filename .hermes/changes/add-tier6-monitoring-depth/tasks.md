# Tasks: Tier 6 — Monitoring Depth (v1 scoped)

## 1. Migration
- [ ] 1.1 migrations/008_monitoring_depth.sql
  - ALTER TABLE metric_points ADD COLUMN message TEXT
  - CREATE INDEX for log queries
- [ ] 1.2 Apply migration to .116 (production)

## 2. ML anomaly (M2)
- [ ] 2.1 internal/ml/anomaly.go — rolling z-score + EWMA Detector
- [ ] 2.2 internal/ml/anomaly_test.go — unit tests (5+ scenarios)
- [ ] 2.3 internal/handler/analytics_anomaly.go — REST endpoints

## 3. Prometheus compat (M3)
- [ ] 3.1 internal/handler/analytics_prom.go — write + query
- [ ] 3.2 internal/promql/parser.go — tiny PromQL-lite parser
- [ ] 3.3 internal/promql/eval.go — query evaluator
- [ ] 3.4 internal/promql/parser_test.go — parser unit tests

## 4. Loki push (M6)
- [ ] 4.1 internal/handler/analytics_loki.go — push + query endpoints

## 5. Routes
- [ ] 5.1 cmd/api-gateway/routes.go: register 5+ endpoints under /api/v1/{prom,loki,anomaly}

## 6. Build + deploy
- [ ] 6.1 go build ./... (with new ml, promql packages)
- [ ] 6.2 Cross-compile api-gateway-linux
- [ ] 6.3 scp + restart on .115

## 7. Verifier
- [ ] 7.1 Write hermes-verify-tier6-full-2026-08-17.py
  - Test /prom/write: send 10 samples, verify 10 rows in DB
  - Test /prom/query: send a query, verify matrix response
  - Test /loki/push: send 5 log entries, verify count
  - Test /loki/query: query by job label
  - Test /anomaly/detect: feed synthetic data, verify z-score
  - Test /anomaly/list: verify recent anomalies surface
- [ ] 7.2 Run 5 back-to-back; verify all PASS
- [ ] 7.3 Cross-check T0+T1+T2+T3+T4+T5 still clean

## 8. Frontend (deferred to Tier 12 polish)

## Done criteria
- [ ] All 5+ handler files exist, build clean
- [ ] Migration applied
- [ ] hermes-verify-tier6-full: ≥15 checks, 5/5 back-to-back PASS
- [ ] T0-T5 cross-check: still clean
- [ ] Commit landed
- [ ] Tier 6 marked ✅ in MASTER_BUILD_PLAN.md
