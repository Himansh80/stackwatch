// Tier 9 Phase 3 — Advanced RBAC (Tier 9.3) — User-role assignments.
//
// HTTP route handlers for the user↔role binding surface:
//
//	POST   /api/v1/enterprise/rbac/users/:user_id/roles         — AssignRBACRole
//	DELETE /api/v1/enterprise/rbac/users/:user_id/roles/:role_id — UnassignRBACRole
//
// The role CRUD endpoints live in handlers_rbac_roles.go; the check
// endpoint + cache helpers live in handlers_rbac_check.go.
//
// Shared types (userRoleAssignmentRow, rbacAssignReq) live in
// handlers_rbac_types.go.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/enterprise/rbac/users/:user_id/roles
// ------------------------------------------------------------------

// AssignRBACRole binds a user to a role. Idempotent via ON CONFLICT
// DO NOTHING — first call returns 201, subsequent calls return 200
// with the existing row. Role must belong to the caller's tenant.
//
// Cache invalidation: bust the user's cache entry on a fresh insert.
func AssignRBACRole(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(c.Param("user_id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var req rbacAssignReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		roleID, err := uuid.Parse(strings.TrimSpace(req.RoleID))
		if err != nil {
			kernel.RespondErrorWithCode(c, 400, "bad_request", "role_id must be a UUID")
			return
		}

		// Verify the role belongs to the caller's tenant — refuses
		// cross-tenant role assignment by ID guessing.
		var roleName string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT name FROM rbac_roles WHERE tenant_id = $1 AND id = $2`,
			tenantID, roleID,
		).Scan(&roleName)
		if err != nil {
			kernel.RespondErrorWithCode(c, 400, "bad_request",
				"role not found in your tenant")
			return
		}

		// ON CONFLICT DO NOTHING + RETURNING — if the (user_id,
		// role_id) pair already exists, RETURNING yields zero rows
		// and we fall back to a SELECT to populate the response
		// (and signal "already assigned" via the 200 status).
		var rowID, assignedAt string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO user_role_assignments (tenant_id, user_id, role_id)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (user_id, role_id) DO NOTHING
			 RETURNING id::text, assigned_at::text`,
			tenantID, userID, roleID,
		).Scan(&rowID, &assignedAt)

		created := err == nil
		if !created {
			// Zero rows → already assigned. Look up the existing row.
			err = pool.Pgx().QueryRow(c.Request.Context(),
				`SELECT id::text, assigned_at::text FROM user_role_assignments
				  WHERE tenant_id = $1 AND user_id = $2 AND role_id = $3`,
				tenantID, userID, roleID,
			).Scan(&rowID, &assignedAt)
			if err != nil {
				kernel.RespondError(c, err)
				return
			}
		}

		// Bust the user's cache (in either case — even the no-op
		// idempotent re-assign gets fresh perms on next /check,
		// which is fine; the cache key includes the userID).
		effectivePermCache.Delete(effectivePermCacheKey(tenantID, userID))

		resp := userRoleAssignmentRow{
			ID:         rowID,
			TenantID:   tenantID.String(),
			UserID:     userID.String(),
			RoleID:     roleID.String(),
			RoleName:   roleName,
			AssignedAt: assignedAt,
		}
		if created {
			kernel.RespondCreated(c, resp)
		} else {
			kernel.RespondOK(c, resp)
		}
	}
}

// ------------------------------------------------------------------
// Protected endpoint: DELETE /api/v1/enterprise/rbac/users/:user_id/roles/:role_id
// ------------------------------------------------------------------

// UnassignRBACRole removes a user↔role binding. Idempotent — if
// the binding doesn't exist, returns 200 with `{removed: false}`
// instead of 404 (avoids noisy retry logs).
func UnassignRBACRole(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(c.Param("user_id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		roleID, err := uuid.Parse(strings.TrimSpace(c.Param("role_id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM user_role_assignments WHERE tenant_id = $1 AND user_id = $2 AND role_id = $3`,
			tenantID, userID, roleID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}

		// Always bust cache (cheap; we don't know if anything was
		// actually deleted).
		effectivePermCache.Delete(effectivePermCacheKey(tenantID, userID))

		kernel.RespondOK(c, gin.H{
			"removed": tag.RowsAffected() > 0,
			"user_id": userID.String(),
			"role_id": roleID.String(),
		})
	}
}
