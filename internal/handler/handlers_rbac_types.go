// Tier 9 Phase 3 — Advanced RBAC (Tier 9.3).
// Shared types + permission allowlist + built-in-role seed definition
// for the 6 protected RBAC endpoints:
//
//	GET    /api/v1/enterprise/rbac/roles                       — ListRBACRoles
//	POST   /api/v1/enterprise/rbac/roles                       — CreateRBACRole
//	PATCH  /api/v1/enterprise/rbac/roles/:id                   — UpdateRBACRole
//	DELETE /api/v1/enterprise/rbac/roles/:id                   — DeleteRBACRole
//	POST   /api/v1/enterprise/rbac/users/:user_id/roles        — AssignRBACRole
//	DELETE /api/v1/enterprise/rbac/users/:user_id/roles/:role_id — UnassignRBACRole
//	GET    /api/v1/enterprise/rbac/check                       — CheckRBACPermission
//
// The HTTP handlers themselves live in handlers_rbac.go; this file
// only carries the JSON row shapes + permission allowlist + the
// definition of the four built-in roles that every tenant gets.
//
// Built-in roles are LAZY-SEEDED (NOT inserted by the migration).
// handlers_rbac.go::seedBuiltinRolesForTenant inserts them on the
// first GET /rbac/roles per tenant — this avoids a slow multi-tenant
// up-front seed when there are many tenants.
//
// Permission allowlist (`builtinPermissions`):
//   Every role's `permissions` array is a slice of strings in the
//   format "<resource>:<verb>" (e.g. "servers:read", "alerts:write").
//   Handlers refuse any string not on this allowlist so a hand-crafted
//   payload can't smuggle in junk like "admin:*" or "system:root".
//
// Effective-permissions cache (`effectivePermCache`):
//   The check endpoint is on the hot path (every page load, every
//   gated API call). To avoid hammering the DB, we cache the union
//   of permissions per (tenant_id, user_id) for 60 seconds. The cache
//   is invalidated when any role definition changes (PATCH role) or
//   when an assignment is added/removed for the affected user.
package handler

import (
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ------------------------------------------------------------------
// Permission allowlist — the SOLE source of truth for what
// "<resource>:<verb>" strings are accepted at the handler layer.
//
// When adding a new permission here, also bump the version comment
// in handlers_rbac.go::seedBuiltinRolesForTenant so existing tenants
// pick up the new permission on next lazy-seed.
// ------------------------------------------------------------------

// builtinPermissions is the canonical whitelist. Order matters for
// stable UI rendering (RbacSection groups them by resource prefix).
var builtinPermissions = []string{
	// Servers (Tier 1-3 inventory + metrics)
	"servers:read",
	"servers:write",
	// Dashboards (Tier 4)
	"dashboards:read",
	"dashboards:write",
	// Alerts (Tier 5)
	"alerts:read",
	"alerts:write",
	// Incidents (Tier 7)
	"incidents:read",
	"incidents:write",
	// SSO (Tier 9.1)
	"sso:read",
	"sso:write",
	// SCIM (Tier 9.2)
	"scim:read",
	"scim:write",
	// RBAC (Tier 9.3 — self)
	"rbac:read",
	"rbac:write",
	// Audit (Tier 9.4 — reserved for future phase)
	"audit:read",
	"audit:write",
	// Compliance (Tier 9.5 — reserved for future phase)
	"compliance:read",
	"compliance:write",
	// Org / tenant settings (Tier 9.6 — reserved for future phase)
	"org:read",
	"org:write",
}

// permissionAllowedSet is the O(1) lookup form of builtinPermissions
// rebuilt at init by init() in handlers_rbac.go.
var permissionAllowedSet = map[string]bool{}

// init is called by the runtime after all package-level vars are set;
// we build the lookup map exactly once so handlers can validate with
// `if !permissionAllowedSet[p] { return 400 }`.
func init() {
	for _, p := range builtinPermissions {
		permissionAllowedSet[p] = true
	}
}

// isAllowedPermission reports whether the given string is on the
// allowlist. Empty string returns false (we never want to persist
// blank permissions — they're meaningless and almost certainly a bug).
func isAllowedPermission(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	return permissionAllowedSet[p]
}

// ------------------------------------------------------------------
// Built-in roles — the four roles every tenant gets on first list.
// Lazy-seeded by handlers_rbac.go::seedBuiltinRolesForTenant.
//
// Permission sets are designed to mirror Datadog / Okta conventions:
//   Admin     — full power, every permission on the allowlist.
//   Operator  — read+write for servers/alerts/incidents/dashboards;
//               NOT rbac/audit/compliance/org/sso/scim (those are
//               "control plane" perms reserved for Admin).
//   Viewer    — read-only across the operational surface.
//   Billing   — just org:read + org:write + dashboards:read (read
//               invoices, change billing settings, view dashboards).
//
// Adding a new built-in role here requires adding it to the seed
// function AND writing the per-tenant name uniqueness so the
// migration's UNIQUE (tenant_id, name) constraint doesn't bite.
// ------------------------------------------------------------------

// builtinRolesSeed is the canonical list of roles lazy-seeded for
// each tenant on first GET /rbac/roles. Order is UI-stable
// (Admin first, Viewer last) so the Built-in tab is always the
// same order across renders.
var builtinRolesSeed = []struct {
	Name        string
	Description string
	Permissions []string
	// allPermissions is true → this role gets every permission on
	// the allowlist at INSERT time. Lets us keep Admin's
	// definition data-driven so future permissions automatically
	// flow in.
	AllPermissions bool
}{
	{
		Name:           "Admin",
		Description:    "Full control of all platform features — assign roles, configure SSO/SCIM, view audit logs, manage billing.",
		AllPermissions: true,
	},
	{
		Name:        "Operator",
		Description: "Day-to-day ops: manage servers, alerts, incidents, and dashboards. No access to RBAC, audit, or billing.",
		Permissions: []string{
			"servers:read", "servers:write",
			"dashboards:read", "dashboards:write",
			"alerts:read", "alerts:write",
			"incidents:read", "incidents:write",
		},
	},
	{
		Name:        "Viewer",
		Description: "Read-only access to servers, alerts, incidents, and dashboards.",
		Permissions: []string{
			"servers:read",
			"dashboards:read",
			"alerts:read",
			"incidents:read",
		},
	},
	{
		Name:        "Billing",
		Description: "Manage billing and organization settings; view dashboards.",
		Permissions: []string{
			"org:read", "org:write",
			"dashboards:read",
		},
	},
}

// builtinRoleNames returns the set of names occupied by built-in
// roles. Used by CreateRBACRole to reject collisions.
var builtinRoleNames = func() map[string]bool {
	m := map[string]bool{}
	for _, r := range builtinRolesSeed {
		m[r.Name] = true
	}
	return m
}()

// isBuiltinRoleName reports whether `name` matches a built-in role
// (case-sensitive — built-in names are Admin / Operator / Viewer /
// Billing with title case).
func isBuiltinRoleName(name string) bool {
	return builtinRoleNames[strings.TrimSpace(name)]
}

// ------------------------------------------------------------------
// JSON row types — mirror the column shape and map directly to the
// `out` slice returned by each List* handler.
// ------------------------------------------------------------------

// rbacRoleRow is the JSON shape for a single RBAC role returned by
// GET /rbac/roles + POST /rbac/roles + PATCH /rbac/roles/:id.
// `description` is omitempty so built-in roles (whose description
// IS populated but is the canonical copy from builtinRolesSeed)
// can still omit it if a future migration nulls it out without
// breaking clients.
type rbacRoleRow struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
	IsBuiltin   bool     `json:"is_builtin"`
	CreatedAt   string   `json:"created_at"`
}

// userRoleAssignmentRow is the JSON shape for a single user↔role
// assignment returned by POST /rbac/users/:user_id/roles +
// DELETE /rbac/users/:user_id/roles/:role_id.
type userRoleAssignmentRow struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenant_id"`
	UserID     string `json:"user_id"`
	RoleID     string `json:"role_id"`
	RoleName   string `json:"role_name,omitempty"`
	AssignedAt string `json:"assigned_at"`
}

// permissionCheckResult is the JSON shape for the response of
// GET /rbac/check?permission=X&user_id=Y. `matching_roles` lists
// every role_id that contributed a positive vote for the requested
// permission so the UI can show "you can do X via the Admin and
// Viewer roles" — useful for debugging over-privileged users.
type permissionCheckResult struct {
	Allowed       bool     `json:"allowed"`
	UserID        string   `json:"user_id"`
	Permission    string   `json:"permission"`
	MatchingRoles []string `json:"matching_roles"`
	// AllPermissions is the full effective permission set for the
	// user (union of all assigned roles' permissions, deduplicated).
	// Helps the UI show "you have N permissions" in a debug panel.
	AllPermissions []string `json:"all_permissions"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST/PATCH endpoints.
// ------------------------------------------------------------------

// rbacRoleReq is the JSON body for POST /rbac/roles. `name` is
// required (operator-chosen display name); `description` and
// `permissions` are optional. The handler rejects `name` collisions
// with built-in role names and validates every entry in `permissions`
// against the allowlist.
type rbacRoleReq struct {
	Name        string   `json:"name"        binding:"required,min=1,max=64"`
	Description string   `json:"description" binding:"max=256"`
	Permissions []string `json:"permissions" binding:"omitempty,dive,min=1,max=64"`
}

// rbacRolePatchReq is the JSON body for PATCH /rbac/roles/:id.
// All fields are optional — only supplied ones are applied. At least
// one must be non-nil (the handler enforces this with a 400).
// Built-in roles (is_builtin=true) cannot be patched — handler returns 403.
type rbacRolePatchReq struct {
	Name        *string   `json:"name"        binding:"omitempty,min=1,max=64"`
	Description *string   `json:"description" binding:"omitempty,max=256"`
	Permissions *[]string `json:"permissions" binding:"omitempty"`
}

// rbacAssignReq is the JSON body for POST /rbac/users/:user_id/roles.
// `role_id` is the UUID of the role to assign. Idempotent: if the
// user already has the role, returns 200 with the existing row;
// returns 201 only on a fresh INSERT.
type rbacAssignReq struct {
	RoleID string `json:"role_id" binding:"required,uuid"`
}

// ------------------------------------------------------------------
// Effective-permissions cache.
//
// Per (tenant_id, user_id) we cache the union of permissions from
// every assigned role, plus the set of role_ids that contributed
// each permission (for matching_roles in the response).
//
// TTL is 60s — short enough that role changes propagate within a
// minute, long enough that a page-load burst of check calls hits
// the cache after the first one.
//
// Invalidation points (handlers_rbac.go):
//   - UpdateRBACRole  → invalidate every user with that role assigned
//   - DeleteRBACRole  → invalidate every user with that role assigned
//   - AssignRBACRole  → invalidate that user
//   - UnassignRBACRole→ invalidate that user
//
// No global mutex — the cache is a sync.Map keyed by
// (tenantID, userID) string. The inner entry is itself a
// pointer to a struct so reads are racy-safe (Go's memory model
// guarantees pointer reads are atomic). We accept the rare
// 60-second-stale-read on bursty role churn; that's bounded by
// the TTL.
// ------------------------------------------------------------------

// effectivePermCacheEntry is one row of the in-memory cache.
type effectivePermCacheEntry struct {
	// AllPermissions is the union of permissions from all assigned
	// roles, deduplicated + sorted (handlers_rbac.go owns the
	// canonical sort order).
	AllPermissions []string
	// PermissionToRoles maps each permission to the set of role
	// IDs that grant it — used to populate `matching_roles` in
	// the check response without a second DB hit.
	PermissionToRoles map[string][]string
	// ExpiresAt is when this entry becomes invalid (time.Now()
	// > ExpiresAt → treat as miss + recompute).
	ExpiresAt time.Time
}

var effectivePermCache sync.Map

// effectivePermCacheKey produces the canonical cache key. Stable
// across processes (UUID string + tenantID string → deterministic).
func effectivePermCacheKey(tenantID, userID uuid.UUID) string {
	return tenantID.String() + "|" + userID.String()
}

// effectivePermCacheTTL is the lifetime of a cached entry.
// Exported as a var (not const) so tests can shorten it.
var effectivePermCacheTTL = 60 * time.Second
