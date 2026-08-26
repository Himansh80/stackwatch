// Package handler — Terminal SSH keys (Tier 3.1 part 1).
//
// List / get / create / delete SSH keys for the tenant.
package handler

import (
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/stackwatch/platform/internal/kernel"
)

// SSHKey is a returned SSH key (caller's view; no private).
type SSHKey struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Fingerprint string    `json:"fingerprint"`
	PublicKey   string    `json:"public_key"`
	KeyType     string    `json:"key_type"`
	Comment     string    `json:"comment"`
	HasPass     bool      `json:"has_passphrase"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (h *TerminalHandler) ListSSHKeys(c *gin.Context) {
	tenantID := h.tenantID(c)
	rows, err := h.pool.Pgx().Query(c.Request.Context(),
		`SELECT id, name, fingerprint, public_key, key_type, comment,
                (passphrase <> ''), created_at, updated_at
                FROM ssh_keys WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer rows.Close()
	keys := []SSHKey{}
	for rows.Next() {
		var k SSHKey
		var hasPass bool
		if err := rows.Scan(&k.ID, &k.Name, &k.Fingerprint, &k.PublicKey,
			&k.KeyType, &k.Comment, &hasPass, &k.CreatedAt, &k.UpdatedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		k.HasPass = hasPass
		keys = append(keys, k)
	}
	kernel.RespondOK(c, gin.H{"keys": keys, "total": len(keys)})
}

// GetSSHKey returns one SSH key (without private).
func (h *TerminalHandler) GetSSHKey(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var k SSHKey
	var hasPass bool
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT id, name, fingerprint, public_key, key_type, comment,
                (passphrase <> ''), created_at, updated_at
                FROM ssh_keys WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&k.ID, &k.Name, &k.Fingerprint, &k.PublicKey, &k.KeyType,
			&k.Comment, &hasPass, &k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	k.HasPass = hasPass
	kernel.RespondOK(c, k)
}

// CreateSSHKey generates a new SSH keypair and stores it.
func (h *TerminalHandler) CreateSSHKey(c *gin.Context) {
	tenantID := h.tenantID(c)
	var req struct {
		Name       string `json:"name" binding:"required"`
		KeyType    string `json:"key_type"`
		RsaBits    int    `json:"rsa_bits"`
		Passphrase string `json:"passphrase"`
		Comment    string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.KeyType == "" {
		req.KeyType = "ed25519"
	}
	if req.KeyType == "rsa" && req.RsaBits == 0 {
		req.RsaBits = 2048
	}

	priv, pub, err := generateKey(req.KeyType, req.RsaBits)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("generate key: %w", err))
		return
	}

	// Marshal private key (with optional passphrase)
	var privPEM *pem.Block
	if req.Passphrase != "" {
		privPEM, err = ssh.MarshalPrivateKeyWithPassphrase(priv, req.Comment, []byte(req.Passphrase))
	} else {
		privPEM, err = ssh.MarshalPrivateKey(priv, req.Comment)
	}
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	privPEMStr := string(pem.EncodeToMemory(privPEM))

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	fp := ssh.FingerprintSHA256(sshPub)

	comment := req.Comment
	if comment == "" {
		comment = req.Name
	}
	authKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))

	var id uuid.UUID
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`INSERT INTO ssh_keys (tenant_id, name, fingerprint, public_key, private_key, key_type, passphrase, comment)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		tenantID, req.Name, fp, authKey, privPEMStr, req.KeyType, req.Passphrase, comment).
		Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			kernel.RespondError(c, kernel.ErrConflict)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":          id,
		"name":        req.Name,
		"fingerprint": fp,
		"public_key":  authKey,
		"key_type":    req.KeyType,
		"comment":     comment,
		"status":      "created",
	})
}

// DeleteSSHKey removes an SSH key.
func (h *TerminalHandler) DeleteSSHKey(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	tag, err := h.pool.Pgx().Exec(c.Request.Context(),
		`DELETE FROM ssh_keys WHERE tenant_id = $1 AND id = $2`, tenantID, id)
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
