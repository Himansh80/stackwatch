package handler

import (
	"strings"
)

// isProxmoxNotFound returns true if err looks like Proxmox's "no such VM" 500.
func isProxmoxNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "configuration error") ||
		strings.Contains(msg, "no such vm") ||
		strings.Contains(msg, "vmid") && strings.Contains(msg, "does not exist")
}
