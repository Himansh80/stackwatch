package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// CreateLXCRequest is the JSON body for POST /proxmox/hosts/:id/nodes/:node/lxc.
type CreateLXCRequest struct {
	VMID         int    `json:"vmid" binding:"required,min=100,max=999999999"`
	Hostname     string `json:"hostname" binding:"required"`
	MemoryMB     int    `json:"memory_mb" binding:"required,min=64"`
	Cores        int    `json:"cores" binding:"required,min=1,max=128"`
	DiskGB       int    `json:"disk_gb" binding:"required,min=1"`
	Storage      string `json:"storage"`
	Password     string `json:"password"`
	OSTemplate   string `json:"ostemplate"`
	Bridge       string `json:"bridge"`
	IPConfig     string `json:"ipconfig"`
	Unprivileged bool   `json:"unprivileged"`
	Description  string `json:"description"`
}

// CreateLXC creates a new LXC container on the given node.
func (h *ProxmoxHandler) CreateLXC(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	var req CreateLXCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	spec := proxmox.LXCSpec{
		VMID:         req.VMID,
		Hostname:     req.Hostname,
		MemoryMB:     req.MemoryMB,
		Cores:        req.Cores,
		DiskGB:       req.DiskGB,
		Storage:      req.Storage,
		Password:     req.Password,
		OSTemplate:   req.OSTemplate,
		Bridge:       req.Bridge,
		IPConfig:     req.IPConfig,
		Unprivileged: req.Unprivileged,
		Description:  req.Description,
	}
	task, err := cli.CreateLXC(c.Request.Context(), node, spec)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{
		"task": task, "vmid": req.VMID, "hostname": req.Hostname,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/%s/tasks/%s", host.ID.String(), node, task),
	})
}

// LXCStatusAction performs lifecycle actions on an LXC container (start/stop/reboot/shutdown).
func (h *ProxmoxHandler) LXCStatusAction(c *gin.Context) {
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
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	task, err := cli.LXCStatus(c.Request.Context(), node, vmid, action)
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

// GetLXCConfig returns the current configuration of an LXC container.
func (h *ProxmoxHandler) GetLXCConfig(c *gin.Context) {
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
	cfg, err := cli.GetLXCConfig(c.Request.Context(), node, vmid)
	if err != nil {
		if isProxmoxNotFound(err) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, cfg)
}

// UpdateLXCConfigRequest is the JSON body for PUT /proxmox/hosts/:id/nodes/:node/lxc/:vmid/config.
type UpdateLXCConfigRequest struct {
	Hostname    string `json:"hostname,omitempty"`
	MemoryMB    int    `json:"memory_mb,omitempty"`
	Cores       int    `json:"cores,omitempty"`
	Description string `json:"description,omitempty"`
}

// UpdateLXCConfig updates editable fields on an LXC container.
func (h *ProxmoxHandler) UpdateLXCConfig(c *gin.Context) {
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
	var req UpdateLXCConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	spec := proxmox.LXCSpec{
		Hostname:    req.Hostname,
		MemoryMB:    req.MemoryMB,
		Cores:       req.Cores,
		Description: req.Description,
	}
	if _, err := cli.UpdateLXCConfig(c.Request.Context(), node, vmid, spec); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"updated": true, "vmid": vmid})
}

// DeleteLXC removes an LXC container (async). Container must be stopped first.
func (h *ProxmoxHandler) DeleteLXC(c *gin.Context) {
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
	task, err := cli.DeleteLXC(c.Request.Context(), node, vmid, true)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"task": task, "deleted": true, "vmid": vmid,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/%s/tasks/%s", host.ID.String(), node, task),
	})
}
