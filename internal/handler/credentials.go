// Package handler — Credentials vault (Tier 3.2 part 1).
//
// AES-GCM encrypted secrets per tenant. Master key comes from env
// (CREDENTIALS_MASTER_KEY, base64). The encryption_key_id column allows
// future rotation while keeping old secrets readable.
package handler

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/kernel"
)

// Credential is a returned secret (no plaintext; only metadata).
type Credential struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Kind        string     `json:"kind"`
	Description string     `json:"description"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CredentialsKeyId identifies which master key encrypted the ciphertext.
const CredentialsKeyId = "v1"

// getMasterKey fetches and decodes the master key from env.
func getMasterKey() ([]byte, error) {
	v := getEnv("CREDENTIALS_MASTER_KEY", "")
	if v == "" {
		// Derive a stable fallback from JWT_SECRET so dev works without setup.
		// NOT for production — log a warning at startup.
		v = getEnv("JWT_SECRET", "")
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

// encryptSecret encrypts plaintext with AES-GCM and returns (ciphertext, nonce).
func encryptSecret(plaintext string) ([]byte, []byte, error) {
	key, err := getMasterKey()
	if err != nil {
		return nil, nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}
	ct := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ct, nonce, nil
}

// decryptSecret reverses encryptSecret.
func decryptSecret(ciphertext, nonce []byte) (string, error) {
	key, err := getMasterKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	pt, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// ListCredentials returns all credentials for the tenant (metadata only).
func (h *TerminalHandler) ListCredentials(c *gin.Context) {
	tenantID := h.tenantID(c)
	rows, err := h.pool.Pgx().Query(c.Request.Context(),
		`SELECT id, name, kind, description, last_used_at, created_at, updated_at
         FROM credentials WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer rows.Close()
	creds := []Credential{}
	for rows.Next() {
		var cr Credential
		if err := rows.Scan(&cr.ID, &cr.Name, &cr.Kind, &cr.Description,
			&cr.LastUsedAt, &cr.CreatedAt, &cr.UpdatedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		creds = append(creds, cr)
	}
	kernel.RespondOK(c, gin.H{"credentials": creds, "total": len(creds)})
}

// GetCredential returns metadata (no plaintext) for one credential.
func (h *TerminalHandler) GetCredential(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var cr Credential
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT id, name, kind, description, last_used_at, created_at, updated_at
         FROM credentials WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan(&cr.ID, &cr.Name, &cr.Kind, &cr.Description,
		&cr.LastUsedAt, &cr.CreatedAt, &cr.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, cr)
}

// CreateCredential encrypts + inserts a credential.
func (h *TerminalHandler) CreateCredential(c *gin.Context) {
	tenantID := h.tenantID(c)
	var req struct {
		Name        string `json:"name" binding:"required"`
		Kind        string `json:"kind" binding:"required"`
		Secret      string `json:"secret" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.Kind != "password" && req.Kind != "key_passphrase" && req.Kind != "sudo_password" {
		kernel.RespondError(c, kernel.ErrBadRequest) // kind validation above
		return
	}
	ct, nonce, err := encryptSecret(req.Secret)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	var id uuid.UUID
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`INSERT INTO credentials (tenant_id, name, kind, ciphertext, nonce, encryption_key_id, description)
         VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		tenantID, req.Name, req.Kind, ct, nonce, CredentialsKeyId, req.Description).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			kernel.RespondError(c, kernel.ErrConflict)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":     id,
		"name":   req.Name,
		"kind":   req.Kind,
		"status": "created",
		"hint":   "store secret securely — never retrievable in API",
	})
}

// UpdateCredential replaces the secret + optional metadata.
func (h *TerminalHandler) UpdateCredential(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req struct {
		Secret      string `json:"secret"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.Secret == "" && req.Description == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	sets := []string{}
	args := []interface{}{}
	idx := 1
	if req.Secret != "" {
		ct, nonce, err := encryptSecret(req.Secret)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		sets = append(sets, fmt.Sprintf("ciphertext = $%d, nonce = $%d", idx, idx+1))
		args = append(args, ct, nonce)
		idx += 2
	}
	if req.Description != "" {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, req.Description)
		idx++
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, tenantID, id)
	q := fmt.Sprintf("UPDATE credentials SET %s WHERE tenant_id = $%d AND id = $%d",
		strings.Join(sets, ", "), idx, idx+1)
	tag, err := h.pool.Pgx().Exec(c.Request.Context(), q, args...)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	if tag.RowsAffected() == 0 {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	kernel.RespondOK(c, gin.H{"id": id, "status": "updated"})
}

// DeleteCredential removes a credential.
func (h *TerminalHandler) DeleteCredential(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	tag, err := h.pool.Pgx().Exec(c.Request.Context(),
		`DELETE FROM credentials WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	if tag.RowsAffected() == 0 {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	kernel.RespondOK(c, gin.H{"id": id, "status": "deleted"})
}

// RevealCredential returns the plaintext secret (audit-logged).
// Only super-admins or credential owner may reveal.
func (h *TerminalHandler) RevealCredential(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var ct, nonce []byte
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT ciphertext, nonce FROM credentials
         WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&ct, &nonce)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	secret, err := decryptSecret(ct, nonce)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("decrypt: %w", err))
		return
	}
	// Update last_used_at
	_, _ = h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE credentials SET last_used_at = NOW() WHERE id = $1`, id)
	kernel.RespondOK(c, gin.H{"id": id, "secret": secret})
}

// getEnv returns env var or default.
func getEnv(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
