// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// HTTP route handlers for the 3 write paths in the backup
// surface:
//
//	POST   /api/v1/platform/backup/create — CreateBackup  (auth-protected)
//	GET    /api/v1/platform/backup/list   — ListBackups  (auth-protected)
//	DELETE /api/v1/platform/backup/:id    — DeleteBackup  (auth-protected)
//
// Why POST /backup/create returns 202:
//   The actual backup runs asynchronously inside
//   platform.CreateBackupAndEncrypt (pg_dump + tar + gzip +
//   AES-256-GCM encrypt can take minutes for a multi-GB
//   tenant). The handler INSERTs a 'pending' row, fires a
//   goroutine, and returns 202 immediately. The UI then
//   polls /backup/list until the row's status flips to
//   'completed' or 'failed'.
//
// Why DELETE doesn't take ownership softly:
//   A backup is a tenant's disaster-recovery record. A
//   delete is FINAL — once the file's gone, restore can't
//   bring it back. The handler UPDATEs status to 'failed'
//   for the audit trail (so the dashboard can show "deleted
//   by Alice on YYYY-MM-DD"), then DELETEs the row + the
//   file. The 'failed' status here is misleading — we
//   instead drop the row entirely and rely on the
//   platform_backup_audit slog line for the paper trail.
//
// Auth & tenancy: all 3 are PROTECTED (RequireAuth); every
// query filters by claims.TenantID. Cross-tenant backup IDs
// return 404 from a SELECT-exists check, never 403 (which
// would leak existence).

package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
	"github.com/stackwatch/platform/internal/platform"
)

// =====================================================================
// POST /api/v1/platform/backup/create
// =====================================================================

// CreateBackup accepts a manual backup trigger. No body
// today — the operator's "click create" is sufficient. The
// handler INSERTs a 'pending' row, fires a background
// goroutine that bumps it to 'running' and ultimately
// 'completed' / 'failed', and returns 202 + the backup_id.
//
// 202 → backupCreateResp{backup_id, status, audit_url, next_step}.
// 400 → invalid JSON body (future body fields). 401 → no
// JWT. 500 → pool error during INSERT.
func CreateBackup(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		claims, ok := auth.ClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		// Optional body — Phase 5 ships a no-arg create.
		// We parse an empty body without complaint so a
		// future body (e.g. {label: "pre-upgrade"}) is
		// forward-compatible without a breaking change.
		var req backupCreateReq
		if c.Request.ContentLength > 0 {
			if err := c.ShouldBindJSON(&req); err != nil {
				kernel.RespondError(c, kernel.ErrBadRequest)
				return
			}
		}

		// INSERT the 'pending' row. Claims carry user_id
		// for the audit column (nullable — the scheduler
		// path uses NULL).
		userID := claims.UserID
		backupID := uuid.New()
		if _, err := pool.Pgx().Exec(c.Request.Context(), `
			INSERT INTO platform_backups
				(id, tenant_id, created_by_user_id, backup_kind, status)
			VALUES ($1, $2, $3, 'manual', 'pending')
		`, backupID, tenantID, userID); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Fire-and-forget. The goroutine uses a fresh
		// background context with a generous timeout —
		// the request context is cancelled on response.
		go func(bid uuid.UUID, tid uuid.UUID, uid uuid.UUID) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			logger := slog.Default().With(
				"tenant_id", tid.String(),
				"backup_id", bid.String(),
			)
			if _, err := platform.CreateBackupAndEncrypt(ctx, pool, tid, &uid, "manual"); err != nil {
				logger.Error("manual backup failed", "err", err)
			}
		}(backupID, tenantID, userID)

		kernel.RespondStatus(c, http.StatusAccepted, backupCreateResp{
			BackupID: backupID.String(),
			Status:   "pending",
			AuditURL: "/api/v1/platform/backup/" + backupID.String(),
			NextStep: "poll_list",
		})
	}
}

// =====================================================================
// GET /api/v1/platform/backup/list
// =====================================================================

// ListBackups returns the tenant's most-recent 20 backups
// plus a header KPI summary. The default limit=20 is
// overridable via the `?limit=N` query string (capped at
// 100 to keep a single SELECT bounded).
//
// 200 → backupListResp{backups, summary}.
func ListBackups(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := int64(20)
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		// List rows. ORDER BY created_at DESC matches the
		// dashboard newest-first display.
		rows, err := pool.Pgx().Query(c.Request.Context(), `
			SELECT id, tenant_id, created_by_user_id, backup_kind, status,
			       file_path, file_size_bytes, uncompressed_bytes,
			       table_count, row_count, encryption_algo,
			       sha256_plaintext, sha256_ciphertext,
			       started_at, completed_at, error_message, expires_at, created_at
			  FROM platform_backups
			 WHERE tenant_id = $1
			 ORDER BY created_at DESC
			 LIMIT $2
		`, tenantID, limit)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		var backups []backupRow
		for rows.Next() {
			r := backupRow{}
			var cbUUID *uuid.UUID
			var fp *string
			var sph *string
			var scc *string
			var compAt *time.Time
			var errMsg *string
			var expAt *time.Time
			if err := rows.Scan(
				&r.ID, &r.TenantID, &cbUUID, &r.BackupKind, &r.Status,
				&fp, &r.FileSizeBytes, &r.UncompressedBytes,
				&r.TableCount, &r.RowCount, &r.EncryptionAlgo,
				&sph, &scc,
				&r.StartedAt, &compAt, &errMsg, &expAt, &r.CreatedAt,
			); err != nil {
				kernel.RespondError(c, err)
				return
			}
			if cbUUID != nil {
				s := cbUUID.String()
				r.CreatedByUserID = &s
			}
			r.FilePath = fp
			r.SHA256Plaintext = sph
			r.SHA256Ciphertext = scc
			r.CompletedAt = compAt
			r.ErrorMessage = errMsg
			r.ExpiresAt = expAt
			backups = append(backups, r)
		}
		if err := rows.Err(); err != nil {
			kernel.RespondError(c, err)
			return
		}

		// KPI summary — one row, scalar projection. SUM +
		// COUNT + MAX all in one SELECT so the dashboard
		// avoids a follow-up fetch.
		var summary backupListSummary
		var lastAt *time.Time
		var lastStatus *string
		row := pool.Pgx().QueryRow(c.Request.Context(), `
			SELECT COUNT(*),
			       COALESCE(MAX(file_size_bytes), 0),
			       COALESCE(SUM(row_count), 0),
			       MAX(completed_at),
			       (
			         SELECT status FROM platform_backups
			          WHERE tenant_id = $1
			          ORDER BY completed_at DESC NULLS LAST
			          LIMIT 1
			       )
			  FROM platform_backups
			 WHERE tenant_id = $1
		`, tenantID)
		if err := row.Scan(&summary.TotalBackups, &summary.TotalStorageBytes,
			&summary.TotalRows, &lastAt, &lastStatus); err != nil {
			kernel.RespondError(c, err)
			return
		}
		summary.LastBackupAt = lastAt
		summary.LastBackupStatus = lastStatus

		kernel.RespondOK(c, backupListResp{
			Backups: backups,
			Summary: summary,
		})
	}
}

// =====================================================================
// DELETE /api/v1/platform/backup/:id
// =====================================================================

// DeleteBackup removes a tenant's backup. Steps:
//   1. SELECT the row to confirm tenant ownership + file_path.
//   2. os.Remove(file_path) if it exists (best-effort; a
//      missing file is not an error).
//   3. DELETE the row.
//
// 200 → backupDeleteResp{backup_id, tenant_id, status: deleted}.
// 404 → not found OR cross-tenant (we never confirm the row
// existed in another tenant).
func DeleteBackup(pool *db.Pool) gin.HandlerFunc {
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

		// Look up the row — limited to this tenant. We
		// need file_path for the unlink.
		var filePath *string
		err = pool.Pgx().QueryRow(c.Request.Context(), `
			SELECT file_path FROM platform_backups
			 WHERE id = $1 AND tenant_id = $2
		`, id, tenantID).Scan(&filePath)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				kernel.RespondErrorWithCode(c, http.StatusNotFound, "not_found",
					"backup not found for this tenant")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// DELETE the row FIRST. If the unlink fails after,
		// the audit log records the orphan path so an
		// operator can clean it up later. Doing it in
		// the reverse order would risk leaving the row
		// dangling with a deleted file.
		tag, err := pool.Pgx().Exec(c.Request.Context(), `
			DELETE FROM platform_backups
			 WHERE id = $1 AND tenant_id = $2
		`, id, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		removed := tag.RowsAffected() > 0

		// Best-effort file removal.
		if filePath != nil && *filePath != "" {
			if rmErr := os.Remove(*filePath); rmErr != nil && !os.IsNotExist(rmErr) {
				slog.Warn("backup: file remove failed",
					"tenant_id", tenantID.String(),
					"backup_id", id.String(),
					"file", *filePath,
					"err", rmErr.Error(),
				)
			}
		}

		// Audit log (slog) — separate row in api-gateway
		// logs that an operator can grep.
		slog.Info("platform_backup_audit",
			"action", "delete",
			"tenant_id", tenantID.String(),
			"backup_id", id.String(),
			"removed", removed,
		)

		kernel.RespondOK(c, backupDeleteResp{
			BackupID: id.String(),
			TenantID: tenantID.String(),
			Status:   "deleted",
			Removed:  removed,
		})
	}
}
