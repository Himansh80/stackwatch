// Tier 11 Phase 5 — Backup / Restore (PL5).
//
// Platform-package mirror of the handler crypto layer.
//
// Why a copy here (not a handler-package import):
//   internal/handler imports internal/platform (handlers call
//   handlers.X(pool)). If internal/platform imported
//   internal/handler, we'd have a circular dep. The shared
//   logic is small enough (master-key fetch + HKDF + AES-GCM
//   encrypt-to-file) that a one-file mirror is cheaper than
//   carving out a fourth package.
//
// The handlers-platform-backup_crypto.go file owns the
// identical routines for the HTTP layer; the two are kept in
// sync manually because the boundary is intentional. Any
// non-trivial change (algorithm switch, layout change) needs
// to be applied to both.
package platform

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/hkdf"

	"github.com/google/uuid"
)

const (
	// platformBackupHKDFInfoTemplate — kept identical to the
	// handler-side constant so a key derivation matches
	// across boundaries.
	platformBackupHKDFInfoTemplate = "stackwatch-backup-tenant:%s"

	// platformBackupEncryptionAlgo — the canonical id
	// matching platform_backups.encryption_algo.
	platformBackupEncryptionAlgo = "aes-256-gcm"
)

// platformGetMasterKey fetches and decodes the master key
// from env. Mirrors handler.getMasterKey (handlers
// credentials.go) — duplicated here so the platform package
// stays import-clean.
func platformGetMasterKey() ([]byte, error) {
	v := os.Getenv("CREDENTIALS_MASTER_KEY")
	if v == "" {
		// Fallback to JWT_SECRET for dev environments. The
		// same caveat as the handler copy applies — not
		// production-safe.
		v = os.Getenv("JWT_SECRET")
		if v == "" {
			return nil, errors.New("CREDENTIALS_MASTER_KEY and JWT_SECRET both unset")
		}
	}
	k, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, fmt.Errorf("decode master key: %w", err)
	}
	if len(k) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes (got %d)", len(k))
	}
	return k, nil
}

// platformDeriveTenantKey — same algorithm as the
// handler-side helper. Kept in sync deliberately.
func platformDeriveTenantKey(masterKey []byte, tenantID uuid.UUID) []byte {
	info := fmt.Sprintf(platformBackupHKDFInfoTemplate, tenantID.String())
	r := hkdf.New(sha256.New, masterKey, nil, []byte(info))
	out := make([]byte, 32)
	if _, err := io.ReadFull(r, out); err != nil {
		for i := range out {
			out[i] = 0
		}
	}
	return out
}

// platformEncryptBackup mirrors handler.encryptBackup but
// returns the (nonce || ciphertext || tag) wire layout so
// the platform-side writeEncryptedBackup can persist
// everything in a single WriteFile call. AES-GCM guarantees
// the nonce is unique per (key, encrypt call) because we
// draw 12 fresh bytes from rand.Reader.
func platformEncryptBackup(plaintext, key []byte) (out []byte, err error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encrypt: key must be 32 bytes (got %d)", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("nonce: %w", err)
	}
	out = gcm.Seal(nil, nonce, plaintext, nil)
	// Prepend the nonce so the on-disk file is
	// self-contained. Layout: {nonce(12) || ciphertext ||
	// tag(16)}.
	prepend := make([]byte, 0, len(nonce)+len(out))
	prepend = append(prepend, nonce...)
	prepend = append(prepend, out...)
	return prepend, nil
}

// encryptFileToPath reads `srcPath`, AES-256-GCM encrypts
// the contents (using `key`), and writes the result to
// `dstPath` with mode 0600. Returns the SHA-256 of the
// encrypted bytes (for the row's sha256_ciphertext column).
func encryptFileToPath(srcPath, dstPath string, key []byte) (string, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("encryptFileToPath read: %w", err)
	}
	enc, err := platformEncryptBackup(data, key)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dstPath, enc, 0600); err != nil {
		return "", fmt.Errorf("encryptFileToPath write: %w", err)
	}
	return hex.EncodeToString(hashBytes(enc)), nil
}

// hashBytes is a tiny SHA-256 wrapper used by encrypt /
// verify helpers in this file.
func hashBytes(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}
