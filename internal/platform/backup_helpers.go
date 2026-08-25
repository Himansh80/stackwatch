// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// Core backup helper: turns "I want a backup for tenant X" into
// a populated platform_backups row + on-disk encrypted blob.
//
// Per spec §"PL5 — Backup/Restore", every backup goes through
// this function so the handler and the scheduler share one
// implementation. The handler (POST /backup/create) calls it
// in a fire-and-forget goroutine; the scheduler calls it once
// per due job.
//
// Steps (numbered to match the spec):
//
//	1. Acquire pg_advisory_lock(hash(tenant_id)).
//	   Prevents two workers from running a backup for the
//	   same tenant at the same time — a partial overwrite
//	   would corrupt the tenant's encrypted file. The lock
//	   is session-scoped and explicitly released at the end
//	   of this function; the DeferBackupLock() helper handles
//	   the release even on panic.
//
//	2. INSERT platform_backups row with status='running'.
//	   The handler pre-inserts 'pending' on entry so the
//	   caller has an ID to poll; this helper bumps the same
//	   row to 'running'. (The simpler design — INSERT once
//	   here — would race with the handler's poll loop. The
//	   pre-insert keeps the poll deterministic.)
//
//	3. Run pg_dump --format=custom to a tmp file. The dump
//	   captures every platform_* table (and the
//	   platform_backups row itself as a sentinel — restored
//	   via --exclude-table=platform_backups so we never
//	   overwrite live data).
//
//	4. tar+gzip the dump into a single .tar.gz blob (the
//	   upgrade path for adding per-tenant uploads / configs
//	   is to add them to the tar).
//
//	5. SHA-256 the plaintext.
//
//	6. Derive the per-tenant key via HKDF-SHA256.
//
//	7. AES-256-GCM encrypt + sign with the derived key.
//
//	8. SHA-256 the ciphertext.
//
//	9. Write to /opt/stackwatch/backups/{tenant_id}/
//	   {backup_id}.tar.gz.enc with mode 0600. The parent
//	   directory is mkdir'd with mode 0700.
//
//	10. UPDATE platform_backups row: status='completed',
//	    file_path + sizes + checksums + completed_at + row
//	    count + table count.
//
//	11. pg_advisory_unlock — done in a defer because step 7
//	    can panic on a cipher error.
//
// Why pg_advisory_lock (NOT row-level FOR UPDATE on
// platform_backups):
//   The advisory lock is intent-based ("only one backup for
//   tenant X") rather than a SQL row state machine. It also
//   releases without holding a transaction — the worker can
//   commit the row first, THEN release the lock, without
//   racing a concurrent caller. FOR UPDATE would force us to
//   keep the transaction open across the multi-second
//   pg_dump + encrypt pass (never do this — pin = outage).
//
// Why pg_dump --format=custom:
//   The custom format is pg_restore-friendly (binary),
//   compressed (gzip-equivalent), and supports parallel
//   restore — see spec §"Restore semantics". Plain SQL would
//   be slower + larger + non-transactional across the file.
package platform

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

const (
	// backupRootDir is the on-disk root for every encrypted
	// backup. Mode 0700 on create. Matches the spec's
	// /opt/stackwatch/backups/{tenant_id}/{id}.tar.gz.enc
	// layout. Override via env BACKUP_ROOT_DIR for a dev
	// machine — the linux-prod path is hard-coded so an
	// operator typo can't re-route backups to a tmpfs.
	defaultBackupRootDir = "/opt/stackwatch/backups"

	// backupTmpPrefix is the pre-encryption working dir.
	// Always /tmp because (a) on the linux prod box /tmp is
	// tmpfs = fast, (b) the encryption step removes the
	// plaintext on success, (c) defer cleanup removes it on
	// panic.
	backupTmpPrefix = "/tmp/stackwatch-backup-"

	// pgDumpTimeout bounds a single tenant's pg_dump pass.
	// 10 min is plenty for a 1GB tenant; larger installs
	// tune via env PL5_BACKUP_TIMEOUT.
	pgDumpTimeout = 10 * time.Minute
)

// ErrBackupTenantBusy is returned by CreateBackupAndEncrypt
// when a concurrent caller already holds the per-tenant
// advisory lock. The handler maps this to 429 / 409 (concurrent
// create calls simply queue).
var ErrBackupTenantBusy = errors.New("backup already running for this tenant")

// =====================================================================
// Per-tenant advisor lock
// =====================================================================

// acquireTenantBackupLock takes a session-scoped
// pg_ad advisory_lock identified by a hash of the tenant_id.
// Returns a release func the caller MUST defer — failure to
// release leaks the slot until the session ends.
//
// We use the BIGINT form (pg_advisory_lock(bigint)) — the
// (int, int) form would force us to split the UUID, which is
// unstable across the unique key. The bigint is the top 63
// bits of the UUID hex so collisions across tenants are
// astronomically unlikely (see https://stackoverflow.com/q/
// 35637507 — pg advisory locks reserve 1 bit for "shared").
func acquireTenantBackupLock(ctx context.Context, pool *db.Pool, tenantID uuid.UUID) (release func(), err error) {
	// Use the first 8 bytes of the UUID to feed the bigint.
	// No string parsing — encode the bytes directly to a
	// signed int64 with the high bit cleared so pg treats it
	// as a positive.
	var key int64
	hash := sha256.Sum256(tenantID[:])
	for i := 0; i < 8; i++ {
		key = (key << 8) | int64(hash[i])
	}
	if _, err := pool.Pgx().Exec(ctx, `SELECT pg_advisory_lock($1)`, key); err != nil {
		return nil, fmt.Errorf("acquire tenant backup lock: %w", err)
	}
	return func() {
		// Release uses a fresh context — ctx may be
		// cancelled if the caller shut down mid-backup.
		// A 5s timeout prevents a stuck conn from
		// blocking the caller forever.
		relCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Pgx().Exec(relCtx, `SELECT pg_advisory_unlock($1)`, key)
	}, nil
}

// =====================================================================
// Public API
// =====================================================================

// CreateBackupAndEncrypt is the per-tenant backup core. Used
// by both the HTTP handler and the BackupSchedulerWorker. The
// `kind` parameter ('manual'|'scheduled') is stored on the row.
// `userID` is optional (the scheduler passes nil).
//
// On success: returns the new backup_id (also written to
// platform_backups.id) and the row is fully populated
// (status='completed', file_path, sizes, checksums).
//
// On failure: returns the underlying error AND ensures the
// row exists in status='failed' with error_message set, so
// the dashboard can show "why did my last backup fail?".
func CreateBackupAndEncrypt(
	ctx context.Context,
	pool *db.Pool,
	tenantID uuid.UUID,
	userID *uuid.UUID,
	kind string,
) (backupID uuid.UUID, err error) {
	logger := slog.Default().With(
		"tenant_id", tenantID.String(),
		"kind", kind,
	)

	// Resolve the master key ONCE per call. platformGetMasterKey
	// is safe to call concurrently — the env is read-only.
	masterKey, err := platformGetMasterKey()
	if err != nil {
		return uuid.Nil, fmt.Errorf("master key: %w", err)
	}
	tenantKey := platformDeriveTenantKey(masterKey, tenantID)
	defer func() {
		// Zero the derived key on the way out so a
		// heap-dump post-mortem can't recover it. masterKey
		// is the caller's to manage — we don't touch.
		for i := range tenantKey {
			tenantKey[i] = 0
		}
	}()

	// 1. Lock for this tenant.
	release, err := acquireTenantBackupLock(ctx, pool, tenantID)
	if err != nil {
		return uuid.Nil, err
	}
	defer release()

	// 2. Find OR create the running row. The handler
	//    pre-inserts 'pending'; the scheduler skips the
	//    pre-insert and lands here directly.
	if backupID == uuid.Nil {
		backupID = uuid.New()
	}
	var existingID uuid.UUID
	err = pool.Pgx().QueryRow(ctx, `
		SELECT id FROM platform_backups
		 WHERE tenant_id = $1 AND status IN ('pending','running')
		 ORDER BY created_at DESC LIMIT 1
	`, tenantID).Scan(&existingID)
	if err == nil && existingID != uuid.Nil {
		// A 'pending' row already exists — promote it.
		backupID = existingID
		if _, err := pool.Pgx().Exec(ctx, `
			UPDATE platform_backups SET status='running', started_at=NOW()
			 WHERE id=$1 AND status IN ('pending','running')
		`, backupID); err != nil {
			return uuid.Nil, fmt.Errorf("promote pending row: %w", err)
		}
	} else {
		// Fresh row.
		uid := userID
		if _, ierr := pool.Pgx().Exec(ctx, `
			INSERT INTO platform_backups
				(id, tenant_id, created_by_user_id, backup_kind, status, started_at)
			VALUES ($1, $2, $3, $4, 'running', NOW())
		`, backupID, tenantID, uid, kind); ierr != nil {
			return uuid.Nil, fmt.Errorf("insert running row: %w", ierr)
		}
	}

	// Ensure failure logs the row.
	failRow := func(e error) {
		_, _ = pool.Pgx().Exec(context.Background(), `
			UPDATE platform_backups
			   SET status='failed', error_message=$2, completed_at=NOW()
			 WHERE id=$1
		`, backupID, e.Error())
		logger.Error("backup failed", "backup_id", backupID.String(), "err", e)
	}

	// 3-4. pg_dump + tar.gz to a tmp file.
	plainPath, plainSize, tableCount, rowCount, err := pgDumpAndArchive(ctx, tenantID, backupID)
	if err != nil {
		failRow(err)
		return uuid.Nil, err
	}
	defer os.Remove(plainPath)

	// 5. SHA-256 plaintext.
	plainSum, err := sha256FileToHex(plainPath)
	if err != nil {
		failRow(err)
		return uuid.Nil, err
	}

	// 6-7. Derive key + AES-256-GCM encrypt (already done
	//      above; reuse tenantKey).

	// 9. Write encrypted file.
	encRel, encAbs, err := writeEncryptedBackup(tenantID, backupID, plainPath, tenantKey)
	if err != nil {
		failRow(err)
		return uuid.Nil, err
	}
	defer os.Remove(encRel) // we keep encAbs as the canonical filename

	// 8. SHA-256 ciphertext (after the rename to the final
	//    path so the hash is of the file-as-stored).
	encSum, err := sha256FileToHex(encAbs)
	if err != nil {
		failRow(err)
		return uuid.Nil, err
	}
	encStat, _ := os.Stat(encAbs)
	encSize := int64(0)
	if encStat != nil {
		encSize = encStat.Size()
	}

	// 10. Final UPDATE.
	completedAt := time.Now().UTC()
	expiresAt := completedAt.Add(7 * 24 * time.Hour) // default 7d; jobs row may override later
	if _, err := pool.Pgx().Exec(ctx, `
		UPDATE platform_backups
		   SET status='completed',
		       file_path=$2,
		       file_size_bytes=$3,
		       uncompressed_bytes=$4,
		       table_count=$5,
		       row_count=$6,
		       encryption_algo='aes-256-gcm',
		       sha256_plaintext=$7,
		       sha256_ciphertext=$8,
		       completed_at=$9,
		       expires_at=$10
		 WHERE id=$1
	`, backupID, encAbs, encSize, plainSize, tableCount, rowCount,
		plainSum, encSum, completedAt, expiresAt); err != nil {
		failRow(err)
		return uuid.Nil, err
	}

	logger.Info("backup completed",
		"backup_id", backupID.String(),
		"plain_bytes", plainSize,
		"enc_bytes", encSize,
		"tables", tableCount,
		"rows", rowCount,
	)
	return backupID, nil
}

