// Tier 14 Phase 14.2 — Proxmox VM QEMU guest agent endpoints.
//
// Currently exposes /agent/network-get-interfaces so the VM detail
// page can show the guest's IP, MAC, and bridge info. Other agent
// endpoints (get-osinfo, get-fsinfo, etc.) will be added in later
// phases as the UI needs them.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// networkInterfacesResponse wraps the guest agent result so we can
// tell the UI whether the agent is actually running inside the VM.
type networkInterfacesResponse struct {
	Interfaces   []proxmox.VMNetworkInterface `json:"interfaces"`
	AgentRunning bool                        `json:"agent_running"`
}

// GetVMNetworkInterfaces proxies GET /proxmox/hosts/:id/nodes/:node/qemu/:vmid/agent/network-get-interfaces
// to the Proxmox API.
//
// Returns:
//   - 200 with {interfaces: [...], agent_running: true} on success
//   - 200 with {interfaces: null, agent_running: false} when the QEMU
//     guest agent is not running inside the VM (Proxmox returns 500)
//   - 404 when the VM does not exist
func (h *ProxmoxHandler) GetVMNetworkInterfaces(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	vmid, err := strconv.Atoi(c.Param("vmid"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	ifaces, err := cli.GetVMNetworkInterfaces(c.Request.Context(), node, vmid)
	if err != nil {
		if isProxmoxNotFound(err) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, networkInterfacesResponse{
		Interfaces:   ifaces,
		AgentRunning: ifaces != nil,
	})
}