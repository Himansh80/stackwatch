// Tier 10 Phase 9 — Task Scheduler (H10). 4 protected
// endpoints back the per-user scheduler definition surface
// (homelab_scheduler_jobs):
//
//	GET    /api/v1/homelab/scheduler/jobs  — ListHomelabSchedulerJobs
//	POST   /api/v1/homelab/scheduler/jobs  — CreateHomelabSchedulerJob
//	PATCH  /api/v1/homelab/scheduler/jobs/:id — PatchHomelabSchedulerJob
//	DELETE /api/v1/homelab/scheduler/jobs/:id — DeleteHomelabSchedulerJob
//
// SECURITY-CRITICAL. Every entry point enforces the
// guardrails documented in handlers_homelab_scheduler_types.go:
//   - action_kind in allowedActionKinds
//   - URL parses as http/https AND resolves to a non-private
//     IP (validateSchedulerURL)
//   - schedule parses as a 5-field cron expression
//     (ValidateCron in handlers_homelab_scheduler_cron.go)
//   - headers pass the allowlist (validateSchedulerHeaders)
//   - body respects the per-action rules
//     (validateSchedulerBody)
//   - per-user job count < schedulerJobCapPerUser (25)
//
// Per-user (NOT per-tenant): every WHERE clause filters by
// both tenant_id and user_id. A user can never see, modify,
// or run another user's job even if they share a tenant.
//
// The SchedulerWorker (internal/homelab/scheduler.go) ticks
// every 60s and dispatches due jobs to
// homelab.RunSchedulerJobAndInsertRun (the same helper the
// POST /jobs/:id/run handler uses for manual triggers).
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// decodeSchedulerHeaders is the JSONB decoder used by
// newSchedulerJobRowFromScan (in scheduler_types.go). Kept
// here so the types file doesn't import encoding/json.
func decodeSchedulerHeaders(raw []byte) (map[string]string, error) {
	out := map[string]string{}
	if len(raw) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode headers: %w", err)
	}
	return out, nil
}

// schedulerJobCount is the per-user job count the
// CreateHandler enforces before INSERT.
func schedulerJobCount(ctx *gin.Context, pool *db.Pool, tenantID, userID uuid.UUID) (int, error) {
	var n int
	if err := pool.Pgx().QueryRow(ctx.Request.Context(),
		`SELECT COUNT(*)::int FROM homelab_scheduler_jobs
		  WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ------------------------------------------------------------------
// GET /api/v1/homelab/scheduler/jobs
// ------------------------------------------------------------------

// ListHomelabSchedulerJobs returns every job the caller has
// defined, ordered by created_at DESC. Each row carries
// cap_remaining so the widget can show "23 / 25 jobs used"
// without a follow-up count call.
func ListHomelabSchedulerJobs(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		limit := schedulerJobListLimit
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			n := 0
			for _, ch := range v {
				if ch < '0' || ch > '9' {
					kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "limit must be a non-negative integer")
					return
				}
				n = n*10 + int(ch-'0')
				if n > schedulerJobListLimitMax {
					n = schedulerJobListLimitMax
				}
			}
			if n > 0 {
				limit = n
			}
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, tenant_id::text, user_id::text,
			        name, action_kind, url, method, headers, body,
			        schedule, enabled,
			        last_run_at::text, next_run_at::text,
			        last_run_status, last_run_error,
			        created_at::text, updated_at::text
			   FROM homelab_scheduler_jobs
			  WHERE tenant_id = $1 AND user_id = $2
			  ORDER BY created_at DESC
			  LIMIT $3`,
			tenantID, userID, limit,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		used := 0
		out := []schedulerJobRow{}
		for rows.Next() {
			var s schedulerJobRowScan
			if err := rows.Scan(&s.ID, &s.TenantID, &s.UserID,
				&s.Name, &s.ActionKind, &s.URL, &s.Method, &s.HeadersRaw, &s.Body,
				&s.Schedule, &s.Enabled,
				&s.LastRunAt, &s.NextRunAt,
				&s.LastRunStatus, &s.LastRunError,
				&s.CreatedAt, &s.UpdatedAt); err != nil {
				kernel.RespondError(c, err)
				return
			}
			row := newSchedulerJobRowFromScan(s)
			out = append(out, row)
			used++
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Stamp every row with the same cap_remaining so
		// the frontend can use any one of them for the badge.
		remaining := schedulerJobCapPerUser - used
		if remaining < 0 {
			remaining = 0
		}
		for i := range out {
			out[i].CapRemaining = remaining
		}
		kernel.RespondOK(c, gin.H{
			"jobs":          out,
			"count":         len(out),
			"cap_per_user":  schedulerJobCapPerUser,
			"cap_remaining": remaining,
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/scheduler/jobs
// ------------------------------------------------------------------

// CreateHomelabSchedulerJob inserts a new job for the caller.
// All security guardrails enforced BEFORE the INSERT (so a
// rejected request never creates a row):
//   - action_kind in allowedActionKinds
//   - URL parses + non-private host
//   - schedule parses as 5-field cron
//   - headers pass allowlist
//   - body respects per-action rules
//   - per-user job count < schedulerJobCapPerUser
//
// On success, computes next_run_at via the cron parser so
// the worker picks it up on its next tick.
func CreateHomelabSchedulerJob(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, ok := auth.UserIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		var req schedulerJobReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if req.Name == nil || req.ActionKind == nil || req.URL == nil || req.Schedule == nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"name, action_kind, url, and schedule are required")
			return
		}
		name := strings.TrimSpace(*req.Name)
		if err := validateSchedulerName(name); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		actionKind := strings.TrimSpace(*req.ActionKind)
		if err := validateSchedulerActionKind(actionKind); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		rawURL := strings.TrimSpace(*req.URL)
		if err := validateSchedulerURL(rawURL); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		schedule := strings.TrimSpace(*req.Schedule)
		if err := validateSchedulerSchedule(schedule); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		method := normalizeSchedulerMethod(actionKind, req.Method)

		// Headers + body
		var headersMap map[string]string
		if req.Headers != nil {
			cleaned, herr := validateSchedulerHeaders(*req.Headers)
			if herr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", herr.Error())
				return
			}
			headersMap = cleaned
		} else {
			headersMap = map[string]string{}
		}
		// Body
		body := ""
		if req.Body != nil {
			body = strings.TrimSpace(*req.Body)
		}
		if err := validateSchedulerBody(actionKind, body); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		// Body content-type check: if a body is provided,
		// the corresponding Content-Type header MUST be in
		// the allowlist.
		if body != "" {
			if !schedulerBodyHasAllowedContentType(headersMap) {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"POST body requires a Content-Type header in the allowlist (text/plain, application/json, application/x-www-form-urlencoded)")
				return
			}
		}

		// Default enabled
		enabled := true
		if req.Enabled != nil {
			enabled = *req.Enabled
		}

		// Per-user cap check (must happen BEFORE the INSERT
		// so a rejected request doesn't leave a half-state).
		used, cerr := schedulerJobCount(c, pool, tenantID, userID)
		if cerr != nil {
			kernel.RespondError(c, cerr)
			return
		}
		if used >= schedulerJobCapPerUser {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "cap_exceeded",
				fmt.Sprintf("per-user job cap of %d reached; delete a job before adding more", schedulerJobCapPerUser))
			return
		}

		headersJSON, herr := json.Marshal(headersMap)
		if herr != nil {
			kernel.RespondError(c, herr)
			return
		}

		// Compute next_run_at from schedule. We use the
		// handler-side parser (matches the worker-side
		// one — see scheduler_cron.go in homelab for the
		// worker mirror).
		expr, perr := ParseCron(schedule)
		var nextRun interface{}
		if perr == nil {
			next, nerr := expr.Next(time.Now().UTC())
			if nerr == nil {
				nextRun = next
			}
		}
		if nextRun == nil {
			// Couldn't parse — should already have been
			// caught by validateSchedulerSchedule above,
			// but defensively fail closed.
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"schedule could not be projected into a future run time")
			return
		}

		var (
			id        string
			createdAt string
			updatedAt string
		)
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO homelab_scheduler_jobs
			    (tenant_id, user_id, name, action_kind, url, method,
			     headers, body, schedule, enabled, next_run_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			 RETURNING id::text, created_at::text, updated_at::text`,
			tenantID, userID, name, actionKind, rawURL, method,
			headersJSON, nullableString(body), schedule, enabled, nextRun,
		).Scan(&id, &createdAt, &updatedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		kernel.RespondCreated(c, gin.H{
			"id":            id,
			"tenant_id":     tenantID.String(),
			"user_id":       userID.String(),
			"name":          name,
			"action_kind":   actionKind,
			"url":           rawURL,
			"method":        method,
			"headers":       headersMap,
			"body":          body,
			"schedule":      schedule,
			"enabled":       enabled,
			"next_run_at":   fmt.Sprintf("%v", nextRun),
			"created_at":    createdAt,
			"updated_at":    updatedAt,
			"cap_remaining": schedulerJobCapPerUser - used - 1,
		})
	}
}
