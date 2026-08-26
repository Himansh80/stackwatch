// Tier 14 Phase 14.3 — Proxmox VM snapshot CRUD client methods.
//
// Proxmox API: /nodes/{node}/qemu/{vmid}/snapshot[/snapname[/rollback]]
package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// VMSnapshot represents a single VM snapshot returned by Proxmox.
type VMSnapshot struct {
	Name        string `json:"name"`               // snapshot name
	Date        string `json:"date,omitempty"`     // ISO-ish date from Proxmox
	VMState     bool   `json:"vmstate,omitempty"`  // whether RAM state was captured
	Description string `json:"description,omitempty"`
	Parent      string `json:"parent,omitempty"`
	Size        int64  `json:"size,omitempty"`     // approximate on-disk size in bytes
}

// VMSnapshotCreateSpec is the body for POST .../snapshot.
type VMSnapshotCreateSpec struct {
	Snapname    string `json:"snapname" binding:"required"`
	Description string `json:"description,omitempty"`
	VMState     bool   `json:"vmstate,omitempty"`
}

// ListVMSnapshots returns all snapshots for a given VM.
func (c *Client) ListVMSnapshots(ctx context.Context, node string, vmid int) ([]VMSnapshot, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", url.PathEscape(node), vmid)
	var out []VMSnapshot
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateVMSnapshot creates a new snapshot. Returns the UPID.
func (c *Client) CreateVMSnapshot(ctx context.Context, node string, vmid int, spec VMSnapshotCreateSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/snapshot", url.PathEscape(node), vmid)
	form := url.Values{}
	form.Set("snapname", spec.Snapname)
	if spec.Description != "" {
		form.Set("description", spec.Description)
	}
	if spec.VMState {
		form.Set("vmstate", "1")
	}
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}

// DeleteVMSnapshot deletes a snapshot by name. Returns the UPID.
func (c *Client) DeleteVMSnapshot(ctx context.Context, node string, vmid int, snapname string, force bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s", url.PathEscape(node), vmid, url.PathEscape(snapname))
	form := url.Values{}
	if force {
		form.Set("force", "1")
	}
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "DELETE", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}

// RollbackVMSnapshot rolls the VM back to a snapshot. Returns the UPID.
func (c *Client) RollbackVMSnapshot(ctx context.Context, node string, vmid int, snapname string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/snapshot/%s/rollback", url.PathEscape(node), vmid, url.PathEscape(snapname))
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, url.Values{}, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}