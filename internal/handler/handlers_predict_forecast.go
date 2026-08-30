// Tier 8 Phase 2 — Predictive Alerting (Tier 8.2). POST /predict/forecast
// HTTP handler.
//
// ForecastPredict runs an OLS linear regression over the last 7 days
// of metric_points for the caller's tenant + (server_id optional).
// Returns the next `horizon_hours` predictions as a p10/p50/p90
// band. If any prediction's p90 crosses the breach_threshold (default
// 90% for percentage metrics), a row is INSERTed into
// intelligence_predictive_alerts and returned in the response so the
// UI can render it immediately.
//
// 400 on bad input. 404 if no metric_points exist in the last 7d.
// 200 on success.
//
// This handler is split out from handlers_predict.go so that file
// stays under the 400-LOC cap. The math itself lives in
// handlers_predict_math.go (forecastSample + olsFit) so it can be
// tested independently of HTTP plumbing.
package handler

import (
	"math"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ForecastPredict — see file header doc.
func ForecastPredict(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req forecastRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		horizon := req.HorizonHours
		if horizon <= 0 {
			horizon = 24
		}
		// Cap horizon at 168h (1 week) to keep the response payload
		// bounded. Anything beyond that isn't useful — the band
		// widens past usefulness anyway.
		if horizon > 168 {
			horizon = 168
		}
		modelType := strings.ToLower(strings.TrimSpace(req.ModelType))
		if modelType == "" {
			modelType = "linear_regression"
		}
		if !allowedPredictModelTypes[modelType] {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		breachPct := req.BreachThresholdPct
		if breachPct <= 0 {
			breachPct = 90.0
		}

		// Resolve server_id param (nullable — tenant-wide forecast if absent).
		var serverParam interface{}
		if req.ServerID != nil && strings.TrimSpace(*req.ServerID) != "" {
			sid, err := uuid.Parse(strings.TrimSpace(*req.ServerID))
			if err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			serverParam = sid
		}

		// Fetch hourly bucketed samples over the last 7 days. The
		// bucketing collapses high-frequency reporters (30s polls,
		// syslog bursts) into a regular hourly series — the OLS fit
		// on a regular series is far more stable than fitting the
		// raw stream.
		const lookbackSQL = `
			SELECT date_trunc('hour', ts) AS bucket_ts,
			       percentile_cont(0.5) WITHIN GROUP (ORDER BY value::double precision) AS median
			  FROM metric_points
			 WHERE tenant_id = $1
			   AND metric_name = $2
			   AND ($3::uuid IS NULL OR server_id = $3)
			   AND ts >= now() - INTERVAL '7 days'
			 GROUP BY bucket_ts
			 ORDER BY bucket_ts ASC`
		rows, err := pool.Pgx().Query(c.Request.Context(), lookbackSQL,
			tenantID, req.MetricName, serverParam)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		var samples []forecastSample
		for rows.Next() {
			var ts time.Time
			var v float64
			if err := rows.Scan(&ts, &v); err == nil {
				samples = append(samples, forecastSample{ts: ts, val: v})
			}
		}
		rows.Close()
		if len(samples) < 4 {
			// Need at least 4 hourly buckets to fit a line + have
			// any confidence in the slope. 404 so the UI can show
			// "not enough history" rather than a generic 500.
			kernel.RespondErrorWithCode(c, 404, "not_found",
				"need at least 4 hourly buckets of metric_points in the last 7d to forecast")
			return
		}

		// Fit OLS: y = m*x + b where x is hours since the first sample.
		slope, intercept, residStddev, mape := olsFit(samples)

		// Generate the next `horizon` hourly predictions starting
		// from the last historical hour.
		lastTs := samples[len(samples)-1].ts
		predicted := make([]predictedPoint, 0, horizon)
		maxP90 := math.Inf(-1)
		var breachAt time.Time
		for i := 1; i <= horizon; i++ {
			ts := lastTs.Add(time.Duration(i) * time.Hour)
			x := float64(len(samples) - 1 + i) // hours since start
			p50 := slope*x + intercept
			p10 := p50 - 1.28*residStddev
			p90 := p50 + 1.28*residStddev
			predicted = append(predicted, predictedPoint{
				TS:  ts.UTC().Format(time.RFC3339),
				P10: p10,
				P50: p50,
				P90: p90,
			})
			if p90 > maxP90 {
				maxP90 = p90
				breachAt = ts
			}
		}

		confidence := 1.0 / (1.0 + mape/100.0)
		// If mape is NaN (zero variance) treat as 100% confidence.
		if math.IsNaN(confidence) || math.IsInf(confidence, 0) {
			confidence = 1.0
		}

		// If the forecast predicts a breach, INSERT a
		// predictive_alerts row.
		var alertID string
		var severity string
		if maxP90 >= breachPct {
			severity = "warning"
			if maxP90 >= breachPct+10 {
				severity = "critical"
			} else if maxP90 < breachPct+5 {
				severity = "info"
			}
			row := pool.Pgx().QueryRow(c.Request.Context(),
				`INSERT INTO intelligence_predictive_alerts
				   (tenant_id, metric_name, server_id,
				    predicted_value, predicted_breach_at,
				    confidence, severity, status)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, 'open')
				 RETURNING id::text`,
				tenantID, req.MetricName, serverParam,
				maxP90, breachAt, confidence, severity,
			)
			if err := row.Scan(&alertID); err != nil {
				// Persistence failure shouldn't fail the forecast —
				// log via kernel envelope and continue with the
				// forecast response.
				kernel.RespondError(c, err)
				return
			}
		}

		// Record the model's accuracy for this forecast run so the
		// /predict/accuracy endpoint can return a meaningful MAPE.
		_, _ = pool.Pgx().Exec(c.Request.Context(),
			`INSERT INTO intelligence_forecast_history
			   (tenant_id, metric_name, server_id, model_type, mape, rmse)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			tenantID, req.MetricName, serverParam, modelType, mape, residStddev,
		)

		resp := gin.H{
			"metric_name":         req.MetricName,
			"model_type":          modelType,
			"horizon_hours":       horizon,
			"predicted_values":    predicted,
			"confidence":          confidence,
			"mape":                mape,
			"rmse":                residStddev,
			"breach_threshold":    breachPct,
			"max_predicted_value": maxP90,
		}
		if alertID != "" {
			resp["alert"] = gin.H{
				"id":                  alertID,
				"predicted_value":     maxP90,
				"predicted_breach_at": breachAt.UTC().Format(time.RFC3339),
				"confidence":          confidence,
				"severity":            severity,
				"status":              "open",
			}
		}
		c.JSON(200, resp)
	}
}
