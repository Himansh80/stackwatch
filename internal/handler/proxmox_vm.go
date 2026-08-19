package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// VMStatusActionRequest is the body for POST /proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/:action.
type VMStatusActionRequest struct {
	Force bool `json:"force"`
}

// waitForTask polls a Proxmox task until it finishes.
// Returns the exit status ("OK" / "ERROR" / "timeout") or empty string if no task.
func waitForTask(c *gin.Context, cli *proxmox.Client, node, upid string) (string, error) {
	if upid == "" {
		return "", nil
	}
	// Long default for busy hosts (reboot/shutdown on hosts with 10+ VMs
	// can take 90+ seconds).
	return waitForTaskBrief(c, cli, node, upid, 180*time.Second)
}

// waitForTaskBrief polls a Proxmox task for a bounded time.
func waitForTaskBrief(c *gin.Context, cli *proxmox.Client, node, upid string, max time.Duration) (string, error) {
	if upid == "" {
		return "", nil
	}
	deadline := time.Now().Add(max)
	for time.Now().Before(deadline) {
		ts, err := cli.GetTaskStatus(c.Request.Context(), node, upid)
		if err != nil {
			return "", err
		}
		if ts.Status == "stopped" {
			return ts.ExitStatus, nil
		}
		time.Sleep(3 * time.Second)
	}
	return "timeout", nil
}

// VMStatusAction performs lifecycle actions on a VM (start/stop/reboot/etc).
// Returns the task ID immediately (async). Use GET /tasks/:upid to poll.
func (h *ProxmoxHandler) VMStatusAction(c *gin.Context) {
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
	action := c.Param("action")
	switch action {
	case proxmox.StatusStart, proxmox.StatusStop, proxmox.StatusReboot,
		proxmox.StatusShutdown, proxmox.StatusSuspend, proxmox.StatusResume:
	default:
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req VMStatusActionRequest
	_ = c.ShouldBindJSON(&req)
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	task, err := cli.VMStatus(c.Request.Context(), node, vmid, action, req.Force)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"task": task, "action": action,
		"vmid": vmid, "node": node,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/%s/tasks/%s", host.ID.String(), node, task),
	})
}

// CreateVMRequest is the body for POST /proxmox/hosts/:id/nodes/:node/qemu.
type CreateVMRequest struct {
	VMID     int    `json:"vmid" binding:"required,min=100,max=999999999"`
	Name     string `json:"name" binding:"required"`
	MemoryMB int    `json:"memory_mb" binding:"required,min=128"`
	Cores    int    `json:"cores" binding:"required,min=1,max=128"`
	DiskGB   int    `json:"disk_gb" binding:"required,min=1"`
	Storage  string `json:"storage"`
	Bridge   string `json:"bridge"`
	ISO      string `json:"iso,omitempty"`
	Boot     string `json:"boot,omitempty"`
}

// CreateVM creates a new QEMU VM on the given node.
// Returns the task ID immediately (async). Poll via /tasks/:upid.
func (h *ProxmoxHandler) CreateVM(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	var req CreateVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	spec := proxmox.VMSpec{
		VMID: req.VMID, Name: req.Name,
		MemoryMB: req.MemoryMB, Cores: req.Cores,
		DiskGB: req.DiskGB, Storage: req.Storage,
		Bridge: req.Bridge, ISO: req.ISO, Boot: req.Boot,
	}
	task, err := cli.CreateVM(c.Request.Context(), node, spec)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{
		"task": task, "vmid": req.VMID, "name": req.Name,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/%s/tasks/%s", host.ID.String(), node, task),
	})
}

// DeleteVM removes a VM and its disks (async). VM must be stopped first.
func (h *ProxmoxHandler) DeleteVM(c *gin.Context) {
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
	var req VMStatusActionRequest
	_ = c.ShouldBindJSON(&req)
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	task, err := cli.DeleteVM(c.Request.Context(), node, vmid, req.Force)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"task": task, "deleted": true, "vmid": vmid,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/%s/tasks/%s", host.ID.String(), node, task),
	})
}

// TaskStatus returns the status of an async Proxmox task.
// GET /api/v1/proxmox/hosts/:id/nodes/:node/tasks/:upid
func (h *ProxmoxHandler) TaskStatus(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	upid := c.Param("upid")
	if upid == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	ts, err := cli.GetTaskStatus(c.Request.Context(), node, upid)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, ts)
}

// GetVMConfig returns the current configuration of a VM.
func (h *ProxmoxHandler) GetVMConfig(c *gin.Context) {
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
	cfg, err := cli.GetVMConfig(c.Request.Context(), node, vmid)
	if err != nil {
		// Proxmox returns 500 for non-existent VM
		if isProxmoxNotFound(err) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, cfg)
}

// UpdateVMConfigRequest is the body for PUT /proxmox/hosts/:id/nodes/:node/qemu/:vmid/config.
type UpdateVMConfigRequest struct {
	Name     string `json:"name,omitempty"`
	MemoryMB int    `json:"memory_mb,omitempty"`
	Cores    int    `json:"cores,omitempty"`
	Boot     string `json:"boot,omitempty"`
}

// UpdateVMConfig updates editable fields on a VM.
// NOTE: Proxmox PUT /config is synchronous — it returns 200 OK directly with
// no UPID. Polling /tasks/:upid is not applicable. Just return success.
func (h *ProxmoxHandler) UpdateVMConfig(c *gin.Context) {
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
	var req UpdateVMConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	spec := proxmox.VMSpec{
		Name: req.Name, MemoryMB: req.MemoryMB,
		Cores: req.Cores, Boot: req.Boot,
	}
	if _, err := cli.UpdateVMConfig(c.Request.Context(), node, vmid, spec); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"updated": true, "vmid": vmid})
}
