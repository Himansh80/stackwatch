// Tier 8 Phase 5 — Intelligence Dashboard + Export (Tier 8.5).
//
// One endpoint backs the dashboard's "Export intelligence report"
// button on IntelligencePage. Returns a JSON document containing the
// last N days (default 7, max 90) of every Tier 8 surface for the
// caller's tenant:
//
//	{
//	  exported_at:   RFC3339,
//	  tenant_id:     uuid,
//	  period_days:   int,
//	  anomalies:     [...last N days of intelligence_anomaly_events...],
//	  predictions:   [...last N days of intelligence_predictive_alerts...],
//	  correlations:  [...last N days of alert_correlations...],
//	  noise_rules:   [...current alert_noise_rules...],
//	  snooze_log:    [...last N days of snooze_log...]
//	}
//
// The response is sent with Content-Disposition: attachment so the
// browser saves the file as `intelligence-export-YYYY-MM-DD.json`.
// Every query filters by claims.TenantID — no cross-tenant data ever
// crosses the wire.
//
// Why one endpoint instead of 5: the spec asks for a single
// downloadable report. Bundling all surfaces into one response
// matches the user's mental model ("export everything about
// intelligence") and keeps the wire-protocol flat.
package handler

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ExportIntelligenceReport writes a JSON document containing the last
// `?days=N` days (default 7, clamped to [1, 90]) of every Tier 8
// surface for the caller's tenant. Content-Disposition forces a
// browser download.
//
//	GET /api/v1/intelligence/export?days=7
//
// The handler is read-only and idempotent. Every query is scoped to
// the JWT tenant_id — no cross-tenant data ever crosses the wire.
func ExportIntelligenceReport(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		// Parse + clamp `days`. Default 7, max 90 (so a curious user
		// can't pull three years of data in one request).
		days := 7
		if raw := strings.TrimSpace(c.Query("days")); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil {
				days = v
			}
		}
		if days < 1 {
			days = 1
		}
		if days > 90 {
			days = 90
		}

		ctx := c.Request.Context()
		cutoff := fmt.Sprintf("now() - (%d || ' days')::interval", days)

		// ---- Anomalies (Phase 1 surface) ----
		anomalyRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, COALESCE(model_id::text, ''),
			        COALESCE(server_id::text, ''),
			        metric_name, anomaly_score, severity,
			        observed_value, expected_range_low, expected_range_high,
			        ts::text, acknowledged,
			        COALESCE(ack_note, ''), COALESCE(ack_user_id::text, '')
			   FROM intelligence_anomaly_events
			  WHERE tenant_id = $1
			    AND ts >= `+cutoff+`
			  ORDER BY ts DESC
			  LIMIT 5000`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		anomalies := []anomalyEventRow{}
		for anomalyRows.Next() {
			var r anomalyEventRow
			var modelID, serverID, ackUserID string
			var ackNote string
			if err := anomalyRows.Scan(&r.ID, &modelID, &serverID,
				&r.MetricName, &r.AnomalyScore, &r.Severity,
				&r.ObservedValue, &r.ExpectedRangeLow, &r.ExpectedRangeHigh,
				&r.TS, &r.Acknowledged, &ackNote, &ackUserID); err == nil {
				if modelID != "" {
					r.ModelID = &modelID
				}
				if serverID != "" {
					r.ServerID = &serverID
				}
				if ackNote != "" {
					r.AckNote = &ackNote
				}
				if ackUserID != "" {
					r.AckUserID = &ackUserID
				}
				anomalies = append(anomalies, r)
			}
		}
		anomalyRows.Close()

		// ---- Predictive alerts (Phase 2 surface) ----
		predictRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, tenant_id::text, metric_name,
			        COALESCE(server_id::text, ''),
			        predicted_value, predicted_breach_at::text,
			        confidence, severity, status,
			        COALESCE(ack_user_id::text, ''),
			        COALESCE(ack_note, ''),
			        created_at::text
			   FROM intelligence_predictive_alerts
			  WHERE tenant_id = $1
			    AND created_at >= `+cutoff+`
			  ORDER BY created_at DESC
			  LIMIT 5000`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		predictions := []predictiveAlertRow{}
		for predictRows.Next() {
			var r predictiveAlertRow
			var serverID, ackUserID, ackNote string
			if err := predictRows.Scan(&r.ID, &r.TenantID, &r.MetricName,
				&serverID, &r.PredictedValue, &r.PredictedBreachAt,
				&r.Confidence, &r.Severity, &r.Status,
				&ackUserID, &ackNote, &r.CreatedAt); err == nil {
				if serverID != "" {
					r.ServerID = &serverID
				}
				if ackUserID != "" {
					r.AckUserID = &ackUserID
				}
				if ackNote != "" {
					r.AckNote = &ackNote
				}
				predictions = append(predictions, r)
			}
		}
		predictRows.Close()

		// ---- Correlations (Phase 3 surface) ----
		// List every group whose root_alert OR any member is within
		// the window — but since correlations are derived from raw
		// alert events, we filter on the correlation's created_at.
		corrRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, tenant_id::text, correlation_id::text,
			        root_alert_id::text, member_alert_ids,
			        similarity_score, auto_detected, created_at::text
			   FROM alert_correlations
			  WHERE tenant_id = $1
			    AND created_at >= `+cutoff+`
			  ORDER BY created_at DESC
			  LIMIT 2000`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		correlations := []alertCorrelationRow{}
		for corrRows.Next() {
			var r alertCorrelationRow
			if err := corrRows.Scan(&r.ID, &r.TenantID, &r.CorrelationID,
				&r.RootAlertID, &r.MemberAlertIDs,
				&r.SimilarityScore, &r.AutoDetected,
				&r.CreatedAt); err == nil {
				r.MemberCount = len(r.MemberAlertIDs)
				correlations = append(correlations, r)
			}
		}
		corrRows.Close()

		// ---- Noise rules (Phase 4 surface — current state, no window) ----
		// Rules are config, not events; we surface all current rules
		// regardless of the days window so the export reflects the
		// tenant's present-day suppression policy.
		ruleRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, tenant_id::text, name,
			        fingerprint_pattern, suppression_window_seconds,
			        COALESCE(channels, '{}'::text[]) AS channels,
			        enabled, created_at::text
			   FROM alert_noise_rules
			  WHERE tenant_id = $1
			  ORDER BY created_at DESC
			  LIMIT 1000`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		noiseRules := []noiseRuleRow{}
		for ruleRows.Next() {
			var r noiseRuleRow
			if err := ruleRows.Scan(&r.ID, &r.TenantID, &r.Name,
				&r.FingerprintPattern, &r.SuppressionWindowSeconds,
				&r.Channels, &r.Enabled, &r.CreatedAt); err == nil {
				noiseRules = append(noiseRules, r)
			}
		}
		ruleRows.Close()

		// ---- Snooze log (Phase 4 surface, windowed) ----
		snoozeRows, err := pool.Pgx().Query(ctx,
			`SELECT id::text, tenant_id::text, alert_id::text,
			        user_id::text, duration_seconds,
			        expires_at::text, COALESCE(reason, ''),
			        (expires_at > now()) AS active,
			        created_at::text
			   FROM snooze_log
			  WHERE tenant_id = $1
			    AND created_at >= `+cutoff+`
			  ORDER BY created_at DESC
			  LIMIT 2000`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		snoozes := []snoozeLogRow{}
		for snoozeRows.Next() {
			var r snoozeLogRow
			var reason string
			if err := snoozeRows.Scan(&r.ID, &r.TenantID, &r.AlertID,
				&r.UserID, &r.DurationSeconds, &r.ExpiresAt,
				&reason, &r.Active, &r.CreatedAt); err == nil {
				if reason != "" {
					r.Reason = &reason
				}
				snoozes = append(snoozes, r)
			}
		}
		snoozeRows.Close()

		// Build the response envelope. exported_at is stamped
		// server-side (never trusted from the caller).
		body := gin.H{
			"exported_at":  time.Now().UTC().Format(time.RFC3339),
			"tenant_id":    tenantID.String(),
			"period_days":  days,
			"anomalies":    anomalies,
			"predictions":  predictions,
			"correlations": correlations,
			"noise_rules":  noiseRules,
			"snooze_log":   snoozes,
			"counts": gin.H{
				"anomalies":    len(anomalies),
				"predictions":  len(predictions),
				"correlations": len(correlations),
				"noise_rules":  len(noiseRules),
				"snooze_log":   len(snoozes),
			},
		}

		// Filename uses today's UTC date so two exports on the same
		// day don't collide and operators can sort chronologically.
		filename := fmt.Sprintf("intelligence-export-%s.json",
			time.Now().UTC().Format("2006-01-02"))

		c.Header("Content-Disposition",
			fmt.Sprintf("attachment; filename=%q", filename))
		c.Header("Content-Type", "application/json; charset=utf-8")
		kernel.RespondOK(c, body)
	}
}