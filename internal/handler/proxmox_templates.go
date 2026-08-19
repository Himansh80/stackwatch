package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

func (h *ProxmoxHandler) templateClient(c *gin.Context) (*proxmox.Client, string, bool) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return nil, "", false
	}
	node := c.Param("node")
	if node == "" {
		node = c.Query("node")
	}
	if node == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return nil, "", false
	}
	return proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS), node, true
}

// ListTemplates returns ISO/template content available on a storage backend.
func (h *ProxmoxHandler) ListTemplates(c *gin.Context) {
	cli, node, ok := h.templateClient(c)
	if !ok {
		return
	}
	storage := c.Query("storage")
	if storage == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	items, err := cli.ListTemplates(c.Request.Context(), node, storage)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"templates": items, "total": len(items), "storage": storage, "node": node})
}

type templateVMRequest struct {
	Kind      string            `json:"kind"`
	CloudInit map[string]string `json:"cloud_init"`
}

// MarkTemplate converts an existing QEMU/LXC guest into a template.
func (h *ProxmoxHandler) MarkTemplate(c *gin.Context) {
	cli, node, ok := h.templateClient(c)
	if !ok {
		return
	}
	kind := c.Param("kind")
	if kind != "qemu" && kind != "lxc" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	vmid, err := parseVMID(c.Param("vmid"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if _, err = cli.MarkVMTemplate(c.Request.Context(), node, vmid, kind); err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "template": true, "kind": kind, "vmid": vmid})
}

// ApplyCloudInit writes supported cloud-init fields to a guest config.
func (h *ProxmoxHandler) ApplyCloudInit(c *gin.Context) {
	cli, node, ok := h.templateClient(c)
	if !ok {
		return
	}
	kind := c.Param("kind")
	if kind != "qemu" && kind != "lxc" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	vmid, err := parseVMID(c.Param("vmid"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil || len(req) == 0 {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if err := cli.ApplyCloudInit(c.Request.Context(), node, kind, vmid, req); err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "cloud_init": true, "kind": kind, "vmid": vmid})
}
