package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/auth"
)

const tenantCtxKey = "auth.tenant_id"

// tenantIDFromContext returns the tenant_id stored in the JWT context.
func tenantIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(tenantCtxKey)
	if !ok {
		return uuid.Nil, false
	}
	tid, ok := v.(uuid.UUID)
	return tid, ok
}

// setTenantInContext stores the tenant id for downstream handlers.
func setTenantInContext(c *gin.Context, claims *auth.Claims) {
	c.Set(tenantCtxKey, claims.TenantID)
}
