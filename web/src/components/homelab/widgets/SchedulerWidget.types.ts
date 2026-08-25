/**
 * Shared types for SchedulerWidget (Tier 10 Phase 9 / H10
 * — Task Scheduler). Split out so SchedulerWidget.tsx
 * stays under the 400-LOC cap.
 *
 * Mirrors the JSON shape returned by
 * /api/v1/homelab/scheduler/jobs + /scheduler/runs.
 */

export type SchedulerActionKind = "http_get" | "http_post";

export interface SchedulerJob {
  id: string;
  tenant_id: string;
  user_id: string;
  name: string;
  action_kind: SchedulerActionKind;
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: string;
  schedule: string;
  enabled: boolean;
  last_run_at?: string;
  next_run_at?: string;
  last_run_status?: string;
  last_run_error?: string;
  created_at: string;
  updated_at: string;
  cap_remaining: number;
}

export interface SchedulerJobsResponse {
  jobs: SchedulerJob[];
  count: number;
  cap_per_user: number;
  cap_remaining: number;
}

export interface SchedulerRun {
  id: string;
  job_id: string;
  started_at: string;
  finished_at?: string;
  status: string;
  http_status_code?: number;
  response_bytes?: number;
  error_message?: string;
  duration_ms?: number;
}

export interface SchedulerRunsResponse {
  runs: SchedulerRun[];
  count: number;
  limit: number;
}

export interface SchedulerFormState {
  name: string;
  action_kind: SchedulerActionKind;
  url: string;
  method: string; // 'GET' or 'POST'
  /** Headers as a list of {name,value} rows so the UI can
   *  edit/delete each one independently. Stored as a map
   *  on submit. */
  headerRows: Array<{ name: string; value: string }>;
  body: string;
  contentType: string;
  schedule: string;
}

export const SCHEDULER_ACTION_KINDS: SchedulerActionKind[] = [
  "http_get",
  "http_post",
];

export const SCHEDULER_CONTENT_TYPES = [
  "text/plain",
  "application/json",
  "application/x-www-form-urlencoded",
];

export const SCHEDULER_HEADER_BLACKLIST = [
  "Host",
  "Cookie",
  "Authorization",
  "Set-Cookie",
  "Proxy-Authenticate",
  "Proxy-Authorization",
  "Transfer-Encoding",
  "Content-Length",
];

export const SCHEDULER_MAX_HEADERS = 10;
export const SCHEDULER_MAX_BODY_BYTES = 4096;

// Casts the loose JSON shape from the API to our typed
// SchedulerJob. Defensive defaults for any missing field
// keep a malformed row from blowing up the widget.
export function asSchedulerJob(raw: unknown): SchedulerJob {
  const r = raw as Partial<SchedulerJob>;
  return {
    id: r.id ?? "",
    tenant_id: r.tenant_id ?? "",
    user_id: r.user_id ?? "",
    name: r.name ?? "(unnamed)",
    action_kind: (r.action_kind as SchedulerActionKind) ?? "http_get",
    url: r.url ?? "",
    method: r.method ?? "GET",
    headers: (r.headers as Record<string, string>) ?? {},
    body: r.body ?? "",
    schedule: r.schedule ?? "",
    enabled: r.enabled ?? true,
    last_run_at: r.last_run_at,
    next_run_at: r.next_run_at,
    last_run_status: r.last_run_status,
    last_run_error: r.last_run_error,
    created_at: r.created_at ?? "",
    updated_at: r.updated_at ?? "",
    cap_remaining: r.cap_remaining ?? 0,
  };
}

export function asSchedulerJobs(raw: unknown): SchedulerJobsResponse {
  const r = raw as Partial<SchedulerJobsResponse>;
  return {
    jobs: Array.isArray(r.jobs) ? r.jobs.map(asSchedulerJob) : [],
    count: r.count ?? 0,
    cap_per_user: r.cap_per_user ?? 25,
    cap_remaining: r.cap_remaining ?? 0,
  };
}

export function asSchedulerRun(raw: unknown): SchedulerRun {
  const r = raw as Partial<SchedulerRun>;
  return {
    id: r.id ?? "",
    job_id: r.job_id ?? "",
    started_at: r.started_at ?? "",
    finished_at: r.finished_at,
    status: r.status ?? "unknown",
    http_status_code: r.http_status_code,
    response_bytes: r.response_bytes,
    error_message: r.error_message,
    duration_ms: r.duration_ms,
  };
}

export function asSchedulerRuns(raw: unknown): SchedulerRunsResponse {
  const r = raw as Partial<SchedulerRunsResponse>;
  return {
    runs: Array.isArray(r.runs) ? r.runs.map(asSchedulerRun) : [],
    count: r.count ?? 0,
    limit: r.limit ?? 50,
  };
}

/** Returns a CSS color hint for the status pill. */
export function schedulerStatusColor(status?: string): string {
  switch (status) {
    case "success":
      return "var(--ok, #10b981)";
    case "failure":
      return "var(--error, #ef4444)";
    case "pending":
      return "var(--warn, #f59e0b)";
    case "skipped":
      return "var(--muted, #6b7280)";
    default:
      return "var(--muted, #6b7280)";
  }
}

/** Tiny relative-time formatter. Same convention the other
 *  widgets use (RSS, calendar). */
export function schedulerRelativeTime(iso?: string): string {
  if (!iso) return "—";
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return iso;
  const sec = Math.floor((Date.now() - t) / 1000);
  if (sec < 0) return "in the future";
  if (sec < 60) return `${sec}s ago`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
  return `${Math.floor(sec / 86400)}d ago`;
}
