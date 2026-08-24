# Proposal: Tier 8 — Intelligence & Alerting

**Change folder**: `.hermes/changes/006-tier8-intelligence-alerting/`
**Spec folder**: `specs/006-tier8-intelligence-alerting/`
**Created**: 2026-08-24
**Status**: Draft

## Why

Tier 7 (Datadog Full Platform) shipped end-to-end on 2026-08-24 with 81 routes live + verified. Tiers 0-6 already shipped infrastructure for monitoring, alerts, and observability. Tier 8 layers **intelligence** on top of that foundation:

- ML-based anomaly detection that learns each host's normal pattern
- Predictive alerting that fires BEFORE the user notices degradation
- Alert correlation that groups related firings so on-call isn't paged 20 times
- Root-cause analysis hints that point at the most likely culprit
- Noise reduction so the same alert doesn't repeat-fire every 30 seconds

Per MASTER_BUILD_PLAN.md: "Tier 8 — Intelligence & Alerting (2-3 sessions)". This proposal estimates 1 change folder with 5 phases.

## What changes

### Backend additions (purely additive)

**4 new tables** (no removals):
- `anomaly_models` — per-metric ML model state (Welford mean + variance, EWMA, last training time)
- `anomaly_events` — when anomaly detection fires (tenant_id, metric_name, server_id, anomaly_score, severity, ts)
- `alert_correlations` — groups of correlated alerts (correlation_id, root_alert_id, member_alert_ids, similarity_score)
- `alert_noise_rules` — deduplication / grouping rules (tenant_id, fingerprint, suppression_window_seconds)

**~24 new routes** (5-6 per phase):
- Phase 1 (ML Anomaly): 5 routes (list models, train, detect, history, ack)
- Phase 2 (Predictive): 4 routes (list predictions, create forecast, model accuracy, ack)
- Phase 3 (Correlation): 5 routes (list groups, group detail, manual correlate, RCA hints, feedback)
- Phase 4 (Noise reduction): 5 routes (list rules, create rule, test rule, snooze alert, history)
- Phase 5 (Dashboard): 5 routes (summary KPIs, top anomalies, alert health, RCA summary, export)

**Migration**: `039_intelligence.sql` (4 tables, 1 materialized view for top anomalies)

### Frontend additions (Datadog style)

- 1 new page: `IntelligencePage.tsx` (Datadog-style 3-tab layout)
- 5 shared components: `AnomalyChart`, `PredictionChart`, `CorrelationCard`, `NoiseRuleEditor`, `RcaPanel`
- AppSidebar: "Intelligence" entry in observability section

### Modular discipline maintained

- All new files ≤400 LOC
- Handlers split by domain (anomaly + predictions + correlation + noise + dashboard)
- Routes split into `routes_intelligence.go` (NEW, similar to routes_incidents.go)
- No new dependencies (pure stdlib + existing motion + tokens)
- Tenant_id isolation enforced everywhere
- All migrations idempotent

## Impact

| Area | Impact |
|---|---|
| Backend | +1 migration, +5 handler files, +24 routes, ~1500 LOC |
| Database | +4 new tables + 1 matview |
| Frontend | +5 components + 1 page, ~900 LOC |
| Build | +50-70 KB JS gzipped |
| Tests | All existing tests must still pass |
| Docs | journal.md updated; archive folder created |
| Breaking changes | None — purely additive |

## Datadog parity

Mirrors Datadog's "Watchdog" + "Anomaly Detection" + "Alert Correlation" surfaces:
- **Watchdog**: detects anomalies automatically (we ship anomaly_models + anomaly_events)
- **Forecast**: predicts future values (we ship predictive alerts with confidence intervals)
- **Correlations**: groups related alerts (we ship alert_correlations with similarity scoring)
- **Noise Reduction**: deduplicates / silences alert storms (we ship alert_noise_rules + snooze)

## Scope discipline

This is Tier 8 ONLY. Out of scope:
- Tier 9 (Security & Enterprise) — SAML SSO, SCIM
- Tier 10 (Homelab Dashboard) — Homarr replacement
- Tier 11 (Platform & Commerce) — billing, quotas
- Tier 12 (Docs & GTM) — customer docs
- Tier 13 (Mobile) — React Native app

Any subagent that adds features outside Tier 8's 5 subtiers is out of scope.

## Done when

- [ ] All 5 user stories pass acceptance criteria
- [ ] All 24 routes registered + return correct status codes
- [ ] Migration applied to prod
- [ ] Bundle JS gzipped ≤ baseline + 70 KB
- [ ] All files under 400 LOC
- [ ] Archive folder created
- [ ] journal.md updated with TIER 8 COMPLETE marker
- [ ] Tier 8 ready for Tier 9 (Security & Enterprise)