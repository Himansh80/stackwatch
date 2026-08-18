package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/auth"
	"github.com/stackwatch/platform/internal/kernel"
)

const userCtxKey = "auth.user"
const authCtxKey = auth.ClaimsCtxKey

// RequireAuth returns a middleware that verifies JWT and stashes the user.
func RequireAuth(issuer *auth.Issuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if len(h) < 8 || !strings.HasPrefix(h, "Bearer ") {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			c.Abort()
			return
		}
		token := strings.TrimPrefix(h, "Bearer ")
		claims, err := issuer.Verify(token)
		if err != nil {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			c.Abort()
			return
		}
		// Enforce must_change_password: if the JWT carries that flag, block
		// every route except the ones the user needs to recover.
		if claims.MustChangePW {
			path := c.Request.URL.Path
			// Strip query for comparison.
			if i := strings.Index(path, "?"); i >= 0 {
				path = path[:i]
			}
			allowed := false
			switch path {
			case "/api/v1/auth/me",
				"/api/v1/auth/change-password",
				"/api/v1/auth/logout",
				"/api/v1/auth/profile": // allow updating name even if forced to change pw
				allowed = true
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{
					"error":   "password change required",
					"code":    "must_change_password",
					"message": "you must change your password before using the platform",
				})
				c.Abort()
				return
			}
		}
		c.Set(userCtxKey, claims)
		// also set the auth-package context key so auth.ClaimsFromContext works
		c.Set(authCtxKey, claims)
		setTenantInContext(c, claims)
		c.Next()
	}
}

// userFromContext returns the user from JWT claims (lightweight; not full DB lookup).
func userFromContext(c *gin.Context) (*kernel.User, bool) {
	v, ok := c.Get(userCtxKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*auth.Claims)
	if !ok {
		return nil, false
	}
	return &kernel.User{
		ID:           claims.UserID,
		TenantID:     claims.TenantID,
		Email:        claims.Email,
		Role:         claims.Role,
		MustChangePW: claims.MustChangePW,
	}, true
}

// lookupTenant fetches a tenant by id.
func (h *AuthHandler) lookupTenant(ctx context.Context, id uuid.UUID) (*kernel.Tenant, error) {
	t := &kernel.Tenant{}
	row := h.pool.Pgx().QueryRow(ctx, `
		SELECT id, name, slug, plan, status, created_at, updated_at
		FROM tenants WHERE id = $1`, id)
	if err := row.Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	return t, nil
}

// hashPassword is a thin wrapper around auth.HashPassword for handler use.
func hashPassword(plain string) (string, error) {
	return auth.HashPassword(plain)
}
