// Tier 9 Phase 3 — Advanced RBAC (Tier 9.3) — Check endpoint +
// cache + shared helpers.
//
// HTTP route handler:
//
//	GET /api/v1/enterprise/rbac/check?permission=X&user_id=Y — CheckRBACPermission
//
// Plus the cross-handler helpers used by the role CRUD + assignment
// files:
//
//	computeEffectivePerms     — DB → cache entry
//	invalidateCacheForRole    — bust cache for all of a role's assignees
//	validateAndNormalizePermissions — trim/dedupe/sort/allowlist-check
//	sliceContainsString       — tiny "in-slice" helper (cache hit path)
//	appendUnique              — preserve-order append-if-absent
//
// The 60-second TTL (effectivePermCacheTTL) lives in
// handlers_rbac_types.go alongside the cache struct.
package handler

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/rbac/check
// ------------------------------------------------------------------

// CheckRBACPermission reports whether the given user has the given
// permission, considering the union of every role they're assigned.
// Reads go through the in-memory effectivePermCache with a 60s TTL.
//
// Query params:
//
//	permission  required — the "<resource>:<verb>" string to check
//	user_id     required — the user to check (UUID)
//	refresh     optional — "true" to bypass the cache
//
// Response (permissionCheckResult):
//
//	allowed          — bool, the answer
//	user_id          — echoed
//	permission       — echoed
//	matching_roles   — list of role_ids that grant this permission
//	all_permissions  — sorted + deduplicated union of the user's perms
//
// If `permission` is not on the allowlist, returns allowed=false +
// empty matching_roles (the unknown-permission case is treated as
// "no one has this permission", which is the correct deny-by-default
// posture).
func CheckRBACPermission(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		permission := strings.TrimSpace(c.Query("permission"))
		if permission == "" {
			kernel.RespondErrorWithCode(c, 400, "bad_request",
				"permission query param is required")
			return
		}
		userID, err := uuid.Parse(strings.TrimSpace(c.Query("user_id")))
		if err != nil {
			kernel.RespondErrorWithCode(c, 400, "bad_request",
				"user_id query param must be a UUID")
			return
		}
		refresh := strings.EqualFold(strings.TrimSpace(c.Query("refresh")), "true")

		// Cache lookup (unless refresh=true).
		key := effectivePermCacheKey(tenantID, userID)
		if !refresh {
			if v, ok := effectivePermCache.Load(key); ok {
				entry := v.(*effectivePermCacheEntry)
				if time.Now().Before(entry.ExpiresAt) {
					kernel.RespondOK(c, permissionCheckResult{
						Allowed:        sliceContainsString(entry.AllPermissions, permission),
						UserID:         userID.String(),
						Permission:     permission,
						MatchingRoles:  append([]string(nil), entry.PermissionToRoles[permission]...),
						AllPermissions: append([]string(nil), entry.AllPermissions...),
					})
					return
				}
			}
		}

		// Cache miss → query DB.
		entry, err := computeEffectivePerms(c, pool, tenantID, userID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		effectivePermCache.Store(key, entry)

		kernel.RespondOK(c, permissionCheckResult{
			Allowed:        sliceContainsString(entry.AllPermissions, permission),
			UserID:         userID.String(),
			Permission:     permission,
			MatchingRoles:  append([]string(nil), entry.PermissionToRoles[permission]...),
			AllPermissions: append([]string(nil), entry.AllPermissions...),
		})
	}
}

// ------------------------------------------------------------------
// Helpers (exported within package — used by handlers_rbac_roles.go
// and handlers_rbac_assignments.go)
// ------------------------------------------------------------------

// computeEffectivePerms queries user_role_assignments joined with
// rbac_roles for the given user, unions + deduplicates the
// permissions, and groups them by contributing role_id so the
// check endpoint can populate matching_roles without a second hit.
//
// The single SQL pass keeps the operation O(1) round-trips even
// when the user has many roles.
func computeEffectivePerms(c *gin.Context, pool *db.Pool, tenantID, userID uuid.UUID) (*effectivePermCacheEntry, error) {
	ctx := c.Request.Context()
	rows, err := pool.Pgx().Query(ctx,
		`SELECT r.id::text, r.permissions
		   FROM user_role_assignments a
		   JOIN rbac_roles r ON r.id = a.role_id
		  WHERE a.tenant_id = $1 AND a.user_id = $2`,
		tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("query effective perms: %w", err)
	}
	defer rows.Close()

	permSet := map[string]struct{}{}
	permToRoles := map[string][]string{}
	for rows.Next() {
		var roleID string
		var perms []string
		if err := rows.Scan(&roleID, &perms); err != nil {
			continue
		}
		for _, p := range perms {
			permSet[p] = struct{}{}
			// Append (avoid duplicate role_id entries in
			// matching_roles when the same role grants the same
			// permission through... well, it can't, but defensive).
			permToRoles[p] = appendUnique(permToRoles[p], roleID)
		}
	}
	// Stable sort for deterministic JSON output.
	all := make([]string, 0, len(permSet))
	for p := range permSet {
		all = append(all, p)
	}
	sort.Strings(all)
	return &effectivePermCacheEntry{
		AllPermissions:    all,
		PermissionToRoles: permToRoles,
		ExpiresAt:         time.Now().Add(effectivePermCacheTTL),
	}, nil
}

// validateAndNormalizePermissions trims, dedupes, sorts, and
// allowlist-checks a slice of permission strings. Returns the
// cleaned slice (always non-nil — empty input → empty slice) and
// an error message string (empty on success).
func validateAndNormalizePermissions(in []string) ([]string, string) {
	if len(in) == 0 {
		return []string{}, ""
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, raw := range in {
		p := strings.TrimSpace(raw)
		if !isAllowedPermission(p) {
			return nil, fmt.Sprintf("unknown or empty permission: %q", raw)
		}
		if _, dup := seen[p]; dup {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out, ""
}

// invalidateCacheForRole deletes the cache entry for every user
// currently assigned to the given role. Called from UpdateRBACRole
// + DeleteRBACRole. Tolerates the case where the role has no
// current assignees (the SELECT simply returns 0 rows).
func invalidateCacheForRole(c *gin.Context, pool *db.Pool, tenantID, roleID uuid.UUID) {
	rows, err := pool.Pgx().Query(c.Request.Context(),
		`SELECT user_id::text FROM user_role_assignments WHERE tenant_id = $1 AND role_id = $2`,
		tenantID, roleID)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var uidStr string
		if err := rows.Scan(&uidStr); err != nil {
			continue
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			continue
		}
		effectivePermCache.Delete(effectivePermCacheKey(tenantID, uid))
	}
}

// sliceContainsString reports whether `needle` appears in `haystack`.
// Tiny helper used by the check endpoint's cache-hit path.
// (Named with the `slice` prefix to avoid colliding with the
// dashboards.go containsString(string, string) helper.)
func sliceContainsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// appendUnique appends v to slice if not already present (preserves
// existing order — used for matching_roles which the UI may render
// in declaration order).
func appendUnique(slice []string, v string) []string {
	for _, s := range slice {
		if s == v {
			return slice
		}
	}
	return append(slice, v)
}
