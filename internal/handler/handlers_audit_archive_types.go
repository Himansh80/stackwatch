// Tier 9 Phase 4 — Audit Log Retention + Export (Tier 9.4).
// Shared types + format allowlist + in-flight job tracker for the
// four protected audit endpoints:
//
//	GET    /api/v1/enterprise/audit/archives                   — ListAuditArchives
//	POST   /api/v1/enterprise/audit/archive                    — CreateAuditArchive
//	GET    /api/v1/enterprise/audit/archives/:id/download      — DownloadAuditArchive
//	POST   /api/v1/enterprise/audit/export                     — ExportAuditEvents
//
// The HTTP handlers themselves live in handlers_audit_archive.go.
// This file only carries the JSON row shapes, the request bodies
// for POST endpoints, the format-allowlist constant, and the
// goroutine-pool tracker the background-compression job uses.
//
// Goroutine pool design (`auditArchiveJobs` + `auditArchiveSem`):
//   The spec requires POST /audit/archive to return immediately and
//   dispatch a background goroutine that SELECTs from audit_log,
//   gzips the JSON-encoded result, and UPDATEs the row. A naive
//   implementation spawns one goroutine per call which can OOM
//   under burst. We cap concurrent jobs at auditArchiveMaxParallel
//   (4) via a buffered channel semaphore; further POSTs queue in
//   `auditArchiveJobs` (sync.Map keyed by the row id) so we can
//   later surface queue depth in a future dashboard.
package handler

import (
	"sync"
	"time"
)

// ------------------------------------------------------------------
// Allowed formats for the CSV/JSON export endpoint.
//
// The download endpoint (Content-Disposition on /archives/:id/download)
// is always gzip so we do not need an allowlist there. For the live
// query+export endpoint the caller picks 'csv' or 'json'; anything
// else is rejected with 400 so a hand-crafted payload can't smuggle
// in arbitrary filenames or content-types (e.g. 'application/xhtml').
// ------------------------------------------------------------------

// allowedAuditExportFormats is the canonical set of format strings
// accepted by POST /api/v1/enterprise/audit/export. The handler
// validates req.Format against this set before dispatching.
//
// Current members:
//   "csv"  — RFC 4180 CSV with header row
//   "json" — JSON-lines (newline-delimited JSON objects)
//
// Add new members here AND extend ExportAuditEvents' `switch f` so
// downstream paths stay explicit. Order is UI-stable.
var allowedAuditExportFormats = map[string]bool{
	"csv":  true,
	"json": true,
}

// ------------------------------------------------------------------
// Background job caps.
//
// auditArchiveMaxParallel caps the number of POST /audit/archive
// compressions running concurrently. The buffered-channel semaphore
// (size = 4) blocks the next goroutine until a slot frees up; if
// calls outpace it, requests will queue at the route handler's
// INSERT step — the response then takes O(queued_jobs) seconds to
// return. That's intentional: refusing the call would surprise the
// operator who just clicked "Archive".
//
// auditArchiveJobTimeout is the per-job upper bound before we
// mark the row 'failed'. A 5-minute timeout caps a single
// million-row archive export. Set as a var (not const) so future
// config plumbing can shrink it under test.
// ------------------------------------------------------------------

const (
	auditArchiveMaxParallel = 4
)

// auditArchiveJobTimeout is the lifetime of a single
// background-compression goroutine. Larger values risk holding an
// open SQL connection for too long; smaller values risk killing
// genuinely-large archives mid-write. 5 minutes is the initial
// sweetspot for a 1M-row archive.
var auditArchiveJobTimeout = 5 * time.Minute

// ------------------------------------------------------------------
// JSON row types — mirror the column shape and map directly to the
// `out` slice returned by each List* handler.
// ------------------------------------------------------------------

// auditArchiveRow is the JSON shape for a single archive returned
// by GET /audit/archives + POST /audit/archive + GET /archive
// /:id/download. The compressed payload itself is NEVER returned
// in a JSON body — only the metadata + a `download_url` that the
// UI uses to fetch the binary via a separate request.
type auditArchiveRow struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	BatchID     string `json:"batch_id"`
	EventCount  int    `json:"event_count"`
	PeriodStart string `json:"period_start"`
	PeriodEnd   string `json:"period_end"`
	SizeBytes   int64  `json:"size_bytes"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// auditExportEvent is the JSON row shape for a single audit_log
// record streamed by POST /audit/export. Mirrors the columns of
// the existing `audit_log` table (migrations/000_init_schema.sql
// line 56): bigserial id, tenant_id, user_id, action, target,
// metadata (jsonb), ip (inet), created_at. The handler chooses the
// concrete fields exported for 'csv' vs 'json'.
type auditExportEvent struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Action    string `json:"action"`
	Target    string `json:"target,omitempty"`
	Metadata  string `json:"metadata,omitempty"`
	IP        string `json:"ip,omitempty"`
	CreatedAt string `json:"created_at"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST endpoints.
// ------------------------------------------------------------------

// auditArchiveReq is the JSON body for POST /audit/archive. The
// caller supplies the period; the handler enforces end > start
// (400 if not), then INSERTs a 'running' row, then spawns the
// background goroutine that fills in compressed_payload +
// event_count + size_bytes + status='completed'.
//
// period_start / period_end are accepted as RFC 3339 strings (the
// default Go json layout for time.Time).
type auditArchiveReq struct {
	PeriodStart time.Time `json:"period_start"     binding:"required"`
	PeriodEnd   time.Time `json:"period_end"       binding:"required"`
}

// auditExportReq is the JSON body for POST /audit/export. The
// caller picks a time window + optional event_type filter +
// format ('csv' | 'json') + limit (caps row count to avoid
// streaming the whole audit_log into memory). The handler uses
// io.Pipe to stream rows directly from the DB to the response
// writer without loading the full resultset in memory.
type auditExportReq struct {
	EventType string    `json:"event_type"     binding:"omitempty,max=64"`
	StartTime time.Time `json:"start_time"     binding:"required"`
	EndTime   time.Time `json:"end_time"       binding:"required"`
	Format    string    `json:"format"         binding:"required,oneof=csv json"`
	Limit     int       `json:"limit"          binding:"omitempty,min=1,max=1000000"`
}

// ------------------------------------------------------------------
// Background job tracker.
//
// In-flight archives live in `auditArchiveJobs` (sync.Map) so a
// future dashboard endpoint (planned for Tier 9.x post-Phase 4)
// could surface queue depth + per-job ETA. We don't currently
// expose it via HTTP; this is purely an in-process bookkeeping
// map so a duplicate POST for the same batch_id is safely
// idempotent (the handler checks sync.Map presence before spawning
// a goroutine).
//
// Each entry's value is a struct holding the row id and the
// insert timestamp so we can compute age; future phases could
// add a status-pending channel for awaiting completion.
// ------------------------------------------------------------------

// auditArchiveJob represents one in-flight compression. The map
// is keyed by row id (uuid string) — the value carries the
// owning tenant + the moment the goroutine was dispatched.
type auditArchiveJob struct {
	RowID       string
	TenantID    string
	BatchID     string
	DispatchedAt time.Time
}

// auditArchiveJobs is the package-level registry of in-flight jobs.
// Lives at package scope (not a struct field on Pool) because the
// 4 protected handlers are instantiated in `mountEnterpriseRoutes`
// outside any longer-lived object — package-scope is the
// standard place for this pattern across the handler package
// (mirrors effectivePermCache in handlers_rbac_types.go).
var auditArchiveJobs sync.Map // key=string(row id), value=*auditArchiveJob

// auditArchiveSem is a buffered-channel semaphore that caps
// concurrent compression goroutines at auditArchiveMaxParallel.
// The POST handler does `auditArchiveSem <- struct{}{}` BEFORE
// `go runArchiveJob(...)`; the goroutine does `<-auditArchiveSem`
// on exit so the slot is released even on panic.
var auditArchiveSem = make(chan struct{}, auditArchiveMaxParallel)
