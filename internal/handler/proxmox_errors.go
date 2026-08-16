package handler

import (
	"strings"
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
