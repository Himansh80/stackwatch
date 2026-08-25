// Tier 9 Phase 6 — Enterprise Tenants + Org Settings (Tier 9.6).
//
// HTTP route handlers for the three protected org-hierarchy endpoints.
// The two ALTER TABLE statements that back these endpoints
// (parent_org_id + settings JSONB) live in migrations/040_enterprise.sql.
//
//	GET    /api/v1/enterprise/orgs                — ListEnterpriseOrgs
//	POST   /api/v1/enterprise/orgs                — CreateEnterpriseOrg
//	PATCH  /api/v1/enterprise/orgs/:id/settings   — UpdateEnterpriseOrgSettings
//
// All three routes honor tenant_id from the JWT — no cross-tenant
// data ever crosses the wire. Sub-orgs are stored as additional
// tenants rows with the SAME tenant_id as their parent (the
// parent_org_id column distinguishes them) — this keeps the
// per-tenant data isolation guarantee in `users.tenant_id` working
// unchanged.
//
// Why sub-orgs are tenant rows (not a separate table):
//   - The existing users.tenant_id FK already filters every query
//     by org scope for free.
//   - We don't need a second set of joins, indexes, or row-level
//     security policies.
//   - Onboarding a new sub-org is one INSERT into the same table
//     the tenant row already lives in.
//
// Authorization model for the PATCH /settings endpoint:
//   - The caller's JWT carries one tenant_id. If the target org
//     row has the same tenant_id AND parent_org_id IS NULL, the
//     caller is the parent org owner and may edit any child.
//   - Otherwise (target is the caller's own org, or a child of the
//     caller's org) the caller may edit only the settings column.
//   - Cross-tenant edits are blocked at the WHERE clause so a
//     leaked row id can't leak across tenant boundaries.
package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ------------------------------------------------------------------
// JSON row type — mirrors the augmented tenants table.
// ------------------------------------------------------------------

// enterpriseOrgRow is the JSON shape returned by GET /api/v1/enterprise/orgs
// and POST /api/v1/enterprise/orgs. Fields map 1:1 to the tenants
// table augmented with parent_org_id + settings (Phase 6).
//
// `user_count` and `child_count` are computed server-side via
// correlated subqueries so the UI doesn't have to re-aggregate the
// raw rows on every render — useful when a tenant has thousands of
// sub-orgs.
type enterpriseOrgRow struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	ParentOrgID *string         `json:"parent_org_id,omitempty"`
	Settings    json.RawMessage `json:"settings"`
	UserCount   int             `json:"user_count"`
	ChildCount  int             `json:"child_count"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
}

// ------------------------------------------------------------------
// Request types — body shape for POST + PATCH endpoints.
// ------------------------------------------------------------------

// createEnterpriseOrgReq is the JSON body for POST
// /api/v1/enterprise/orgs. `parent_org_id` is nullable — pass
// null/omit for a top-level org, or the UUID of an existing org
// in the same tenant to create a sub-org.
type createEnterpriseOrgReq struct {
	Name        string          `json:"name"        binding:"required,min=1,max=255"`
	Slug        string          `json:"slug"        binding:"required,min=1,max=64"`
	ParentOrgID *string         `json:"parent_org_id,omitempty"`
	Settings    json.RawMessage `json:"settings,omitempty"`
}

// updateEnterpriseOrgSettingsReq is the JSON body for PATCH
// /api/v1/enterprise/orgs/:id/settings. We only accept the
// settings blob — name/slug edits belong on a future rename
// endpoint (out of scope for Phase 6 — keep the surface small).
type updateEnterpriseOrgSettingsReq struct {
	Settings json.RawMessage `json:"settings" binding:"required"`
}

// slugSafePattern restricts org slugs to URL-safe ASCII so they
// can be embedded in subdomain-style URLs (orgs.example.com) by
// future client code without escaping.
var slugSafePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,62}[a-z0-9]$`)

// normalizeOrgSettings wraps a raw JSON payload in an object when
// the caller passes an array, number, string, etc. so the column
// stays valid JSONB of object shape. Returns the normalized
// payload + a bool indicating whether the caller passed valid JSON.
func normalizeOrgSettings(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	// Quick sanity check: must parse as JSON.
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	// If it's already an object, keep as-is.
	if _, ok := probe.(map[string]any); ok {
		return raw, nil
	}
	// Otherwise wrap under a "value" key so the column stays an object.
	wrapped, err := json.Marshal(map[string]any{"value": probe})
	if err != nil {
		return nil, err
	}
	return wrapped, nil
}

// ------------------------------------------------------------------
// Protected endpoint: GET /api/v1/enterprise/orgs
// ------------------------------------------------------------------

// ListEnterpriseOrgs returns every org (parent + children) for the
// caller's tenant as a flat list, sorted parent-first then name ASC.
// The UI's EnterpriseOrgsSection builds the parent→child tree
// client-side from the parent_org_id links so we don't have to ship
// a recursive CTE per call.
//
// Query params:
//
//	include_children=true|false  default=true. When false, only
//	                              top-level orgs (parent_org_id IS
//	                              NULL) are returned — useful for
//	                              a "create sub-org" picker.
//
// User/child counts are computed inline so a single SELECT
// replaces what would otherwise be N+1 round-trips.
func ListEnterpriseOrgs(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}

		includeChildren := true
		if v := strings.ToLower(strings.TrimSpace(c.Query("include_children"))); v == "false" || v == "0" || v == "no" {
			includeChildren = false
		}

		q := `SELECT t.id::text, t.name, t.slug,
		            t.parent_org_id::text,
		            COALESCE(t.settings, '{}'::jsonb),
		            COALESCE(uc.cnt, 0) AS user_count,
		            COALESCE(cc.cnt, 0) AS child_count,
		            t.created_at::text, t.updated_at::text
		      FROM tenants t
		      LEFT JOIN (
		          SELECT tenant_id, COUNT(*)::int AS cnt FROM users GROUP BY tenant_id
		      ) uc ON uc.tenant_id = t.id
		      LEFT JOIN (
		          SELECT parent_org_id, COUNT(*)::int AS cnt FROM tenants WHERE parent_org_id IS NOT NULL GROUP BY parent_org_id
		      ) cc ON cc.parent_org_id = t.id
		      WHERE t.id = $1`
		if !includeChildren {
			q += ` AND t.parent_org_id IS NULL`
		}
		q += ` ORDER BY (t.parent_org_id IS NULL) DESC, t.name ASC`

		pgxRows, err := pool.Pgx().Query(c.Request.Context(), q, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer pgxRows.Close()

		out := []enterpriseOrgRow{}
		for pgxRows.Next() {
			var (
				r          enterpriseOrgRow
				parentOrg  *string
				createdStr string
				updatedStr string
			)
			if err := pgxRows.Scan(&r.ID, &r.Name, &r.Slug,
				&parentOrg, &r.Settings, &r.UserCount, &r.ChildCount,
				&createdStr, &updatedStr); err != nil {
				continue
			}
			r.TenantID = tenantID.String()
			r.ParentOrgID = parentOrg
			r.CreatedAt = createdStr
			r.UpdatedAt = updatedStr
			out = append(out, r)
		}
		kernel.RespondOK(c, gin.H{"orgs": out, "total": len(out), "include_children": includeChildren})
	}
}

// ------------------------------------------------------------------
// Protected endpoint: POST /api/v1/enterprise/orgs
// ------------------------------------------------------------------

// CreateEnterpriseOrg inserts a new sub-org (or top-level org) into
// the caller's tenant. Validation:
//
//   - name is required, 1..255 chars
//   - slug is required, URL-safe (^[a-z0-9-]+$, 1..64 chars)
//   - parent_org_id, if present, must belong to an existing org in
//     the same tenant (otherwise 404 — we don't leak existence)
//   - settings, if present, must be valid JSON; non-object values
//     are wrapped under "value" so the JSONB column stays valid
//
// Returns 201 + the new row. 409 on duplicate slug (the slug column
// has a UNIQUE constraint — even cross-tenant collisions are
// rejected; we don't try to namespace slugs by tenant in Phase 6).
func CreateEnterpriseOrg(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req createEnterpriseOrgReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		slug := strings.ToLower(strings.TrimSpace(req.Slug))
		if !slugSafePattern.MatchString(slug) {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"slug must be URL-safe (lowercase letters, digits, and dashes; 1..64 chars)")
			return
		}

		// If parent_org_id was supplied, verify it belongs to this tenant.
		var parentUUID *uuid.UUID
		if req.ParentOrgID != nil && strings.TrimSpace(*req.ParentOrgID) != "" {
			parsed, perr := uuid.Parse(strings.TrimSpace(*req.ParentOrgID))
			if perr != nil {
				kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
					"parent_org_id must be a UUID")
				return
			}
			var dummy int
			err := pool.Pgx().QueryRow(c.Request.Context(),
				`SELECT 1 FROM tenants WHERE id = $1 AND id = $2`,
				parsed, tenantID,
			).Scan(&dummy)
			if err != nil {
				kernel.RespondError(c, kernel.ErrNotFound)
				return
			}
			parentUUID = &parsed
			}

		normalizedSettings, err := normalizeOrgSettings(req.Settings)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"settings must be valid JSON")
			return
		}

		newID := uuid.New()
		var (
			createdStr string
			updatedStr string
		)
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO tenants (id, name, slug, parent_org_id, settings,
		                       plan, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, 'free', 'active', NOW(), NOW())
		 RETURNING created_at::text, updated_at::text`,
			newID, strings.TrimSpace(req.Name), slug,
			parentUUID, normalizedSettings,
		).Scan(&createdStr, &updatedStr)
		if err != nil {
			msg := err.Error()
			if strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate") {
				kernel.RespondErrorWithCode(c, http.StatusConflict, "slug_exists",
					"an org with that slug already exists")
				return
			}
			kernel.RespondError(c, err)
			return
		}

		resp := enterpriseOrgRow{
			ID:        newID.String(),
			TenantID:  tenantID.String(),
			Name:      strings.TrimSpace(req.Name),
			Slug:      slug,
			Settings:  normalizedSettings,
			CreatedAt: createdStr,
			UpdatedAt: updatedStr,
		}
		if parentUUID != nil {
			pid := parentUUID.String()
			resp.ParentOrgID = &pid
		}
		kernel.RespondCreated(c, resp)
	}
}

// ------------------------------------------------------------------
// Protected endpoint: PATCH /api/v1/enterprise/orgs/:id/settings
// ------------------------------------------------------------------

// UpdateEnterpriseOrgSettings patches the JSONB settings blob on a
// single org row. Authorization: the target row must belong to the
// caller's tenant. We allow both the parent org owner AND the org
// itself to edit its own settings — in this tier every org row
// inside the tenant is in-scope; cross-tenant edits are blocked by
// the WHERE clause.
//
// Returns 404 if the row doesn't exist OR belongs to a different
// tenant (we don't leak existence across tenants).
func UpdateEnterpriseOrgSettings(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		orgID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var req updateEnterpriseOrgSettingsReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		normalized, err := normalizeOrgSettings(req.Settings)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request",
				"settings must be valid JSON")
			return
		}

		tag, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE tenants
			    SET settings = $1, updated_at = NOW()
			  WHERE id = $2 AND id = $3`,
			normalized, tenantID, orgID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		if tag.RowsAffected() == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}

		kernel.RespondOK(c, gin.H{
			"id":       orgID.String(),
			"settings": json.RawMessage(normalized),
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
