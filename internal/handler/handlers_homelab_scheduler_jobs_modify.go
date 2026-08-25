// Tier 10 Phase 9 — Task Scheduler (H10). PATCH + DELETE
// handlers + the body content-type helper. Split out of
// handlers_homelab_scheduler_jobs.go so each file stays
// under the 400-LOC cap.
//
//	PATCH  /api/v1/homelab/scheduler/jobs/:id — PatchHomelabSchedulerJob
//	DELETE /api/v1/homelab/scheduler/jobs/:id — DeleteHomelabSchedulerJob
//
// Same security model as the create handler: every PATCH
// field is re-validated before reaching the SQL. ON DELETE
// CASCADE on homelab_scheduler_runs.job_id purges the run
// history in one transaction.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// PATCH /api/v1/homelab/scheduler/jobs/:id
// ------------------------------------------------------------------

// PatchHomelabSchedulerJob updates any subset of the mutable
// fields. user_id and tenant_id are immutable (the URL
// itself is the security boundary — you can't move a job
// to another user's namespace via PATCH).
//
// Every PATCHed field goes through the same validators as
// POST so a PATCH can't bypass them.
func PatchHomelabSchedulerJob(pool *db.Pool) gin.HandlerFunc {
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

		var req schedulerJobReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Build the SET clause dynamically based on which
		// fields were supplied.
		sets := []string{}
		args := []interface{}{id, tenantID, userID}
		argIdx := 4 // $1=id, $2=tenant, $3=user, others start at $4

		if req.Name != nil {
			name := strings.TrimSpace(*req.Name)
			if err := validateSchedulerName(name); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
			args = append(args, name)
			argIdx++
		}
		if req.ActionKind != nil {
			ak := strings.TrimSpace(*req.ActionKind)
			if err := validateSchedulerActionKind(ak); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			sets = append(sets, fmt.Sprintf("action_kind = $%d", argIdx))
			args = append(args, ak)
			argIdx++
		}
		if req.URL != nil {
			rawURL := strings.TrimSpace(*req.URL)
			if err := validateSchedulerURL(rawURL); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			sets = append(sets, fmt.Sprintf("url = $%d", argIdx))
			args = append(args, rawURL)
			argIdx++
		}
		if req.Method != nil {
			m := strings.ToUpper(strings.TrimSpace(*req.Method))
			if m != "GET" && m != "POST" {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "method must be GET or POST")
				return
			}
			sets = append(sets, fmt.Sprintf("method = $%d", argIdx))
			args = append(args, m)
			argIdx++
		}
		if req.Headers != nil {
			cleaned, herr := validateSchedulerHeaders(*req.Headers)
			if herr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", herr.Error())
				return
			}
			headersJSON, jerr := json.Marshal(cleaned)
			if jerr != nil {
				kernel.RespondError(c, jerr)
				return
			}
			sets = append(sets, fmt.Sprintf("headers = $%d", argIdx))
			args = append(args, headersJSON)
			argIdx++
		}
		if req.Body != nil {
			body := strings.TrimSpace(*req.Body)
			// We need the action_kind to validate the body
			// rules — read it from the row if not supplied
			// in the PATCH.
			effectiveKind := ""
			if req.ActionKind != nil {
				effectiveKind = strings.TrimSpace(*req.ActionKind)
			} else {
				if err := pool.Pgx().QueryRow(c.Request.Context(),
					`SELECT action_kind FROM homelab_scheduler_jobs
					  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
					id, tenantID, userID,
				).Scan(&effectiveKind); err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						kernel.RespondErrorWithCode(c, http.StatusNotFound, "not_found", "job not found")
						return
					}
					kernel.RespondError(c, err)
					return
				}
			}
			if err := validateSchedulerBody(effectiveKind, body); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			sets = append(sets, fmt.Sprintf("body = $%d", argIdx))
			args = append(args, nullableString(body))
			argIdx++
		}
		if req.Schedule != nil {
			sched := strings.TrimSpace(*req.Schedule)
			if err := validateSchedulerSchedule(sched); err != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
				return
			}
			sets = append(sets, fmt.Sprintf("schedule = $%d", argIdx))
			args = append(args, sched)
			argIdx++
			// Recompute next_run_at from the new schedule.
			expr, perr := ParseCron(sched)
			if perr == nil {
				next, nerr := expr.Next(time.Now().UTC())
				if nerr == nil {
					sets = append(sets, fmt.Sprintf("next_run_at = $%d", argIdx))
					args = append(args, next)
					argIdx++
				}
			}
		}
		if req.Enabled != nil {
			sets = append(sets, fmt.Sprintf("enabled = $%d", argIdx))
			args = append(args, *req.Enabled)
			argIdx++
		}

		if len(sets) == 0 {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "no fields to update")
			return
		}

		// Always update updated_at.
		sets = append(sets, "updated_at = NOW()")

		query := fmt.Sprintf(
			`UPDATE homelab_scheduler_jobs
			    SET %s
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			strings.Join(sets, ", "),
		)
		tag, err := pool.Pgx().Exec(c.Request.Context(), query, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondErrorWithCode(c, http.StatusNotFound, "not_found", "job not found")
			return
		}
		kernel.RespondOK(c, gin.H{
			"id":            id.String(),
			"updated":       tag.RowsAffected(),
			"idempotent_ok": false,
		})
	}
}

// ------------------------------------------------------------------
// DELETE /api/v1/homelab/scheduler/jobs/:id
// ------------------------------------------------------------------

// DeleteHomelabSchedulerJob removes a job (and its runs via
// ON DELETE CASCADE FK). Idempotent: 200 with {deleted: 0}
// when the row doesn't exist or isn't the caller's.
func DeleteHomelabSchedulerJob(pool *db.Pool) gin.HandlerFunc {
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
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM homelab_scheduler_jobs
			  WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
			id, tenantID, userID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{
			"deleted":       tag.RowsAffected(),
			"id":            id.String(),
			"idempotent_ok": true,
		})
	}
}

// schedulerBodyHasAllowedContentType returns true when the
// header map contains a Content-Type header whose value is
// in allowedSchedulerBodyContentTypes (case-insensitive on
// the type/subtype; the params are ignored).
func schedulerBodyHasAllowedContentType(headers map[string]string) bool {
	var ct string
	for k, v := range headers {
		if strings.EqualFold(k, "Content-Type") {
			ct = v
			break
		}
	}
	if ct == "" {
		return false
	}
	// Strip "; charset=utf-8" etc.
	if idx := strings.Index(ct, ";"); idx >= 0 {
		ct = strings.TrimSpace(ct[:idx])
	}
	ct = strings.ToLower(ct)
	_, ok := allowedSchedulerBodyContentTypes[ct]
	return ok
}
