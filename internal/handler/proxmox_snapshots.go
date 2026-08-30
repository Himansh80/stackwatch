// Tier 14 Phase 14.3 — Proxmox VM snapshot CRUD handlers.
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListVMSnapshots — GET /proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot
func (h *ProxmoxHandler) ListVMSnapshots(c *gin.Context) {
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
	snaps, err := cli.ListVMSnapshots(c.Request.Context(), node, vmid)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": snaps})
}

// CreateVMSnapshotRequest is the body for POST .../snapshot.
type CreateVMSnapshotRequest struct {
	Snapname    string `json:"snapname" binding:"required"`
	Description string `json:"description"`
	VMState     bool   `json:"vmstate"`
}

// CreateVMSnapshot — POST /proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot
func (h *ProxmoxHandler) CreateVMSnapshot(c *gin.Context) {
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
	var req CreateVMSnapshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	upid, err := cli.CreateVMSnapshot(c.Request.Context(), node, vmid, proxmox.VMSnapshotCreateSpec{
		Snapname:    req.Snapname,
		Description: req.Description,
		VMState:     req.VMState,
	})
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"upid": upid})
}

// DeleteVMSnapshot — DELETE .../snapshot?snapname=X
func (h *ProxmoxHandler) DeleteVMSnapshot(c *gin.Context) {
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
	snapname := c.Query("snapname")
	if snapname == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	force := c.Query("force") == "1"
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	upid, err := cli.DeleteVMSnapshot(c.Request.Context(), node, vmid, snapname, force)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"upid": upid})
}

// RollbackVMSnapshot — POST .../snapshot/:snapname/rollback
func (h *ProxmoxHandler) RollbackVMSnapshot(c *gin.Context) {
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
	snapname := c.Param("snapname")
	if snapname == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	upid, err := cli.RollbackVMSnapshot(c.Request.Context(), node, vmid, snapname)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"upid": upid})
}
