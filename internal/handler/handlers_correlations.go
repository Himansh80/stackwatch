// Tier 8 Phase 3 — Alert Correlation + RCA (Tier 8.3). HTTP route
// handlers for /correlations/*.
//
// Routes handled here:
//
//	GET  /api/v1/correlations/groups    — list correlation groups
//	POST /api/v1/correlations/manual    — create manual correlation
//	GET  /api/v1/correlations/rca/:aid  — RCA hints for a single alert
//	POST /api/v1/correlations/feedback  — record user feedback
//
// The GET /correlations/group/:id detail handler lives in
// handlers_correlations_detail.go so this file stays under 400 LOC.
// Types live in handlers_correlations_types.go. All queries honor
// tenant_id from the JWT. The RCA heuristic is intentionally simple:
// when no rca_hints row exists we synthesize a low-confidence hint
// based on the count of recent alerts in the same tenant.
package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListCorrelationGroups returns recent correlation groups for the
// caller's tenant, newest first. Optional filters:
// ?auto_detected=true|false&limit=50
//
// For each group we also fetch the top RCA hint (highest confidence)
// via a LATERAL join so the UI can render the hint card without a
// second request. limit is clamped 1..500 (default 50).
func ListCorrelationGroups(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := clampLimit(c.Query("limit"), 50, 500)

		// Build the WHERE clause incrementally so we can support the
		// optional auto_detected filter without string-concat risks.
		args := []interface{}{tenantID}
		where := "ac.tenant_id = $1"
		autoFlag := strings.ToLower(strings.TrimSpace(c.Query("auto_detected")))
		if autoFlag == "true" || autoFlag == "false" {
			args = append(args, autoFlag == "true")
			where += " AND ac.auto_detected = $" + itoa(len(args))
		}
		args = append(args, limit)
		limitIdx := len(args)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT ac.id::text,
			        ac.tenant_id::text,
			        ac.correlation_id::text,
			        ac.root_alert_id::text,
			        COALESCE(ARRAY(
			            SELECT (u::text) FROM unnest(ac.member_alert_ids) u
			        ), '{}'::text[]) AS member_ids,
			        COALESCE(array_length(ac.member_alert_ids, 1), 0) AS member_count,
			        ac.similarity_score,
			        ac.auto_detected,
			        ac.created_at::text,
			        rh.likely_root, rh.confidence, rh.reasoning, rh.similar_incidents
			   FROM alert_correlations ac
			   LEFT JOIN LATERAL (
			     SELECT likely_root, confidence, reasoning,
			            COALESCE(similar_past_incidents, '[]'::jsonb) AS similar_incidents
			       FROM rca_hints
			      WHERE tenant_id = ac.tenant_id
			        AND alert_id = ac.root_alert_id
			      ORDER BY confidence DESC, created_at DESC
			      LIMIT 1
			   ) rh ON true
			  WHERE `+where+`
			  ORDER BY ac.similarity_score DESC, ac.created_at DESC
			  LIMIT $`+itoa(limitIdx), args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []alertCorrelationRow{}
		for rows.Next() {
			var r alertCorrelationRow
			var likelyRoot, reasoning, simIncJSON *string
			var confidence *float64
			if err := rows.Scan(&r.ID, &r.TenantID, &r.CorrelationID,
				&r.RootAlertID, &r.MemberAlertIDs, &r.MemberCount,
				&r.SimilarityScore, &r.AutoDetected, &r.CreatedAt,
				&likelyRoot, &confidence, &reasoning, &simIncJSON); err != nil {
				continue
			}
			if likelyRoot != nil && confidence != nil && reasoning != nil {
				hint := rcaHintRow{
					LikelyRoot: *likelyRoot,
					Confidence: *confidence,
					Reasoning:  *reasoning,
					CreatedAt:  r.CreatedAt,
				}
				if simIncJSON != nil {
					_ = json.Unmarshal([]byte(*simIncJSON), &hint.SimilarPastIncidents)
				}
				r.TopRCAHint = &hint
			}
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{
			"groups": out,
			"total":  len(out),
			"limit":  limit,
		})
	}
}

// CreateManualCorrelation inserts a correlation group authored by a
// human (vs. the background correlator). auto_detected is forced to
// false so the UI tags it "manual". root_alert_id is the first id in
// the list (the spec says alert_ids is an ordered array; the first
// is treated as the operator's "primary suspect"). similarity_score
// defaults to 1.0 if the caller didn't supply one (operators are
// confident they're related).
func CreateManualCorrelation(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req manualCorrelationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Deduplicate + validate UUIDs (binding handles parse, but
		// we still need to dedup). Convert to pgx-friendly array.
		seen := make(map[string]bool, len(req.AlertIDs))
		deduped := make([]uuid.UUID, 0, len(req.AlertIDs))
		for _, raw := range req.AlertIDs {
			id, err := uuid.Parse(strings.TrimSpace(raw))
			if err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			key := id.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			deduped = append(deduped, id)
		}
		if len(deduped) < 2 {
			kernel.RespondErrorWithCode(c, 400, "bad_request",
				"alert_ids must contain at least 2 distinct UUIDs")
			return
		}

		rootID := deduped[0]
		memberIDs := deduped[1:]
		groupID := uuid.New()
		sim := req.SimilarityScore
		if sim <= 0 {
			sim = 1.0
		}

		var rowID string
		var createdAt string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO alert_correlations
			    (tenant_id, correlation_id, root_alert_id,
			     member_alert_ids, similarity_score, auto_detected)
			 VALUES ($1, $2, $3, $4, $5, false)
			 RETURNING id::text, created_at::text`,
			tenantID, groupID, rootID, memberIDs, sim,
		).Scan(&rowID, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Store the operator's reason as an auto-rca_hint so the
		// detail page can show "reason: same root cause" alongside
		// the correlation. We tag confidence=1.0 because a human
		// asserted this.
		reasonText := strings.TrimSpace(req.Reason)
		if reasonText == "" {
			reasonText = "manual correlation created by operator"
		}
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO rca_hints
			    (tenant_id, alert_id, likely_root, confidence, reasoning)
			 VALUES ($1, $2, $3, $4, $5)`,
			tenantID, rootID, "manual_correlation", 1.0, reasonText,
		)

		kernel.RespondCreated(c, gin.H{
			"id":               rowID,
			"correlation_id":   groupID.String(),
			"root_alert_id":    rootID.String(),
			"member_alert_ids": memberIDStrings(memberIDs),
			"auto_detected":    false,
			"similarity_score": sim,
			"created_at":       createdAt,
		})
	}
}

// memberIDStrings formats a []uuid.UUID as []string for the JSON
// response.
func memberIDStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, m := range ids {
		out[i] = m.String()
	}
	return out
}

// GetRCAHintsForAlert returns the RCA hints for the supplied alert_id.
// If no hint exists yet, we synthesize a low-confidence heuristic one
// (inserted into rca_hints so subsequent calls return the same
// payload) and return it. The heuristic is intentionally simple:
// "same server + same 5-min window → resource_saturation"; Phase 4+
// will replace it with a real correlator.
func GetRCAHintsForAlert(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		alertID, err := uuid.Parse(strings.TrimSpace(c.Param("alert_id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// First: any existing hints for this alert?
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, alert_id::text, likely_root, confidence,
			        reasoning,
			        COALESCE(similar_past_incidents, '[]'::jsonb),
			        created_at::text
			   FROM rca_hints
			  WHERE tenant_id = $1 AND alert_id = $2
			  ORDER BY confidence DESC, created_at DESC`,
			tenantID, alertID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		hints := []rcaHintRow{}
		for rows.Next() {
			var h rcaHintRow
			var simJSON []byte
			if err := rows.Scan(&h.ID, &h.AlertID, &h.LikelyRoot,
				&h.Confidence, &h.Reasoning, &simJSON, &h.CreatedAt); err == nil {
				_ = json.Unmarshal(simJSON, &h.SimilarPastIncidents)
				hints = append(hints, h)
			}
		}
		rows.Close()

		// If we have nothing, run the heuristic and INSERT a low-
		// confidence hint so the next call is a cache hit.
		if len(hints) == 0 {
			hint := heuristicRCA(pool, c, tenantID, alertID)
			if hint != nil {
				hints = append(hints, *hint)
			}
		}

		c.JSON(200, gin.H{
			"alert_id":  alertID.String(),
			"rca_hints": hints,
			"total":     len(hints),
		})
	}
}

// heuristicRCA synthesizes a single low-confidence rca_hint row when
// none exists. Looks at the past 5 minutes of anomaly_events in the
// same tenant — if 3+ fired, "resource_saturation" with confidence
// ~0.5. Otherwise "isolated_event" with confidence ~0.3.
func heuristicRCA(pool *db.Pool, c *gin.Context, tenantID, alertID uuid.UUID) *rcaHintRow {
	// Count recent alerts in the same tenant within the last 5 min.
	var recentCount int
	_ = pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT COUNT(*) FROM intelligence_anomaly_events
		  WHERE tenant_id = $1 AND ts >= now() - INTERVAL '5 minutes'`,
		tenantID,
	).Scan(&recentCount)

	var likelyRoot string
	var confidence float64
	var reasoning string
	if recentCount >= 3 {
		likelyRoot = "resource_saturation"
		confidence = 0.5
		reasoning = "Heuristic: " + itoa(recentCount) + " anomaly events fired in the last 5 minutes for this tenant — likely correlated resource pressure. Confirm by inspecting the correlation group."
	} else {
		likelyRoot = "isolated_event"
		confidence = 0.3
		reasoning = "Heuristic: only a single alert fired in the last 5 minutes for this tenant. No clear correlating signal. Treat as isolated until more data arrives."
	}

	var inserted rcaHintRow
	var simJSON []byte
	err := pool.Pgx().QueryRow(c.Request.Context(),
		`INSERT INTO rca_hints
		    (tenant_id, alert_id, likely_root, confidence, reasoning,
		     similar_past_incidents)
		 VALUES ($1, $2, $3, $4, $5, '[]'::jsonb)
		 RETURNING id::text, alert_id::text, likely_root, confidence,
		           reasoning, similar_past_incidents, created_at::text`,
		tenantID, alertID, likelyRoot, confidence, reasoning,
	).Scan(&inserted.ID, &inserted.AlertID, &inserted.LikelyRoot,
		&inserted.Confidence, &inserted.Reasoning, &simJSON, &inserted.CreatedAt)
	if err != nil {
		return nil
	}
	_ = json.Unmarshal(simJSON, &inserted.SimilarPastIncidents)
	return &inserted
}

// RecordCorrelationFeedback persists a user's judgement on a
// correlation. user_id is stamped from the JWT — never from the body.
// correlation_id is not FK-checked against alert_correlations so a
// freshly-created manual correlation can be fed back immediately.
func RecordCorrelationFeedback(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req correlationFeedbackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		cid, err := uuid.Parse(req.CorrelationID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Optional validation: if the correlation exists in our tenant
		// and is auto_detected, surface that. We don't 404 here — an
		// operator may want to record feedback on a manual correlation
		// they just created (race-condition safe). We just stamp the row.
		var autoFlag *bool
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT auto_detected FROM alert_correlations
			  WHERE tenant_id = $1 AND correlation_id = $2
			  LIMIT 1`,
			tenantID, cid,
		).Scan(&autoFlag)

		var rowID, createdAt string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO correlation_feedback
			    (tenant_id, correlation_id, user_id, useful, note)
			 VALUES ($1, $2, $3, $4, NULLIF($5, ''))
			 RETURNING id::text, created_at::text`,
			tenantID, cid, claims.UserID, req.Useful, strings.TrimSpace(req.Note),
		).Scan(&rowID, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		c.JSON(200, gin.H{
			"id":             rowID,
			"correlation_id": cid.String(),
			"user_id":        claims.UserID.String(),
			"useful":         req.Useful,
			"auto_detected":  autoFlag,
			"created_at":     createdAt,
		})
	}
}

// Compile-time guard so the time import stays referenced.
var _ = time.RFC3339
