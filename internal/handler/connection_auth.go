// Package handler — Auth method management (Tier 3.6).
package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/kernel"
)

// GetConnectionAuth returns the current auth_method + credential_id for a connection.
func (h *TerminalHandler) GetConnectionAuth(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var method string
	var credentialID *uuid.UUID
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT auth_method, credential_id FROM connections WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan(&method, &credentialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":           id,
		"auth_method":  method,
		"credential_id": credentialID,
	})
}

// UpdateConnectionAuthRequest is the body for PUT /auth.
type UpdateConnectionAuthRequest struct {
	AuthMethod   string  `json:"auth_method" binding:"required"` // "key" | "password" | "key_with_passphrase"
	CredentialID *string `json:"credential_id"`                  // required for password + key_with_passphrase
}

// UpdateConnectionAuth sets the auth_method + credential_id for a connection.
func (h *TerminalHandler) UpdateConnectionAuth(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req UpdateConnectionAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.AuthMethod != "key" && req.AuthMethod != "password" && req.AuthMethod != "key_with_passphrase" {
		kernel.RespondError(c, kernel.ErrBadRequest) // expects sentinel
		return
	}
	var credUUID *uuid.UUID
	if req.CredentialID != nil && *req.CredentialID != "" {
		u, err := uuid.Parse(*req.CredentialID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Verify credential exists in this tenant
		var exists bool
		err = h.pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT EXISTS(SELECT 1 FROM credentials WHERE tenant_id = $1 AND id = $2)`,
			tenantID, u).Scan(&exists)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if !exists {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		credUUID = &u
	}
	if (req.AuthMethod == "password" || req.AuthMethod == "key_with_passphrase") && credUUID == nil {
		kernel.RespondError(c, kernel.ErrBadRequest) // need credential
		return
	}
	tag, err := h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE connections SET auth_method = $1, credential_id = $2, updated_at = NOW()
         WHERE tenant_id = $3 AND id = $4`, req.AuthMethod, credUUID, tenantID, id)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	if tag.RowsAffected() == 0 {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":            id,
		"auth_method":   req.AuthMethod,
		"credential_id": credUUID,
		"status":        "updated",
	})
}
