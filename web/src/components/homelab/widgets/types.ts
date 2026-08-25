/**
 * Shared types for ServiceStatusWidget — split out so
 * ServiceStatusWidget.tsx stays under the 400-LOC cap.
 *
 * Importing from `'./types'` keeps the import graph shallow and
 * avoids the (currently unused but future-friendly) `react` import
 * here — every consumer pulls these shapes via type-only imports.
 */

export type ServiceStatus = "up" | "degraded" | "down" | "unknown";

export type ServiceKind = "http" | "https" | "tcp" | "icmp";

export interface ServiceHealth {
  status: ServiceStatus;
  latency_ms?: number | null;
  status_code?: number | null;
  error_message?: string | null;
  checked_at: string;
}

export interface PinnedService {
  id: string;
  name: string;
  url: string;
  kind: ServiceKind;
  icon?: string | null;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  health?: ServiceHealth | null;
}

export interface ServicesResponse {
  services: PinnedService[];
  count: number;
}

export interface AggregateResponse {
  up: number;
  degraded: number;
  down: number;
  unknown: number;
  total: number;
}

export interface PinFormState {
  name: string;
  url: string;
  kind: ServiceKind;
  icon: string;
}

// Re-exported under a `Status` alias for places that read the
// shape without wanting to type out "ServiceStatus" everywhere.
export type Status = ServiceStatus;

// ------------------------------------------------------------------
// Phase 3 (H3 — Personal Notes + Todos) types.
// ------------------------------------------------------------------

export type TodoPriority = "low" | "medium" | "high" | "urgent";

export interface NoteRow {
  id: string;
  title: string;
  body: string;
  tags: string[];
  pinned: boolean;
  created_at: string;
  updated_at: string;
}

export interface NotesResponse {
  notes: NoteRow[];
  count: number;
}

export interface NoteFormState {
  title: string;
  body: string;
  tagsRaw: string; // comma-separated, split on submit
  pinned: boolean;
}

export interface TodoRow {
  id: string;
  title: string;
  description?: string | null;
  priority: TodoPriority;
  due_date?: string | null;
  completed_at?: string | null;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface TodosResponse {
  todos: TodoRow[];
  count: number;
}

export interface DueSoonResponse {
  todos: TodoRow[];
  count: number;
  overdue_count: number;
  window_days: number;
}

export interface TodoFormState {
  title: string;
  description: string;
  priority: TodoPriority;
  dueDate: string; // datetime-local input value
  tagsRaw: string;
}

// ------------------------------------------------------------------
// Phase 4 (H4 — Calendar) types.
// ------------------------------------------------------------------

export interface CalendarRow {
  id: string;
  name: string;
  ical_url: string;
  color: string;
  enabled: boolean;
  last_synced_at?: string;
  last_sync_status?: string;
  last_sync_error?: string;
  event_count: number;
  created_at: string;
  updated_at: string;
}

export interface CalendarsResponse {
  calendars: CalendarRow[];
  count: number;
}

export interface EventRow {
  id: string;
  calendar_id: string;
  uid: string;
  summary: string;
  description?: string;
  location?: string;
  starts_at: string;
  ends_at?: string;
  all_day: boolean;
  created_at: string;
  updated_at: string;
}

export interface EventsResponse {
  events: EventRow[];
  count: number;
  from: string;
  to: string;
  limit: number;
}

export interface CalendarFormState {
  name: string;
  ical_url: string;
  color: string; // hex or 'default'
}

// ------------------------------------------------------------------
// Phase 5 (H5 — Download Stats) types.
// ------------------------------------------------------------------

export type DownloadKind =
  "sonarr" | "radarr" | "qbittorrent" | "sabnzbd" | "lidarr" | "readarr";

export interface DownloadLatestSnapshot {
  client_id: string;
  queue_count: number;
  queue_size_bytes: number;
  download_speed_bytes_per_sec: number;
  upload_speed_bytes_per_sec: number;
  today_downloaded_bytes: number;
  today_uploaded_bytes: number;
  polled_at: string;
}

export interface DownloadClientRow {
  id: string;
  name: string;
  kind: DownloadKind;
  base_url: string;
  enabled: boolean;
  last_polled_at?: string;
  last_poll_status?: string;
  last_poll_error?: string;
  latest_snapshot?: DownloadLatestSnapshot | null;
  created_at: string;
  updated_at: string;
}

export interface DownloadsResponse {
  clients: DownloadClientRow[];
  count: number;
}

export interface DownloadSnapshotRow {
  id: string;
  client_id: string;
  queue_count: number;
  queue_size_bytes: number;
  download_speed_bytes_per_sec: number;
  upload_speed_bytes_per_sec: number;
  today_downloaded_bytes: number;
  today_uploaded_bytes: number;
  polled_at: string;
}

export interface DownloadSnapshotsResponse {
  snapshots: DownloadSnapshotRow[];
  count: number;
  limit: number;
}

export interface DownloadClientFormState {
  name: string;
  kind: DownloadKind;
  base_url: string;
  api_key: string;
  username: string;
  password: string;
}
