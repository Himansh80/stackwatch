package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// CreateNetworkRequest is the JSON body for POST /proxmox/hosts/:id/nodes/:node/network.
type CreateNetworkRequest struct {
	Iface           string `json:"iface" binding:"required,min=1,max=16"`
	Type            string `json:"type" binding:"required,oneof=bridge bond vlan eth"`
	Autostart       int    `json:"autostart"`
	BridgePorts     string `json:"bridge_ports"`
	BridgeVlanAware int    `json:"bridge_vlan_aware"`
	BridgeSTP       int    `json:"bridge_stp"`
	BondSlaves      string `json:"bond_slaves"`
	BondMode        string `json:"bond_mode"`
	IP              string `json:"ip"`
	Gateway         string `json:"gateway"`
	CIDR            string `json:"cidr"`
	MTU             int    `json:"mtu"`
	VlanID          int    `json:"vlan_id"`
	VlanRawDevice   string `json:"vlan_raw_device"`
	Comments        string `json:"comments"`
}

// UpdateNetworkRequest is the JSON body for PUT /proxmox/hosts/:id/nodes/:node/network/:iface.
type UpdateNetworkRequest struct {
	Autostart       *int   `json:"autostart,omitempty"`
	BridgePorts     string `json:"bridge_ports,omitempty"`
	BridgeVlanAware *int   `json:"bridge_vlan_aware,omitempty"`
	BridgeSTP       *int   `json:"bridge_stp,omitempty"`
	BondSlaves      string `json:"bond_slaves,omitempty"`
	BondMode        string `json:"bond_mode,omitempty"`
	IP              string `json:"ip,omitempty"`
	Gateway         string `json:"gateway,omitempty"`
	CIDR            string `json:"cidr,omitempty"`
	MTU             *int   `json:"mtu,omitempty"`
	Comments        string `json:"comments,omitempty"`
}

// ListNetwork returns all network interfaces on a node.
func (h *ProxmoxHandler) ListNetwork(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	ifaces, err := cli.ListNetwork(c.Request.Context(), node)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"interfaces": ifaces, "total": len(ifaces)})
}

// CreateNetwork adds a new interface (sync).
func (h *ProxmoxHandler) CreateNetwork(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	var req CreateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	spec := proxmox.NetworkSpec{
		Iface:           req.Iface,
		Type:            req.Type,
		Autostart:       req.Autostart,
		BridgePorts:     req.BridgePorts,
		BridgeVlanAware: req.BridgeVlanAware,
		BridgeSTP:       req.BridgeSTP,
		BondSlaves:      req.BondSlaves,
		BondMode:        req.BondMode,
		IP:              req.IP,
		Gateway:         req.Gateway,
		CIDR:            req.CIDR,
		MTU:             req.MTU,
		VlanID:          req.VlanID,
		VlanRawDevice:   req.VlanRawDevice,
		Comments:        req.Comments,
	}
	if _, err := cli.CreateNetwork(c.Request.Context(), node, spec); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{
		"created": true, "iface": req.Iface, "type": req.Type,
	})
}

// UpdateNetwork modifies an interface (sync).
func (h *ProxmoxHandler) UpdateNetwork(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	iface := c.Param("iface")
	var req UpdateNetworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	fields := map[string]string{}
	if req.Autostart != nil {
		if *req.Autostart > 0 {
			fields["autostart"] = "1"
		} else {
			fields["autostart"] = "0"
		}
	}
	if req.BridgePorts != "" {
		fields["bridge_ports"] = req.BridgePorts
	}
	if req.BridgeVlanAware != nil {
		if *req.BridgeVlanAware > 0 {
			fields["bridge_vlan_aware"] = "1"
		} else {
			fields["bridge_vlan_aware"] = "0"
		}
	}
	if req.BridgeSTP != nil {
		if *req.BridgeSTP > 0 {
			fields["bridge_stp"] = "1"
		} else {
			fields["bridge_stp"] = "0"
		}
	}
	if req.BondSlaves != "" {
		fields["bond_slaves"] = req.BondSlaves
	}
	if req.BondMode != "" {
		fields["bond_mode"] = req.BondMode
	}
	if req.IP != "" {
		fields["ip"] = req.IP
	}
	if req.Gateway != "" {
		fields["gateway"] = req.Gateway
	}
	if req.CIDR != "" {
		fields["cidr"] = req.CIDR
	}
	if req.MTU != nil {
		fields["mtu"] = fmt.Sprintf("%d", *req.MTU)
	}
	if req.Comments != "" {
		fields["comments"] = req.Comments
	}

	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.UpdateNetwork(c.Request.Context(), node, iface, fields); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"updated": true, "iface": iface})
}

// DeleteNetwork removes an interface (sync).
func (h *ProxmoxHandler) DeleteNetwork(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	iface := c.Param("iface")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteNetwork(c.Request.Context(), node, iface); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"deleted": true, "iface": iface})
}
