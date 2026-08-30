// Tier 9 Phase 5 — Compliance Reports (Tier 9.5).
//
// Shared types + framework/frequency allowlists + in-flight job tracker
// for the seven protected compliance endpoints:
//
//	GET    /api/v1/enterprise/compliance/reports                  — ListComplianceReports
//	POST   /api/v1/enterprise/compliance/reports                  — CreateComplianceReport
//	GET    /api/v1/enterprise/compliance/reports/:id               — GetComplianceReport
//	GET    /api/v1/enterprise/compliance/reports/:id/download     — DownloadComplianceReport
//	GET    /api/v1/enterprise/compliance/schedules                — ListComplianceSchedules
//	POST   /api/v1/enterprise/compliance/schedules                — CreateComplianceSchedule
//	DELETE /api/v1/enterprise/compliance/schedules/:id            — DeleteComplianceSchedule
//
// The HTTP handlers themselves live in handlers_compliance.go (the four
// report endpoints + the background generator goroutine) and
// handlers_compliance_schedules.go (the three schedule endpoints).
// This file only carries the JSON row shapes, the request bodies for
// POST endpoints, the framework/frequency allowlists, and the
// goroutine-pool tracker the background-report job uses.
//
// Framework allowlist:
//
//	Phase 5 ships five well-known frameworks: 'soc2' | 'iso27001' |
//	'hipaa' | 'pci' | 'gdpr'. The same set is enforced at the DB
//	layer via a CHECK constraint (see migrations/040_enterprise.sql)
//	and at the handler layer via allowedComplianceFrameworks —
//	keeping the two in sync is intentional: a hand-crafted INSERT
//	can't smuggle in 'sox' / 'fedramp' even if a future regression
//	adds a missing constant to the handler map.
//
// Goroutine pool design (`complianceReportJobs` + `complianceReportSem`):
//
//	Mirrors the audit-archive pattern from Phase 4. POST
//	/compliance/reports returns 201 IMMEDIATELY with the row id after
//	INSERTing a status='pending' row. A background goroutine then
//	flips the row to 'running', assembles a synthetic evidence
//	document from real DB counts (audit log / anomaly events /
//	active users / etc.), writes it to the filesystem at
//	/opt/stackwatch/reports/{tenant_id}/{report_id}.txt, and
//	finally updates the row to status='completed' (with artifact_path)
//	or 'failed' (with error_message). Concurrent generators are
//	capped at complianceReportMaxParallel (4) via a buffered-channel
//	semaphore; further POSTs queue without spawning unbounded
//	goroutines.
package handler

import (
	"sync"
	"time"
)

// ------------------------------------------------------------------
// Framework + frequency allowlists.
//
// allowedComplianceFrameworks is the canonical set accepted by
// POST /api/v1/enterprise/compliance/reports and POST
// /api/v1/enterprise/compliance/schedules. The CHECK constraint on
// the DB column mirrors this set so a hand-crafted INSERT can't
// smuggle in junk like 'sox' or 'fedramp'.
//
//	"soc2"      — SOC 2 (Service Organization Control 2)
//	"iso27001"  — ISO/IEC 27001 (information security management)
//	"hipaa"     — HIPAA (US healthcare data privacy)
//	"pci"       — PCI-DSS (payment-card industry data security)
//	"gdpr"      — GDPR (EU general data protection regulation)
//
// Add new frameworks here AND extend the DB CHECK in
// migrations/040_enterprise.sql so the two layers stay in sync.
// ------------------------------------------------------------------
var allowedComplianceFrameworks = map[string]bool{
	"soc2":     true,
	"iso27001": true,
	"hipaa":    true,
	"pci":      true,
	"gdpr":     true,
}

// allowedComplianceFrequencies is the canonical set accepted by
// POST /api/v1/enterprise/compliance/schedules. Same pattern as
// the framework allowlist — handler validates against this map
// AND the DB CHECK constraint mirrors it.
//
//	"monthly"   — every 30 days
//	"quarterly" — every 90 days
//	"yearly"    — every 365 days
var allowedComplianceFrequencies = map[string]bool{
	"monthly":   true,
	"quarterly": true,
	"yearly":    true,
}

// ------------------------------------------------------------------
// Background job caps.
//
// complianceReportMaxParallel caps concurrent generator goroutines
// at 4 via a buffered-channel semaphore — a burst of 10 parallel
// POSTs only spawns 4 concurrent workers + 6 queued submits. See
// handlers_compliance.go::CreateComplianceReport for the send /
// run / receive dance.
//
// complianceReportJobTimeout is the per-job upper bound before we
// mark the row 'failed'. A 2-minute window caps a single report
// assembly — in practice the generator is fast (a few COUNT queries
// + a small text-file write) but we don't want a stuck fsync to
// pin a goroutine forever.
const (
	complianceReportMaxParallel = 4
)

var complianceReportJobTimeout = 2 * time.Minute

// complianceReportsBaseDir is the directory the background generator
// writes artifacts into. Each tenant gets a sub-directory at
// /opt/stackwatch/reports/{tenant_id}/{report_id}.txt. The path is
// stored verbatim in compliance_reports.artifact_path.
const complianceReportsBaseDir = "/opt/stackwatch/reports"

// ------------------------------------------------------------------
// JSON row types — mirror the column shape of compliance_reports /
// compliance_schedules.
// ------------------------------------------------------------------

// complianceReportRow is the JSON shape for a single compliance
// report returned by GET /compliance/reports, POST
// /compliance/reports, and GET /compliance/reports/:id.
//
// Fields map 1:1 to the compliance_reports table. The artifact
// blob itself is NEVER returned in a JSON body — only the
// artifact_path so the UI can call /download separately.
type complianceReportRow struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	Framework    string `json:"framework"`
	PeriodStart  string `json:"period_start"`
	PeriodEnd    string `json:"period_end"`
	Status       string `json:"status"`
	ArtifactPath string `json:"artifact_path,omitempty"`
	ScheduledID  string `json:"scheduled_id,omitempty"`
	RequestedBy  string `json:"requested_by,omitempty"`
	CreatedAt    string `json:"created_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// complianceScheduleRow is the JSON shape for a single recurring
// report schedule returned by GET /compliance/schedules and POST
// /compliance/schedules.
type complianceScheduleRow struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Framework  string    `json:"framework"`
	Frequency  string    `json:"frequency"`
	Recipients []string  `json:"recipients"`
	Enabled    bool      `json:"enabled"`
	NextRunAt  time.Time `json:"next_run_at"`
	CreatedAt  string    `json:"created_at"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST endpoints.
// ------------------------------------------------------------------

// complianceReportReq is the JSON body for POST
// /compliance/reports. The handler validates the framework against
// allowedComplianceFrameworks and end > start before INSERT.
type complianceReportReq struct {
	Framework   string    `json:"framework"     binding:"required"`
	PeriodStart time.Time `json:"period_start"  binding:"required"`
	PeriodEnd   time.Time `json:"period_end"    binding:"required"`
}

// complianceScheduleReq is the JSON body for POST
// /compliance/schedules. Recipients is the email list to mail the
// report to when the future cron sweep fires.
type complianceScheduleReq struct {
	Framework  string    `json:"framework"   binding:"required"`
	Frequency  string    `json:"frequency"   binding:"required"`
	Recipients []string  `json:"recipients"`
	Enabled    bool      `json:"enabled"`
	NextRunAt  time.Time `json:"next_run_at" binding:"required"`
}

// ------------------------------------------------------------------
// Background job tracker — mirrors handlers_audit_archive_types.go.
//
// In-flight reports live in `complianceReportJobs` (sync.Map) so a
// future dashboard endpoint (planned for Tier 9.x post-Phase 5)
// could surface queue depth + per-job ETA. We don't currently
// expose it via HTTP; this is purely in-process bookkeeping.
// ------------------------------------------------------------------

// complianceReportJob represents one in-flight report generator.
type complianceReportJob struct {
	RowID        string
	TenantID     string
	Framework    string
	DispatchedAt time.Time
}

// complianceReportJobs is the package-level registry of in-flight
// report jobs. Keyed by row id (uuid string) so a future operator
// can look up by id.
var complianceReportJobs sync.Map // key=string(row id), value=*complianceReportJob

// complianceReportSem is a buffered-channel semaphore that caps
// concurrent report generators at complianceReportMaxParallel.
var complianceReportSem = make(chan struct{}, complianceReportMaxParallel)
