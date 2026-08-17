package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ClaimsCtxKey is the gin.Context key under which RequireAuth stores the
// verified JWT claims so auth.ClaimsFromContext can retrieve them.
const ClaimsCtxKey = "auth.claims"

// ClaimsFromContext extracts the JWT claims stored on the gin context by
// RequireAuth. Returns false if the caller is unauthenticated.
func ClaimsFromContext(c *gin.Context) (*Claims, bool) {
	v, ok := c.Get(ClaimsCtxKey)
	if !ok {
		return nil, false
	}
	claims, ok := v.(*Claims)
	return claims, ok
}

// UserIDFromContext is a convenience for handlers that only need the user id.
func UserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	claims, ok := ClaimsFromContext(c)
	if !ok {
		return uuid.Nil, false
	}
	return claims.UserID, true
}
