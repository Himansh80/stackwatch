// Tier 9 Phase 3 — Advanced RBAC (Tier 9.3) — Role CRUD.
//
// HTTP route handlers for the role CRUD surface:
//
//	GET    /api/v1/enterprise/rbac/roles          — ListRBACRoles
//	POST   /api/v1/enterprise/rbac/roles          — CreateRBACRole
//	PATCH  /api/v1/enterprise/rbac/roles/:id      — UpdateRBACRole
//	DELETE /api/v1/enterprise/rbac/roles/:id      — DeleteRBACRole
//
// The assignment endpoints (AssignRBACRole / UnassignRBACRole) live
// in handlers_rbac_assignments.go and the check endpoint +
// cache helpers live in handlers_rbac_check.go — splitting keeps
// each file under the 500-LOC modular rule.
//
// Shared types (rbacRoleRow, rbacRoleReq, rbacRolePatchReq,
// permission allowlist, built-in role seed) live in
// handlers_rbac_types.go.
package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/rbac/roles
// ------------------------------------------------------------------

// ListRBACRoles returns every role (built-in + custom) for the
// caller's tenant. Lazy-seeds the four built-in roles on first call
// (idempotent — re-calls are a no-op).
//
// Query params:
//
//	include_builtin=true|false  default=true; set false to skip
//	                              built-in rows (used by the UI's
//	                              "Custom" tab to avoid duplicate
//	                              fetch).
//
// The 4 built-in roles are seeded by seedBuiltinRolesForTenant on
// first call. We use INSERT ... ON CONFLICT DO NOTHING so the seed
// is fully idempotent — no race between two parallel first-calls.
func ListRBACRoles(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		// Lazy-seed built-ins. If any of the 4 names already exists
		// for this tenant, ON CONFLICT DO NOTHING makes this a no-op.
		if err := seedBuiltinRolesForTenant(c, pool, tenantID); err != nil {
			kernel.RespondError(c, err)
			return
		}

		includeBuiltin := true
		if v := strings.ToLower(strings.TrimSpace(c.Query("include_builtin"))); v == "false" || v == "0" || v == "no" {
			includeBuiltin = false
		}

		q := `SELECT id::text, tenant_id::text, name, COALESCE(description, ''), permissions, is_builtin, created_at::text
		      FROM rbac_roles WHERE tenant_id = $1`
		if !includeBuiltin {
			q += ` AND is_builtin = false`
		}
		q += ` ORDER BY is_builtin DESC, name ASC`

		rows, err := pool.Pgx().Query(c.Request.Context(), q, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []rbacRoleRow{}
		for rows.Next() {
			var r rbacRoleRow
			var isBuiltin bool
			if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.Description, &r.Permissions, &isBuiltin, &r.CreatedAt); err != nil {
				continue
			}
			r.IsBuiltin = isBuiltin
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"roles": out, "total": len(out), "include_builtin": includeBuiltin})
	}
}

// seedBuiltinRolesForTenant inserts the four built-in roles
// (Admin / Operator / Viewer / Billing) for the given tenant if
// they don't already exist. Idempotent via ON CONFLICT DO NOTHING
// against the UNIQUE (tenant_id, name) constraint on rbac_roles.
//
// Admin's permissions = entire builtinPermissions allowlist (so
// future permission additions flow in automatically for Admin).
// Other built-ins have their permissions hard-coded in
// builtinRolesSeed.
func seedBuiltinRolesForTenant(c *gin.Context, pool *db.Pool, tenantID uuid.UUID) error {
	ctx := c.Request.Context()
	for _, br := range builtinRolesSeed {
		perms := br.Permissions
		if br.AllPermissions {
			perms = make([]string, len(builtinPermissions))
			copy(perms, builtinPermissions)
		}
		_, err := pool.Pgx().Exec(ctx,
			`INSERT INTO rbac_roles (tenant_id, name, description, permissions, is_builtin)
			 VALUES ($1, $2, $3, $4, true)
			 ON CONFLICT (tenant_id, name) DO NOTHING`,
			tenantID, br.Name, br.Description, perms)
		if err != nil {
			return fmt.Errorf("seed %s: %w", br.Name, err)
		}
	}
	return nil
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/enterprise/rbac/roles
// ------------------------------------------------------------------

// CreateRBACRole inserts a custom role for the caller's tenant.
// Rejects:
//   - name collisions with built-in role names (Admin/Operator/Viewer/Billing)
//   - permissions outside the allowlist
//   - duplicate names within the same tenant (UNIQUE constraint)
func CreateRBACRole(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req rbacRoleReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		name := strings.TrimSpace(req.Name)
		if isBuiltinRoleName(name) {
			kernel.RespondErrorWithCode(c, http.StatusConflict, "builtin_role",
				fmt.Sprintf("%q is a built-in role name; pick a different name", name))
			return
		}
		// Validate permissions against allowlist; dedupe + sort
		// so storage is stable across requests.
		perms, errStr := validateAndNormalizePermissions(req.Permissions)
		if errStr != "" {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", errStr)
			return
		}

		var rowID, createdAt string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO rbac_roles (tenant_id, name, description, permissions, is_builtin)
			 VALUES ($1, $2, NULLIF($3, ''), $4, false)
			 RETURNING id::text, created_at::text`,
			tenantID, name, strings.TrimSpace(req.Description), perms,
		).Scan(&rowID, &createdAt)
		if err != nil {
			// UNIQUE (tenant_id, name) violation → 409.
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "role_exists",
					fmt.Sprintf("a role named %q already exists for this tenant", name))
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Custom role creation changes the set of available roles
		// but doesn't directly change anyone's effective permissions,
		// so we DON'T bust the cache here.

		kernel.RespondCreated(c, rbacRoleRow{
			ID:          rowID,
			TenantID:    tenantID.String(),
			Name:        name,
			Description: strings.TrimSpace(req.Description),
			Permissions: perms,
			IsBuiltin:   false,
			CreatedAt:   createdAt,
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: PATCH /api/v1/enterprise/rbac/roles/:id
// ------------------------------------------------------------------

// UpdateRBACRole patches a custom role. 403 if the role is built-in.
// On success, invalidates the effective-permissions cache for every
// user currently assigned to this role so the next /check call sees
// the new permissions.
func UpdateRBACRole(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		roleID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var req rbacRolePatchReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if req.Name == nil && req.Description == nil && req.Permissions == nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"at least one of name, description, permissions must be supplied")
			return
		}

		// Load current row to enforce built-in protection and to
		// capture the affected user list for cache invalidation.
		var (
			currentName, currentDesc string
			currentPerms             []string
			isBuiltin                bool
		)
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT name, COALESCE(description, ''), permissions, is_builtin
			   FROM rbac_roles WHERE tenant_id = $1 AND id = $2`,
			tenantID, roleID,
		).Scan(&currentName, &currentDesc, &currentPerms, &isBuiltin)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				kernel.RespondError(c, kernel.ErrNotFound)
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if isBuiltin {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "builtin_readonly",
				"built-in roles are read-only")
			return
		}

		newName := currentName
		newDesc := currentDesc
		newPerms := currentPerms
		if req.Name != nil {
			candidate := strings.TrimSpace(*req.Name)
			if isBuiltinRoleName(candidate) {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "builtin_role",
					fmt.Sprintf("%q is a built-in role name", candidate))
				return
			}
			newName = candidate
		}
		if req.Description != nil {
			newDesc = strings.TrimSpace(*req.Description)
		}
		if req.Permissions != nil {
			normalized, errStr := validateAndNormalizePermissions(*req.Permissions)
			if errStr != "" {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", errStr)
				return
			}
			newPerms = normalized
		}

		_, err = pool.Pgx().Exec(c.Request.Context(),
			`UPDATE rbac_roles SET name = $1, description = NULLIF($2, ''), permissions = $3
			   WHERE tenant_id = $4 AND id = $5 AND is_builtin = false`,
			newName, newDesc, newPerms, tenantID, roleID)
		if err != nil {
			if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "role_exists",
					fmt.Sprintf("a role named %q already exists for this tenant", newName))
				return
			}
			kernel.RespondError(c, err)
			return
		}

		// Bust the cache for every user with this role so the
		// next /check sees the new permissions. (No-op if there
		// are no current assignees.)
		invalidateCacheForRole(c, pool, tenantID, roleID)

		kernel.RespondOK(c, rbacRoleRow{
			ID:          roleID.String(),
			TenantID:    tenantID.String(),
			Name:        newName,
			Description: newDesc,
			Permissions: newPerms,
			IsBuiltin:   false,
			CreatedAt:   "", // not returned on patch (no SELECT)
		})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: DELETE /api/v1/enterprise/rbac/roles/:id
// ------------------------------------------------------------------

// DeleteRBACRole hard-deletes a custom role. ON DELETE CASCADE on
// user_role_assignments.role_id removes the assignments for free.
// 403 if the role is built-in.
//
// Cache invalidation: bust the cache for every user that had this
// role assigned (we read them before the DELETE).
func DeleteRBACRole(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		roleID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}

		// Capture assignee list before DELETE so we can bust their
		// cache entries.
		invalidateCacheForRole(c, pool, tenantID, roleID)

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`DELETE FROM rbac_roles WHERE tenant_id = $1 AND id = $2 AND is_builtin = false`,
			tenantID, roleID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			// Could be: doesn't exist, OR is built-in. Distinguish
			// so the operator gets a useful error.
			var isBuiltin bool
			err := pool.Pgx().QueryRow(c.Request.Context(),
				`SELECT is_builtin FROM rbac_roles WHERE tenant_id = $1 AND id = $2`,
				tenantID, roleID,
			).Scan(&isBuiltin)
			if err != nil {
				kernel.RespondError(c, kernel.ErrNotFound)
				return
			}
			if isBuiltin {
				kernel.RespondErrorWithCode(c, http.StatusForbidden, "builtin_readonly",
					"built-in roles cannot be deleted")
				return
			}
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondOK(c, gin.H{"deleted": true, "id": roleID.String()})
	}
}
