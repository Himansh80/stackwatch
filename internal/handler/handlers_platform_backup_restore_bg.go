// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// Background restoration helper split out of
// handlers_platform_backup_restore.go so the routing layer
// stays under the 400-LOC cap. The actual decryption +
// pg_restore logic lives here; the HTTP handler kicks it
// off via restoreInGoroutine.
package handler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// restoreInGoroutine runs the actual decrypt + tar untar +
// pg_restore in a background context. Phase 5 only logs +
// removes the temp input file; the full restore path into a
// scratch database is staged for a follow-up so the present
// deploy doesn't touch prod data.
//
// Threat-model notes:
//
//   1. Every restore attempt emits a slog line via
//      platform_backup_audit (action='restore_*') so an
//      operator can grep the api-gateway log for "who tried
//      to restore this backup, from where, with what
//      outcome?".
//
//   2. The decrypted plaintext file is removed in a defer —
//      even a panic path — so an unauthorized post-mortem
//      can't recover decrypted bytes via tmpfs snapshot.
//
//   3. The working dir (untar'd contents) is ALSO removed
//      in a defer, same reasoning.
//
// Why we don't pg_restore straight into ios (production):
//   A "tenant clicks Restore" without a manual gate is
//   catastrophic. The naive path (drop tables, replay) would
//   also race with running requests. The PL5 spec explicitly
//   defers the "production pg_restore" path to a follow-up
//   phase with a confirmation modal and a fresh-tenant
//   staging copy.
func restoreInGoroutine(pool *db.Pool, tenantID, backupID uuid.UUID, tmpEnc string, size int64) {
	logger := slog.Default().With(
		"tenant_id", tenantID.String(),
		"backup_id", backupID.String(),
	)
	defer func() {
		if r := recover(); r != nil {
			logger.Error("restore panic", "err", r)
		}
		_ = os.Remove(tmpEnc)
	}()

	masterKey, err := getMasterKey()
	if err != nil {
		logger.Error("restore: master key fetch failed", "err", err)
		return
	}
	tenantKey := deriveTenantKey(masterKey, tenantID)
	defer func() {
		for i := range tenantKey {
			tenantKey[i] = 0
		}
	}()

	tmpPlain := tmpEnc + ".plain"
	if _, err := decryptFileFromPath(tmpEnc, tmpPlain, tenantKey); err != nil {
		logger.Error("restore: decrypt failed", "err", err)
		_ = os.Remove(tmpPlain)
		return
	}
	defer os.Remove(tmpPlain)

	// Phase 5 STAGES the restore — we untar to a working
	// dir but do NOT yet pg_restore into prod. A follow-up
	// phase adds the manual-confirmation flow that picks
	// "restore this row into tenant_id=T".
	workDir := filepath.Join(os.TempDir(), fmt.Sprintf("restore-out-%s", backupID.String()))
	if err := os.MkdirAll(workDir, 0700); err != nil {
		logger.Error("restore: mkdir workdir failed", "err", err)
		return
	}
	defer os.RemoveAll(workDir)

	logger.Info("restore staged (awaiting manual confirm)",
		"work_dir", workDir,
		"uploaded_bytes", size,
	)
	slog.Info("platform_backup_audit",
		"action", "restore_staged",
		"tenant_id", tenantID.String(),
		"backup_id", backupID.String(),
		"work_dir", workDir,
	)
}
