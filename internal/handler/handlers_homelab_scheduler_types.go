// Tier 10 Phase 9 — Task Scheduler (H10). SECURITY-CRITICAL.
//
// Shared types + the per-handler allowlist/validation primitives
// for the user-schedulable HTTP-probe cron. Lives in its own
// file so the 4 routes in handlers_homelab_scheduler_jobs.go
// can stay under the 400-LOC cap.
//
// Storage: homelab_scheduler_jobs + homelab_scheduler_runs
// (migrations/041_homelab.sql — Phase 9 extension).
//
// Per-user design: every query is filtered by both tenant_id
// and user_id. A user in tenant A cannot see (or modify) or
// run another user's jobs even if they share a tenant —
// matches US-8 in the speckit proposal.
//
// SECURITY MODEL — every entry point enforces these guardrails:
//   1. Action allowlist (allowedActionKinds): only http_get +
//      http_post are valid. NO shell-out, NO exec.Command.
//   2. URL allowlist (validateSchedulerURL): http/https ONLY;
//      host must NOT resolve to a private/loopback/link-local
//      CIDR (validateSchedulerHostNotPrivate) — defense against
//      SSRF into the homelab. Re-checked at runtime because
//      DNS could change between POST and the worker tick.
//   3. Header allowlist (allowedSchedulerHeaderMax,
//      headerNameRe, validateSchedulerHeaderName +
//      blockedHeaderNames): name regex, max 10, no Host/Cookie/
//      Authorization/Set-Cookie/etc. The user must NOT be able
//      to forge credentials they don't already have.
//   4. Body limit (schedulerBodyMaxBytes = 4KB) — caps memory
//      per probe, and content-type allowlist
//      (allowedSchedulerBodyContentTypes).
//   5. Schedule (validateSchedulerSchedule): must parse as a
//      5-field cron expression. We don't shell out to validate
//      against a tool — parseCron + ValidateCron return errors
//      for malformed input.
//   6. Per-user job cap (schedulerJobCapPerUser = 25): enforced
//      in the handler before INSERT.
//   7. URL-redaction in audit messages (RedactSchedulerURL):
//      credentials embedded in query strings (?api_key=…) are
//      stripped before logging.
package handler

// ------------------------------------------------------------------
// Job-row shape — what /homelab/scheduler/jobs returns + accepts.
// ------------------------------------------------------------------

// schedulerJobRow is the JSON shape returned by GET/POST/PATCH
// /scheduler/jobs. Mirrors the columns of homelab_scheduler_jobs
// plus a computed cap_remaining (against schedulerJobCapPerUser)
// so the frontend can show "23 / 25 jobs used" without a
// follow-up call. last_run_at + next_run_at + last_run_status
// are RFC3339-ish strings so the frontend can render relative
// timestamps directly.
type schedulerJobRow struct {
	ID            string            `json:"id"`
	TenantID      string            `json:"tenant_id"`
	UserID        string            `json:"user_id"`
	Name          string            `json:"name"`
	ActionKind    string            `json:"action_kind"`
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	Body          string            `json:"body,omitempty"`
	Schedule      string            `json:"schedule"`
	Enabled       bool              `json:"enabled"`
	LastRunAt     string            `json:"last_run_at,omitempty"`
	NextRunAt     string            `json:"next_run_at,omitempty"`
	LastRunStatus string            `json:"last_run_status,omitempty"`
	LastRunError  string            `json:"last_run_error,omitempty"`
	CreatedAt     string            `json:"created_at"`
	UpdatedAt     string            `json:"updated_at"`
	CapRemaining  int               `json:"cap_remaining"`
}

// schedulerJobReq is the body for POST + PATCH /scheduler/jobs.
// All pointer fields so PATCH semantics work (a missing key
// means "don't update").
//
// action_kind: must be 'http_get' or 'http_post' (validateSchedulerActionKind).
// url: must be http/https + non-private host (validateSchedulerURL).
// method: optional; defaults to 'GET' for http_get and 'POST'
// for http_post (normalizeSchedulerMethod).
// headers: optional; max 10 entries, name regex enforced, no
// blocked header names (validateSchedulerHeaders).
// body: optional; max 4KB; content-type allowlist
// (validateSchedulerBody). Empty for http_get.
// schedule: must be a valid 5-field cron expression
// (validateSchedulerSchedule).
// enabled: optional; defaults to true on POST.
type schedulerJobReq struct {
	Name       *string            `json:"name"`
	ActionKind *string            `json:"action_kind"`
	URL        *string            `json:"url"`
	Method     *string            `json:"method"`
	Headers    *map[string]string `json:"headers"`
	Body       *string            `json:"body"`
	Schedule   *string            `json:"schedule"`
	Enabled    *bool              `json:"enabled"`
}

// schedulerJobRowScan is the raw shape pgx returns for a SELECT
// on homelab_scheduler_jobs. Carries nullable timestamps as
// pointers; newSchedulerJobRowFromScan copies them into the
// final shape.
type schedulerJobRowScan struct {
	ID            string
	TenantID      string
	UserID        string
	Name          string
	ActionKind    string
	URL           string
	Method        string
	HeadersRaw    []byte
	Body          *string
	Schedule      string
	Enabled       bool
	LastRunAt     *string
	NextRunAt     *string
	LastRunStatus *string
	LastRunError  *string
	CreatedAt     string
	UpdatedAt     string
}

func newSchedulerJobRowFromScan(s schedulerJobRowScan) schedulerJobRow {
	row := schedulerJobRow{
		ID:        s.ID,
		TenantID:  s.TenantID,
		UserID:    s.UserID,
		Name:      s.Name,
		ActionKind: s.ActionKind,
		URL:       s.URL,
		Method:    s.Method,
		Schedule:  s.Schedule,
		Enabled:   s.Enabled,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
	if s.Body != nil {
		row.Body = *s.Body
	}
	if s.LastRunAt != nil {
		row.LastRunAt = *s.LastRunAt
	}
	if s.NextRunAt != nil {
		row.NextRunAt = *s.NextRunAt
	}
	if s.LastRunStatus != nil {
		row.LastRunStatus = *s.LastRunStatus
	}
	if s.LastRunError != nil {
		row.LastRunError = *s.LastRunError
	}
	// Headers is JSONB → []byte. Decoded lazily so we don't
	// blow up on an empty {} blob. DecodeHeadersForScheduler
	// lives in scheduler_jobs.go and returns map[string]string.
	row.Headers = map[string]string{}
	if len(s.HeadersRaw) > 0 {
		decoded, derr := decodeSchedulerHeaders(s.HeadersRaw)
		if derr == nil {
			row.Headers = decoded
		}
	}
	return row
}

// ------------------------------------------------------------------
// Run-row shape — what /homelab/scheduler/runs returns.
// ------------------------------------------------------------------

// schedulerRunRow is the JSON shape returned by GET
// /scheduler/runs. error_message is shown as a tooltip on the
// run row in the widget's history list. duration_ms + http_status_code
// are surfaced as a small "200 in 312ms" pill.
type schedulerRunRow struct {
	ID             string `json:"id"`
	JobID          string `json:"job_id"`
	StartedAt      string `json:"started_at"`
	FinishedAt     string `json:"finished_at,omitempty"`
	Status         string `json:"status"`
	HTTPStatusCode *int   `json:"http_status_code,omitempty"`
	ResponseBytes  *int   `json:"response_bytes,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	DurationMS     *int   `json:"duration_ms,omitempty"`
}

