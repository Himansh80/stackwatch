// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// JSON row shapes + request/response bodies for the 5 backup
// endpoints:
//
//	POST /api/v1/platform/backup/create        — CreateBackup (auth-protected)
//	GET  /api/v1/platform/backup/list          — ListBackups  (auth-protected)
//	GET  /api/v1/platform/backup/:id/download  — DownloadBackup (auth-protected)
//	DELETE /api/v1/platform/backup/:id         — DeleteBackup (auth-protected)
//	POST /api/v1/platform/backup/restore       — RestoreBackup (auth-protected, multipart upload)
//
// The handlers themselves live in handlers_platform_backup.go +
// handlers_platform_backup_restore.go (kept under the 400-LOC
// cap by the split). The crypto layer (AES-256-GCM + HKDF + SHA-
// 256) lives in handlers_platform_backup_crypto.go. The
// underlying per-tenant pg_dump / encrypt helper is
// internal/platform/backup_helpers.go (CreateBackupAndEncrypt +
// advisory lock + audit log).
//
// Why these shapes:
//   - backupRow mirrors the platform_backups row directly minus
//     a few handler-irrelevant columns (sha256_plaintext
//     intentionally exposed so an operator can verify download
//     integrity client-side; sha256_ciphertext also exposed so
//     the UI can verify against the disk blob).
//
//   - backupCreateResp echoes {backup_id, status} — CreateBackup
//     returns 202 because encryption is fire-and-forget; the
//     UI then polls GET /backup/list until status flips to
//     'completed' or 'failed'.
//
//   - backupListResp batches the rows + a tenant-wide KPI
//     summary so the dashboard can render the header strip and
//     the row list from a single fetch.
//
//   - backupRestoreResp mirrors backupCreateResp — the multipart
//     upload returns immediately and runs in a background
//     goroutine for the actual pg_restore phase.
//
//   - allowedBackupKinds / allowedBackupStatuses /
//     allowedBackupFrequencies are server-side allowlists
//     so a future "bulk import" tool can't smuggle
//     'system' or other reserved values past the validator.

package handler

import "time"

// backupRow mirrors one row of platform_backups. JSON tags
// match the column names so a future endpoint can switch to
// straight json.Marshal over the row without re-mapping.
type backupRow struct {
	ID                 string     `json:"id"`
	TenantID           string     `json:"tenant_id"`
	CreatedByUserID    *string    `json:"created_by_user_id,omitempty"`
	BackupKind         string     `json:"backup_kind"`
	Status             string     `json:"status"`
	FilePath           *string    `json:"file_path,omitempty"`
	FileSizeBytes      int64      `json:"file_size_bytes"`
	UncompressedBytes  int64      `json:"uncompressed_bytes"`
	TableCount         int        `json:"table_count"`
	RowCount           int64      `json:"row_count"`
	EncryptionAlgo     string     `json:"encryption_algo"`
	SHA256Plaintext    *string    `json:"sha256_plaintext,omitempty"`
	SHA256Ciphertext   *string    `json:"sha256_ciphertext,omitempty"`
	StartedAt          time.Time  `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	ErrorMessage       *string    `json:"error_message,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// backupJobRow mirrors one row of platform_backup_jobs. Today
// only used by ListBackupJobs (helper used by the scheduler
// worker; not exposed on a route — the dashboard reads it via
// the backup list rollup).
type backupJobRow struct {
	ID             string     `json:"id"`
	TenantID       string     `json:"tenant_id"`
	Frequency      string     `json:"frequency"`
	RetentionDays  int        `json:"retention_days"`
	Enabled        bool       `json:"enabled"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	LastRunStatus  *string    `json:"last_run_status,omitempty"`
	LastBackupID   *string    `json:"last_backup_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// backupCreateReq is the optional body for POST /backup/create.
// Phase 5 ships a no-arg create; the struct exists so a
// future label / encryption-override field can be added
// without a breaking handler signature change. The handler
// checks ContentLength before parsing so an empty POST is
// still accepted.
type backupCreateReq struct {
	Label string `json:"label" binding:"omitempty,max=64"`
}

// backupCreateResp is the immediate 202 reply for
// POST /backup/create. The caller polls /backup/list until
// the row's status flips to a terminal state. Restated the
// status here so the UI doesn't need a second fetch to render
// "we're working on it".
type backupCreateResp struct {
	BackupID    string `json:"backup_id"`
	Status      string `json:"status"`        // pending | running
	AuditURL    string `json:"audit_url"`     // /api/v1/platform/backup/{id}
	NextStep    string `json:"next_step"`     // poll_list | wait
}

// backupDeleteResp is the immediate reply for DELETE /backup/:id.
// Mirrors the platform_backups row that was removed so a UI
// that cached the row can drop it without a refetch.
type backupDeleteResp struct {
	BackupID  string `json:"backup_id"`
	TenantID  string `json:"tenant_id"`
	Status    string `json:"status"`         // deleted
	Removed   bool   `json:"removed"`        // true if a row was actually removed
}

// backupListResp is the JSON for GET /backup/list. The Dashboard
// renders the summary strip from Summary + the row list from
// Backups in a single fetch.
type backupListResp struct {
	Backups []backupRow    `json:"backups"`
	Summary backupListSummary `json:"summary"`
}

// backupListSummary is the header KPI strip for the
// dashboard. Computed in SQL (COUNT + SUM + MAX) so the API
// stays a single round-trip.
type backupListSummary struct {
	TotalBackups       int     `json:"total_backups"`
	LastBackupAt       *time.Time `json:"last_backup_at,omitempty"`
	LastBackupStatus   *string `json:"last_backup_status,omitempty"`
	TotalStorageBytes  int64   `json:"total_storage_bytes"`
	TotalRows          int64   `json:"total_rows"`
}

// backupRestoreResp is the immediate reply for POST
// /backup/restore. Same fire-and-forget semantics as
// CreateBackup — the restore runs in a goroutine; the UI
// polls /backup/list to observe the (separately-created)
// audit row.
type backupRestoreResp struct {
	RestoreID  string `json:"restore_id"`
	BackupID   string `json:"backup_id"`
	Status     string `json:"status"`        // pending
	NextStep   string `json:"next_step"`     // poll_list
	AuditNote  string `json:"audit_note"`    // every attempt logged
}

// =====================================================================
// Server-side allowlists
// =====================================================================

// allowedBackupKinds restricts what the API will accept for
// backup_kind. 'manual' = triggered via POST /backup/create;
// 'scheduled' = produced by BackupSchedulerWorker. The reserved
// 'system' value is left for a future backup-of-platform-tables
// hook (PL5 does not produce or accept 'system' rows today).
var allowedBackupKinds = map[string]struct{}{
	"manual":   {},
	"scheduled": {},
}

// allowedBackupStatuses is the canonical status machine.
// pending → running → completed | failed. The handler forces a
// new row into ('pending') and the worker bumps it to
// ('running') → ('completed'|'failed').
var allowedBackupStatuses = map[string]struct{}{
	"pending":   {},
	"running":   {},
	"completed": {},
	"failed":    {},
}

// allowedBackupFrequencies restricts the /backup/jobs config.
// 'manual' is the de-facto "no schedule" sentinel — the handler
// maps it to disabled=true so the scheduler worker ignores the
// row.
var allowedBackupFrequencies = map[string]struct{}{
	"manual":  {},
	"daily":   {},
	"weekly":  {},
	"monthly": {},
}

// isAllowedBackupKind predicate used in CreateBackup to
// validate the `kind` field if the operator wants to override
// the default ('manual'). Today the handler always defaults
// to 'manual' and the scheduler always uses 'scheduled';
// the predicate is here for a future "create as scheduled"
// override.
func isAllowedBackupKind(k string) bool {
	_, ok := allowedBackupKinds[k]
	return ok
}

// isAllowedBackupStatus predicate used in handler validation
// (e.g. the RESTORE handler checking that the referenced
// backup_id row exists in a terminal state before queuing the
// restore).
func isAllowedBackupStatus(s string) bool {
	_, ok := allowedBackupStatuses[s]
	return ok
}

// isAllowedBackupFrequency predicate used in (future) PATCH
// /backup/jobs handler — the scheduler worker also gates on
// this so a typo'd frequency string can never enqueue a job.
func isAllowedBackupFrequency(f string) bool {
	_, ok := allowedBackupFrequencies[f]
	return ok
}
