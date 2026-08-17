package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// DownloadStorageFile returns a URL the caller can use to download a file
// from Proxmox storage. We don't proxy the actual binary stream — instead
// we return the URL + auth hint, since Proxmox needs the API token directly.
//
// Proxmox URL: GET /api2/json/nodes/{node}/storage/{storage}/download/{volume}
// The :volume param uses Gin catch-all to allow paths like "vztmpl/file.tar.zst".
func (h *ProxmoxHandler) DownloadStorageFile(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	storage := c.Param("storage")
	// Gin catch-all (*) includes the leading slash — strip it.
	volume := strings.TrimPrefix(c.Param("volume"), "/")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	urlStr := cli.DownloadStorageFile(node, storage, volume)
	kernel.RespondOK(c, gin.H{
		"download_url": urlStr,
		"auth_header":  "Authorization: " + host.APIToken,
		"storage":      storage,
		"volume":       volume,
		"node":         node,
		"instructions": "Curl with -H auth_header to download file directly from Proxmox",
	})
}

// GetStorageUploadURL returns the URL to POST multipart upload to.
func (h *ProxmoxHandler) GetStorageUploadURL(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	storage := c.Param("storage")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	urlStr := cli.UploadStorageFileToURL(node, storage)
	kernel.RespondOK(c, gin.H{
		"upload_url":   urlStr,
		"auth_header":  "Authorization: " + host.APIToken,
		"storage":      storage,
		"node":         node,
		"instructions": "POST multipart/form-data with file field 'content' (volume param required): curl -F content=@file.iso -F filename=file.iso -H auth_header upload_url",
	})
}

// ListISCSI returns iSCSI targets configured on a node.
func (h *ProxmoxHandler) ListISCSI(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	targets, err := cli.ListISCSI(c.Request.Context(), node)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"targets": targets, "total": len(targets)})
}