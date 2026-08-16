package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/kernel"
)

// SignupRequest is the JSON body for POST /auth/signup.
type SignupRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	FullName   string `json:"full_name" binding:"required"`
	TenantName string `json:"tenant_name" binding:"required"`
}

// Signup creates a new tenant + admin user + returns a token.
func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}

	tenantID := generateTenantID()
	userID := generateUserID()

	tx, err := h.pool.Pgx().Begin(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	if _, err := tx.Exec(c.Request.Context(), `
		INSERT INTO tenants (id, name, slug, plan, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
	`, tenantID, req.TenantName, slugify(req.TenantName), kernel.PlanFree, kernel.TenantActive); err != nil {
		if isUniqueViolation(err) {
			kernel.RespondError(c, kernel.ErrConflict)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	if _, err := tx.Exec(c.Request.Context(), `
		INSERT INTO users (id, tenant_id, email, full_name, password_hash, role, status, must_change_password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, false, NOW(), NOW())
	`, userID, tenantID, email, req.FullName, hash, kernel.RoleAdmin, kernel.UserActive); err != nil {
		if isUniqueViolation(err) {
			kernel.RespondError(c, kernel.ErrConflict)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		kernel.RespondError(c, err)
		return
	}

	tok, err := h.issuer.Issue(userID, tenantID, email, kernel.RoleAdmin, false)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondCreated(c, LoginResponse{
		Token: tok,
		User: &kernel.User{
			ID: userID, TenantID: tenantID, Email: email, FullName: req.FullName,
			Role: kernel.RoleAdmin, Status: kernel.UserActive,
		},
		Tenant: &kernel.Tenant{
			ID: tenantID, Name: req.TenantName, Slug: slugify(req.TenantName),
			Plan: kernel.PlanFree, Status: kernel.TenantActive,
		},
	})
}

// slugify lowercases and replaces spaces with dashes.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '_' || r == '-':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
