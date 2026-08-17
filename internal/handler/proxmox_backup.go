package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListBackupJobs returns all scheduled backup jobs.
func (h *ProxmoxHandler) ListBackupJobs(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	jobs, err := cli.ListBackupJobs(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"jobs": jobs, "total": len(jobs)})
}

// GetBackupJob returns one backup job.
func (h *ProxmoxHandler) GetBackupJob(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	jobID := c.Param("jobid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	job, err := cli.GetBackupJob(c.Request.Context(), jobID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, job)
}

// CreateBackupJob schedules a new backup.
func (h *ProxmoxHandler) CreateBackupJob(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req proxmox.BackupJob
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.ID == "" || req.Schedule == "" || req.Storage == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.CreateBackupJob(c.Request.Context(), req); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"id": req.ID, "schedule": req.Schedule, "storage": req.Storage, "status": "created"})
}

// UpdateBackupJob modifies a backup job.
func (h *ProxmoxHandler) UpdateBackupJob(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	jobID := c.Param("jobid")
	var req proxmox.BackupJob
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.UpdateBackupJob(c.Request.Context(), jobID, req); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"id": jobID, "status": "updated"})
}

// DeleteBackupJob removes a backup job.
func (h *ProxmoxHandler) DeleteBackupJob(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	jobID := c.Param("jobid")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteBackupJob(c.Request.Context(), jobID); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"id": jobID, "status": "deleted"})
}

// BackupNow starts an immediate backup of a VM.
// Body: {"vmid": 100, "storage": "local", "mode": "snapshot"}
// mode: snapshot|suspend|stop (default snapshot)
// Returns UPID for status polling via /tasks/:upid/status.
func (h *ProxmoxHandler) BackupNow(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	var req struct {
		VMID    int    `json:"vmid" binding:"required"`
		Storage string `json:"storage"`
		Mode    string `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	upid, err := cli.BackupNow(c.Request.Context(), node, req.VMID, req.Storage, req.Mode, nil)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"vmid":  req.VMID,
		"node":  node,
		"upid":  upid,
		"status": "started",
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/%s/tasks/%s/status", host.ID.String(), node, upid),
	})
}