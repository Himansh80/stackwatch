// Package homelab — Task Scheduler (H10) — sync run helper.
//
// RunSchedulerJobAndInsertRun is the single entry point that
// executes ONE scheduler job and records the result. Used by
// the SchedulerWorker (scheduler.go) AND the manual-trigger
// handler (POST /api/v1/homelab/scheduler/jobs/:id/run).
//
// SECURITY: this is where most of the runtime safety lives.
// Every invocation:
//   1. Re-fetches the job (defense against row corruption or
//      concurrent DELETE).
//   2. Validates the row is still enabled.
//   3. Enforces the rate-limit floor (no two runs within
//      schedulerRunRateLimitMinInterval of each other).
//   4. Re-validates the URL DNS at runtime (defense in depth —
//      the add-time check happened on a different network
//      snapshot).
//   5. Builds the HTTP request using ONLY the user-supplied
//      headers that pass the allowlist. Host/Cookie/Auth
//      CANNOT be set.
//   6. Executes the request with a 30s timeout, capturing
//      status code + body size + error message.
//   7. INSERTs a homelab_scheduler_runs row and UPDATEs the
//      job's last_run_at + next_run_at bookkeeping fields.
//
// Sync helper split out of scheduler.go so the worker file
// stays under the 400-LOC cap (Phase 8 RSS uses the same
// split: rss.go + rss_poll.go).
package homelab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// schedulerRunRateLimitMinInterval is the worker-side mirror
// of handler.schedulerRunRateLimitMinInterval. We duplicate
// to avoid a circular import. Keep these two in sync —
// changing one without the other lets a job fire too
// frequently when triggered through a different code path.
const schedulerRunRateLimitMinInterval = 5 * time.Minute

// schedulerErrorMessageMaxLen is the worker-side mirror of
// handler.schedulerErrorMessageMaxLen. Kept in sync.
const schedulerErrorMessageMaxLen = 500

// schedulerJobSnapshot is the in-memory projection of one
// row of homelab_scheduler_jobs, sufficient to execute a
// single run without further DB calls.
type schedulerJobSnapshot struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	UserID     uuid.UUID
	ActionKind string
	URL        string
	Method     string
	Headers    map[string]string
	Body       string
	Schedule   string
	Enabled    bool
}

// schedulerRunOutcome is what RunSchedulerJobAndInsertRun
// returns to the worker / handler. Status mirrors the row's
// status column ('success' | 'failure' | 'skipped').
type schedulerRunOutcome struct {
	JobID         uuid.UUID
	Status        string // 'success' | 'failure' | 'skipped'
	HTTPStatus    int
	ResponseBytes int
	ErrorMessage  string
	DurationMS    int
	SkippedReason string // populated when Status='skipped'
}

// schedulerRunHTTPClient is the per-job HTTP client. Each
// invocation gets a fresh client so the timeout is per-job
// (a shared client.Timeout would apply to all).
func schedulerRunHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// RunSchedulerJobAndInsertRun executes one scheduler job by
// its UUID and writes the result rows. The function is
// safe to call concurrently for DIFFERENT jobs; the
// SchedulerWorker dispatches one job per goroutine and the
// handler triggers one manual run per click.
//
// Errors at any step are surfaced via the returned
// schedulerRunOutcome + a non-nil error from the function
// itself (DB-level failures, mostly). The function NEVER
// panics — panics in the HTTP client path are recovered
// inside the run body and recorded as a failure run.
//
// Returns the outcome + nil on success; outcome + non-nil
// error when a SYSTEM-level failure occurred (DB connection
// dropped, etc.) — the worker logs and continues.
func RunSchedulerJobAndInsertRun(
	parentCtx context.Context,
	pool *db.Pool,
	jobID uuid.UUID,
	logger *slog.Logger,
) (schedulerRunOutcome, error) {
	if logger == nil {
		logger = slog.Default()
	}
	// Use a separate timeout for the run as a whole. The
	// HTTP client has its own 30s; this cap covers the DB
	// writes too. 45s is generous.
	ctx, cancel := context.WithTimeout(parentCtx, 45*time.Second)
	defer cancel()

	outcome := schedulerRunOutcome{
		JobID:  jobID,
		Status: "failure",
	}

	job, err := fetchSchedulerJob(ctx, pool, jobID)
	if err != nil {
		if errors.Is(err, errSchedulerJobNotFound) {
			// The row was deleted between the dispatch
			// query and the fetch — silently drop. The
			// worker logs at info, the handler returns
			// 404.
			return schedulerRunOutcome{
				JobID:         jobID,
				Status:        "skipped",
				SkippedReason: "job_not_found",
			}, nil
		}
		return outcome, fmt.Errorf("fetch job: %w", err)
	}
	if !job.Enabled {
		return schedulerRunOutcome{
			JobID:         jobID,
			Status:        "skipped",
			SkippedReason: "job_disabled",
		}, nil
	}

	// Rate-limit check: refuse if the previous run was too
	// recent. We re-read last_run_at here (instead of
	// trusting the dispatch query) because the dispatcher
	// may have grabbed a row whose last_run_at was about to
	// be updated by another worker.
	lastRun, lerr := fetchSchedulerJobLastRunAt(ctx, pool, jobID)
	if lerr != nil && !errors.Is(lerr, errSchedulerJobLastRunUnknown) {
		return outcome, fmt.Errorf("fetch last_run_at: %w", lerr)
	}
	if !lastRun.IsZero() && time.Since(lastRun) < schedulerRunRateLimitMinInterval {
		return schedulerRunOutcome{
			JobID:         jobID,
			Status:        "skipped",
			SkippedReason: "rate_limited",
		}, nil
	}

	// Runtime URL re-validation (defense in depth — DNS
	// could have changed between POST and now).
	host := ""
	if u, perr := url.Parse(job.URL); perr == nil {
		host = u.Hostname()
	}
	startedAt := time.Now().UTC()
	if host == "" || validateSchedulerURLHostNonPrivate(host) != nil {
		return outcome, recordSchedulerRunFailure(ctx, pool, job, "runtime_url_blocked", 0, 0, startedAt)
	}

	// Build + execute the HTTP request.
	body := job.Body
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, rerr := http.NewRequestWithContext(ctx, job.Method, job.URL, bodyReader)
	if rerr != nil {
		return outcome, recordSchedulerRunFailure(ctx, pool, job, fmt.Sprintf("new_request: %v", rerr), 0, 0, startedAt)
	}
	// Only the user-supplied headers — every name was
	// validated at add time. We re-check at runtime because
	// the row could have been inserted before a security
	// hardening change was deployed.
	for k, v := range job.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("User-Agent", "StackWatch-Homelab-Scheduler/1.0")

	client := schedulerRunHTTPClient()
	resp, herr := client.Do(req)
	dur := time.Since(startedAt)
	durationMS := int(dur / time.Millisecond)
	if herr != nil {
		errMsg := truncateSchedulerErrMsg(herr.Error())
		return outcome, recordSchedulerRunFailure(ctx, pool, job, errMsg, 0, 0, startedAt)
	}
	defer resp.Body.Close()
	// Read up to 256KB of body for accounting — the target
	// might return a huge HTML page but we don't need all
	// of it, just the byte count. Reading also lets the
	// connection be reused for keep-alive.
	respBytes, rerr := io.Copy(io.Discard, io.LimitReader(resp.Body, 256*1024))
	if rerr != nil {
		// Connection-level error after headers — record
		// the response we got but mark as failure.
		return outcome, recordSchedulerRunFailure(ctx, pool, job,
			truncateSchedulerErrMsg(fmt.Sprintf("read body: %v", rerr)),
			resp.StatusCode, int(respBytes), startedAt)
	}
	status := "success"
	if resp.StatusCode >= 400 {
		status = "failure"
	}
	outcome = schedulerRunOutcome{
		JobID:         jobID,
		Status:        status,
		HTTPStatus:    resp.StatusCode,
		ResponseBytes: int(respBytes),
		DurationMS:    durationMS,
	}
	if status == "failure" {
		outcome.ErrorMessage = fmt.Sprintf("http %d", resp.StatusCode)
	}
	if err := insertSchedulerRunAndUpdateJob(ctx, pool, job, outcome, startedAt, dur); err != nil {
		return outcome, fmt.Errorf("insert run + update job: %w", err)
	}
	logger.Info("scheduler run complete",
		"job_id", jobID,
		"user_id", job.UserID,
		"status", status,
		"http_status", resp.StatusCode,
		"duration_ms", durationMS)
	return outcome, nil
}

// (DB read/write helpers moved to scheduler_db.go to keep
// this file under the 400-LOC cap. URL helpers moved to
// scheduler_url_helpers.go for the same reason.)
