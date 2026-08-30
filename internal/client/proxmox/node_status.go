// Package proxmox — Tier 14 fill-in: node status + node services.
// These use the generic get() helper, so only types + wrappers needed.
package proxmox

import (
	"context"
	"fmt"
)

// NodeStatus is the response from GET /nodes/{node}/status.
// Only fields the dashboard actually uses are mapped.
type NodeStatus struct {
	Cpu         float64  `json:"cpu"`
	MemTotal    int64    `json:"memory.total"`
	MemUsed     int64    `json:"memory.used"`
	RootfsAvai  int64    `json:"rootfs.avail"`
	RootfsTotal int64    `json:"rootfs.total"`
	LoadAvg     []string `json:"loadavg"`
	Uptime      int64    `json:"uptime"`
	CpuCount    int64    `json:"cpuinfo.cpus"`
	CpuModel    string   `json:"cpuinfo.model"`
}

// NodeService is one systemd unit returned by /nodes/{node}/services.
type NodeService struct {
	Service  string `json:"service"`
	Name     string `json:"name"`
	State    string `json:"state"`    // running / stopped / failed
	Active   string `json:"active"`   // active / inactive
	SubState string `json:"substate"` // running / dead / listening / etc.
	Desc     string `json:"desc"`
}

// GetNodeStatus returns high-level node metrics.
func (c *Client) GetNodeStatus(ctx context.Context, node string) (*NodeStatus, error) {
	// Proxmox nests values deeply; simplest is to grab the raw JSON object
	// and re-serialize into a flat struct. We write our own minimal decoder.
	path := fmt.Sprintf("/nodes/%s/status", node)
	var raw map[string]any
	if err := c.get(ctx, path, &raw); err != nil {
		return nil, err
	}
	out := &NodeStatus{}
	if v, ok := raw["cpu"].(float64); ok {
		out.Cpu = v
	}
	if v, ok := raw["uptime"].(float64); ok {
		out.Uptime = int64(v)
	}
	if mem, ok := raw["memory"].(map[string]any); ok {
		if v, ok := mem["total"].(float64); ok {
			out.MemTotal = int64(v)
		}
		if v, ok := mem["used"].(float64); ok {
			out.MemUsed = int64(v)
		}
	}
	if rootfs, ok := raw["rootfs"].(map[string]any); ok {
		if v, ok := rootfs["total"].(float64); ok {
			out.RootfsTotal = int64(v)
		}
		if v, ok := rootfs["avail"].(float64); ok {
			out.RootfsAvai = int64(v)
		}
	}
	if la, ok := raw["loadavg"].([]any); ok {
		out.LoadAvg = make([]string, 0, len(la))
		for _, x := range la {
			if s, ok := x.(string); ok {
				out.LoadAvg = append(out.LoadAvg, s)
			}
		}
	}
	if ci, ok := raw["cpuinfo"].(map[string]any); ok {
		if v, ok := ci["cpus"].(float64); ok {
			out.CpuCount = int64(v)
		}
		if v, ok := ci["model"].(string); ok {
			out.CpuModel = v
		}
	}
	return out, nil
}

// ListNodeServices returns systemd services for a node.
func (c *Client) ListNodeServices(ctx context.Context, node string) ([]NodeService, error) {
	var out []NodeService
	err := c.get(ctx, fmt.Sprintf("/nodes/%s/services", node), &out)
	return out, err
}
