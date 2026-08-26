// Tier 7 Phase 3 — CI/CD Visibility (D8). Public webhook receivers
// (no JWT — providers can't carry one; tenant is resolved from the
// X-Tenant-Id header for the simple case):
//
//	POST   /api/v1/cicd/webhook/github           — GitHub push event
//	POST   /api/v1/cicd/webhook/gitlab           — GitLab push event
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/stackwatch/platform/internal/db"
)

// GitHubCICDWebhook accepts a GitHub push event and records it as a
// pipeline. No JWT — providers don't carry one. Tenant resolution:
// the X-Tenant-Id header carries the tenant uuid; in a production
// deployment this would be replaced by a repo-slug→tenant lookup
// table, but for Phase 3 the header keeps the surface small.
func GitHubCICDWebhook(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantIDStr := strings.TrimSpace(c.GetHeader("X-Tenant-Id"))
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-Id header missing or invalid"})
			return
		}
		var ev ghPushEvent
		if err := c.ShouldBindJSON(&ev); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if ev.Repository.FullName == "" || ev.HeadCommit.ID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing repository or head_commit"})
			return
		}
		status := "success"
		switch strings.ToLower(ev.Conclusion) {
		case "failure", "failed":
			status = "failed"
		case "cancelled", "canceled":
			status = "cancelled"
		}
		branch := ev.Ref
		if strings.HasPrefix(branch, "refs/heads/") {
			branch = strings.TrimPrefix(branch, "refs/heads/")
		}
		var id string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO cicd_pipelines (tenant_id, provider, repo, branch,
			                              commit_sha, status)
			 VALUES ($1, 'github', $2, $3, $4, $5)
			 RETURNING id::text`,
			tenantID, ev.Repository.FullName, branch, ev.HeadCommit.ID, status,
		).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "insert pipeline: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"pipeline_id": id,
			"provider":    "github",
			"repo":        ev.Repository.FullName,
			"conclusion":  status,
		})
	}
}

// GitLabCICDWebhook accepts a GitLab push event. Same auth model as
// the GitHub webhook (X-Tenant-Id header).
func GitLabCICDWebhook(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantIDStr := strings.TrimSpace(c.GetHeader("X-Tenant-Id"))
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-Id header missing or invalid"})
			return
		}
		var ev glPushEvent
		if err := json.NewDecoder(c.Request.Body).Decode(&ev); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad json: " + err.Error()})
			return
		}
		if ev.Project.PathWithNamespace == "" || ev.CheckoutSHA == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing project or checkout_sha"})
			return
		}
		branch := ev.Ref
		if strings.HasPrefix(branch, "refs/heads/") {
			branch = strings.TrimPrefix(branch, "refs/heads/")
		}
		var id string
		err = pool.Pgx().QueryRow(c.Request.Context(),
			`INSERT INTO cicd_pipelines (tenant_id, provider, repo, branch,
			                              commit_sha, status)
			 VALUES ($1, 'gitlab', $2, $3, $4, 'success')
			 RETURNING id::text`,
			tenantID, ev.Project.PathWithNamespace, branch, ev.CheckoutSHA,
		).Scan(&id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "insert pipeline: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"pipeline_id": id,
			"provider":    "gitlab",
			"repo":        ev.Project.PathWithNamespace,
		})
	}
}