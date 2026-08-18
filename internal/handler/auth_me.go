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

// ChangePassword + ForgotPassword + ResetPassword live in auth_recovery.go.

// RefreshTokenRequest is the body for POST /auth/refresh.
type RefreshTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// Refresh issues a new JWT for a still-valid token. This is the cheapest
// way to keep a long-lived client logged in without re-entering credentials.
//
// We do NOT rotate the underlying subject — refresh just re-signs the
// existing claims with a fresh ExpiresAt. Token rotation (issue a fresh
// sub each refresh) is deferred to Tier 11 / OAuth2.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	// Re-verify; this returns the same claims if still valid.
	claims, err := h.issuer.Verify(req.Token)
	if err != nil {
		h.logger.Error("refresh verify failed", "err", err, "token_len", len(req.Token))
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	newToken, err := h.issuer.Issue(claims.UserID, claims.TenantID, claims.Email, claims.Role, claims.MustChangePW)
	if err != nil {
		kernel.RespondError(c, kernel.ErrInternal)
		return
	}
	kernel.RespondOK(c, gin.H{
		"ok":    true,
		"token": newToken,
		"ttl":   h.issuer.TTLSeconds(),
	})
}
