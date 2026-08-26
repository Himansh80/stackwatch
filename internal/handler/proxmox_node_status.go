// Package handler — Tier 14 fill-in: Node status + Node services.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// GetNodeStatus → GET /proxmox/hosts/:id/nodes/:node/status
func (h *ProxmoxHandler) GetNodeStatus(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	node := c.Param("node")
	status, err := cli.GetNodeStatus(c.Request.Context(), node)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, status)
}

// ListNodeServices → GET /proxmox/hosts/:id/nodes/:node/services
func (h *ProxmoxHandler) ListNodeServices(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	node := c.Param("node")
	svcs, err := cli.ListNodeServices(c.Request.Context(), node)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"services": svcs, "total": len(svcs)})
}