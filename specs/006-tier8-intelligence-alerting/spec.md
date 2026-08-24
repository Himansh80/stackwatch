# Feature Specification: Tier 8 — Intelligence & Alerting

**Feature Branch**: `006-tier8-intelligence-alerting`
**Created**: 2026-08-24
**Status**: Draft

## Overview

Tier 8 layers intelligence on top of Tier 0-7's monitoring foundation. After this ships, StackWatch doesn't just collect and alert — it predicts, correlates, and explains.

## User Stories

### Story 1 — ML Anomaly Detection (Tier 8.1, P1 priority)

**As an** on-call engineer
**I want** StackWatch to automatically detect anomalies in my fleet's metrics
**So that** I learn about problems before users complain

**Why this priority**: Foundation for everything else. Every other Tier 8 feature builds on anomaly detection.

**Independent test**: Inject a known anomaly pattern, verify it's flagged within 1 eval cycle.

**Acceptance scenarios**:
1. **Given** a server has reported `cpu.user_pct` every 30s for 24 hours
   **When** `POST /api/v1/anomaly/train` is called with `{metric_name: "cpu.user_pct", server_id: X}`
   **Then** an `anomaly_models` row is created with mean + variance from the last 24h
2. **Given** a trained model exists for `cpu.user_pct`
   **When** the metric value spikes to >3σ above mean
   **Then** `POST /api/v1/anomaly/detect` returns `{anomaly_score: 4.2, severity: "critical"}`
3. **Given** an anomaly is detected
   **When** `GET /api/v1/anomaly/events?server_id=X&limit=50` is called
   **Then** the latest anomaly is returned with timestamp + score
4. **Given** an anomaly event exists
   **When** `POST /api/v1/anomaly/ack` is called with `{event_id, note}`
   **Then** the event is marked `acknowledged=true` with the operator's note
5. **Given** an anomaly is acknowledged
   **When** `GET /api/v1/anomaly/events` is called
   **Then** acknowledged events appear with a green checkmark badge in the UI

---

### Story 2 — Predictive Alerting (Tier 8.2, P1 priority)

**As a** SRE
**I want** StackWatch to forecast metric values 1-24 hours into the future
**So that** I can act on predicted degradation before it happens

**Why this priority**: Reactive alerts are too late. Predictive alerts let you preempt incidents.

**Independent test**: Use historical data to predict a known trend (e.g., disk fill rate), verify the prediction matches.

**Acceptance scenarios**:
1. **Given** a metric has 7+ days of history
   **When** `POST /api/v1/predict/forecast` is called with `{metric_name, server_id, horizon_hours: 24}`
   **Then** returns `{predicted_values: [{ts, p10, p50, p90}, ...], confidence: 0.85}`
2. **Given** a forecast predicts `disk.used_pct > 90%` in <24 hours
   **When** the prediction is generated
   **Then** a `predictive_alerts` event is fired (severity based on confidence + breach magnitude)
3. **Given** multiple forecasts exist for the same metric
   **When** `GET /api/v1/predict/accuracy` is called with `{metric_name, server_id}`
   **Then** returns `{mape: 0.12, rmse: 4.5, last_evaluated: ts}` — model accuracy metrics
4. **Given** a predictive alert fires
   **When** `POST /api/v1/predict/ack` is called with `{prediction_id, note}`
   **Then** the alert is acknowledged and won't re-fire for the same prediction window

---

### Story 3 — Alert Correlation + RCA (Tier 8.3, P2 priority)

**As an** on-call engineer
**I want** StackWatch to group related alerts and suggest root causes
**So that** I don't get paged 20 times for one underlying issue

**Why this priority**: Reduces alert fatigue — directly improves on-call quality of life.

**Independent test**: Fire a CPU spike + memory spike on the same host at the same time, verify they correlate.

**Acceptance scenarios**:
1. **Given** 3 alerts fire for the same server within 5 minutes
   **When** the correlator runs (background job)
   **Then** an `alert_correlations` row is created with `member_alert_ids: [a, b, c]` and `similarity_score: 0.92`
2. **Given** a correlation group exists
   **When** `GET /api/v1/correlations/group/:id` is called
   **Then** returns the group with all member alerts + the auto-detected root alert
3. **Given** a correlation exists
   **When** `POST /api/v1/correlations/manual` is called with `{alert_ids: [a, b], reason: "same root cause"}`
   **Then** a new correlation is created (manual override of auto-detect)
4. **Given** an alert is part of a correlation
   **When** `GET /api/v1/correlations/rca/:alert_id` is called
   **Then** returns RCA hints: `{likely_root: "server_id=X", confidence: 0.78, similar_past_incidents: [...]}`
5. **Given** an RCA suggestion was wrong
   **When** `POST /api/v1/correlations/feedback` is called with `{correlation_id, useful: false, note}`
   **Then** the feedback is stored for future model improvement

---

### Story 4 — Alert Deduplication + Noise Reduction (Tier 8.4, P2 priority)

**As an** SRE
**I want** StackWatch to silence noisy alerts automatically
**So that** I focus on real problems, not alert storms

**Why this priority**: Alert noise is the #1 cause of on-call burnout.

**Independent test**: Create a noisy alert rule, verify it fires once per window instead of every minute.

**Acceptance scenarios**:
1. **Given** a rule exists that would fire every 30s
   **When** a noise rule with `suppression_window: 3600` matches the alert fingerprint
   **Then** the alert is suppressed (not delivered) for 1 hour after the first firing
2. **Given** an alert is firing
   **When** `POST /api/v1/noise/snooze` is called with `{alert_id, duration: "1h"}`
   **Then** the alert is snoozed for 1 hour, then re-fires if still relevant
3. **Given** a noise rule is too aggressive
   **When** `POST /api/v1/noise/test` is called with `{fingerprint: "X", window: 300}`
   **Then** returns how many alerts WOULD be suppressed in the next 5 minutes (preview)
4. **Given** a noise rule has been working
   **When** `GET /api/v1/noise/history?rule_id=X` is called
   **Then** returns how many alerts were suppressed vs delivered in the last 24h
5. **Given** an operator wants custom noise rules
   **When** `POST /api/v1/noise/rules` is called with `{fingerprint_pattern, suppression_window_seconds, channels}`
   **Then** the rule is created and applied to future firings

---

### Story 5 — Intelligence Dashboard (Tier 8.5, P3 priority)

**As a** team lead
**I want** a single page that shows fleet intelligence at a glance
**So that** I can see what's anomalous, what's predicted, and what's correlated

**Why this priority**: The 4 algorithmic features need a single pane that ties them together.

**Independent test**: Open the dashboard, verify each KPI strip + tab renders with real data.

**Acceptance scenarios**:
1. **Given** the page is opened
   **When** it loads
   **Then** 3 KPI cards render (anomalies today / predictions fired / correlations detected) using kpiStagger
2. **Given** anomalies exist
   **When** the "Top Anomalies" tab is selected
   **Then** a Datadog-style anomaly chart renders per server (with sparkline + score)
3. **Given** correlations exist
   **When** the "Alert Health" tab is selected
   **Then** correlation cards render grouped by similarity, with member alert counts
4. **Given** RCA hints exist
   **When** the "RCA" tab is selected
   **Then** a panel shows top 5 most likely root causes with confidence scores
5. **Given** the user wants to export
   **When** "Export intelligence report" is clicked
   **Then** a JSON file with all anomalies + predictions + correlations for the last 7 days is downloaded

---

## Out of scope

- Tier 9 (Security & Enterprise) — SAML SSO, SCIM, advanced RBAC
- Tier 10 (Homelab Dashboard) — Homarr replacement, drag-drop widgets
- Tier 11 (Platform & Commerce) — billing, quotas, Stripe integration
- Tier 12 (Docs & GTM) — customer-facing docs, marketing landing
- Tier 13 (Mobile) — React Native app

Any subagent that adds features from these tiers is out of scope.

## Done criteria (measured at the END of Tier 8)

| ID | Criterion | Status |
|----|-----------|--------|
| SC-001 | All 5 user stories pass their acceptance scenarios | [ ] |
| SC-002 | 24 routes registered + return correct status codes | [ ] |
| SC-003 | Migration 039 applied to prod | [ ] |
| SC-004 | go build + npm run type-check + lint + build exit 0 | [ ] |
| SC-005 | Bundle JS gzipped ≤ baseline + 70 KB | [ ] |
| SC-006 | Live 4× verifier PASS | [ ] |
| SC-007 | All files under 400 LOC | [ ] |
| SC-008 | Archive folder created | [ ] |
| SC-009 | journal.md updated with TIER 8 COMPLETE marker | [ ] |
| SC-010 | Tier 8 ready for Tier 9 (Security & Enterprise) | [ ] |

## Risks

| Risk | Mitigation |
|------|------------|
| routes_protected.go will overflow (currently 380 LOC; +24 routes would push it ~480 LOC) | Plan for routes_intelligence.go split (new file) BEFORE Phase 1 starts |
| ML model training is slow on large fleets | Use Welford's algorithm (O(1) updates) + per-tenant partitioning |
| Predictive forecasting accuracy on noisy metrics | Show MAPE + RMSE in UI; mark low-confidence predictions as "experimental" |
| Alert correlation could be wrong | Provide feedback endpoint; let users override; store feedback for future improvement |
| Noise rules could hide real problems | Always show suppressed count in UI; log every suppressed event to audit_log |

## Modular approach (binding for all phases)

- **One concern per file.** Handlers split into 5 files: anomaly + predict + correlations + noise + dashboard.
- **Routes split.** Create `routes_intelligence.go` (NEW) similar to routes_incidents.go. After Phase 1: routes_protected.go should drop from 380 → ~340 LOC.
- **Shared components only.** Datadog-style chart components in `web/src/components/shared/`. Reuse tokens (`--surface`, `--border`, `--accent`, `--green`, `--amber`, `--red`).
- **No new dependencies.** Pure stdlib + existing tokens + existing motion variants.
- **Tenant_id isolation.** Every query filters by claims.TenantID.
- **Idempotent migrations.** `CREATE TABLE IF NOT EXISTS`.

## Notes for implementers

- Build env gotcha: ALWAYS use `export GOOS=linux; export GOARCH=amd64; go build -o ./<local>` (not `GOOS=linux go build -o /tmp/...` which silently fails).
- After completing each phase, ALWAYS commit. Subagents in prior sessions forgot this and left uncommitted files. Add "git commit hash (mandatory)" to output format.
- Use the existing `apm_*` tables (Tier 7.1) as a reference for time-series patterns.
- Use the existing `synthetics_*` tables (Tier 7.4) as a reference for background-job patterns.
- For ML math, use the existing `internal/ml/anomaly.go` (Welford + EWMA) as a starting point.