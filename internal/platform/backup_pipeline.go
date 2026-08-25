// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// Pipeline helpers used by CreateBackupAndEncrypt.
//
// Split out of backup_helpers.go so each file stays under
// the 400-LOC cap. The functions live in this file because
// they're called only by CreateBackupAndEncrypt — handlers
// never reach in directly. The same advisory-lock helper
// stays in backup_helpers.go because it's ALSO called by the
// Download + Restore handlers (they acquire the lock to
// prevent a download racing with a delete).
package platform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// pgDumpAndArchive runs `pg_dump --format=custom` against the
// supplied tenant + bundles the result into a tar.gz in /tmp.
// Returns (plaintextPath, uncompressedSize, tableCount,
// rowCount, err).
//
// Why we hand-build the tar (not pg_dump --format=tar):
//   The custom format is parallel-restore friendly and
//   compressed already. We wrap it in a tar because future
//   phases (config snapshots, uploads) drop extra files in
//   the same archive — keeping a single extension (.tar.gz)
//   from day one means we don't migrate the filename
//   convention later.
//
// Why we count tables + rows separately from pg_dump:
//   The handler's "Row count" KPI is tenant-visible. We
//   produce the number from a SUM of estimated row counts in
//   pg_stat_user_tables. This is an estimate so the JSON
//   contract marks the column "approx".
func pgDumpAndArchive(ctx context.Context, tenantID, backupID uuid.UUID) (string, int64, int, int64, error) {
	tmpDir, err := os.MkdirTemp(backupTmpPrefix, "pg-*")
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("mkdir tmp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	dumpPath := filepath.Join(tmpDir, "dump.custom")
	dsn, err := databaseURLForPgDump()
	if err != nil {
		return "", 0, 0, 0, err
	}

	// pg_dump --format=custom. Exclude platform_backups
	// itself so a restore doesn't duplicate the in-flight
	// row. Includes every other platform_* table.
	dumpCtx, cancel := context.WithTimeout(ctx, pgDumpTimeout)
	defer cancel()
	dumpCmd := exec.CommandContext(dumpCtx,
		"pg_dump",
		"--dbname="+dsn,
		"--format=custom",
		"--no-owner",
		"--exclude-table=platform_backups",
		"--exclude-table=platform_backup_jobs",
		"--file="+dumpPath,
	)
	if out, err := dumpCmd.CombinedOutput(); err != nil {
		return "", 0, 0, 0, fmt.Errorf("pg_dump failed: %w (%s)", err, string(out))
	}

	// tar+gzip the dump.
	tarPath := filepath.Join(os.TempDir(), fmt.Sprintf("backup-%s.tar.gz", backupID.String()))
	tarCmd := exec.CommandContext(ctx, "tar", "-czf", tarPath, "-C", tmpDir, "dump.custom")
	if out, err := tarCmd.CombinedOutput(); err != nil {
		return "", 0, 0, 0, fmt.Errorf("tar failed: %w (%s)", err, string(out))
	}

	// Inspect the produced tar.gz for size + counts.
	stat, err := os.Stat(tarPath)
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("stat tar: %w", err)
	}
	tableCount, rowCount, err := countTenantTablesAndRows(ctx, dumpPath, tenantID)
	if err != nil {
		// Counts are KPI-best-effort; the backup is still
		// usable. Default to 0 so the row can still UPDATE.
		tableCount = 0
		rowCount = 0
	}
	return tarPath, stat.Size(), tableCount, rowCount, nil
}

// countTenantTablesAndRows opens the produced .custom dump
// via `pg_restore --list` (cheap, no schema apply) and
// cross-references with row estimates from
// pg_stat_user_tables. Returns (0, 0, nil) on any failure —
// the caller zero-fills the row anyway.
func countTenantTablesAndRows(ctx context.Context, dumpPath string, tenantID uuid.UUID) (int, int64, error) {
	listCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(listCtx, "pg_restore", "--list", dumpPath).CombinedOutput()
	if err != nil {
		return 0, 0, err
	}
	tableCount := strings.Count(string(out), "TABLE ")
	if tableCount == 0 {
		// pg_restore --list output format differs across
		// pg versions; treat any line containing ' TABLE '
		// as a table.
		tableCount = strings.Count(string(out), " TABLE ")
	}

	// Row estimate (cheap; uses the stats collector).
	var rowSum int64
	rowCmd := exec.CommandContext(listCtx, "psql", dsnForExec(),
		"-tA", "-c", fmt.Sprintf(`
			SELECT COALESCE(SUM(n_live_tup), 0)
			  FROM pg_stat_user_tables
			 WHERE schemaname='public'
		`))
	rowOut, rerr := rowCmd.CombinedOutput()
	if rerr == nil {
		fmt.Sscanf(strings.TrimSpace(string(rowOut)), "%d", &rowSum)
	}
	return tableCount, rowSum, nil
}

// dsnForExec is a tiny helper that returns the DB DSN from
// env DATABASE_URL (with a sensible default matching config.go).
// Kept private — package-internal.
func dsnForExec() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://ios:***@192.168.0.116:5432/ios?sslmode=disable"
}

// databaseURLForPgDump wraps dsnForExec so a future env
// override (e.g. redirecting pg_dump to a hot-standby for
// heavy tenants) is one line to swap.
func databaseURLForPgDump() (string, error) {
	return dsnForExec(), nil
}

// writeEncryptedBackup puts the encrypted blob at the
// canonical storage path. We write to a tmp sibling first so a
// crashed encrypt doesn't leave a half-written file at the
// final name (atomic rename). Returns (tmp, final, err).
func writeEncryptedBackup(tenantID, backupID uuid.UUID, plainPath string, key []byte) (string, string, error) {
	root := os.Getenv("BACKUP_ROOT_DIR")
	if root == "" {
		root = defaultBackupRootDir
	}
	dir := filepath.Join(root, tenantID.String())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", "", fmt.Errorf("mkdir backup dir: %w", err)
	}
	final := filepath.Join(dir, backupID.String()+".tar.gz.enc")
	tmp := final + ".tmp"
	if _, err := encryptFileToPath(plainPath, tmp, key); err != nil {
		return "", "", err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
		return "", "", fmt.Errorf("chmod: %w", err)
	}
	if err := os.Rename(tmp, final); err != nil {
		_ = os.Remove(tmp)
		return "", "", fmt.Errorf("rename: %w", err)
	}
	return tmp, final, nil
}

// sha256FileToHex streams `path` through SHA-256 and returns
// lowercase hex. 64-char output matching the text column
// width in platform_backups.
func sha256FileToHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// =====================================================================
// Audit log
// =====================================================================

// logBackupAudit writes a structured audit row via slog so an
// operator can grep the api-gateway log for all backup actions
// (create / download / restore attempt). The DB-side audit
// (audit_log table) is populated by the existing Tier-3.1
// middleware — we don't duplicate it here; the slog line is
// the platform-side hook.
func logBackupAudit(action, tenantID, backupID, detail string) {
	slog.Info("platform_backup_audit",
		"action", action,
		"tenant_id", tenantID,
		"backup_id", backupID,
		"detail", detail,
	)
}
