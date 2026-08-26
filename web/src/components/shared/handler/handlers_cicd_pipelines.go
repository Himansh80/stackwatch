// Tier 7 Phase 3 — CI/CD Visibility (D8). Pipelines + deployments.
//
// Every protected query honors tenant_id from the JWT — no cross-
// tenant data ever crosses the wire.
//
// Routes:
//
//	GET    /api/v1/cicd/pipelines                — list pipelines
//	POST   /api/v1/cicd/pipelines                — create a pipeline record
//	GET    /api/v1/cicd/pipelines/:id            — pipeline detail
//	GET    /api/v1/cicd/deployments              — list deployments
//	POST   /api/v1/cicd/deployments              — create a deployment
package handler

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListCICDPipelines returns the caller's tenant pipelines, newest first.
// Optional filters: ?repo=X&status=failed&provider=github&limit=N.
func ListCICDPipelines(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		repo := strings.TrimSpace(c.Query("repo"))
		status := strings.ToLower(strings.TrimSpace(c.Query("status")))
		provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
		limit := clampLimit(c.Query("limit"), 100, 500)

		args := []any{tenantID}
		q := `SELECT id::text, provider, repo, COALESCE(branch, ''),
		             COALESCE(commit_sha, ''), status,
		             started_at::text, finished_at::text, duration_ms
		      FROM cicd_pipelines
		      WHERE tenant_id = $1`
		if repo != "" {
			args = append(args, repo)
			q += " AND repo = $" + itoa(len(args))
		}
		if allowedCICDStatuses[status] {
			args = append(args, status)
			q += " AND status = $" + itoa(len(args))
		}
		if allowedCICDProviders[provider] {
			args = append(args, provider)
			q += " AND provider = $" + itoa(len(args))
		}
		args = append(args, limit)
		q += " ORDER BY started_at DESC LIMIT $" + itoa(len(args))

		rows, err := pool.Pgx().Query(c.Request.Context(), q, args...)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()
		out := []cicdPipelineRow{}
		for rows.Next() {
			var p cicdPipelineRow
			var finished *string
			if err := rows.Scan(&p.ID, &p.Provider, &p.Repo, &p.Branch,
				&p.CommitSHA, &p.Status, &p.StartedAt, &finished, &p.DurationMS); err != nil {
				continue
			}
			p.FinishedAt = finished
			out = append(out, p)
		}
		kernel.RespondOK(c, gin.H{"pipelines": out, "total": len(out)})
	}
}

// CreateCICDPipeline inserts a pipeline row for the caller's tenant.
func CreateCICDPipeline(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var r cicdPipelineReq
		if err := c.ShouldBindJSON(&r); err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var finished interface{}
		if r.FinishedAt != "" {
			if t, err := time.Parse(time.RFC3339, r.FinishedAt); err == nil {
				finished = t
			}
		}
		var p cicdPipelineRow
		var fin *string
		err := pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO cicd_pipelines (tenant_id, provider, repo, branch,
			                              commit_sha, status, finished_at, duration_ms)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING id::text, provider, repo, COALESCE(branch, ''),
			           COALESCE(commit_sha, ''), status,
			           started_at::text, finished_at::text, duration_ms`,
			tenantID, r.Provider, r.Repo, r.Branch, r.CommitSHA, r.Status,
			finished, r.DurationMS,
		).Scan(&p.ID, &p.Provider, &p.Repo, &p.Branch, &p.CommitSHA,
			&p.Status, &p.StartedAt, &fin, &p.DurationMS)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		p.FinishedAt = fin
		kernel.RespondCreated(c, gin.H{"pipeline": p})
	}
}

// GetCICDPipeline returns a single pipeline + a count of deployments.
func GetCICDPipeline(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var p cicdPipelineRow
		var fin *string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT id::text, provider, repo, COALESCE(branch, ''),
			        COALESCE(commit_sha, ''), status,
			        started_at::text, finished_at::text, duration_ms
			 FROM cicd_pipelines
			 WHERE id = $1 AND tenant_id = $2`, id, tenantID,
		).Scan(&p.ID, &p.Provider, &p.Repo, &p.Branch, &p.CommitSHA,
			&p.Status, &p.StartedAt, &fin, &p.DurationMS)
		if err != nil {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		p.FinishedAt = fin
		// Count deployments for this pipeline so the UI can badge it.
		var deployCount int
		_ = pool.Pgx().QueryRow(c.Request.Context(),
			`SELECT COUNT(*) FROM cicd_deployments WHERE pipeline_id = $1`, id,
		).Scan(&deployCount)
		kernel.RespondOK(c, gin.H{
			"pipeline":            p,
			"deployment_count":    deployCount,
		})
	}
}