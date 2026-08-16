package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/kernel"
)

// LoginRequest is the JSON body for POST /auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse is the JSON body for successful login.
type LoginResponse struct {
	Token  string `json:"token"`
	User   any    `json:"user"`
	Tenant any    `json:"tenant"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	user, tenant, err := h.lookupUserAndTenant(c.Request.Context(), email)
	if err != nil {
		// Hide whether email exists or not
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	if !user.IsActive() {
		kernel.RespondError(c, kernel.ErrForbidden)
		return
	}
	if !checkHash(user.PasswordHash, req.Password) {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}

	tok, err := h.issuer.Issue(user.ID, user.TenantID, user.Email, user.Role, user.MustChangePW)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}

	kernel.RespondOK(c, LoginResponse{Token: tok, User: user, Tenant: tenant})
}
