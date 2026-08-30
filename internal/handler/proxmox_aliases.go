// Package handler — Tier 14 fill-in: firewall aliases (datacenter-scoped).
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// fetchHostCredsDatacenter: same as fetchHostCreds but for cluster-scoped endpoints.
// We re-use fetchHostCreds because the API token is per-host (not per-datacenter).
func (h *ProxmoxHandler) ListAliases(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	items, err := cli.ListAliases(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"aliases": items, "total": len(items)})
}

// AliasSpec is the body for POST/PUT /proxmox/hosts/:id/cluster/firewall/aliases/:name.
type AliasSpec struct {
	Name    string `json:"name" binding:"required"`
	CIDR    string `json:"cidr" binding:"required"`
	Comment string `json:"comment"`
}

func (h *ProxmoxHandler) CreateAlias(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req AliasSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.CreateAlias(c.Request.Context(), req.Name, req.CIDR, req.Comment); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{"ok": true, "name": req.Name})
}

func (h *ProxmoxHandler) UpdateAlias(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	var req AliasSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.UpdateAlias(c.Request.Context(), name, req.CIDR, req.Comment); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"ok": true, "name": name})
}

func (h *ProxmoxHandler) DeleteAlias(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.DeleteAlias(c.Request.Context(), name); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"ok": true, "deleted": name})
}
