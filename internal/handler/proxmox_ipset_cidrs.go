// Package handler — Tier 14 fill-in: IPSet CIDR CRUD.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// IPSetCidrSpec is the body for POST /proxmox/hosts/:id/firewall/ipset/:name/cidr.
type IPSetCidrSpec struct {
	CIDR    string `json:"cidr" binding:"required"`
	Comment string `json:"comment"`
	NoMatch bool   `json:"nomatch"`
}

// ListIPsetCidrs → GET /proxmox/hosts/:id/firewall/ipset/:name
func (h *ProxmoxHandler) ListIPsetCidrs(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	items, err := cli.ListIPsetCidrs(c.Request.Context(), name)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"cidrs": items, "total": len(items)})
}

// AddIPsetCidr → POST /proxmox/hosts/:id/firewall/ipset/:name/cidr
func (h *ProxmoxHandler) AddIPsetCidr(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	var req IPSetCidrSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.AddIPsetCidr(c.Request.Context(), name, req.CIDR, req.Comment, req.NoMatch); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{"ok": true, "cidr": req.CIDR})
}

// DeleteIPsetCidr → DELETE /proxmox/hosts/:id/firewall/ipset/:name/cidr?cidr=X
func (h *ProxmoxHandler) DeleteIPsetCidr(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	cidr := c.Query("cidr")
	if cidr == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.DeleteIPsetCidr(c.Request.Context(), name, cidr); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"ok": true, "deleted": cidr})
}