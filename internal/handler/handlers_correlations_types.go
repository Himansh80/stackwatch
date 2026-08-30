// Tier 8 Phase 3 — Alert Correlation + RCA (Tier 8.3). Shared types
// + access-control maps for the 5 correlation endpoints.
//
// Five endpoints back the correlation / RCA surface:
//
//	GET  /api/v1/correlations/groups    — list correlation groups (filter ?auto_detected&limit)
//	GET  /api/v1/correlations/group/:id — group detail with all member alerts + RCA + feedback
//	POST /api/v1/correlations/manual    — create a manual correlation (override auto-detect)
//	GET  /api/v1/correlations/rca/:aid  — RCA hints for a single alert
//	POST /api/v1/correlations/feedback  — record user feedback on a correlation
//
// Every protected query honors tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. Manual correlations always set
// auto_detected=false (vs. the background correlator's true default).
//
// Tables backing this surface are created by the idempotent migration
// migrations/039_correlations.sql.
package handler

import "time"

// Compile-time guard so the time import stays referenced even if a
// downstream refactor trims a usage — keeps the import list honest.
var _ = time.RFC3339

// ------------------------------------------------------------------
// JSON row types — mirror the column shape and map directly to the
// `out` slice returned by each List* handler.
// ------------------------------------------------------------------

// alertCorrelationRow is the JSON shape for a single correlation
// group returned by GET /correlations/groups. The GroupDetail
// endpoint nests an extra `members` array of correlationMemberRow
// (plus `rca_hints` + `feedback_history`) but the list endpoint
// stays flat so the UI can render top-N cards cheaply.
type alertCorrelationRow struct {
	ID              string      `json:"id"`
	TenantID        string      `json:"tenant_id"`
	CorrelationID   string      `json:"correlation_id"`
	RootAlertID     string      `json:"root_alert_id"`
	MemberAlertIDs  []string    `json:"member_alert_ids"`
	MemberCount     int         `json:"member_count"`
	SimilarityScore float64     `json:"similarity_score"`
	AutoDetected    bool        `json:"auto_detected"`
	CreatedAt       string      `json:"created_at"`
	TopRCAHint      *rcaHintRow `json:"top_rca_hint,omitempty"`
}

// correlationMemberRow is the per-alert shape inside a group's
// `members` array. We pull `severity` from whichever underlying
// event the member_id points at (anomaly_events preferred;
// predictive_alerts fallback). The handler does a UNION ALL to
// surface both — the UI uses severity to color the member chips.
type correlationMemberRow struct {
	AlertID    string  `json:"alert_id"`
	Source     string  `json:"source"`   // 'anomaly' | 'predict'
	Severity   string  `json:"severity"` // 'info' | 'warning' | 'critical'
	MetricName string  `json:"metric_name"`
	ServerID   *string `json:"server_id,omitempty"`
	DetectedAt string  `json:"detected_at"`
}

// rcaHintRow is the JSON shape returned for a single RCA hint
// (GET /correlations/rca/:alert_id or nested inside a group's
// `rca_hints` array).
type rcaHintRow struct {
	ID                   string   `json:"id"`
	AlertID              string   `json:"alert_id"`
	LikelyRoot           string   `json:"likely_root"`
	Confidence           float64  `json:"confidence"`
	Reasoning            string   `json:"reasoning"`
	SimilarPastIncidents []string `json:"similar_past_incidents"`
	CreatedAt            string   `json:"created_at"`
}

// correlationFeedbackRow is the JSON shape for a single feedback
// submission (POST /correlations/feedback). The detail endpoint
// returns the history as `feedback_history`.
type correlationFeedbackRow struct {
	ID            string  `json:"id"`
	CorrelationID string  `json:"correlation_id"`
	UserID        string  `json:"user_id"`
	Useful        bool    `json:"useful"`
	Note          *string `json:"note,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST endpoints.
// ------------------------------------------------------------------

// manualCorrelationRequest is the JSON body for POST /correlations/manual.
// alert_ids is required (must contain at least 2 UUIDs); reason is an
// optional operator note that gets stored alongside the auto_detected=false
// row for audit purposes.
type manualCorrelationRequest struct {
	AlertIDs        []string `json:"alert_ids" binding:"required,min=2,max=50,dive,uuid"`
	Reason          string   `json:"reason"    binding:"max=2048"`
	SimilarityScore float64  `json:"similarity_score" binding:"omitempty,min=0,max=1"`
}

// correlationFeedbackRequest is the JSON body for POST /correlations/feedback.
// user_id is intentionally NOT in the body — we stamp it from the JWT
// (claims.UserID) so a user can't submit feedback on behalf of someone else.
type correlationFeedbackRequest struct {
	CorrelationID string `json:"correlation_id" binding:"required,uuid"`
	Useful        bool   `json:"useful"`
	Note          string `json:"note"           binding:"max=2048"`
}
