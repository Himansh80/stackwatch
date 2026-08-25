// Tier 10 Phase 9 — Task Scheduler (H10). 2 protected
// endpoints for the run-history + manual-trigger surfaces
// (homelab_scheduler_runs + the manual-trigger invocation
// of homelab.RunSchedulerJobAndInsertRun):
//
//	GET  /api/v1/homelab/scheduler/runs       — ListHomelabSchedulerRuns
//	POST /api/v1/homelab/scheduler/jobs/:id/run — RunHomelabSchedulerJobNow
//
// Per-user (NOT per-tenant): every WHERE clause filters by
// both tenant_id and user_id. A user can never see or run
// another user's jobs even if they share a tenant.
//
// Manual trigger semantics: 202 Accepted. The actual run
// executes synchronously inside the request because runs
// are bounded (30s timeout) and we want the caller to
// immediately know whether the job's URL is reachable —
// the dashboard's "Run now" button surfaces the latest
// run row when the user refreshes the modal.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/homelab"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// GET /api/v1/homelab/scheduler/runs
// ------------------------------------------------------------------

// ListHomelabSchedulerRuns returns recent runs for the
// caller's jobs. Query params:
//   job_id — optional, restrict to one job
//   limit  — optional (default 50, max 200)
//
// Results ordered by started_at DESC.
func ListHomelabSchedulerRuns(pool *db.Pool) gin.HandlerFunc {
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

		// Parse + validate query params.
		jobIDStr := strings.TrimSpace(c.Query("job_id"))
		limit := schedulerRunListLimit
		if v := strings.TrimSpace(c.Query("limit")); v != "" {
			n := 0
			for _, ch := range v {
				if ch < '0' || ch > '9' {
					kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "limit must be a non-negative integer")
					return
				}
				n = n*10 + int(ch-'0')
				if n > schedulerRunListLimitMax {
					n = schedulerRunListLimitMax
				}
			}
			if n > 0 {
				limit = n
			}
		}

		args := []interface{}{tenantID, userID}
		conds := []string{"r.tenant_id = $1", "r.user_id = $2"}
		if jobIDStr != "" {
			jfid, ferr := uuid.Parse(jobIDStr)
			if ferr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "job_id must be a uuid")
				return
			}
			args = append(args, jfid)
			conds = append(conds, fmt.Sprintf("r.job_id = $%d", len(args)))
		}
		args = append(args, limit)
		limitPlaceholder := fmt.Sprintf("$%d", len(args))

		query := fmt.Sprintf(
			`SELECT r.id::text, r.job_id::text, r.started_at::text,
			        COALESCE(r.finished_at::text, ''),
			        r.status, r.http_status_code, r.response_bytes,
			        COALESCE(r.error_message, ''), r.duration_ms
			   FROM homelab_scheduler_runs r
			  WHERE %s
			  ORDER BY r.started_at DESC
			  LIMIT %s`,
			strings.Join(conds, " AND "),
			limitPlaceholder,
		)

		rows, err := pool.Pgx().Query(c.Request.Context(), query, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		out := []schedulerRunRow{}
		for rows.Next() {
			var (
				id, jobID, startedAt, finishedAt, status, errMsg string
				httpStatus                                       *int
				respBytes                                        *int
				durationMS                                       *int
			)
			if err := rows.Scan(&id, &jobID, &startedAt, &finishedAt,
				&status, &httpStatus, &respBytes, &errMsg, &durationMS); err != nil {
				kernel.RespondError(c, err)
				return
			}
			row := schedulerRunRow{
				ID:           id,
				JobID:        jobID,
				StartedAt:    startedAt,
				FinishedAt:   finishedAt,
				Status:       status,
				ErrorMessage: errMsg,
			}
			if httpStatus != nil {
				row.HTTPStatusCode = httpStatus
			}
			if respBytes != nil {
				row.ResponseBytes = respBytes
			}
			if durationMS != nil {
				row.DurationMS = durationMS
			}
			out = append(out, row)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"runs":  out,
			"count": len(out),
			"limit": limit,
		})
	}
}

// ------------------------------------------------------------------
// POST /api/v1/homelab/scheduler/jobs/:id/run
// ------------------------------------------------------------------

// RunHomelabSchedulerJobNow synchronously triggers one run
// of the given job. Returns 202 Accepted with the outcome
// of the run (or a descriptive error if the run failed
// before any HTTP request).
//
// The job must belong to the caller (tenant_id + user_id).
// The handler re-validates URL + DNS at runtime via the
// same RunSchedulerJobAndInsertRun helper the worker uses —
// keeping the two paths in lock-step.
func RunHomelabSchedulerJobNow(pool *db.Pool) gin.HandlerFunc {
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
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "id must be a uuid")
			return
		}

		// Verify the job belongs to the caller before
		// invoking RunSchedulerJobAndInsertRun (which has
		// its own check, but doing it here too gives a
		// cleaner 404 vs 403 distinction for the API
		// consumer).
		var exists bool
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT EXISTS (
			    SELECT 1 FROM homelab_scheduler_jobs
			     WHERE id = $1 AND tenant_id = $2 AND user_id = $3
			   )`,
			id, tenantID, userID,
		).Scan(&exists); err != nil {
			kernel.RespondError(c, err)
			return
		}
		if !exists {
			kernel.RespondErrorWithCode(c, http.StatusNotFound, "not_found", "job not found")
			return
		}

		out, rerr := homelab.RunSchedulerJobAndInsertRun(c.Request.Context(), pool, id, nil)
		if rerr != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "run_failed", rerr.Error())
			return
		}
		kernel.RespondOK(c, gin.H{
			"accepted":      true,
			"job_id":        out.JobID.String(),
			"status":        out.Status,
			"http_status":   out.HTTPStatus,
			"response_bytes": out.ResponseBytes,
			"duration_ms":   out.DurationMS,
			"error_message": out.ErrorMessage,
			"skipped_reason": out.SkippedReason,
		})
	}
}
