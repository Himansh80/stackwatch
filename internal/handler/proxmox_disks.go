package handler

import (
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// CreateZFSPoolRequest is the JSON body for POST /disks/zfs.
type CreateZFSPoolRequest struct {
	Name      string `json:"name" binding:"required,min=1,max=64"`
	RaidLevel string `json:"raidlevel" binding:"required,oneof=single mirror raid10 raidz raidz2 raidz3"`
	Devices   string `json:"devices" binding:"required"`  // comma-separated, e.g. "/dev/sdb,/dev/sdc"
}

// DestroyZFSPoolRequest includes name + confirmation for destructive operation.
type DestroyZFSPoolRequest struct {
	Name string `json:"name" binding:"required"`
}

// ListDisks returns physical disks on a node.
func (h *ProxmoxHandler) ListDisks(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	disks, err := cli.ListDisks(c.Request.Context(), node)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"disks": disks, "total": len(disks)})
}

// ListZFSPools returns all ZFS pools on a node.
func (h *ProxmoxHandler) ListZFSPools(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	pools, err := cli.ListZFSPools(c.Request.Context(), node)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"pools": pools, "total": len(pools)})
}

// CreateZFSPool creates a new ZFS pool.
func (h *ProxmoxHandler) CreateZFSPool(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	var req CreateZFSPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	// Validate devices string: each device must be /dev/sdX, /dev/nvme0n1, etc.
	devRe := regexp.MustCompile(`^/dev/[a-zA-Z0-9]+$`)
	seen := map[string]bool{}
	devices := ""
	for _, dev := range splitCSV(req.Devices) {
		if !devRe.MatchString(dev) {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		if seen[dev] {
			kernel.RespondError(c, kernel.ErrBadRequest)
			return
		}
		seen[dev] = true
		if devices != "" {
			devices += ","
		}
		devices += dev
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.CreateZFSPool(c.Request.Context(), node, req.Name, req.RaidLevel, devices); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondCreated(c, gin.H{"created": true, "name": req.Name, "raidlevel": req.RaidLevel})
}

// DestroyZFSPool destroys a ZFS pool (destructive).
func (h *ProxmoxHandler) DestroyZFSPool(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	name := c.Param("name")
	if name == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DestroyZFSPool(c.Request.Context(), node, name); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"deleted": true, "name": name})
}

// splitCSV is a small helper to split comma-separated values without
// pulling in the entire strings package.
func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}