// Tier 7 Phase 3 — CI/CD Visibility (D8). Deployment CRUD.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListCICDDeployments returns deployments for the caller's tenant.
// Optional filters: ?service_id=X&environment=prod&limit=N.
func ListCICDDeployments(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		serviceIDStr := strings.TrimSpace(c.Query("service_id"))
		environment := strings.TrimSpace(c.Query("environment"))
		limit := clampLimit(c.Query("limit"), 100, 500)

		args := []any{tenantID}
		q := `SELECT d.id::text, d.pipeline_id::text, d.service_id::text,
		             COALESCE(s.name, ''), d.environment, d.version, d.deployed_at::text
		      FROM cicd_deployments d
		      LEFT JOIN apm_services s ON s.id = d.service_id
		      WHERE d.tenant_id = $1`
		if serviceIDStr != "" {
			if sid, err := uuid.Parse(serviceIDStr); err == nil {
				args = append(args, sid)
				q += " AND d.service_id = $" + itoa(len(args))
			}
		}
		if environment != "" {
			args = append(args, environment)
			q += " AND d.environment = $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY d.deployed_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []cicdDeploymentRow{}
		for rows.Next() {
			var d cicdDeploymentRow
			if err := rows.Scan(&d.ID, &d.PipelineID, &d.ServiceID,
				&d.ServiceName, &d.Environment, &d.Version, &d.DeployedAt); err != nil {
				continue
			}
			out = append(out, d)
		}
		kernel.RespondOK(c, gin.H{"deployments": out, "total": len(out)})
	}
}

// CreateCICDDeployment inserts a deployment linking a pipeline to an
// APM service. The handler verifies the pipeline + service both belong
// to the caller's tenant before inserting.
func CreateCICDDeployment(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r cicdDeploymentReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		pipelineID, err := uuid.Parse(r.PipelineID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		serviceID, err := uuid.Parse(r.ServiceID)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		// Tenant ownership check on both parents.
		var ownCount int
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT
			   (SELECT COUNT(*) FROM cicd_pipelines WHERE id = $1 AND tenant_id = $2)
			 + (SELECT COUNT(*) FROM apm_services    WHERE id = $3 AND tenant_id = $2)`,
			pipelineID, tenantID, serviceID,
		).Scan(&ownCount)
		if err != nil || ownCount != 2 {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var d cicdDeploymentRow
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO cicd_deployments (tenant_id, pipeline_id, service_id, environment, version)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING id::text, pipeline_id::text, service_id::text,
			           environment, version, deployed_at::text`,
			tenantID, pipelineID, serviceID, r.Environment, r.Version,
		).Scan(&d.ID, &d.PipelineID, &d.ServiceID, &d.Environment, &d.Version, &d.DeployedAt)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		// Backfill service name for the response shape (so the UI
		// can render the row without a follow-up call).
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT name FROM apm_services WHERE id = $1`, serviceID,
		).Scan(&d.ServiceName)
		kernel.RespondCreated(c, gin.H{"deployment": d})
	}
}