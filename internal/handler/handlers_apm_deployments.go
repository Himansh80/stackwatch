// Tier 7 — APM (D2) — Deployment endpoints.
//
//	POST /api/v1/apm/deployments   — record a deployment marker
//	GET  /api/v1/apm/deployments   — list deployments (filter ?service_id=)
//
// Deployments are immutable markers — once recorded they stay. To
// indicate a rollback, callers should record a new deployment with the
// previous version + set `rolled_back = true` on the old row via
// direct SQL (no API surface needed in Phase 1).
package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// apmDeploymentReq is the JSON shape for creating a deployment.
type apmDeploymentReq struct {
	ServiceID   string `json:"service_id" binding:"required,uuid"`
	Environment string `json:"environment" binding:"max=64"`
	Version     string `json:"version" binding:"required,min=1,max=64"`
	CommitSHA   string `json:"commit_sha" binding:"max=64"`
}

// RecordAPMDeployment inserts a deployment marker.
func RecordAPMDeployment(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r apmDeploymentReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		svcID, _ := uuid.Parse(r.ServiceID)
		env := r.Environment
		if env == "" {
			env = "production"
		}
		var id uuid.UUID
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO apm_deployments
			 (tenant_id, service_id, environment, version, commit_sha)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id`,
			tenantID, svcID, env, r.Version, nullIfEmpty(r.CommitSHA),
		).Scan(&id)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		kernel.RespondCreated(c, gin.H{
			"id":          id.String(),
			"service_id":  svcID.String(),
			"environment": env,
			"version":     r.Version,
			"commit_sha":  r.CommitSHA,
			"deployed_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ListAPMDeployments returns deployments for the tenant, optionally
// filtered by service_id.
func ListAPMDeployments(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		limit := 50
		if v := c.Query("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 200 {
				limit = n
			}
		}
		args := []any{tenantID}
		q := `SELECT d.id::text, d.service_id::text, s.name, d.environment, d.version,
		             COALESCE(d.commit_sha, ''), d.deployed_at::text, d.rolled_back
		      FROM apm_deployments d
		      JOIN apm_services s ON s.id = d.service_id
		      WHERE d.tenant_id = $1`
		if svcID := c.Query("service_id"); svcID != "" {
			if id, err := uuid.Parse(svcID); err == nil {
				args = append(args, id)
				q += " AND d.service_id = $2"
			}
		}
		args = append(args, limit)
		q += " ORDER BY d.deployed_at DESC LIMIT $" + strconv.Itoa(len(args))
		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var id, svcID, svcName, env, ver, sha, deployedAt string
			var rolled bool
			if err := rows.Scan(&id, &svcID, &svcName, &env, &ver, &sha, &deployedAt, &rolled); err != nil {
				continue
			}
			out = append(out, gin.H{
				"id":           id,
				"service_id":   svcID,
				"service_name": svcName,
				"environment":  env,
				"version":      ver,
				"commit_sha":   sha,
				"deployed_at":  deployedAt,
				"rolled_back":  rolled,
			})
		}
		kernel.RespondOK(c, gin.H{"deployments": out, "total": len(out)})
	}
}
