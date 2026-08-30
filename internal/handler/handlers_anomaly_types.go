// Tier 8 Phase 1 — ML Anomaly Detection (D9). Shared types + access-control maps.
//
// Five endpoints backed by the internal/ml.Detector (Welford + EWMA):
//
//	GET  /api/v1/anomaly/models    — list trained models for tenant
//	POST /api/v1/anomaly/train     — train/update a model (last 24h)
//	POST /api/v1/anomaly/detect    — detect anomalies now
//	GET  /api/v1/anomaly/events    — list recent events
//	POST /api/v1/anomaly/ack       — acknowledge an event
//
// Every protected query honors tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. model_type is restricted to 'welford' /
// 'ewma' at the API edge. severity is restricted to the canonical
// 3-value set the UI knows how to render.
//
// Tables backing this surface are created by the idempotent migration
// migrations/039_intelligence.sql.
//
// Severity → color mapping (the UI uses this):
//
//	'info'     → green   (mild z-score, mostly noise)
//	'warning'  → amber   (above threshold but not critical)
//	'critical' → red     (well beyond threshold, page-worthy)
package handler

import (
	"time"
)

// Compile-time guard so unused-time import is intentional in this file
// (re-exports below reference time.Time in their JSON tags).
var _ = time.RFC3339

// allowedModelTypes — defense in depth at the API edge. Phase 1 only
// writes 'welford'; 'ewma' is reserved for Phase 2 (predictive) but
// we accept it here so the schema is forward-compatible.
var allowedModelTypes = map[string]bool{
	"welford": true,
	"ewma":    true,
}

// allowedSeverities — the severity set the AnomalyChart UI knows how
// to render. Anything else would render as a blank circle.
var allowedSeverities = map[string]bool{
	"info":     true,
	"warning":  true,
	"critical": true,
}

// anomalyModelRow is the JSON shape returned for a single trained model.
//
// All fields are nullable in the underlying SQL but every field here
// is non-null — the schema defaults and the ON CONFLICT UPSERT in
// TrainAnomalyModel guarantee we only ever return populated rows.
type anomalyModelRow struct {
	ID            string  `json:"id"`
	TenantID      string  `json:"tenant_id"`
	MetricName    string  `json:"metric_name"`
	ServerID      *string `json:"server_id,omitempty"`
	ModelType     string  `json:"model_type"`
	Mean          float64 `json:"mean"`
	Variance      float64 `json:"variance"`
	EWMA          float64 `json:"ewma"`
	LastTrainedAt string  `json:"last_trained_at"`
	SampleCount   int64   `json:"sample_count"`
}

// anomalyEventRow is the JSON shape returned for a single detection.
//
// expected_range is rendered as a band on the AnomalyChart; we keep
// the low/high as flat scalars in the response so the frontend can
// avoid parsing arrays. server_id is nullable because the model can
// be tenant-wide.
type anomalyEventRow struct {
	ID                string  `json:"id"`
	ModelID           *string `json:"model_id,omitempty"`
	ServerID          *string `json:"server_id,omitempty"`
	MetricName        string  `json:"metric_name"`
	AnomalyScore      float64 `json:"anomaly_score"`
	Severity          string  `json:"severity"`
	ObservedValue     float64 `json:"observed_value"`
	ExpectedRangeLow  float64 `json:"expected_range_low"`
	ExpectedRangeHigh float64 `json:"expected_range_high"`
	TS                string  `json:"ts"`
	Acknowledged      bool    `json:"acknowledged"`
	AckNote           *string `json:"ack_note,omitempty"`
	AckUserID         *string `json:"ack_user_id,omitempty"`
}

// trainRequest is the JSON shape for POST /anomaly/train.
type trainRequest struct {
	MetricName string  `json:"metric_name" binding:"required,min=1,max=256"`
	ServerID   *string `json:"server_id"` // optional → tenant-wide model
	ModelType  string  `json:"model_type" binding:"omitempty,oneof=welford ewma"`
}

// anomalyDetectRequest is the JSON shape for POST /anomaly/detect.
// (Named with the anomaly prefix to avoid collision with the older
// Tier 6 streaming `detectRequest` type in analytics_anomaly.go —
// the two endpoints share the path /api/v1/anomaly/detect but the
// Tier 8 version is the new ML-model-backed one; Tier 6's stateless
// variant is still wired under the same route by handler ordering.)
type anomalyDetectRequest struct {
	MetricName     string  `json:"metric_name"     binding:"required,min=1,max=256"`
	ServerID       *string `json:"server_id"`
	Value          float64 `json:"value"           binding:"required"`
	ThresholdSigma float64 `json:"threshold_sigma" binding:"omitempty,min=0.1,max=20"`
}

// ackRequest is the JSON shape for POST /anomaly/ack.
type ackRequest struct {
	EventID string `json:"event_id" binding:"required,uuid"`
	Note    string `json:"note"     binding:"max=2048"`
}
