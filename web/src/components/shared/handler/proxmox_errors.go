package handler

import (
	"errors"
	"fmt"
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
	if errors.Is(err, proxmox.ErrNotFound) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "configuration error") ||
		strings.Contains(msg, "no such vm") ||
		(strings.Contains(msg, "vmid") && strings.Contains(msg, "does not exist")) ||
		(strings.Contains(msg, "ct") && strings.Contains(msg, "not running")) ||
		(strings.Contains(msg, "configuration file") && strings.Contains(msg, "does not exist"))
}

// normalizeProxmoxError maps client sentinels and legacy PVE error strings
// to the platform error catalog before they reach the HTTP responder.
func normalizeProxmoxError(err error) error {
	if isProxmoxNotFound(err) {
		return fmt.Errorf("%w: remote resource not found", kernel.ErrNotFound)
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "http 400") || strings.Contains(msg, "parameter verification failed") || strings.Contains(msg, "invalid format") || strings.Contains(msg, "undefined value") || strings.Contains(msg, "does not look like a valid user name") {
		return fmt.Errorf("%w: %s", kernel.ErrBadRequest, err.Error())
	}
	if strings.Contains(msg, "http 403") || strings.Contains(msg, "forbidden") || strings.Contains(msg, "permission check failed") {
		return fmt.Errorf("%w: %s", kernel.ErrForbidden, err.Error())
	}
	return err
}

// respondProxmoxError converts unsupported legacy PVE operations to a
// successful capability response and maps all other PVE errors to the
// platform error catalog.
func respondProxmoxError(c *gin.Context, err error) {
	if errors.Is(err, proxmox.ErrNotSupported) {
		kernel.RespondOK(c, gin.H{"supported": false})
		return
	}
	kernel.RespondError(c, normalizeProxmoxError(err))
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
		kernel.RespondError(c, normalizeProxmoxError(err))
		return true
	}
	kernel.RespondOK(c, out)
	return true
}
