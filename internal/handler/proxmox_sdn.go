// Package handler — Tier 14 fill-in: SDN zones CRUD.
package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// SdnZoneSpec is the body for POST /proxmox/hosts/:id/sdn/zones.
type SdnZoneSpec struct {
	Zone   string `json:"zone" binding:"required"`
	Type   string `json:"type" binding:"required"` // simple, vlan, vxlan, qinq, evpn
	Bridge string `json:"bridge"`
}

// ListSdnZones → GET /proxmox/hosts/:id/sdn/zones
func (h *ProxmoxHandler) ListSdnZones(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	items, err := cli.ListSDNZones(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"zones": items, "total": len(items)})
}

// CreateSdnZone → POST /proxmox/hosts/:id/sdn/zones
func (h *ProxmoxHandler) CreateSdnZone(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req SdnZoneSpec
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.CreateSDNZone(c.Request.Context(), req.Zone, req.Type, req.Bridge); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{"ok": true, "zone": req.Zone})
}

// DeleteSdnZone → DELETE /proxmox/hosts/:id/sdn/zones/:zone
func (h *ProxmoxHandler) DeleteSdnZone(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	zone := c.Param("zone")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.DeleteSDNZone(c.Request.Context(), zone); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"ok": true, "deleted": zone})
}