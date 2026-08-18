// Tier 4 S2: Storage (block devices, mounts, SMART, ZFS).
//
//   GET /api/v1/admin/storage?connection_id=X
//
// Runs multiple commands in one round-trip:
//   - lsblk -Jp          → block devices + partitions
//   - findmnt -J         → mount points
//   - df -h --output=json → filesystem usage (optional)
//   - smartctl --scan    → list of SMART-capable disks (optional)
//
// All are JSON-shaped so we can parse them locally.
package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
)

// lsblkOutput is the JSON shape of `lsblk -Jp`.
type lsblkOutput struct {
	Blockdevices []lsblkDevice `json:"blockdevices"`
}
type lsblkDevice struct {
	Name       string        `json:"name"`
	Size       int64         `json:"size"`
	Rota       bool          `json:"rota"`   // true = HDD, false = SSD
	Type       string        `json:"type"`   // disk / part / rom / lvm / raid
	Model      string        `json:"model"`
	Serial     string        `json:"serial"`
	Mountpoint string        `json:"mountpoint"`
	FSType     string        `json:"fstype"`
	Children   []lsblkDevice `json:"children,omitempty"`
}

// findmntOutput is the JSON shape of `findmnt -J`.
type findmntOutput struct {
	Filesystems []findmntEntry `json:"filesystems"`
}
type findmntEntry struct {
	Target     string `json:"target"`
	Source     string `json:"source"`
	FSType     string `json:"fstype"`
	Options    string `json:"options"`
	Size       int64  `json:"size"`       // bytes
	Used       int64  `json:"used"`       // bytes
	Available  int64  `json:"avail"`      // bytes
	UsePercent string `json:"usepercent"` // e.g. "23%"
}

// ListStorage returns storage info: block devices + mount points + filesystem usage.
func (h *AdminHandler) ListStorage(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	// 1. Block devices
	var blockdevs lsblkOutput
	var blockdevErr error
	if r, err := h.SSHExec(c, cid, "lsblk -Jp 2>/dev/null", 15000); err == nil && r != nil {
		_ = json.Unmarshal([]byte(r.Stdout), &blockdevs)
		blockdevErr = nil
	}

	// 2. Mounts
	var mounts findmntOutput
	var mountErr error
	if r, err := h.SSHExec(c, cid, "findmnt -J -o TARGET,SOURCE,FSTYPE,OPTIONS,SIZE,USED,AVAIL,USEPERCENT 2>/dev/null", 15000); err == nil && r != nil {
		_ = json.Unmarshal([]byte(r.Stdout), &mounts)
		mountErr = nil
	}

	// 3. SMART scan (optional - may not be installed)
	smartOut, _ := h.SSHExec(c, cid, "which smartctl >/dev/null 2>&1 && smartctl --scan 2>/dev/null || echo ''", 10000)
	if smartOut == nil {
		smartOut = &sshExecResponse{}
	}

	// Build response
	resp := gin.H{
		"block_devices":   []lsblkDevice{},
		"mounts":          []findmntEntry{},
		"smart_available": false,
		"smart_devices":   []string{},
	}
	if blockdevErr == nil {
		resp["block_devices"] = blockdevs.Blockdevices
	} else {
		resp["warning_blockdevs"] = blockdevErr.Error()
	}
	if mountErr == nil {
		resp["mounts"] = mounts.Filesystems
	} else {
		resp["warning_mounts"] = mountErr.Error()
	}
	if smartOut.ExitCode == 0 && smartOut.Stdout != "" {
		resp["smart_available"] = true
		resp["smart_devices"] = splitLines(smartOut.Stdout)
	}

	c.JSON(200, resp)
}

// splitLines splits on newlines and drops empty lines.
func splitLines(s string) []string {
	out := []string{}
	start := 0
	for i, ch := range s {
		if ch == '\n' {
			line := s[start:i]
			if line != "" {
				out = append(out, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		if tail := s[start:]; tail != "" {
			out = append(out, tail)
		}
	}
	return out
}
