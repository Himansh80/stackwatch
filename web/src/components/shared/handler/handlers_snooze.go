// Tier 8 Phase 4 — Alert Deduplication + Noise Reduction (Tier 8.4).
// HTTP route handlers for the 2 snooze endpoints:
//
//	POST /api/v1/noise/snooze   — snooze an alert (inserts into snooze_log)
//	GET  /api/v1/noise/history  — snooze history (filter ?alert_id&limit)
//
// The 4 noise-rule endpoints live in handlers_noise.go. Types live
// in handlers_noise_types.go. Both files split this way so each
// stays under the 400-LOC cap.
//
// Snooze semantics: every snooze inserts a snooze_log row with
// `expires_at = now() + duration_seconds`. The noise-reduction
// engine (Phase 5+) treats an alert as suppressed while at least
// one snooze_log row for it has `expires_at > now()`. Multiple
// concurrent snoozes are kept as separate rows (audit trail) — the
// query "active snoozes" filters on `expires_at > now()`.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// SnoozeAlert inserts a new snooze row. user_id is stamped from the
// JWT (never from the body) so a user can't snooze on behalf of
// someone else. expires_at is computed server-side so a tampered
// body can't extend a snooze indefinitely.
func SnoozeAlert(pool *db.Pool) gin.HandlerFunc {
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
		var req snoozeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		alertID, err := uuid.Parse(strings.TrimSpace(req.AlertID))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Validate duration (binding already enforces 60..604800).
		if req.DurationSeconds < 60 {
			kernel.RespondErrorWithCode(c, 400, "bad_request",
				"duration_seconds must be >= 60")
			return
		}

		// We don't currently FK-check the alert_id against the
		// anomaly / predictive tables because the underlying alert
		// could be from a stream we don't have in this tenant, OR
		// it could be a manually-snoozed incident id. We just stamp
		// the row. Phase 5 will add the cross-table validation.
		reasonText := strings.TrimSpace(req.Reason)
		var rowID, expiresAt, createdAt string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO snooze_log
			    (tenant_id, alert_id, user_id, duration_seconds,
			     expires_at, reason)
			 VALUES ($1, $2, $3, $4,
			         now() + ($4 || ' seconds')::interval,
			         NULLIF($5, ''))
			 RETURNING id::text, expires_at::text, created_at::text`,
			tenantID, alertID, claims.UserID, req.DurationSeconds, reasonText,
		).Scan(&rowID, &expiresAt, &createdAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		reasonPtr := stringPtrIf(reasonText)
		kernel.RespondCreated(c, snoozeLogRow{
			ID:              rowID,
			TenantID:        tenantID.String(),
			AlertID:         alertID.String(),
			UserID:          claims.UserID.String(),
			DurationSeconds: req.DurationSeconds,
			ExpiresAt:       expiresAt,
			Reason:          reasonPtr,
			Active:          true,
			CreatedAt:       createdAt,
		})
	}
}

// ListSnoozeHistory returns the most recent snooze_log rows for the
// caller's tenant, newest first. Optional filters:
//
//	?alert_id=<uuid>   — only snoozes for this alert
//	?active=true|false — only currently-active or expired rows
//	?limit=50          — clamp 1..500 (default 50)
//
// The `active` column is computed at scan time (expires_at > now())
// so the UI can show "still snoozed" vs "expired" without a second
// pass.
func ListSnoozeHistory(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := clampLimit(c.Query("limit"), 50, 500)

		args := []interface{}{tenantID}
		where := "tenant_id = $1"

		if raw := strings.ToLower(strings.TrimSpace(c.Query("alert_id"))); raw != "" {
			if _, err := uuid.Parse(raw); err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
			args = append(args, raw)
			where += " AND alert_id = $" + itoa(len(args))
		}
		activeRaw := strings.ToLower(strings.TrimSpace(c.Query("active")))
		if activeRaw == "true" {
			where += " AND expires_at > now()"
		} else if activeRaw == "false" {
			where += " AND expires_at <= now()"
		}

		args = append(args, limit)
		limitIdx := len(args)

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, alert_id::text,
			        user_id::text, duration_seconds,
			        expires_at::text, reason, created_at::text,
			        (expires_at > now()) AS active
			   FROM snooze_log
			  WHERE `+where+`
			  ORDER BY created_at DESC
			  LIMIT $`+itoa(limitIdx), args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []snoozeLogRow{}
		for rows.Next() {
			var r snoozeLogRow
			var reason *string
			var active bool
			if err := rows.Scan(&r.ID, &r.TenantID, &r.AlertID,
				&r.UserID, &r.DurationSeconds, &r.ExpiresAt,
				&reason, &r.CreatedAt, &active); err == nil {
				r.Reason = reason
				r.Active = active
				out = append(out, r)
			}
		}
		kernel.RespondOK(c, gin.H{
			"snoozes": out,
			"total":   len(out),
			"limit":   limit,
		})
	}
}

// stringPtrIf returns a pointer to s if non-empty, else nil. Used so
// the JSON response uses `null` (omitempty) for an unset reason.
func stringPtrIf(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
