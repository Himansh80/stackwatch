// Package homelab — Task Scheduler (H10) DB helpers (split
// out of scheduler_run.go to keep that file under the 400-
// LOC cap). The functions here are the read + write sides
// of the run cycle: fetch a job's row, write the run row,
// update the parent's bookkeeping fields. RunSchedulerJobAndInsertRun
// (in scheduler_run.go) is the orchestration layer.
package homelab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/db"
)

var errSchedulerJobNotFound = errors.New("scheduler job not found")

var errSchedulerJobLastRunUnknown = errors.New("scheduler job last_run_at unknown")

// fetchSchedulerJob loads one row by ID. Returns
// errSchedulerJobNotFound when no row matches.
func fetchSchedulerJob(ctx context.Context, pool *db.Pool, id uuid.UUID) (schedulerJobSnapshot, error) {
	var (
		out     schedulerJobSnapshot
		headers []byte
		body    *string
		enabled bool
	)
	row := pool.Pgx().QueryRow(ctx,
		`SELECT id::text, tenant_id::text, user_id::text,
		        action_kind, url, method, headers, body,
		        schedule, enabled
		   FROM homelab_scheduler_jobs
		  WHERE id = $1`, id)
	var (
		idStr, tStr, uStr, action, urlStr, method, sched string
	)
	if err := row.Scan(&idStr, &tStr, &uStr, &action, &urlStr, &method, &headers, &body, &sched, &enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return out, errSchedulerJobNotFound
		}
		return out, err
	}
	out.ID = id
	if v, perr := uuid.Parse(tStr); perr == nil {
		out.TenantID = v
	}
	if v, perr := uuid.Parse(uStr); perr == nil {
		out.UserID = v
	}
	out.ActionKind = action
	out.URL = urlStr
	out.Method = method
	out.Schedule = sched
	out.Enabled = enabled
	if body != nil {
		out.Body = *body
	}
	out.Headers = decodeSchedulerHeadersBytes(headers)
	return out, nil
}

// fetchSchedulerJobLastRunAt returns the job's last_run_at
// or the zero time when it has never run.
func fetchSchedulerJobLastRunAt(ctx context.Context, pool *db.Pool, id uuid.UUID) (time.Time, error) {
	var lastRun *time.Time
	if err := pool.Pgx().QueryRow(ctx,
		`SELECT last_run_at FROM homelab_scheduler_jobs WHERE id = $1`, id,
	).Scan(&lastRun); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, errSchedulerJobLastRunUnknown
		}
		return time.Time{}, err
	}
	if lastRun == nil {
		return time.Time{}, errSchedulerJobLastRunUnknown
	}
	return *lastRun, nil
}

// recordSchedulerRunFailure is the write side for a failed
// run. It INSERTs a homelab_scheduler_runs row + UPDATEs the
// parent's bookkeeping fields. Used by RunSchedulerJobAndInsertRun
// when the HTTP request errors before any response.
func recordSchedulerRunFailure(
	ctx context.Context,
	pool *db.Pool,
	job schedulerJobSnapshot,
	errMsg string,
	httpStatus int,
	respBytes int,
	startedAt time.Time,
) error {
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	duration := time.Since(startedAt)
	out := schedulerRunOutcome{
		JobID:         job.ID,
		Status:        "failure",
		HTTPStatus:    httpStatus,
		ResponseBytes: respBytes,
		ErrorMessage:  truncateSchedulerErrMsg(errMsg),
		DurationMS:    int(duration / time.Millisecond),
	}
	return insertSchedulerRunAndUpdateJob(ctx, pool, job, out, startedAt, duration)
}

// insertSchedulerRunAndUpdateJob writes the run row + the
// parent's next_run_at in one helper. Used by both the
// success and failure paths.
func insertSchedulerRunAndUpdateJob(
	ctx context.Context,
	pool *db.Pool,
	job schedulerJobSnapshot,
	out schedulerRunOutcome,
	startedAt time.Time,
	duration time.Duration,
) error {
	// Compute next_run_at via the cron parser. The parser
	// lives in scheduler_cron.go in this package.
	next, nerr := NextCronRun(job.Schedule, startedAt)
	if nerr != nil {
		// Schedule became invalid since insert (e.g. the
		// admin tightened the parser). Leave next_run_at
		// alone — the job will keep firing on its current
		// next_run_at, and the next PATCH/DELETE cycle
		// will fix it.
		next = time.Time{}
	}

	tx, terr := pool.Pgx().Begin(ctx)
	if terr != nil {
		return terr
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// 1. INSERT the run row.
	var httpStatusArg interface{}
	if out.HTTPStatus > 0 {
		httpStatusArg = out.HTTPStatus
	}
	var respBytesArg interface{}
	if out.ResponseBytes > 0 {
		respBytesArg = out.ResponseBytes
	}
	var errMsgArg interface{}
	if out.ErrorMessage != "" {
		errMsgArg = out.ErrorMessage
	}
	if _, ierr := tx.Exec(ctx,
		`INSERT INTO homelab_scheduler_runs
		    (tenant_id, user_id, job_id, started_at, finished_at,
		     status, http_status_code, response_bytes, error_message,
		     duration_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		job.TenantID, job.UserID, job.ID,
		startedAt, startedAt.Add(duration),
		out.Status, httpStatusArg, respBytesArg, errMsgArg,
		out.DurationMS,
	); ierr != nil {
		return fmt.Errorf("insert run: %w", ierr)
	}

	// 2. UPDATE the parent job's bookkeeping fields.
	if _, uerr := tx.Exec(ctx,
		`UPDATE homelab_scheduler_jobs
		    SET last_run_at     = $2,
		        last_run_status = $3,
		        last_run_error  = NULLIF($4, ''),
		        next_run_at     = COALESCE($5, next_run_at),
		        updated_at      = NOW()
		  WHERE id = $1`,
		job.ID, startedAt, out.Status, out.ErrorMessage,
		nullableTime(next),
	); uerr != nil {
		return fmt.Errorf("update job: %w", uerr)
	}

	return tx.Commit(ctx)
}

// decodeSchedulerHeadersBytes turns the JSONB []byte from
// pgx into a plain map[string]string. Empty blob → empty
// map. Malformed blob → empty map (defensive — the handler
// validates the headers on POST so a malformed blob would
// only happen if a future migration corrupts the column).
func decodeSchedulerHeadersBytes(raw []byte) map[string]string {
	out := map[string]string{}
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]string{}
	}
	return out
}

// truncateSchedulerErrMsg caps the error message at the
// homelab_scheduler_runs.error_message column's soft cap.
func truncateSchedulerErrMsg(s string) string {
	if len(s) <= schedulerErrorMessageMaxLen {
		return s
	}
	return s[:schedulerErrorMessageMaxLen]
}

// nullableTime returns nil for the zero time so the
// COALESCE in the UPDATE keeps the existing next_run_at
// when the cron parser failed.
func nullableTime(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
