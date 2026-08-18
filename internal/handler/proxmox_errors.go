package handler

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// isProxmoxNotFound returns true if err looks like Proxmox's "no such VM/CT" 500.
func isProxmoxNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "configuration error") ||
		strings.Contains(msg, "no such vm") ||
		strings.Contains(msg, "vmid") && strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "ct") && strings.Contains(msg, "not running") ||
		strings.Contains(msg, "configuration file") && strings.Contains(msg, "does not exist")
}

// handleProxmoxCall runs fn and writes the response. If fn returns
// proxmox.ErrNotSupported (HTTP 501 from PVE), responds with 200 +
// {"supported": false} so the UI can render the section as unavailable
// instead of a hard error. Returns true if it handled the response.
func handleProxmoxCall(c *gin.Context, fn func() (any, error)) bool {
	out, err := fn()
	if errors.Is(err, proxmox.ErrNotSupported) {
		kernel.RespondOK(c, gin.H{"supported": false})
		return true
	}
	if err != nil {
		kernel.RespondError(c, err)
		return true
	}
	kernel.RespondOK(c, out)
	return true
}
