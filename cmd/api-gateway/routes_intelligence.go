package main

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/handler"
)

// mountIntelligenceRoutes registers the Tier 8 Intelligence & Alerting
// endpoints — speckit change 006-tier8-intelligence-alerting.
//
// Extracted to its own file in Phase 0 of the 006 change because
// routes_protected.go was at the 380-LOC cap after Tier 7.5 Team Collab
// (D12). Adding 24 new Tier 8 routes without a split would push the
// file past the 400-LOC limit. Splitting keeps routes_protected.go
// focused on top-level tier wiring while the intelligence CRUD lives
// here.
//
// Phases 1-5 add their routes here in this order:
//
//	Phase 1: ML Anomaly Detection (5 routes)        ← added in Phase 1
//	Phase 2: Predictive Alerting (4 routes)          ← future
//	Phase 3: Alert Correlation + RCA (5 routes)       ← future
//	Phase 4: Alert Noise Reduction (5 routes)         ← future
//	Phase 5: Intelligence Dashboard + Export (5)     ← future
//
// All handlers honor tenant_id from the JWT — no cross-tenant data
// ever crosses the wire. Idempotent migrations in
// migrations/039_intelligence.sql set up the backing tables.
func mountIntelligenceRoutes(protected *gin.RouterGroup, pool *db.Pool) {
	// ---- Tier 8.1: ML Anomaly Detection (Phase 1) ----
	// Per the spec, the existing internal/ml.Detector (Welford + EWMA)
	// does the math — these handlers are thin HTTP shells that fetch
	// historical metric_points, run the detector, and persist the
	// results to intelligence_anomaly_events.
	//
	// Routes (5):
	//   GET  /api/v1/anomaly/models    — list trained models for tenant
	//   POST /api/v1/anomaly/train     — train/update a model (last 24h)
	//   POST /api/v1/anomaly/detect    — detect anomalies now
	//   GET  /api/v1/anomaly/events    — list recent events
	//   POST /api/v1/anomaly/ack       — acknowledge an event
	protected.GET("/anomaly/models", handler.ListAnomalyModels(pool))
	protected.POST("/anomaly/train", handler.TrainAnomalyModel(pool))
	protected.POST("/anomaly/detect", handler.DetectAnomalyFromModel(pool))
	protected.GET("/anomaly/events", handler.ListAnomalyEvents(pool))
	protected.POST("/anomaly/ack", handler.AckAnomalyEvent(pool))

	// ---- Tier 8.2: Predictive Alerting (Phase 2) ----
	// Backed by an OLS linear regression fit to the last 7 days of
	// metric_points (hourly bucketed median). The math lives in
	// handlers_predict.go (olsFit); these handlers are thin HTTP
	// shells that fetch historical samples, run the fit, persist
	// the forecast result, and (when p90 crosses breach_threshold)
	// INSERT a predictive_alerts row.
	//
	// Routes (4):
	//   POST /api/v1/predict/forecast   — generate forecast + optional alert
	//   GET  /api/v1/predict/alerts     — list predictive alerts
	//   GET  /api/v1/predict/accuracy   — MAPE / RMSE for a metric
	//   POST /api/v1/predict/ack        — acknowledge a predictive alert
	protected.POST("/predict/forecast", handler.ForecastPredict(pool))
	protected.GET("/predict/alerts", handler.ListPredictiveAlerts(pool))
	protected.GET("/predict/accuracy", handler.PredictAccuracy(pool))
	protected.POST("/predict/ack", handler.AckPredictiveAlert(pool))

	// ---- Tier 8.3: Alert Correlation + RCA (Phase 3) ----
		// Surfaces correlation groups (auto-detected by the background
		// correlator or manually created by an operator) and root-cause
		// analysis hints. The detail handler (GET /group/:id) lives in
		// handlers_correlations_detail.go; the rest of the 5 routes are
		// in handlers_correlations.go.
		//
		// Routes (5):
		//   GET  /api/v1/correlations/groups     — list groups
		//   GET  /api/v1/correlations/group/:id  — group detail
		//   POST /api/v1/correlations/manual     — create manual group
		//   GET  /api/v1/correlations/rca/:aid   — RCA hints for an alert
		//   POST /api/v1/correlations/feedback   — record user feedback
		protected.GET("/correlations/groups", handler.ListCorrelationGroups(pool))
		protected.GET("/correlations/group/:id", handler.GetCorrelationGroup(pool))
		protected.POST("/correlations/manual", handler.CreateManualCorrelation(pool))
		protected.GET("/correlations/rca/:alert_id", handler.GetRCAHintsForAlert(pool))
		protected.POST("/correlations/feedback", handler.RecordCorrelationFeedback(pool))

		// Phase 4 — Alert Noise Reduction (5 routes)        ← future
		// Phase 5 — Intelligence Dashboard (5 routes)       ← future
	}
