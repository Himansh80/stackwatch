// Tier 9 Phase 2 — SCIM Provisioning (Tier 9.2).
// HTTP route handlers for the 5 PUBLIC SCIM 2.0 endpoints. Bearer-
// token auth via scimAuthMiddleware (defined in
// handlers_scim_protected.go alongside the 4 protected endpoints).
//
// Public SCIM 2.0 endpoints:
//
//	GET    /scim/v2/Users           — SCIMListUsers
//	POST   /scim/v2/Users           — SCIMCreateUser
//	PUT    /scim/v2/Users/:id       — SCIMUpdateUser
//	DELETE /scim/v2/Users/:id       — SCIMDisableUser
//	POST   /scim/v2/Groups          — SCIMCreateGroup (Phase 2 stub)
//
// We implement the MINIMUM subset of SCIM 2.0 that real IdPs (Okta,
// Azure AD, Google Workspace) exercise for user lifecycle. Group
// support is a stub for Phase 2 and lands fully in a later phase.
//
// Spec reference: RFC 7644 (SCIM Protocol) + RFC 7643 (SCIM Core
// Schema). We don't pull in a SCIM library — the JSON shapes in
// handlers_scim_types.go are hand-rolled and intentionally permissive
// (IdPs each add their own custom fields which we ignore gracefully).
package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Public endpoint: GET /scim/v2/Users
// ------------------------------------------------------------------

// SCIMListUsers lists active + disabled users for the caller's tenant.
// Honors the SCIM 2.0 ListResponse envelope (RFC 7644 §3.4.2).
func SCIMListUsers(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := scimClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if !scimHasScope(cl, "users:read") {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "forbidden", "token missing users:read scope")
			return
		}

		startIndex := 1
		count := 50
		if v := c.Query("startIndex"); v != "" {
			fmt.Sscanf(v, "%d", &startIndex)
			if startIndex < 1 {
				startIndex = 1
			}
		}
		if v := c.Query("count"); v != "" {
			fmt.Sscanf(v, "%d", &count)
			if count < 1 || count > 200 {
				count = 50
			}
		}
		offset := startIndex - 1

		var total int
		if err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT count(*) FROM users WHERE tenant_id = $1`, cl.TenantID,
		).Scan(&total); err != nil {
			kernel.RespondError(c, err)
			return
		}

		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, email, full_name, status
			   FROM users WHERE tenant_id = $1
			  ORDER BY created_at ASC
			  LIMIT $2 OFFSET $3`,
			cl.TenantID, count, offset)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		resources := []scimUserResource{}
		for rows.Next() {
			var id, email, fullName, status string
			if err := rows.Scan(&id, &email, &fullName, &status); err != nil {
				continue
			}
			active := status != "disabled"
			resources = append(resources, scimUserResource{
				Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
				ID:       id,
				UserName: email,
				Name:     &scimName{Formatted: fullName},
				Emails:   []scimEmail{{Value: email, Type: "work", Primary: true}},
				Active:   &active,
			})
		}

		logSCIMSync(c, pool, cl.TenantID, "list", "", "User", "success", "")

		c.JSON(http.StatusOK, scimListResponse{
			Schemas:      []string{"urn:ietf:params:scim:schemas:core:2.0:ListResponse"},
			TotalResults: total,
			ItemsPerPage: count,
			StartIndex:   startIndex,
			Resources:    resources,
		})
	}
}

// ------------------------------------------------------------------
// Public endpoint: POST /scim/v2/Users
// ------------------------------------------------------------------

// SCIMCreateUser provisions a new user. Requires `users:write` scope.
//
// password_hash cannot be NULL on the users table — for SCIM-
// provisioned users we seed an UNUSABLE bcrypt hash (a fresh random
// string) so they can't log in via password until a human sets one
// (typically via the SSO JIT path, or a "set initial password" email
// sent later). This matches the spec scenario 2 — the IdP creates
// the user but doesn't have to manage a password.
func SCIMCreateUser(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := scimClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if !scimHasScope(cl, "users:write") {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "forbidden", "token missing users:write scope")
			return
		}
		var req scimUserResource
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		email := deriveSCIMEmail(&req)
		fullName := deriveSCIMFullName(&req)
		if email == "" {
			logSCIMSync(c, pool, cl.TenantID, "create", req.ExternalID, "User", "error",
				"missing email/userName")
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", "userName/email required")
			return
		}
		active := true
		if req.Active != nil {
			active = *req.Active
		}
		status := "active"
		if !active {
			status = "disabled"
		}
		placeholderHash, _ := bcrypt.GenerateFromPassword(
			[]byte(time.Now().UTC().Format(time.RFC3339Nano)+":"+email),
			bcrypt.MinCost,
		)
		var userID string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO users (tenant_id, email, full_name, password_hash, role, status, email_verified)
			 VALUES ($1, $2, $3, $4, 'viewer', $5, true)
			 RETURNING id::text`,
			cl.TenantID, email, fullName, string(placeholderHash), status,
		).Scan(&userID)
		if err != nil {
			logSCIMSync(c, pool, cl.TenantID, "create", req.ExternalID, "User", "error", err.Error())
			if isUniqueViolation(err) {
				kernel.RespondError(c, kernel.ErrConflict)
				return
			}
			kernel.RespondError(c, err)
			return
		}
		logSCIMSync(c, pool, cl.TenantID, "create", req.ExternalID, "User", "success", "")
		active = status == "active"
		c.JSON(http.StatusCreated, scimUserResource{
			Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			ID:       userID,
			UserName: email,
			Name:     &scimName{Formatted: fullName},
			Emails:   []scimEmail{{Value: email, Type: "work", Primary: true}},
			Active:   &active,
		})
	}
}

// ------------------------------------------------------------------
// Public endpoint: PUT /scim/v2/Users/:id
// ------------------------------------------------------------------

// SCIMUpdateUser updates an existing user (email, full_name, active).
// Requires `users:write` scope.
func SCIMUpdateUser(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := scimClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if !scimHasScope(cl, "users:write") {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "forbidden", "token missing users:write scope")
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var req scimUserResource
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		email := deriveSCIMEmail(&req)
		fullName := deriveSCIMFullName(&req)
		status := "active"
		if req.Active != nil && !*req.Active {
			status = "disabled"
		}
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE users
			    SET email = $1, full_name = $2, status = $3, updated_at = NOW()
			  WHERE id = $4 AND tenant_id = $5`,
			email, fullName, status, id, cl.TenantID)
		if err != nil {
			logSCIMSync(c, pool, cl.TenantID, "update", req.ExternalID, "User", "error", err.Error())
			if isUniqueViolation(err) {
				kernel.RespondError(c, kernel.ErrConflict)
				return
			}
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			logSCIMSync(c, pool, cl.TenantID, "update", req.ExternalID, "User", "error", "user not found")
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		logSCIMSync(c, pool, cl.TenantID, "update", req.ExternalID, "User", "success", "")
		active := status == "active"
		c.JSON(http.StatusOK, scimUserResource{
			Schemas:  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
			ID:       id.String(),
			UserName: email,
			Name:     &scimName{Formatted: fullName},
			Emails:   []scimEmail{{Value: email, Type: "work", Primary: true}},
			Active:   &active,
		})
	}
}

// ------------------------------------------------------------------
// Public endpoint: DELETE /scim/v2/Users/:id
// ------------------------------------------------------------------

// SCIMDisableUser implements spec §"Story 2 — SCIM Provisioning"
// scenario 4 — this is a SOFT delete. We set status='disabled' (NOT a
// row DELETE) so the audit trail (login history, alert
// acknowledgements, etc.) is preserved.
func SCIMDisableUser(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := scimClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if !scimHasScope(cl, "users:write") {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "forbidden", "token missing users:write scope")
			return
		}
		id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE users SET status = 'disabled', updated_at = NOW()
			  WHERE id = $1 AND tenant_id = $2`,
			id, cl.TenantID)
		if err != nil {
			logSCIMSync(c, pool, cl.TenantID, "delete", "", "User", "error", err.Error())
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			logSCIMSync(c, pool, cl.TenantID, "delete", "", "User", "error", "user not found")
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		logSCIMSync(c, pool, cl.TenantID, "delete", "", "User", "success", "")
		// SCIM 2.0 says DELETE should return 204 No Content.
		c.Status(http.StatusNoContent)
	}
}

// ------------------------------------------------------------------
// Public endpoint: POST /scim/v2/Groups
// ------------------------------------------------------------------

// SCIMCreateGroup is a Phase 2 stub. We accept the request, log the
// call to scim_sync_log so the IdP doesn't give up retrying, and
// return a 202 Accepted with a placeholder Group resource. A future
// phase will persist groups to a `groups` table and link them to
// user_role_assignments; for now this satisfies the IdP handshake.
func SCIMCreateGroup(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := scimClaimsFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		if !scimHasScope(cl, "groups:write") {
			kernel.RespondErrorWithCode(c, http.StatusForbidden, "forbidden", "token missing groups:write scope")
			return
		}
		// Parse minimally — we only care about displayName for the log.
		var req struct {
			DisplayName string `json:"displayName"`
		}
		_ = c.ShouldBindJSON(&req) // tolerate empty bodies
		logSCIMSync(c, pool, cl.TenantID, "create", "", "Group", "success",
			"groups: phase 2 stub — full support in a later phase")
		c.JSON(http.StatusAccepted, gin.H{
			"schemas":     []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
			"id":          uuid.New().String(),
			"displayName": req.DisplayName,
			"members":     []any{},
			"_stub":       true,
		})
	}
}
