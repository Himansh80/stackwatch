package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListPools returns all resource pools.
func (h *ProxmoxHandler) ListPools(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	pools, err := cli.ListPools(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	// Strip members from list (use GET pool for detail)
	for i := range pools {
		pools[i].Members = nil
	}
	kernel.RespondOK(c, gin.H{"pools": pools, "total": len(pools)})
}

// GetPool returns one pool with members.
func (h *ProxmoxHandler) GetPool(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	poolID := c.Param("poolid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	pool, err := cli.GetPool(c.Request.Context(), poolID)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, pool)
}

// CreatePool creates a new pool. Body: {"poolid": "...", "comment": "..."}
func (h *ProxmoxHandler) CreatePool(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req struct {
		PoolID  string `json:"poolid" binding:"required"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	upid, err := cli.CreatePool(c.Request.Context(), req.PoolID, req.Comment)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"poolid":  req.PoolID,
		"comment": req.Comment,
		"upid":    upid,
		"status":  "created",
	})
}

// UpdatePool modifies a pool's comment.
func (h *ProxmoxHandler) UpdatePool(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	poolID := c.Param("poolid")
	var req struct {
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.UpdatePool(c.Request.Context(), poolID, req.Comment); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"poolid": poolID, "comment": req.Comment, "status": "updated"})
}

// DeletePool removes a pool.
func (h *ProxmoxHandler) DeletePool(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	poolID := c.Param("poolid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeletePool(c.Request.Context(), poolID); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"poolid": poolID, "status": "deleted"})
}

// ListClusterResources returns aggregated cluster resources.
func (h *ProxmoxHandler) ListClusterResources(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	typeFilter := c.Query("type")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	resources, err := cli.ListClusterResources(c.Request.Context(), typeFilter)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"resources": resources, "total": len(resources), "type": typeFilter})
}

// GetClusterStatus returns cluster status (nodes + quorum).
func (h *ProxmoxHandler) GetClusterStatus(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	nodes, err := cli.GetClusterStatus(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"nodes": nodes, "total": len(nodes)})
}

// GetClusterInfo returns basic cluster details (name, ID, version).
func (h *ProxmoxHandler) GetClusterInfo(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	handleProxmoxCall(c, func() (any, error) {
		return cli.GetClusterInfo(c.Request.Context())
	})
}
