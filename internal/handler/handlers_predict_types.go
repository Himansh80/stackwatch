// Tier 8 Phase 2 — Predictive Alerting (Tier 8.2). Shared types +
// access-control maps for the 4 predict endpoints.
//
// Four endpoints backed by a simple ordinary-least-squares linear
// regression fit to the last 7 days of metric_points:
//
//	POST /api/v1/predict/forecast   — generate forecast + optional alert
//	GET  /api/v1/predict/alerts     — list predictive alerts
//	GET  /api/v1/predict/accuracy   — MAPE / RMSE for a metric
//	POST /api/v1/predict/ack        — acknowledge a predictive alert
//
// Every protected query honors tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. model_type is restricted to
// 'linear_regression' (default) / 'exponential_smoothing' /
// 'holt_winters' at the API edge. severity is restricted to the same
// 3-value set Phase 1 uses so the UI chart legend stays uniform.
//
// Tables backing this surface are created by the idempotent migration
// migrations/039_predictive.sql.
package handler

import "time"

// Compile-time guard so the time import is referenced even if a
// downstream refactor trims a usage — keeps the import list honest.
var _ = time.RFC3339

// allowedPredictModelTypes — defense in depth at the API edge.
// Phase 2 only writes 'linear_regression'; the others are reserved
// for future phases but we accept them here so the schema is
// forward-compatible.
var allowedPredictModelTypes = map[string]bool{
	"linear_regression":      true,
	"exponential_smoothing":  true,
	"holt_winters":           true,
}

// allowedPredictiveSeverities — the severity set the prediction UI
// knows how to render (matches Phase 1's allowedSeverities so the
// KPI strip / chart legend can share a single color mapping).
var allowedPredictiveSeverities = map[string]bool{
	"info":     true,
	"warning":  true,
	"critical": true,
}

// allowedPredictiveStatuses — matches the schema default 'open' and
// the values written by AckPredictiveAlert.
var allowedPredictiveStatuses = map[string]bool{
	"open":         true,
	"acknowledged": true,
	"resolved":     true,
}

// predictedPoint is the per-hour JSON shape returned inside the
// `predicted_values` array of POST /predict/forecast. ts is RFC3339;
// p10/p50/p90 are the 10th / 50th / 90th percentiles of the
// forecast distribution (±1.28σ around the linear point estimate).
type predictedPoint struct {
	TS  string  `json:"ts"`
	P10 float64 `json:"p10"`
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
}

// predictiveAlertRow is the JSON shape returned for a single
// predicted-breach alert (GET /predict/alerts).
//
// `predicted_value` is the forecast's p90 at the predicted breach
// timestamp. confidence is the model's overall confidence
// (1 / (1 + mape/100)). ack_note / ack_user_id follow the Phase 1
// pattern — set by POST /predict/ack.
type predictiveAlertRow struct {
	ID                 string  `json:"id"`
	TenantID           string  `json:"tenant_id"`
	MetricName         string  `json:"metric_name"`
	ServerID           *string `json:"server_id,omitempty"`
	PredictedValue     float64 `json:"predicted_value"`
	PredictedBreachAt  string  `json:"predicted_breach_at"`
	Confidence         float64 `json:"confidence"`
	Severity           string  `json:"severity"`
	Status             string  `json:"status"`
	AckUserID          *string `json:"ack_user_id,omitempty"`
	AckNote            *string `json:"ack_note,omitempty"`
	CreatedAt          string  `json:"created_at"`
}

// forecastRequest is the JSON shape for POST /predict/forecast.
type forecastRequest struct {
	MetricName   string  `json:"metric_name"   binding:"required,min=1,max=256"`
	ServerID     *string `json:"server_id"`                                // optional → tenant-wide
	HorizonHours int     `json:"horizon_hours" binding:"omitempty,min=1,max=168"`
	ModelType    string  `json:"model_type" binding:"omitempty,oneof=linear_regression exponential_smoothing holt_winters"`
	BreachThresholdPct float64 `json:"breach_threshold_pct" binding:"omitempty,min=1,max=100"`
}

// predictiveAckRequest is the JSON shape for POST /predict/ack.
type predictiveAckRequest struct {
	AlertID string `json:"alert_id" binding:"required,uuid"`
	Note    string `json:"note"     binding:"max=2048"`
}