package handler

import (
	"strings"
	"time"

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
	ip := clientIP(c)

	// Rate limit BEFORE doing any DB lookup. 5 attempts per ip+email per 5min.
	if wait, err := checkLoginRateLimit(ip, email); err != nil {
		kernel.RespondError(c, err)
		c.Header("Retry-After", wait.Round(time.Second).String())
		return
	}

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
		// Failed login → audit
		h.auditLog(c.Request.Context(), &user.TenantID, &user.ID, "login.failed", user.Email, ip, map[string]any{"reason": "bad_password"})
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}

	tok, err := h.issuer.Issue(user.ID, user.TenantID, user.Email, user.Role, user.MustChangePW)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}

	// Update last_login_at + audit success.
	_, _ = h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE users SET last_login_at=NOW() WHERE id=$1`, user.ID)
	h.auditLog(c.Request.Context(), &user.TenantID, &user.ID, "login.success", user.Email, ip, nil)

	kernel.RespondOK(c, LoginResponse{Token: tok, User: user, Tenant: tenant})
}
