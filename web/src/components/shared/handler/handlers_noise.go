// Tier 8 Phase 4 — Alert Deduplication + Noise Reduction (Tier 8.4).
// HTTP route handlers for the 4 noise-rule endpoints:
//
//	GET    /api/v1/noise/rules      — list noise rules (filter ?enabled)
//	POST   /api/v1/noise/rules      — create a noise rule
//	DELETE /api/v1/noise/rules/:id  — delete a noise rule
//	POST   /api/v1/noise/test       — preview suppression count
//
// The 2 snooze endpoints (POST /noise/snooze + GET /noise/history)
// live in handlers_snooze.go so this file stays under the 400-LOC cap.
// Types live in handlers_noise_types.go.
//
// Pattern matching: `fingerprint_pattern` is stored verbatim; the
// SQL `LIKE` operator is used with `%` wildcards at evaluation time.
// The handler converts any leading / trailing `*` from the UI to
// `%` so the glob-style "filter" UI feels natural.
package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListNoiseRules returns all noise rules for the caller's tenant,
// newest first. Optional filter: ?enabled=true|false.
func ListNoiseRules(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := clampLimit(c.Query("limit"), 100, 500)

		args := []interface{}{tenantID}
		where := "tenant_id = $1"
		enabledRaw := strings.ToLower(strings.TrimSpace(c.Query("enabled")))
		if enabledRaw == "true" || enabledRaw == "false" {
			args = append(args, enabledRaw == "true")
			where += " AND enabled = $" + itoa(len(args))
		}
		args = append(args, limit)
		limitIdx := len(args)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, name,
			        fingerprint_pattern, suppression_window_seconds,
			        COALESCE(channels, '{}'::text[]) AS channels,
			        enabled, created_at::text
			   FROM alert_noise_rules
			  WHERE `+where+`
			  ORDER BY created_at DESC
			  LIMIT $`+itoa(limitIdx), args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []noiseRuleRow{}
		for rows.Next() {
			var r noiseRuleRow
			if err := rows.Scan(&r.ID, &r.TenantID, &r.Name,
				&r.FingerprintPattern, &r.SuppressionWindowSeconds,
				&r.Channels, &r.Enabled, &r.CreatedAt); err == nil {
				out = append(out, r)
			}
		}
		kernel.RespondOK(c, gin.H{
			"rules": out,
			"total": len(out),
			"limit": limit,
		})
	}
}

// CreateNoiseRule inserts a new noise rule for the caller's tenant.
// The fingerprint_pattern is stored verbatim; we accept either glob
// ("*cpu*") or LIKE ("%cpu%") syntax and normalize internally at
// evaluation time (preview handler). On the storage path we keep it
// as-is so the UI can render the original pattern back to the user.
func CreateNoiseRule(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req noiseRuleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Normalize channels (lowercase + dedup + whitelist).
		channels := normalizeChannels(req.Channels)

		// Default window: 1h. Default enabled: true.
		window := req.SuppressionWindowSeconds
		if window <= 0 {
			window = 3600
		}
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		var rowID, createdAt string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO alert_noise_rules
			    (tenant_id, name, fingerprint_pattern,
			     suppression_window_seconds, channels, enabled)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 RETURNING id::text, created_at::text`,
			tenantID, strings.TrimSpace(req.Name),
			strings.TrimSpace(req.FingerprintPattern),
			window, channels, enabled,
		).Scan(&rowID, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, noiseRuleRow{
			ID:                       rowID,
			TenantID:                 tenantID.String(),
			Name:                     strings.TrimSpace(req.Name),
			FingerprintPattern:       strings.TrimSpace(req.FingerprintPattern),
			SuppressionWindowSeconds: window,
			Channels:                 channels,
			Enabled:                  enabled,
			CreatedAt:                createdAt,
		})
	}
}

// DeleteNoiseRule removes a noise rule by id. 404 if the rule
// doesn't exist for the caller's tenant (we don't leak existence
// of rules in other tenants).
func DeleteNoiseRule(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM alert_noise_rules
			  WHERE tenant_id = $1 AND id = $2`,
			tenantID, id,
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
			"id":      id.String(),
			"deleted": true,
		})
	}
}

// PreviewNoiseRule answers "how many alerts would this rule
// suppress right now?" Returns {would_suppress_count, sample_alerts}.
//
// We match by joining the rules table against the union of
// intelligence_anomaly_events + intelligence_predictive_alerts on the
// metric_name fingerprint via SQL LIKE. We use the rule's window
// literally: "alerts in the last `window_seconds` that match the
// pattern" — so the preview window is exactly what would happen on
// the next evaluation.
func PreviewNoiseRule(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req noiseTestRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Convert glob → LIKE pattern. We replace any `*` with `%`
		// and escape `%` / `_` that the user might have meant
		// literally. This keeps the storage verbatim ("*cpu*") but
		// lets the preview use native LIKE semantics.
		like := globToLike(req.FingerprintPattern)

		var count int
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM (
			   SELECT ae.id::text AS alert_id,
			          ae.metric_name,
			          ae.severity,
			          ae.ts::text AS detected_at
			     FROM intelligence_anomaly_events ae
			    WHERE ae.tenant_id = $1
			      AND ae.metric_name LIKE $2
			      AND ae.ts >= now() - ($3 || ' seconds')::interval
			   UNION ALL
			   SELECT pa.id::text,
			          pa.metric_name,
			          pa.severity,
			          pa.created_at::text
			     FROM intelligence_predictive_alerts pa
			    WHERE pa.tenant_id = $1
			      AND pa.metric_name LIKE $2
			      AND pa.created_at >= now() - ($3 || ' seconds')::interval
			 ) m`,
			tenantID, like, req.WindowSeconds,
		).Scan(&count)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Sample the most recent 5 matching alert IDs so the UI
		// can show "this rule would catch: …" without forcing a
		// second request.
		sampleRows, _ := pool.Pgx().Query(c.Request.Context(),
			`SELECT alert_id FROM (
			   SELECT ae.id::text AS alert_id, ae.ts::text AS ts
			     FROM intelligence_anomaly_events ae
			    WHERE ae.tenant_id = $1
			      AND ae.metric_name LIKE $2
			      AND ae.ts >= now() - ($3 || ' seconds')::interval
			   UNION ALL
			   SELECT pa.id::text, pa.created_at::text
			     FROM intelligence_predictive_alerts pa
			    WHERE pa.tenant_id = $1
			      AND pa.metric_name LIKE $2
			      AND pa.created_at >= now() - ($3 || ' seconds')::interval
			 ) m
			 ORDER BY ts DESC
			 LIMIT 5`,
			tenantID, like, req.WindowSeconds,
		)
		samples := []string{}
		if sampleRows != nil {
			defer sampleRows.Close()
			for sampleRows.Next() {
				var aid string
				if err := sampleRows.Scan(&aid); err == nil {
					samples = append(samples, aid)
				}
			}
		}

		kernel.RespondOK(c, gin.H{
			"would_suppress_count": count,
			"window_seconds":       req.WindowSeconds,
			"sample_alerts":        samples,
			"matched_pattern":      like,
		})
	}
}

// globToLike converts a user-friendly glob pattern to SQL LIKE
// syntax. `*` → `%`, `?` → `_`. Existing `%` and `_` characters
// in the input are escaped so they don't accidentally behave as
// wildcards. The result is safe to bind as a parameter.
func globToLike(glob string) string {
	var sb strings.Builder
	sb.Grow(len(glob) + 4)
	for _, ch := range glob {
		switch ch {
		case '*':
			sb.WriteByte('%')
		case '?':
			sb.WriteByte('_')
		case '%', '_':
			sb.WriteByte('\\')
			sb.WriteRune(ch)
		default:
			sb.WriteRune(ch)
		}
	}
	return sb.String()
}

// normalizeChannels lowercases, dedups, and whitelists the channel
// list. Returns an empty slice (NOT nil) when no valid channels are
// supplied so the JSON serializes as `[]` rather than `null`.
func normalizeChannels(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		v := strings.ToLower(strings.TrimSpace(raw))
		if v == "" {
			continue
		}
		if !allowedChannels[v] {
			continue
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

// Compile-time guard so the time import stays referenced even if a
// downstream refactor trims a usage — keeps the import list honest.
var _ = time.RFC3339
