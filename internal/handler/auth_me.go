package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/kernel"
)

// Me returns the current user + tenant.
func (h *AuthHandler) Me(c *gin.Context) {
	claimsUser, ok := userFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	// Fetch full user from DB so we get the latest full_name, status, etc.
	u, t, err := h.lookupUserAndTenant(c.Request.Context(), claimsUser.Email)
	if err != nil {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	kernel.RespondOK(c, gin.H{"user": u, "tenant": t})
}

// Logout is a no-op for stateless JWT. Client discards the token.
func (h *AuthHandler) Logout(c *gin.Context) {
	kernel.RespondOK(c, gin.H{"ok": true})
}

// ChangePasswordRequest is the JSON body for POST /auth/change-password.
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=8"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword verifies the current password and updates it.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	claimsUser, ok := userFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	// Fetch the full user record from DB (claims don't include password_hash).
	u, _, err := h.lookupUserAndTenant(c.Request.Context(), claimsUser.Email)
	if err != nil {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if !checkHash(u.PasswordHash, req.OldPassword) {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	hash, err := hashPassword(req.NewPassword)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if _, err := h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE users SET password_hash=$1, must_change_password=false, updated_at=NOW() WHERE id=$2`,
		hash, u.ID,
	); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"ok": true})
}

// ForgotPassword is a stub for Tier 0. Sends no email yet.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	kernel.RespondOK(c, gin.H{"ok": true, "message": "if email exists, reset link sent"})
}

// ResetPassword is a stub for Tier 0.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	kernel.RespondOK(c, gin.H{"ok": true})
}
