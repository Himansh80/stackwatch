// Tier 8 Phase 2 — Predictive Alerting (Tier 8.2). Three of the
// four HTTP route handlers for /predict/* — the forecast handler
// lives in handlers_predict_forecast.go so this file stays under
// the 400-LOC cap.
//
// Routes handled here:
//
//	GET  /api/v1/predict/alerts    — list predictive alerts (filter ?status&severity&limit)
//	GET  /api/v1/predict/accuracy  — model MAPE / RMSE for a metric
//	POST /api/v1/predict/ack       — acknowledge a predictive alert
//
// The forecast handler (POST /api/v1/predict/forecast) lives in
// handlers_predict_forecast.go. The OLS math (forecastSample +
// olsFit) lives in handlers_predict_math.go.
package handler

import (
	"math"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListPredictiveAlerts returns recent predictive alerts for the
// caller's tenant, newest first. Optional filters:
// ?status=open|acknowledged|resolved&severity=Y&limit=50
func ListPredictiveAlerts(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := clampLimit(c.Query("limit"), 50, 500)
		status := strings.ToLower(strings.TrimSpace(c.Query("status")))
		severity := strings.ToLower(strings.TrimSpace(c.Query("severity")))

		args := []interface{}{tenantID}
		where := "tenant_id = $1"
		if status != "" {
			if !allowedPredictiveStatuses[status] {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			args = append(args, status)
			where += " AND status = $" + itoa(len(args))
		}
		if severity != "" {
			if !allowedPredictiveSeverities[severity] {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			args = append(args, severity)
			where += " AND severity = $" + itoa(len(args))
		}
		args = append(args, limit)
		limitIdx := len(args)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, metric_name, server_id::text,
			        predicted_value,
			        predicted_breach_at::text,
			        confidence, severity, status,
			        ack_user_id::text, ack_note,
			        created_at::text
			   FROM intelligence_predictive_alerts
			  WHERE `+where+`
			  ORDER BY created_at DESC
			  LIMIT $`+itoa(limitIdx), args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []predictiveAlertRow{}
		for rows.Next() {
			var r predictiveAlertRow
			var serverID, ackUserID, ackNote *string
			if err := rows.Scan(&r.ID, &r.TenantID, &r.MetricName, &serverID,
				&r.PredictedValue, &r.PredictedBreachAt,
				&r.Confidence, &r.Severity, &r.Status,
				&ackUserID, &ackNote, &r.CreatedAt); err != nil {
				continue
			}
			r.ServerID = serverID
			r.AckUserID = ackUserID
			r.AckNote = ackNote
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{
			"alerts": out,
			"total":  len(out),
			"limit":  limit,
		})
	}
}

// PredictAccuracy returns the latest MAPE / RMSE for the caller's
// tenant + metric. Accepts the metric via query string (?metric_name=X)
// OR JSON body (POST variant) so curl callers can use either. If
// no forecast_history rows exist for the metric, returns 404 so the
// UI can show "no forecasts yet".
func PredictAccuracy(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		metricName := strings.TrimSpace(c.Query("metric_name"))
		serverQ := strings.TrimSpace(c.Query("server_id"))
		// Allow the body variant for POST-style accuracy probes.
		if metricName == "" && c.Request.ContentLength > 0 {
			var body struct {
				MetricName string  `json:"metric_name"`
				ServerID   *string `json:"server_id"`
			}
			if err := c.ShouldBindJSON(&body); err == nil {
				metricName = strings.TrimSpace(body.MetricName)
				if body.ServerID != nil {
					serverQ = strings.TrimSpace(*body.ServerID)
				}
			}
		}
		if metricName == "" {
			kernel.RespondErrorWithCode(c, 400, "bad_request", "metric_name is required")
			return
		}
		var serverParam interface{}
		if serverQ != "" {
			sid, err := uuid.Parse(serverQ)
			if err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			serverParam = sid
		}

		// Aggregate the latest 20 evaluations. AVG gives a smoother
		// answer than taking the single newest row.
		var (
			avgMAPE   float64
			avgRMSE   float64
			count     int64
			lastEval  *string
			modelType *string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT AVG(mape)::double precision,
			        AVG(rmse)::double precision,
			        COUNT(*),
			        MAX(evaluated_at)::text,
			        (ARRAY_AGG(model_type ORDER BY evaluated_at DESC))[1]
			   FROM (
			     SELECT mape, rmse, model_type, evaluated_at
			       FROM intelligence_forecast_history
			      WHERE tenant_id = $1
			        AND metric_name = $2
			        AND ($3::uuid IS NULL OR server_id = $3)
			      ORDER BY evaluated_at DESC
			      LIMIT 20
			   ) recent`,
			tenantID, metricName, serverParam,
		).Scan(&avgMAPE, &avgRMSE, &count, &lastEval, &modelType)
		if err != nil {
			kernel.RespondErrorWithCode(c, 404, "not_found",
				"no forecast_history for this metric — POST /predict/forecast first")
			return
		}
		confidence := 1.0 / (1.0 + avgMAPE/100.0)
		if math.IsNaN(confidence) || math.IsInf(confidence, 0) {
			confidence = 1.0
		}
		resp := gin.H{
			"metric_name":    metricName,
			"mape":           avgMAPE,
			"rmse":           avgRMSE,
			"sample_count":   count,
			"confidence":     confidence,
			"last_evaluated": lastEval,
			"model_type":     modelType,
		}
		c.JSON(200, resp)
	}
}

// AckPredictiveAlert marks a predictive alert as acknowledged. The
// user_id is stamped from the JWT (claims.UserID), never from the body.
// 404 if the alert isn't in the caller's tenant (we don't leak
// existence of foreign rows).
func AckPredictiveAlert(pool *db.Pool) gin.HandlerFunc {
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
		var req predictiveAckRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		alertID, err := uuid.Parse(req.AlertID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE intelligence_predictive_alerts
			    SET status      = 'acknowledged',
			        ack_user_id = $3,
			        ack_note    = $4
			  WHERE id = $1 AND tenant_id = $2`,
			alertID, tenantID, claims.UserID, strings.TrimSpace(req.Note),
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{
			"alert_id":    alertID.String(),
			"status":      "acknowledged",
			"ack_user_id": claims.UserID.String(),
		})
	}
}
