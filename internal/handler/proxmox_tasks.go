package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListNodeTasks returns task history for a node.
func (h *ProxmoxHandler) ListNodeTasks(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	limit := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	tasks, err := cli.ListNodeTasks(c.Request.Context(), node, limit)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"tasks": tasks, "total": len(tasks)})
}

// GetTaskStatus returns the current status of a task by UPID.
// Returns the existing TaskStatus struct (which has Status/ExitStatus/UPID).
func (h *ProxmoxHandler) GetTaskStatus(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	upid := c.Param("upid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	status, err := cli.GetTaskStatus(c.Request.Context(), node, upid)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, status)
}

// GetTaskLog returns the log output of a task.
func (h *ProxmoxHandler) GetTaskLog(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	upid := c.Param("upid")
	lines := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			lines = v
		}
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	log, err := cli.GetTaskLog(c.Request.Context(), node, upid, lines)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, log)
}

// StopTask cancels a running task.
func (h *ProxmoxHandler) StopTask(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	upid := c.Param("upid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.StopTask(c.Request.Context(), node, upid); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"stopped": true, "upid": upid})
}

// ListClusterTasks returns cluster-wide recent tasks.
func (h *ProxmoxHandler) ListClusterTasks(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	limit := 0
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil {
			limit = v
		}
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	tasks, err := cli.ListClusterTasks(c.Request.Context(), limit)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"tasks": tasks, "total": len(tasks)})
}