// Package handler — Terminal module root.
//
// Defines TerminalHandler — instantiated once in routes, then dispatched to:
//   - terminal_keys.go         (SSH key CRUD)
//   - terminal_connections.go  (connections + history + SSH dial)
package handler

import (

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// TerminalHandler bundles all Tier-3 endpoints.
type TerminalHandler struct {
	pool *db.Pool
}

// NewTerminalHandler constructs the handler.
func NewTerminalHandler(pool *db.Pool) *TerminalHandler {
	return &TerminalHandler{pool: pool}
}

func (h *TerminalHandler) tenantID(c *gin.Context) uuid.UUID {
	tid, ok := tenantIDFromContext(c)
	if !ok {
		return uuid.Nil
	}
	return tid
}

func (h *TerminalHandler) userID(c *gin.Context) uuid.UUID {
	u, ok := userFromContext(c)
	if !ok || u == nil {
		return uuid.Nil
	}
	return u.ID
}
