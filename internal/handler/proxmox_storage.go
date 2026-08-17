package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// CreateStorageRequest is the JSON body for POST /proxmox/hosts/:id/storage.
type CreateStorageRequest struct {
	Name        string   `json:"name" binding:"required,min=1,max=32"`
	StorageType string   `json:"type" binding:"required,oneof=dir lvm lvmthin zfspool nfs cifs cephfs rbd pbs"`
	Content     []string `json:"content"`
	Path        string   `json:"path"`
	Server      string   `json:"server"`
	Export      string   `json:"export"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	VGName      string   `json:"vgname"`
	ThinPool    string   `json:"thinpool"`
	Pool        string   `json:"pool"`
	Nodes       []string `json:"nodes"`
}

// ListStorageEntries returns all storage entries across the cluster.
func (h *ProxmoxHandler) ListStorageEntries(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	entries, err := cli.ListStorageEntries(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"storages": entries, "total": len(entries)})
}

// CreateStorage registers a new storage backend on the cluster.
func (h *ProxmoxHandler) CreateStorage(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req CreateStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	spec := proxmox.StorageSpec{
		Name:        req.Name,
		StorageType: proxmox.StorageType(req.StorageType),
		Content:     req.Content,
		Path:        req.Path,
		Server:      req.Server,
		Export:      req.Export,
		Username:    req.Username,
		Password:    req.Password,
		VGName:      req.VGName,
		ThinPool:    req.ThinPool,
		Pool:        req.Pool,
		Nodes:       req.Nodes,
	}
	task, err := cli.CreateStorage(c.Request.Context(), spec)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{
		"task": task, "name": req.Name, "type": req.StorageType,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/router/tasks/%s", host.ID.String(), task),
	})
}

// DeleteStorage removes a storage registration (not the underlying data).
func (h *ProxmoxHandler) DeleteStorage(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	if name == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	task, err := cli.DeleteStorage(c.Request.Context(), name)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"task": task, "deleted": true, "name": name,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/router/tasks/%s", host.ID.String(), task),
	})
}

// ListStorageContent returns the volumes (ISOs, templates, backups) in a storage.
// Optional `content` query param filters by type: iso, vztmpl, backup, rootdir, images.
func (h *ProxmoxHandler) ListStorageContent(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	storage := c.Param("storage")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	contentFilter := c.Query("content")
	content, err := cli.ListContent(c.Request.Context(), node, storage, contentFilter)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"content": content, "total": len(content), "storage": storage, "filter": contentFilter})
}

// DeleteStorageContent removes a specific volume from a storage.
func (h *ProxmoxHandler) DeleteStorageContent(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	storage := c.Param("storage")
	// Volume is colon-separated like "local:iso/debian-12.iso" — needs full path
	volume := c.Param("volume")
	// The volume param might contain slashes; gin doesn't capture those in :param.
	// Instead, get from query param or full path.
	rawVol := c.Query("volume")
	if rawVol != "" {
		volume = rawVol
	}
	if volume == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	task, err := cli.DeleteContent(c.Request.Context(), node, storage, volume)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"task": task, "deleted": true, "volume": volume,
		"status_url": fmt.Sprintf("/api/v1/proxmox/hosts/%s/nodes/%s/tasks/%s", host.ID.String(), node, task),
	})
}

// suppress unused import
var _ = strings.Contains
