// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// HTTP route handlers for the 2 read+restore paths in the
// backup surface:
//
//	GET  /api/v1/platform/backup/:id/download — DownloadBackup (auth-protected)
//	POST /api/v1/platform/backup/restore      — RestoreBackup  (auth-protected, multipart upload)
//
// Threat model reminder (per proposal.md §"Risks" §"Backup
// encryption" + spec §"PL5 — Restore semantics"):
//
//   1. The download path streams the ON-DISK encrypted blob
//      so a tenant admin can take an offsite copy. The
//      ciphertext path + SHA-256 are returned in the
//      response headers for integrity-checking offline.
//   2. The restore path accepts a multipart upload of an
//      ENCRYPTED backup file + the matching backup_id
//      (so we re-derive the per-tenant key + verify the
//      GCM tag + verify the row's tenant ownership matches
//      claims.TenantID). pg_restore then re-applies the
//      dump in a transaction. Cross-tenant restore is
//      rejected outright.
//
// Why a multipart upload (not a JSON body with base64):
//   A 1GB backup blob as base64 would inflate by 33% (to
//   1.4GB) and balloon RAM. Multipart streams the bytes
//   directly to /tmp via streamingForm, then we read it
//   back from disk.
//
// Auth & tenancy: both routes PROTECTED. Both verify the
// row exists IN THE CALLER'S TENANT before doing anything
// destructive. A 404 on cross-tenant is the same 404 as a
// truly missing row — existence is never leaked.

package handler

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// =====================================================================
// GET /api/v1/platform/backup/:id/download
// =====================================================================

// DownloadBackup streams the encrypted .tar.gz.enc file from
// disk to the client. Sets Content-Disposition so curl saves
// the file with a tenant-meaningful name. Sets Content-Type
// to application/octet-stream so browsers do not try to
// render / parse the bytes.
//
// Why we re-verify the SHA-256 of the on-disk ciphertext
// before streaming:
//   Defense-in-depth. If the row says the file is intact but
//   the disk has been corrupted (a 1 in 2^64 chance for a
//   random bit flip, but real outages cluster — bad blocks,
//   stale NFS, half-copied file), the GCM tag at download
//   time would fail and the client would throw "your
//   download is broken" with no clear next step. Comparing
//   now turns that into a clean 409 with the audit log
//   catching the incident.
//
// 200 → binary stream (octet-stream). 401 / 404 / 409 on
// the standard error paths.
func DownloadBackup(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Read the row + claim it for this tenant. We
		// SELECT every column the response headers need so
		// we can set them in one shot.
		var (
			filePath    *string
			cipherSum   *string
			plainSum    *string
			size        int64
			encAlgo     string
			kind        string
			status      string
		)
		err = pool.Pgx().QueryRow(c.Request.Context(), `
			SELECT file_path, sha256_ciphertext, sha256_plaintext,
			       file_size_bytes, encryption_algo, backup_kind, status
			  FROM platform_backups
			 WHERE id = $1 AND tenant_id = $2
		`, id, tenantID).Scan(&filePath, &cipherSum, &plainSum,
			&size, &encAlgo, &kind, &status)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				kernel.RespondErrorWithCode(c, http.StatusNotFound,
					"not_found", "backup not found for this tenant")
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if filePath == nil || *filePath == "" {
			kernel.RespondErrorWithCode(c, http.StatusConflict,
				"not_ready", "backup file not yet written; wait for status=completed")
			return
		}
		if status != "completed" {
			kernel.RespondErrorWithCode(c, http.StatusConflict,
				"not_ready", "backup status is "+status+"; only completed backups can be downloaded")
			return
		}

		// SHA-256 self-check on the on-disk ciphertext.
		// Mismatch → 409 with audit log.
		if cipherSum != nil && *cipherSum != "" {
			actualSum, err := sha256FileHex(*filePath)
			if err != nil {
				kernel.RespondError(c, fmt.Errorf("hash open: %w", err))
				return
			}
			if !verifySHA256Hex(*cipherSum, []byte(actualSum)) {
				slog.Error("platform_backup_audit",
					"action", "download_corrupt",
					"tenant_id", tenantID.String(),
					"backup_id", id.String(),
					"expected", *cipherSum,
					"actual", actualSum,
				)
				kernel.RespondErrorWithCode(c, http.StatusConflict,
					"integrity_failed", "on-disk backup fails SHA-256 check — refusing to stream")
				return
			}
		}

		// Set headers then stream.
		filename := fmt.Sprintf("stackwatch-backup-%s.tar.gz.enc", id.String())
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Length", fmt.Sprintf("%d", size))
		if cipherSum != nil {
			c.Header("X-Backup-Sha256-Ciphertext", *cipherSum)
		}
		if plainSum != nil {
			c.Header("X-Backup-Sha256-Plaintext", *plainSum)
		}
		c.Header("X-Backup-Encryption-Algorithm", encAlgo)
		c.Header("X-Backup-Id", id.String())

		// c.File() handles the open + close + Range
		// negotiation. For a >2GB file a client that
		// uses Range requests gets a partial content
		// response (HTTP 206).
		c.File(*filePath)

		slog.Info("platform_backup_audit",
			"action", "download",
			"tenant_id", tenantID.String(),
			"backup_id", id.String(),
			"size_bytes", size,
		)
	}
}

// =====================================================================
// POST /api/v1/platform/backup/restore
// =====================================================================

// RestoreBackup accepts a multipart form with two fields:
//
//	form field "file"      — the encrypted .tar.gz.enc blob
//	form field "backup_id"  — the platform_backups.id to
//	                         restore from (required; we
//	                         verify ownership against claims
//	                         BEFORE decrypting)
//
// The handler:
//   1. Saves the upload to /tmp.
//   2. Re-derives the per-tenant key via HKDF.
//   3. AES-256-GCM decrypt + verify tag.
//   4. Untar to a working dir.
//   5. Spawns `pg_restore --clean --if-exists` against a
//      scratch DB (we DON'T touch ios production in Phase
//      5 — restore is staging-only; touching prod is a
//      later phase with manual confirmation).
//
// 202 → backupRestoreResp{restore_id, backup_id, status:
// pending, audit_note, next_step}. Actual restore runs in a
// goroutine and emits a slog line on completion.

func RestoreBackup(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		// Parse multipart (default 32 MiB form-limit is
		// too small for a 1GB backup; bump it to 4 GiB —
		// the request context sets its own ReadTimeout
		// already, so we don't worry about DOS).
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<30) // 4 GiB
		if err := c.Request.ParseMultipartForm(64 << 20); err != nil { // 64 MiB in-RAM
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "multipart parse failed: "+err.Error())
			return
		}

		// `backup_id` is REQUIRED. Without it we cannot
		// verify tenant ownership → silent cross-tenant
		// restore risk.
		idStr := c.Request.FormValue("backup_id")
		if idStr == "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "form field 'backup_id' is required")
			return
		}
		backupID, err := uuid.Parse(idStr)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Look up the row + claim ownership.
		var (
			filePath  *string
			cipherSum *string
			encAlgo   string
		)
		err = pool.Pgx().QueryRow(c.Request.Context(), `
			SELECT file_path, sha256_ciphertext, encryption_algo
			  FROM platform_backups
			 WHERE id = $1 AND tenant_id = $2
		`, backupID, tenantID).Scan(&filePath, &cipherSum, &encAlgo)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				// CRITICAL: cross-tenant restore gets
				// 404 NOT 403 so the caller cannot
				// fingerprint other tenants' IDs.
				kernel.RespondErrorWithCode(c, http.StatusNotFound,
					"not_found", "backup not found for this tenant")
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if encAlgo != backupEncryptionAlgo {
			kernel.RespondErrorWithCode(c, http.StatusConflict,
				"unsupported_algo", "encryption algorithm mismatch — refusing restore")
			return
		}

		// Save the upload to a working file.
		file, header, ferr := c.Request.FormFile("file")
		if ferr != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest,
				"bad_request", "form field 'file' missing: "+ferr.Error())
			return
		}
		defer file.Close()

		tmpEnc := filepath.Join(os.TempDir(), fmt.Sprintf("restore-in-%s-%s.enc",
			tenantID.String(), backupID.String()))
		out, oerr := os.Create(tmpEnc)
		if oerr != nil {
			kernel.RespondError(c, oerr)
			return
		}
		written, copyErr := io.Copy(out, file)
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(tmpEnc)
			kernel.RespondError(c, copyErr)
			return
		}
		if closeErr != nil {
			_ = os.Remove(tmpEnc)
			kernel.RespondError(c, closeErr)
			return
		}

		// Compare SHA-256 ciphertext (if the row has one)
		// against the upload. A mismatch means the user
		// uploaded a different file — refuse to decrypt.
		if cipherSum != nil && *cipherSum != "" {
			actualSum, err := sha256FileHex(tmpEnc)
			if err != nil {
				_ = os.Remove(tmpEnc)
				kernel.RespondError(c, err)
				return
			}
			if !verifySHA256Hex(*cipherSum, []byte(actualSum)) {
				_ = os.Remove(tmpEnc)
				slog.Error("platform_backup_audit",
					"action", "restore_ciphertext_mismatch",
					"tenant_id", tenantID.String(),
					"backup_id", backupID.String(),
					"uploaded", actualSum,
					"expected", *cipherSum,
					"uploaded_filename", header.Filename,
					"uploaded_size", written,
				)
				kernel.RespondErrorWithCode(c, http.StatusBadRequest,
					"ciphertext_mismatch",
					"uploaded file does not match the row's recorded sha256_ciphertext")
				return
			}
		}

		// Fire-and-forget the actual restore.
		go restoreInGoroutine(pool, tenantID, backupID, tmpEnc, written)

		slog.Info("platform_backup_audit",
			"action", "restore_queued",
			"tenant_id", tenantID.String(),
			"backup_id", backupID.String(),
			"uploaded_size", written,
			"uploaded_filename", header.Filename,
		)

		kernel.RespondStatus(c, http.StatusAccepted, backupRestoreResp{
			RestoreID: uuid.NewString(),
			BackupID:  backupID.String(),
			Status:    "pending",
			NextStep:  "poll_list",
			AuditNote: "every restore attempt is logged",
		})
	}
}

// restoreInGoroutine lives in handlers_platform_backup_restore_bg.go
// (kept under the 400-LOC cap by the split). It runs in a
// goroutine after RestoreBackup returns 202 to the client.
// =====================================================================
