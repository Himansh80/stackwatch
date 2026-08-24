# Tasks: Tier 8 — Intelligence & Alerting

**Feature**: 006-tier8-intelligence-alerting
**Spec**: [spec.md](spec.md)
**Status**: Draft

## Phase 0 — Modular pre-work

- [ ] 0.1 Create `cmd/api-gateway/routes_intelligence.go` (~60 LOC) — empty placeholder `mountIntelligenceRoutes()` function that returns nil
- [ ] 0.2 Modify `cmd/api-gateway/routes_protected.go` — add a comment line near end: "Intelligence routes registered via mountIntelligenceRoutes() in routes_intelligence.go (Phases 1-5)"
- [ ] 0.3 Verify `go build ./cmd/api-gateway` exit 0
- [ ] 0.4 Verify `go vet ./cmd/api-gateway` exit 0
- [ ] 0.5 Confirm all existing routes still return 401 (no regression)

## Phase 1 — ML Anomaly Detection (Tier 8.1)

- [ ] 1.1 Migration `migrations/039_intelligence.sql` — table `anomaly_models` (id, tenant_id, metric_name, server_id nullable, model_type ('welford'|'ewma'), mean, variance, ewma, last_trained_at, sample_count, UNIQUE on (tenant_id, metric_name, server_id)). Index on (tenant_id, metric_name).
- [ ] 1.2 Migration `migrations/039_intelligence.sql` — table `anomaly_events` (id, tenant_id, model_id FK CASCADE, server_id nullable, metric_name, anomaly_score (real), severity ('info'|'warning'|'critical'), observed_value (real), expected_range_low (real), expected_range_high (real), ts, acknowledged (bool default false), ack_note (text), ack_user_id (uuid nullable). Index on (tenant_id, ts DESC), Index on (tenant_id, severity, ts DESC).
- [ ] 1.3 Apply migration to prod DB on `.116`
- [ ] 1.4 Handler `internal/handler/handlers_anomaly.go` (~250 LOC, 5 routes):
  - `GET /api/v1/anomaly/models` — list trained models
  - `POST /api/v1/anomaly/train` — train a model (body: {metric_name, server_id nullable, model_type default 'welford'})
  - `POST /api/v1/anomaly/detect` — detect anomalies now (body: {metric_name, server_id nullable, value, threshold_sigma default 3.0})
  - `GET /api/v1/anomaly/events` — list recent events (filter ?server_id=X&severity=Y&limit=50)
  - `POST /api/v1/anomaly/ack` — acknowledge an event (body: {event_id, note})
- [ ] 1.5 Register 5 routes via `mountIntelligenceRoutes` in `routes_intelligence.go`
- [ ] 1.6 Shared component `web/src/components/shared/AnomalyChart.tsx` (~100 LOC):
  - Props: `{events: Array<{ts, anomaly_score, observed_value}>}`
  - Layout: sparkline showing anomaly_score over time (red points = critical, amber = warning, green = info)
  - Hover shows timestamp + score + value
- [ ] 1.7 Add anomaly section to `web/src/pages/IntelligencePage.tsx` (top section, KPI strip: anomalies today / critical count / acknowledged count)
- [ ] 1.8 Verify all gates green (go + npm)
- [ ] 1.9 Commit Phase 1 — `feat(tier8): ML Anomaly Detection`

## Phase 2 — Predictive Alerting (Tier 8.2)

- [ ] 2.1 Migration `migrations/039_predictive.sql` — table `predictive_alerts` (id, tenant_id, metric_name, server_id nullable, predicted_value (real), predicted_breach_at (timestamptz), confidence (real), severity, status ('open'|'acknowledged'|'resolved'), ack_user_id, created_at). Index on (tenant_id, status, predicted_breach_at).
- [ ] 2.2 Migration `migrations/039_predictive.sql` — table `forecast_history` (id, tenant_id, metric_name, server_id nullable, model_type ('linear_regression'|'exponential_smoothing'|'holt_winters'), mape (real), rmse (real), evaluated_at). Index on (tenant_id, metric_name).
- [ ] 2.3 Apply migration to prod
- [ ] 2.4 Handler `internal/handler/handlers_predict.go` (~200 LOC, 4 routes):
  - `POST /api/v1/predict/forecast` — generate forecast (body: {metric_name, server_id nullable, horizon_hours})
  - `GET /api/v1/predict/alerts` — list predictive alerts (filter ?status=open&severity=Y)
  - `GET /api/v1/predict/accuracy` — model accuracy for metric (body or query: {metric_name, server_id nullable})
  - `POST /api/v1/predict/ack` — acknowledge predictive alert (body: {alert_id, note})
- [ ] 2.5 Register 4 routes
- [ ] 2.6 Shared component `web/src/components/shared/PredictionChart.tsx` (~100 LOC):
  - Props: `{historical: Array<{ts, value}>, forecast: Array<{ts, p10, p50, p90}>}`
  - Layout: line chart with historical solid line + forecast dashed line + shaded p10-p90 confidence band (light accent color)
  - Hover shows timestamp + all 3 percentiles
- [ ] 2.7 Add predict section to IntelligencePage (KPI strip + PredictionChart for top 3 metrics)
- [ ] 2.8 Verify all gates green
- [ ] 2.9 Commit Phase 2 — `feat(tier8): Predictive Alerting`

## Phase 3 — Alert Correlation + RCA (Tier 8.3)

- [ ] 3.1 Migration `migrations/039_correlations.sql` — table `alert_correlations` (id, tenant_id, correlation_id (uuid — group id), root_alert_id (uuid), member_alert_ids (uuid[]), similarity_score (real), created_at, auto_detected (bool default true)). Index on (tenant_id, correlation_id).
- [ ] 3.2 Migration `migrations/039_correlations.sql` — table `rca_hints` (id, tenant_id, alert_id (uuid), likely_root (text), confidence (real), reasoning (text), similar_past_incidents (jsonb default '[]'), created_at). Index on (tenant_id, alert_id).
- [ ] 3.3 Migration `migrations/039_correlations.sql` — table `correlation_feedback` (id, tenant_id, correlation_id (uuid), user_id (uuid), useful (bool), note (text), created_at). Index on (tenant_id, correlation_id).
- [ ] 3.4 Apply migration to prod
- [ ] 3.5 Handler `internal/handler/handlers_correlations.go` (~250 LOC, 5 routes):
  - `GET /api/v1/correlations/groups` — list correlation groups (filter ?auto_detected=true)
  - `GET /api/v1/correlations/group/:id` — group detail with all member alerts
  - `POST /api/v1/correlations/manual` — create manual correlation (body: {alert_ids, reason})
  - `GET /api/v1/correlations/rca/:alert_id` — RCA hints for an alert
  - `POST /api/v1/correlations/feedback` — feedback on a correlation (body: {correlation_id, useful, note})
- [ ] 3.6 Register 5 routes
- [ ] 3.7 Shared component `web/src/components/shared/CorrelationCard.tsx` (~100 LOC):
  - Props: `{group: {id, root_alert, member_count, similarity_score, auto_detected, member_alerts}}`
  - Layout: card with: root alert badge + member count + similarity score + list of member alerts (collapsible)
  - Click expands to show all member alerts
- [ ] 3.8 Add correlation section to IntelligencePage
- [ ] 3.9 Verify all gates green
- [ ] 3.10 Commit Phase 3 — `feat(tier8): Alert Correlation + RCA`

## Phase 4 — Alert Deduplication + Noise Reduction (Tier 8.4)

- [ ] 4.1 Migration `migrations/039_noise.sql` — table `alert_noise_rules` (id, tenant_id, name, fingerprint_pattern (text NOT NULL), suppression_window_seconds (int NOT NULL default 3600), channels (text[] default '{}'), enabled (bool default true), created_at). Index on (tenant_id, fingerprint_pattern).
- [ ] 4.2 Migration `migrations/039_noise.sql` — table `snooze_log` (id, tenant_id, alert_id (uuid), user_id (uuid), duration_seconds (int), expires_at (timestamptz), reason (text), created_at). Index on (tenant_id, alert_id), Index on (tenant_id, expires_at).
- [ ] 4.3 Apply migration to prod
- [ ] 4.4 Handler `internal/handler/handlers_noise.go` (~200 LOC, 5 routes):
  - `GET /api/v1/noise/rules` — list noise rules
  - `POST /api/v1/noise/rules` — create rule (body: {name, fingerprint_pattern, suppression_window_seconds, channels})
  - `DELETE /api/v1/noise/rules/:id` — delete rule
  - `POST /api/v1/noise/test` — preview suppression (body: {fingerprint_pattern, window_seconds})
  - `POST /api/v1/noise/snooze` — snooze an alert (body: {alert_id, duration_seconds, reason})
  - `GET /api/v1/noise/history` — snooze history (filter ?rule_id=X&limit=50)
- [ ] 4.5 Register 5 routes
- [ ] 4.6 Shared component `web/src/components/shared/NoiseRuleEditor.tsx` (~80 LOC):
  - Props: `{rule?: {id, name, fingerprint_pattern, ...}, onSave: (rule) => void}`
  - Layout: form with name input, fingerprint pattern input, suppression window dropdown (5m/15m/1h/6h/24h), channels checkboxes
- [ ] 4.7 Add noise section to IntelligencePage (rule list + editor)
- [ ] 4.8 Verify all gates green
- [ ] 4.9 Commit Phase 4 — `feat(tier8): Alert Noise Reduction`

## Phase 5 — Intelligence Dashboard (Tier 8.5) — Final

- [ ] 5.1 Build unified `web/src/pages/IntelligencePage.tsx` (~200 LOC, 4 tabs: Anomalies / Predictions / Correlations / Noise)
- [ ] 5.2 Shared component `web/src/components/shared/RcaPanel.tsx` (~80 LOC):
  - Props: `{hints: Array<{likely_root, confidence, reasoning}>}`
  - Layout: list of top 5 RCA hints with confidence bars + reasoning preview
- [ ] 5.3 Handler `internal/handler/handlers_intelligence_export.go` (~80 LOC, 1 route):
  - `GET /api/v1/intelligence/export?days=7` — returns JSON file with last N days of anomalies + predictions + correlations
- [ ] 5.4 Register export route
- [ ] 5.5 AppSidebar: add "Intelligence" entry in observability section
- [ ] 5.6 Verify all gates green
- [ ] 5.7 Commit Phase 5 — `feat(tier8): Intelligence Dashboard`

## Phase 6 — Final verify + deploy + archive

- [ ] 6.1 Final go build + npm run type-check + lint + build all green
- [ ] 6.2 Bundle JS gzipped ≤ baseline + 70 KB
- [ ] 6.3 Deploy binary to `.115` (use skill `stackwatch-build-deploy`)
- [ ] 6.4 Live 4× verifier PASS (curl all 24 routes, expect 401)
- [ ] 6.5 Archive to `.hermes/changes/archive/2026-08-24-006-tier8-intelligence-alerting/`
- [ ] 6.6 Update journal.md with TIER 8 COMPLETE marker
- [ ] 6.7 Tier 8 ready for Tier 9 (Security & Enterprise)

## Final success criteria

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 5 user stories pass acceptance scenarios | [ ] |
| SC-002 | 24 routes registered + return correct status codes | [ ] |
| SC-003 | Migration 039 applied to prod | [ ] |
| SC-004 | go build + npm run type-check + lint + build exit 0 | [ ] |
| SC-005 | Bundle JS gzipped ≤ baseline + 70 KB | [ ] |
| SC-006 | Live 4× verifier PASS | [ ] |
| SC-007 | All files under 400 LOC | [ ] |
| SC-008 | Archive folder created | [ ] |
| SC-009 | journal.md updated with TIER 8 COMPLETE marker | [ ] |
| SC-010 | Tier 8 ready for Tier 9 (Security & Enterprise) | [ ] |