package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// Tenant is the REST shape of a row in tenants.
type Tenant struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Plan      string `json:"plan"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ListTenants returns tenants the caller can see.
//
// In the simplest model: a caller always sees their own tenant. Super admins
// can list ALL tenants by passing ?all=true (deferred to role check).
func ListTenants(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id, name, slug, plan, status, created_at, updated_at
			 FROM tenants WHERE id = $1 ORDER BY created_at DESC`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []Tenant{}
		for rows.Next() {
			var t Tenant
			var created, updated time.Time
			if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &created, &updated); err != nil {
				kernel.RespondError(c, err)
				return
			}
			t.CreatedAt = created.UTC().Format(time.RFC3339)
			t.UpdatedAt = updated.UTC().Format(time.RFC3339)
			out = append(out, t)
		}
		kernel.RespondOK(c, gin.H{"tenants": out, "total": len(out)})
	}
}

// GetTenantMe returns the caller's own tenant.
func GetTenantMe(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var t Tenant
		var created, updated time.Time
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id, name, slug, plan, status, created_at, updated_at
			 FROM tenants WHERE id = $1`, tenantID,
		).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &created, &updated)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		t.CreatedAt = created.UTC().Format(time.RFC3339)
		t.UpdatedAt = updated.UTC().Format(time.RFC3339)
		kernel.RespondOK(c, gin.H{"tenant": t})
	}
}

// GetTenant returns one tenant by id (super-admin only — for now we just
// match by id and return the row; the role check is in the route wrapper).
func GetTenant(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var t Tenant
		var created, updated time.Time
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id, name, slug, plan, status, created_at, updated_at
			 FROM tenants WHERE id = $1`, id,
		).Scan(&t.ID, &t.Name, &t.Slug, &t.Plan, &t.Status, &created, &updated)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		t.CreatedAt = created.UTC().Format(time.RFC3339)
		t.UpdatedAt = updated.UTC().Format(time.RFC3339)
		kernel.RespondOK(c, gin.H{"tenant": t})
	}
}

// UpdateTenant body
type updateTenantReq struct {
	Name   *string `json:"name,omitempty"`
	Slug   *string `json:"slug,omitempty"`
	Plan   *string `json:"plan,omitempty"`
	Status *string `json:"status,omitempty"`
}

// UpdateTenant lets the calling tenant update its own name/slug/plan/status.
// Tenant_id is from JWT, never from body — prevents privilege escalation.
func UpdateTenant(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req updateTenantReq
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		_, err := pool.Pgx().Exec(c.Request.Context(),
			`UPDATE tenants SET
				name = COALESCE($2, name),
				slug = COALESCE($3, slug),
				plan = COALESCE($4, plan),
				status = COALESCE($5, status),
				updated_at = now()
			 WHERE id = $1`,
			tenantID, req.Name, req.Slug, req.Plan, req.Status,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondOK(c, gin.H{"updated": true})
	}
}
