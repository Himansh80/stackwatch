// Tier 8 Phase 3 — Alert Correlation + RCA (Tier 8.3). GET
// /api/v1/correlations/group/:id — full group detail.
//
// Extracted from handlers_correlations.go so that file stays under
// the 400-LOC cap. The detail handler is the heaviest of the 5
// (members UNION + rca_hints scan + feedback history in one request)
// and benefits from being its own file.
//
// Returns:
//   - group:           alertCorrelationRow
//   - members:         []correlationMemberRow (severity + metric + server + ts)
//   - rca_hints:       []rcaHintRow (one per member alert)
//   - feedback_history: []correlationFeedbackRow (all judgements on the group)
package handler

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// GetCorrelationGroup returns the full detail of a single correlation
// group. :id is the GROUP correlation_id (not the row id — the spec
// says "GET /correlations/group/:id" is keyed by the group identifier
// operators see in the UI).
//
// Members are fetched via UNION anomaly_events + predictive_alerts so
// the UI can render member chips regardless of which underlying stream
// fired the alert. RCA hints are fetched per-member; feedback history
// is keyed by group_id. 404 if no group exists for the caller's
// tenant + correlation_id (we don't leak existence of foreign rows).
func GetCorrelationGroup(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		groupID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Fetch the group itself (latest row for that correlation_id —
		// a tenant could in theory have multiple rows per correlation_id
		// if the background correlator re-fires; we surface the newest).
		var r alertCorrelationRow
		err = pool.Pgx().QueryRow(c.Request.Context(),
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
			        ac.created_at::text
			   FROM alert_correlations ac
			  WHERE ac.tenant_id = $1 AND ac.correlation_id = $2
			  ORDER BY ac.created_at DESC
			  LIMIT 1`, tenantID, groupID,
		).Scan(&r.ID, &r.TenantID, &r.CorrelationID, &r.RootAlertID,
			&r.MemberAlertIDs, &r.MemberCount,
			&r.SimilarityScore, &r.AutoDetected, &r.CreatedAt)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		// The members array includes the root + every member.
		allIDs := append([]string{r.RootAlertID}, r.MemberAlertIDs...)
		members := fetchGroupMembers(c, pool, tenantID, allIDs)
		hints := fetchGroupRCAHints(c, pool, tenantID, allIDs)
		feedback := fetchGroupFeedback(c, pool, tenantID, groupID)

		c.JSON(200, gin.H{
			"group":            r,
			"members":          members,
			"rca_hints":        hints,
			"feedback_history": feedback,
		})
	}
}

// fetchGroupMembers builds the union of anomaly_events +
// predictive_alerts that match the supplied alert IDs. Returns an
// empty slice if the SQL fails (defensive — the UI can render the
// "group" card without members).
func fetchGroupMembers(c *gin.Context, pool *db.Pool, tenantID uuid.UUID, alertIDs []string) []correlationMemberRow {
	memberRows, err := pool.Pgx().Query(c.Request.Context(),
		`SELECT ae.id::text, 'anomaly' AS source,
		        ae.severity, ae.metric_name,
		        ae.server_id::text, ae.ts::text
		   FROM intelligence_anomaly_events ae
		  WHERE ae.tenant_id = $1 AND ae.id = ANY($2::uuid[])
		  UNION ALL
		 SELECT pa.id::text, 'predict' AS source,
		        pa.severity, pa.metric_name,
		        pa.server_id::text, pa.created_at::text AS ts
		   FROM intelligence_predictive_alerts pa
		  WHERE pa.tenant_id = $1 AND pa.id = ANY($2::uuid[])`,
		tenantID, alertIDs,
	)
	if err != nil {
		return []correlationMemberRow{}
	}
	defer memberRows.Close()
	members := []correlationMemberRow{}
	for memberRows.Next() {
		var m correlationMemberRow
		var serverID *string
		if err := memberRows.Scan(&m.AlertID, &m.Source, &m.Severity,
			&m.MetricName, &serverID, &m.DetectedAt); err == nil {
			m.ServerID = serverID
			members = append(members, m)
		}
	}
	return members
}

// fetchGroupRCAHints returns the RCA hints for every supplied alert.
// Similar to fetchGroupMembers, returns an empty slice on SQL error.
func fetchGroupRCAHints(c *gin.Context, pool *db.Pool, tenantID uuid.UUID, alertIDs []string) []rcaHintRow {
	hintRows, err := pool.Pgx().Query(c.Request.Context(),
		`SELECT id::text, alert_id::text, likely_root, confidence,
		        reasoning,
		        COALESCE(similar_past_incidents, '[]'::jsonb),
		        created_at::text
		   FROM rca_hints
		  WHERE tenant_id = $1 AND alert_id = ANY($2::uuid[])
		  ORDER BY confidence DESC, created_at DESC`,
		tenantID, alertIDs,
	)
	if err != nil {
		return []rcaHintRow{}
	}
	defer hintRows.Close()
	hints := []rcaHintRow{}
	for hintRows.Next() {
		var h rcaHintRow
		var simJSON []byte
		if err := hintRows.Scan(&h.ID, &h.AlertID, &h.LikelyRoot,
			&h.Confidence, &h.Reasoning, &simJSON, &h.CreatedAt); err == nil {
			_ = json.Unmarshal(simJSON, &h.SimilarPastIncidents)
			hints = append(hints, h)
		}
	}
	return hints
}

// fetchGroupFeedback returns all feedback rows for the group.
func fetchGroupFeedback(c *gin.Context, pool *db.Pool, tenantID, groupID uuid.UUID) []correlationFeedbackRow {
	fbRows, err := pool.Pgx().Query(c.Request.Context(),
		`SELECT id::text, correlation_id::text, user_id::text,
		        useful, note, created_at::text
		   FROM correlation_feedback
		  WHERE tenant_id = $1 AND correlation_id = $2
		  ORDER BY created_at DESC`,
		tenantID, groupID,
	)
	if err != nil {
		return []correlationFeedbackRow{}
	}
	defer fbRows.Close()
	feedback := []correlationFeedbackRow{}
	for fbRows.Next() {
		var f correlationFeedbackRow
		var note *string
		if err := fbRows.Scan(&f.ID, &f.CorrelationID, &f.UserID,
			&f.Useful, &note, &f.CreatedAt); err == nil {
			f.Note = note
			feedback = append(feedback, f)
		}
	}
	return feedback
}
