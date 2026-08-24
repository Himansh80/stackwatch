# Implementation Plan: Tier 8 — Intelligence & Alerting

**Feature**: 006-tier8-intelligence-alerting
**Status**: Draft

## Approach

Six phases. Phase 0 first (modular split + foundation) so Phase 1-5 can each add their routes without hitting the 400 cap. Then 5 subtier phases (anomaly, predict, correlations, noise, dashboard), each end-to-end. Final phase: verify + archive.

## Phase 0 — Modular pre-work + foundation

**Why**: routes_protected.go is currently 380 LOC. Adding 24 new routes would push it past 400. We need a split BEFORE Phase 1.

Tasks:
- 0.1 Extract `cmd/api-gateway/routes_intelligence.go` (~60 LOC) — placeholder `mountIntelligenceRoutes()` that returns nil. We'll add the actual route registrations in Phase 1+ as we go.
- 0.2 Modify `cmd/api-gateway/routes_protected.go` — remove the (empty) placeholder registration. Add a comment explaining that intelligence routes will be added in Phases 1-5.
- 0.3 Verify `go build ./cmd/api-gateway` and `go vet` still exit 0
- 0.4 Live check: existing routes still return 401.

After Phase 0: routes_protected.go drops slightly (placeholder removed); routes_intelligence.go is empty waiting for Phase 1.

## Phase 1 — ML Anomaly Detection (Tier 8.1)

Tasks (per tasks.md):
1.1 Migration `migrations/039_intelligence.sql` — anomaly_models + anomaly_events tables
1.2 Apply migration to prod
1.3 Handler `handlers_anomaly.go` (~250 LOC, 5 routes)
1.4 Register 5 routes via `mountIntelligenceRoutes` (Phase 1 routes only)
1.5 Shared `AnomalyChart.tsx` (~100 LOC, Datadog-style sparkline)
1.6 Page section: top section of IntelligencePage with anomaly KPI strip
1.7 Verify + commit (Phase 1 done — `feat(tier8): ML Anomaly Detection`)

Note: We will BUILD the IntelligencePage incrementally. Phase 1 adds the top section (anomalies), Phase 5 unifies it.

## Phase 2 — Predictive Alerting (Tier 8.2)

Tasks:
2.1 Migration `039_predictive.sql` — predictive_alerts + forecast_history tables (separate migration or part of 039, your call)
2.2 Handler `handlers_predict.go` (~200 LOC, 4 routes)
2.3 Register 4 routes
2.4 Shared `PredictionChart.tsx` (~100 LOC, Datadog-style prediction band with p10/p50/p90)
2.5 Add predict section to IntelligencePage
2.6 Verify + commit (Phase 2 done — `feat(tier8): Predictive Alerting`)

## Phase 3 — Alert Correlation + RCA (Tier 8.3)

Tasks:
3.1 Migration `039_correlations.sql` — alert_correlations + rca_hints + correlation_feedback tables
3.2 Handler `handlers_correlations.go` (~250 LOC, 5 routes)
3.3 Register 5 routes
3.4 Shared `CorrelationCard.tsx` (~100 LOC, Datadog-style correlation cluster)
3.5 Add correlation section to IntelligencePage
3.6 Verify + commit (Phase 3 done — `feat(tier8): Alert Correlation + RCA`)

## Phase 4 — Alert Deduplication + Noise Reduction (Tier 8.4)

Tasks:
4.1 Migration `039_noise.sql` — alert_noise_rules + snooze_log tables
4.2 Handler `handlers_noise.go` (~200 LOC, 5 routes)
4.3 Register 5 routes
4.4 Shared `NoiseRuleEditor.tsx` (~80 LOC, simple form)
4.5 Add noise section to IntelligencePage
4.6 Verify + commit (Phase 4 done — `feat(tier8): Alert Noise Reduction`)

## Phase 5 — Intelligence Dashboard (Tier 8.5) — Final

Tasks:
5.1 Build the unified `IntelligencePage.tsx` (~200 LOC, 4 tabs: Anomalies / Predictions / Correlations / Noise)
5.2 Shared `RcaPanel.tsx` (~80 LOC, top-5 root cause summary)
5.3 Export endpoint: `GET /api/v1/intelligence/export` — JSON download of last 7 days of data
5.4 AppSidebar: add "Intelligence" entry in observability section
5.5 Verify all routes live + final commit (Phase 5 done — `feat(tier8): Intelligence Dashboard`)

## Phase 6 — Final verify + deploy + archive

Tasks:
6.1 Final go build + npm run type-check + lint + build all green
6.2 Bundle JS gzipped ≤ baseline + 70 KB
6.3 Deploy binary to `.115`
6.4 Live 4× verifier PASS
6.5 Archive to `.hermes/changes/archive/2026-08-24-006-tier8-intelligence-alerting/`
6.6 Update journal.md with TIER 8 COMPLETE marker
6.7 Tier 8 ready for Tier 9 (Security & Enterprise)

## Risks

| Risk | Mitigation |
|------|------------|
| routes_protected.go overflow | Phase 0 split (routes_intelligence.go) BEFORE Phase 1 starts |
| ML model training is slow | Use Welford's algorithm (O(1) updates) |
| Predictive forecasting accuracy | Show MAPE + RMSE; mark low-confidence as experimental |
| Correlation could be wrong | Feedback endpoint for users to override |
| Noise rules could hide real problems | Log every suppressed event to audit_log |

## Done criteria

- [ ] Phase 0: routes_intelligence.go created, placeholder ready
- [ ] Phases 1-5: each subtier passes verification + commits cleanly
- [ ] All 24 routes registered
- [ ] All files < 400 LOC
- [ ] Tier 8 COMPLETE marker in journal
- [ ] Archive created
- [ ] Ready for Tier 9 (Security & Enterprise)