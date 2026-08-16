package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// DiskInfo describes a block device (physical disk or partition).
//
// Proxmox returns most fields as strings (vendor, serial, by-id-link),
// but osdid is sometimes int (-1 means "no OS disk ID").
type DiskInfo struct {
	DevPath    string `json:"devpath"`           // e.g. /dev/sda
	ByIDLink   string `json:"by_id_link,omitempty"`  // stable device ID
	OSType     string `json:"ostype,omitempty"`
	Model      string `json:"model,omitempty"`
	Serial     string `json:"serial,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	Type       string `json:"type,omitempty"`       // hdd, ssd, nvme
	Size       uint64 `json:"size,omitempty"`       // bytes
	WWN        string `json:"wwn,omitempty"`
	Health     string `json:"health,omitempty"`     // PASSED, FAILED, UNKNOWN
	RPM        int    `json:"rpm,omitempty"`        // HDD rotation speed (0 for SSD)
	Used        string `json:"used,omitempty"`         // "used" by storage as partition/LVM/ZFS, or "none"
	Wearout    int    `json:"wearout,omitempty"`   // SSD wear percentage (100 = new, 0 = end of life)
	GPT        int    `json:"gpt,omitempty"`        // 1 = has GPT partition table
	OSDID      int    `json:"osdid,omitempty"`      // OS disk ID (-1 = none)
	OSDIDList  interface{} `json:"osdid-list,omitempty"` // can be null or string
	VendorPlain string `json:"vendor_plain,omitempty"`
}

// ZFSPool is a ZFS pool on a Proxmox node.
type ZFSPool struct {
	Name   string `json:"name"`            // pool name
	State  string `json:"state"`           // ONLINE, DEGRADED, OFFLINE, REMOVED, UNAVAIL, FAULTED
	Status string `json:"status,omitempty"` // detailed status
	Size   uint64 `json:"size,omitempty"`   // total bytes
	Free   uint64 `json:"free,omitempty"`   // free bytes
	Alloc  uint64 `json:"alloc,omitempty"`  // allocated bytes
	Frag   string `json:"frag,omitempty"`  // fragmentation percentage
	Dedup  string `json:"dedup,omitempty"` // dedup ratio
}

// ListDisks returns physical disk info (without partition info).
// Equivalent to `lsblk` from the host's perspective.
func (c *Client) ListDisks(ctx context.Context, node string) ([]DiskInfo, error) {
	path := fmt.Sprintf("/nodes/%s/disks/list", url.PathEscape(node))
	var out []DiskInfo
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListZFSPools returns all ZFS pools on a node.
func (c *Client) ListZFSPools(ctx context.Context, node string) ([]ZFSPool, error) {
	path := fmt.Sprintf("/nodes/%s/disks/zfs", url.PathEscape(node))
	var out []ZFSPool
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateZFSPool creates a new ZFS pool (sync).
// devices is a comma-separated list of disk paths (e.g. "/dev/sdb,/dev/sdc").
// raidlevel is one of: single, mirror, raid10, raidz, raidz2, raidz3.
func (c *Client) CreateZFSPool(ctx context.Context, node, name, raidlevel, devices string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/disks/zfs", url.PathEscape(node))
	form := url.Values{}
	form.Set("name", name)
	if raidlevel != "" {
		form.Set("raidlevel", raidlevel)
	}
	form.Set("devices", devices)
	if err := c.postForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DestroyZFSPool destroys a ZFS pool (sync, DESTRUCTIVE — requires explicit name confirmation).
func (c *Client) DestroyZFSPool(ctx context.Context, node, name string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/disks/zfs/%s",
		url.PathEscape(node), url.PathEscape(name))
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

// InitiateFSTHIN creates a thin pool (LVM) on a volume group.
// Use for advanced storage backends.
func (c *Client) InitiateFST(ctx context.Context, node, name, vgname string, thinpoolSizeGB int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/disks/initgpt", url.PathEscape(node))
	_ = path
	_ = ctx
	return "", fmt.Errorf("not yet implemented")
}

// DirectoryCreate creates a directory on the host filesystem for storage.
// Returns the created directory path.
func (c *Client) DirectoryCreate(ctx context.Context, node, parent, name string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/disks/directory", url.PathEscape(node))
	form := url.Values{}
	if parent != "" {
		form.Set("parent", parent)
	}
	if name != "" {
		form.Set("name", name)
	}
	if err := c.postForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}