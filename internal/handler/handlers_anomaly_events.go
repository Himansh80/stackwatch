// Tier 8 Phase 1 — ML Anomaly Detection. Events list + detect + ack.
//
// Routes:
//
//	POST /api/v1/anomaly/detect  — run a value through the trained model
//	GET  /api/v1/anomaly/events  — list recent events (filter by server/severity)
//	POST /api/v1/anomaly/ack     — mark an event as acknowledged
//
// detect is the hot path: feed in a new value, get back a score +
// severity + the cached expected range, and (if above threshold) have
// a row written to intelligence_anomaly_events for the UI to render.
//
// ack is a soft-delete equivalent: the row stays in the table (we
// want historical "what was acked when" data) but the UI shows it
// dimmed with a green checkmark.
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

// DetectAnomalyFromModel runs a single value through the trained model
// for (metric_name, server_id). Returns the z-score, severity, the
// expected range, and (if above threshold) a fresh row in
// intelligence_anomaly_events.
//
// 404 if no trained model exists for the caller's tenant + metric
// (the UI will surface "train a model first"). 200 otherwise.
func DetectAnomalyFromModel(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req anomalyDetectRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		threshold := req.ThresholdSigma
		if threshold <= 0 {
			threshold = 3.0
		}

		// Resolve server_id param (nullable — tenant-wide model if absent).
		var serverParam interface{}
		if req.ServerID != nil && strings.TrimSpace(*req.ServerID) != "" {
			sid, err := uuid.Parse(strings.TrimSpace(*req.ServerID))
			if err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			serverParam = sid
		}

		// Look up the model.
		var (
			modelID   uuid.UUID
			mean      float64
			variance  float64
			sampleCnt int64
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id, mean, variance, sample_count
			   FROM intelligence_anomaly_models
			  WHERE tenant_id = $1
			    AND metric_name = $2
			    AND ($3::uuid IS NULL AND server_id IS NULL
			         OR server_id = $3)
			  LIMIT 1`,
			tenantID, req.MetricName, serverParam,
		).Scan(&modelID, &mean, &variance, &sampleCnt)
		if err != nil {
			kernel.RespondErrorWithCode(c, 404, "not_found",
				"no trained model exists for this metric — POST /anomaly/train first")
			return
		}

		// Compute the z-score. If variance is 0, every training point
		// was identical — any non-matching value is an infinite
		// anomaly; a matching value is a non-anomaly.
		var zscore float64
		if variance <= 0 {
			if math.Abs(req.Value-mean) > 0 {
				zscore = math.Inf(1)
			}
		} else {
			stddev := math.Sqrt(variance)
			zscore = (req.Value - mean) / stddev
		}
		isAnomaly := math.Abs(zscore) >= threshold
		severity := severityFromZscore(zscore)

		expectedLow := mean - threshold*math.Sqrt(variance)
		expectedHigh := mean + threshold*math.Sqrt(variance)

		// If above threshold, persist the event so the UI can show it.
		var eventID string
		if isAnomaly {
			row := pool.Pgx().QueryRow(c.Request.Context(),
				`INSERT INTO intelligence_anomaly_events
				   (tenant_id, model_id, server_id, metric_name,
				    anomaly_score, severity, observed_value,
				    expected_range_low, expected_range_high)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
				 RETURNING id::text`,
				tenantID, modelID, serverParam, req.MetricName,
				zscore, severity, req.Value, expectedLow, expectedHigh,
			)
			if err := row.Scan(&eventID); err != nil {
				// Persist failure shouldn't fail the detection — log
				// via kernel error envelope and continue.
				kernel.RespondError(c, err)
				return
			}
		}

		c.JSON(200, gin.H{
			"anomaly_score":      zscore,
			"is_anomaly":         isAnomaly,
			"severity":           severity,
			"observed_value":     req.Value,
			"expected_range":     []float64{expectedLow, expectedHigh},
			"expected_range_low": expectedLow,
			"expected_range_high": expectedHigh,
			"threshold_sigma":    threshold,
			"model_id":           modelID.String(),
			"sample_count":       sampleCnt,
			"event_id":           eventID,
		})
	}
}

// ListAnomalyEvents returns recent events for the caller's tenant,
// newest first. Optional filters: ?server_id=X&severity=info|warning|critical&limit=50
func ListAnomalyEvents(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := clampLimit(c.Query("limit"), 50, 500)
		severity := strings.ToLower(strings.TrimSpace(c.Query("severity")))
		serverQ := strings.TrimSpace(c.Query("server_id"))

		args := []interface{}{tenantID}
		where := "tenant_id = $1"
		if severity != "" {
			if !allowedSeverities[severity] {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			args = append(args, severity)
			where += " AND severity = $" + itoa(len(args))
		}
		if serverQ != "" {
			sid, err := uuid.Parse(serverQ)
			if err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			args = append(args, sid)
			where += " AND server_id = $" + itoa(len(args))
		}
		args = append(args, limit)
		limitIdx := len(args)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, model_id::text, server_id::text,
			        metric_name, anomaly_score, severity,
			        observed_value, expected_range_low, expected_range_high,
			        ts::text, acknowledged, ack_note, ack_user_id::text
			 FROM intelligence_anomaly_events
			 WHERE `+where+`
			 ORDER BY ts DESC
			 LIMIT $`+itoa(limitIdx), args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []anomalyEventRow{}
		for rows.Next() {
			var r anomalyEventRow
			var modelID, serverID, ackUserID *string
			var ackNote *string
			if err := rows.Scan(&r.ID, &modelID, &serverID,
				&r.MetricName, &r.AnomalyScore, &r.Severity,
				&r.ObservedValue, &r.ExpectedRangeLow, &r.ExpectedRangeHigh,
				&r.TS, &r.Acknowledged, &ackNote, &ackUserID); err != nil {
				continue
			}
			r.ModelID = modelID
			r.ServerID = serverID
			r.AckNote = ackNote
			r.AckUserID = ackUserID
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{
			"events": out,
			"total":  len(out),
			"limit":  limit,
		})
	}
}

// AckAnomalyEvent marks an event as acknowledged. The user_id is
// stamped from the JWT (claims.UserID), never from the body — that
// way a user can't ack on behalf of someone else. Note is operator-
// supplied and stored verbatim. 404 if the event isn't in the
// caller's tenant (we don't leak existence of foreign rows).
func AckAnomalyEvent(pool *db.Pool) gin.HandlerFunc {
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
		var req ackRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		eventID, err := uuid.Parse(req.EventID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE intelligence_anomaly_events
			    SET acknowledged = true,
			        ack_note     = $3,
			        ack_user_id  = $4
			  WHERE id = $1 AND tenant_id = $2`,
			eventID, tenantID, strings.TrimSpace(req.Note), claims.UserID,
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
			"event_id":    eventID.String(),
			"acknowledged": true,
			"ack_user_id": claims.UserID.String(),
		})
	}
}
