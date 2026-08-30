// Tier 7 Phase 2 — Synthetics Full (D5) — Test location CRUD.
//
//	GET  /api/v1/synthetics/locations — list locations for the caller's tenant
//	POST /api/v1/synthetics/locations — create a location (UNIQUE on tenant_id+name)
package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// synthLocationReq is the JSON shape for POST /locations.
type synthLocationReq struct {
	Name    string `json:"name" binding:"required,min=1,max=64"`
	Region  string `json:"region" binding:"required,min=1,max=64"`
	Enabled *bool  `json:"enabled"`
}

// ListSynthLocations returns all locations for the caller's tenant.
func ListSynthLocations(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`SELECT id::text, name, region, enabled
			 FROM synthetics_locations
			 WHERE tenant_id = $1
			 ORDER BY name ASC`, tenantID)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var id, name, region string
			var enabled bool
			if err := rows.Scan(&id, &name, &region, &enabled); err != nil {
				continue
			}
			out = append(out, gin.H{
				"id":      id,
				"name":    name,
				"region":  region,
				"enabled": enabled,
			})
		}
		kernel.RespondOK(c, gin.H{"locations": out, "total": len(out)})
	}
}

// CreateSynthLocation creates a new location. The DB enforces UNIQUE
// (tenant_id, name) — duplicates surface as a 409 via RespondError.
func CreateSynthLocation(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r synthLocationReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		r.Name = strings.TrimSpace(r.Name)
		r.Region = strings.TrimSpace(r.Region)
		enabled := true
		if r.Enabled != nil {
			enabled = *r.Enabled
		}
		var id uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO synthetics_locations (tenant_id, name, region, enabled)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			tenantID, r.Name, r.Region, enabled,
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"id":         id.String(),
			"name":       r.Name,
			"region":     r.Region,
			"enabled":    enabled,
			"created_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
