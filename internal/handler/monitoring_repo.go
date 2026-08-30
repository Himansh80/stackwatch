package handler

// monitoring_repo.go — Repository-backed Tier 0.5 monitoring endpoints.
// These replace the raw-SQL handlers in monitoring_servers.go / monitoring_agents_ingest.go.
// Repository interface is defined in internal/repository.

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
	"github.com/stackwatch/platform/internal/repository"
)

// MonitoringRepository is the subset of repository.DB we actually need for Tier 0.5.
type MonitoringRepository = repository.PgxServerRepo

// NewMonitoringRepository constructs the production repository against the given pool.
func NewMonitoringRepository(pool *db.Pool) *MonitoringRepository {
	return repository.NewPgxServerRepo(pool)
}

//
// --- Servers ---
//

// ListServersRepo returns all servers for the tenant.
func ListServersRepo(repo *MonitoringRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		servers, err := repo.ListServers(c.Request.Context(), tenantID)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"servers": servers, "total": len(servers)})
	}
}

// GetServerRepo returns a single server by id.
func GetServerRepo(repo *MonitoringRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		s, err := repo.GetServer(c.Request.Context(), tenantID, id)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				kernel.RespondError(c, kernel.ErrNotFound)
				return
			}
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

// CreateServerRepo creates a new server.
func CreateServerRepo(repo *MonitoringRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req ServerCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		s, err := repo.CreateServer(c.Request.Context(), tenantID, repository.CreateServerInput{
			Name:      req.Name,
			Hostname:  req.Hostname,
			IPAddress: req.IPAddress,
			OS:        req.OS,
			OSVersion: req.OSVersion,
			Arch:      req.Arch,
			Tags:      req.Tags,
		})
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		c.JSON(http.StatusCreated, s)
	}
}

// UpdateServerRepo applies a partial update.
func UpdateServerRepo(repo *MonitoringRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		var req ServerUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		n, err := repo.UpdateServer(c.Request.Context(), tenantID, id, repository.UpdateServerInput{
			Name:          req.Name,
			Hostname:      req.Hostname,
			IPAddress:     req.IPAddress,
			OS:            req.OS,
			OSVersion:     req.OSVersion,
			Arch:          req.Arch,
			KernelVersion: req.KernelVersion,
			CPUCores:      req.CPUCores,
			CPUModel:      req.CPUModel,
			MemoryTotal:   req.MemoryTotal,
			DiskTotal:     req.DiskTotal,
			Tags:          req.Tags,
			Status:        req.Status,
		})
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		if n == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		// Return updated server
		s, err := repo.GetServer(c.Request.Context(), tenantID, id)
		if err != nil {
			kernel.RespondError(c, kernel.ErrInternal)
			return
		}
		c.JSON(http.StatusOK, s)
	}
}

// DeleteServerRepo soft-deletes a server.
func DeleteServerRepo(repo *MonitoringRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		n, err := repo.DeleteServer(c.Request.Context(), tenantID, id)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		if n == 0 {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

//
// --- Agents ---
//

// ListAgentsRepo returns all agents for the tenant.
func ListAgentsRepo(repo *MonitoringRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		agents, err := repo.ListAgents(c.Request.Context(), tenantID)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"agents": agents, "total": len(agents)})
	}
}

// EnrollAgentRepo creates an agent and returns the ingest key.
func EnrollAgentRepo(repo *MonitoringRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := getTenantID(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		var req AgentCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			kernel.RespondErrorWithCode(c, http.StatusBadRequest, "bad_request", err.Error())
			return
		}

		ingestKey := "ios_a_" + uuid.NewString()[:32]
		keyHash := ingestKey // Phase 1 parity: no hashing yet

		labels := req.Labels
		if labels == nil {
			labels = []byte("{}")
		}

		agent, err := repo.CreateAgent(c.Request.Context(), tenantID, req.Name, ingestKey, keyHash, labels)
		if err != nil {
			kernel.RespondErrorWithCode(c, http.StatusInternalServerError, "internal", err.Error())
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":            agent.ID,
			"agent_id":      agent.ID,
			"name":          agent.Name,
			"ingest_key":    agent.IngestKey,
			"ingest_url":    "/api/v1/ingest/heartbeat",
			"poll_interval": "30s",
		})
	}
}
