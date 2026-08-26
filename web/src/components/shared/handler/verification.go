// Package handler — Connection verification mode (Tier 3.4).
//
// Allows per-connection control of host-key verification:
//   - "strict"   (default): require known_hosts entry with matching fingerprint
//   - "insecure": skip verification (InsecureIgnoreHostKey) — backwards compat
package handler

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// GetConnectionVerificationMode returns the current mode for a connection.
func (h *TerminalHandler) GetConnectionVerificationMode(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var mode string
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT verification_mode FROM connections WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan(&mode)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":                id,
		"verification_mode": mode,
	})
}

// UpdateConnectionVerificationMode sets the mode for a connection.
type UpdateVerificationModeRequest struct {
	Mode string `json:"mode" binding:"required"` // "strict" or "insecure"
}

func (h *TerminalHandler) UpdateConnectionVerificationMode(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req UpdateVerificationModeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.Mode != "strict" && req.Mode != "insecure" {
		kernel.RespondError(c, kernel.ErrBadRequest) // expects string but sentinel works
		return
	}
	tag, err := h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE connections SET verification_mode = $1, updated_at = NOW()
         WHERE tenant_id = $2 AND id = $3`, req.Mode, tenantID, id)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	if tag.RowsAffected() == 0 {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":                id,
		"verification_mode": req.Mode,
		"status":            "updated",
	})
}

// makeStrictHostKeyCallback returns an ssh.HostKeyCallback that looks up the
// stored fingerprint for (tenantID, host, port) and requires an exact match.
func makeStrictHostKeyCallback(pool *db.Pool, ctx context.Context, tenantID uuid.UUID, host string, port int) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		got := ssh.FingerprintSHA256(key)
		var storedFp string
		err := pool.Pgx().QueryRow(ctx,
			`SELECT fingerprint FROM known_hosts
             WHERE tenant_id = $1 AND host = $2 AND port = $3`,
			tenantID, host, port).Scan(&storedFp)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("host %s:%d not trusted — POST /api/v1/terminal/known-hosts/trust first", host, port)
			}
			return fmt.Errorf("lookup known_hosts: %w", err)
		}
		if storedFp != got {
			return fmt.Errorf("host key mismatch! stored=%s got=%s (possible MITM)", storedFp, got)
		}
		return nil
	}
}
