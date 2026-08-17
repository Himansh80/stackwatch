package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// Pool represents a Proxmox resource pool.
type Pool struct {
	PoolID   string `json:"poolid"`        // unique pool name
	Comment  string `json:"comment,omitempty"`
	Members  []PoolMember `json:"members,omitempty"` // only on GET single
}

// PoolMember is a VM/LXC or storage inside a pool.
type PoolMember struct {
	ID         string `json:"id"`           // unique member id
	Type       string `json:"type"`         // "qemu", "lxc", "storage"
	VMID       int    `json:"vmid,omitempty"`
	Storage    string `json:"storage,omitempty"`
	Content    string `json:"content,omitempty"`
	Pool       string `json:"pool,omitempty"`
}

// ListPools returns all resource pools.
func (c *Client) ListPools(ctx context.Context) ([]Pool, error) {
	var out []Pool
	if err := c.get(ctx, "/pools", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPool returns one pool with its members.
func (c *Client) GetPool(ctx context.Context, poolID string) (Pool, error) {
	var out Pool
	path := "/pools/" + url.PathEscape(poolID)
	if err := c.get(ctx, path, &out); err != nil {
		return out, err
	}
	return out, nil
}

// CreatePool creates a new resource pool.
func (c *Client) CreatePool(ctx context.Context, poolID, comment string) (string, error) {
	form := url.Values{}
	form.Set("poolid", poolID)
	if comment != "" {
		form.Set("comment", comment)
	}
	if err := c.postForm(ctx, "/pools", form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// UpdatePool modifies a pool's comment.
func (c *Client) UpdatePool(ctx context.Context, poolID, comment string) (string, error) {
	form := url.Values{}
	form.Set("comment", comment)
	if err := c.putForm(ctx, "/pools/"+url.PathEscape(poolID), form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeletePool removes a pool (must be empty).
func (c *Client) DeletePool(ctx context.Context, poolID string) (string, error) {
	if err := c.deleteForm(ctx, "/pools/"+url.PathEscape(poolID)); err != nil {
		return "", err
	}
	return "", nil
}

// ClusterResource is an aggregated view of nodes/VMs/LXC/storage/network.
type ClusterResource struct {
	ResourceType string  `json:"type"`          // "node", "qemu", "lxc", "storage", "network"
	ID           string  `json:"id"`            // unique id (e.g. "qemu/100")
	Node         string  `json:"node,omitempty"`
	Status       string  `json:"status,omitempty"`  // "running", "stopped"
	VMID         int     `json:"vmid,omitempty"`
	Name         string  `json:"name,omitempty"`
	CPU          float64 `json:"cpu,omitempty"`    // 0.0-1.0 fraction
	Mem          int64   `json:"mem,omitempty"`   // bytes used
	MaxMem       int64   `json:"maxmem,omitempty"`
	Disk         int64   `json:"disk,omitempty"`
	MaxDisk      int64   `json:"maxdisk,omitempty"`
	Uptime       int64   `json:"uptime,omitempty"`
	Template     int     `json:"template,omitempty"`
	Pool         string  `json:"pool,omitempty"`
	Storage      string  `json:"storage,omitempty"`
	Content      string  `json:"content,omitempty"`
	Total        int64   `json:"total,omitempty"`
	Used         int64   `json:"used,omitempty"`
	Avail        int64   `json:"avail,omitempty"`
	Level        string  `json:"level,omitempty"`
	IP           string  `json:"ip,omitempty"`
}

// ListClusterResources returns aggregated cluster resources.
// Optional type filter: "vm", "lxc", "node", "storage", "network" or "" for all.
func (c *Client) ListClusterResources(ctx context.Context, typeFilter string) ([]ClusterResource, error) {
	path := "/cluster/resources"
	if typeFilter != "" {
		path += "?type=" + url.QueryEscape(typeFilter)
	}
	var out []ClusterResource
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterStatusNode describes one node in /cluster/status.
type ClusterStatusNode struct {
	Name   string `json:"name"`
	Type   string `json:"type"`     // "node"
	Level  string `json:"level"`    // "" for normal nodes
	IP     string `json:"ip,omitempty"`
	ID     string `json:"id"`
	Local  int    `json:"local,omitempty"`
	Online int    `json:"online,omitempty"`
	NodeID int    `json:"nodeid"`
}

// ClusterQuorumInfo describes cluster quorum status.
type ClusterQuorumInfo struct {
	Name   string             `json:"name"`
	Type   string             `json:"type"`
	Quorate int              `json:"quorate,omitempty"`
	Version int              `json:"version,omitempty"`
	Nodes  []ClusterStatusNode `json:"nodes,omitempty"`
}

// GetClusterStatus returns cluster status including nodes + quorum.
func (c *Client) GetClusterStatus(ctx context.Context) ([]ClusterStatusNode, error) {
	var out []ClusterStatusNode
	if err := c.get(ctx, "/cluster/status", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetClusterInfo returns cluster name + ID (single object, not array).
type ClusterInfo struct {
	Name  string `json:"name"`
	IPID  string `json:"id"`
	Nodes int    `json:"max_nodes"`
	Quorate int   `json:"quorate"`
	Version int  `json:"version"`
}

// GetClusterInfo returns basic cluster information.
func (c *Client) GetClusterInfo(ctx context.Context) (ClusterInfo, error) {
	var out ClusterInfo
	if err := c.get(ctx, "/cluster/info", &out); err != nil {
		return out, err
	}
	return out, nil
}

// _ to keep fmt import
var _ = fmt.Sprintf