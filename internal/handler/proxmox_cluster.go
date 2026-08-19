package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

func (h *ProxmoxHandler) clusterClient(c *gin.Context) (*proxmox.Client, bool) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return nil, false
	}
	return proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS), true
}

func (h *ProxmoxHandler) ListHAResources(c *gin.Context) {
	cli, ok := h.clusterClient(c)
	if !ok {
		return
	}
	items, err := cli.GetHAResources(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"resources": items, "total": len(items)})
}

func (h *ProxmoxHandler) GetHAStatus(c *gin.Context) {
	cli, ok := h.clusterClient(c)
	if !ok {
		return
	}
	status, err := cli.GetHAStatus(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *ProxmoxHandler) JoinCluster(c *gin.Context) {
	cli, ok := h.clusterClient(c)
	if !ok {
		return
	}
	var fields map[string]string
	if err := c.ShouldBindJSON(&fields); err != nil || fields["hostname"] == "" || fields["password"] == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if err := cli.JoinCluster(c.Request.Context(), fields); err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "joined": true})
}

func (h *ProxmoxHandler) LeaveCluster(c *gin.Context) {
	cli, ok := h.clusterClient(c)
	if !ok {
		return
	}
	if err := cli.LeaveCluster(c.Request.Context()); err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "left": true})
}

func (h *ProxmoxHandler) MigrateResource(c *gin.Context) {
	cli, ok := h.clusterClient(c)
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
	var req struct {
		Target string `json:"target"`
		Online bool   `json:"online"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Target == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	task, err := cli.Migrate(c.Request.Context(), c.Param("node"), kind, vmid, req.Target, req.Online)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"ok": true, "task": task, "target": req.Target})
}
