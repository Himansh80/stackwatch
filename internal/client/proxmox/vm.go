package proxmox

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// VMStatus constants for the status/* endpoints.
const (
	StatusStart    = "start"
	StatusStop     = "stop"
	StatusReboot   = "reboot"
	StatusShutdown = "shutdown"
	StatusSuspend  = "suspend"
	StatusResume   = "resume"
)

// VMStatus sends a lifecycle action to a VM.
//
// POST /api2/json/nodes/{node}/qemu/{vmid}/status/{action}
func (c *Client) VMStatus(ctx context.Context, node string, vmid int, action string, force bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/%s",
		url.PathEscape(node), vmid, url.PathEscape(action))
	form := url.Values{}
	// Note: 'force' is NOT a valid query param for /status/* endpoints in
	// recent Proxmox versions. It's silently accepted but the schema
	// rejects it with 400. We omit it.
	_ = force
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	task, err := c.doTask(req)
	if err != nil {
		return "", err
	}
	return task, nil
}

// LXCStatus sends a lifecycle action to an LXC container.
//
// POST /api2/json/nodes/{node}/lxc/{vmid}/status/{action}
func (c *Client) LXCStatus(ctx context.Context, node string, vmid int, action string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/status/%s",
		url.PathEscape(node), vmid, url.PathEscape(action))
	form := url.Values{}
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}

// LXCSpec describes inputs for LXC container creation/update.
type LXCSpec struct {
	VMID         int
	Hostname     string
	MemoryMB     int
	Cores        int
	DiskGB       int
	Storage      string
	Password     string  // root password (optional)
	OSTemplate   string  // e.g. "local:vztmpl/ubuntu-22.04-standard_22.04-1_amd64.tar.zst"
	Bridge       string
	NetName      string  // network interface name (default: eth0)
	IPConfig     string  // e.g. "dhcp" or "10.0.0.5/24,gw=10.0.0.1"
	Unprivileged bool
	Description  string
}

// CreateLXC creates a new LXC container.
func (c *Client) CreateLXC(ctx context.Context, node string, spec LXCSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc", url.PathEscape(node))
	form := url.Values{}
	form.Set("vmid", strconv.Itoa(spec.VMID))
	if spec.Hostname != "" {
		form.Set("hostname", spec.Hostname)
	}
	if spec.MemoryMB > 0 {
		form.Set("memory", strconv.Itoa(spec.MemoryMB))
	}
	if spec.Cores > 0 {
		form.Set("cores", strconv.Itoa(spec.Cores))
	}
	if spec.DiskGB > 0 {
		size := fmt.Sprintf("%d", spec.DiskGB)
		if spec.Storage != "" {
			form.Set("rootfs", spec.Storage+":"+size)
		} else {
			form.Set("rootfs", size)
		}
	}
	if spec.Storage != "" {
		form.Set("storage", spec.Storage)
	}
	if spec.Password != "" {
		form.Set("password", spec.Password)
	}
	if spec.OSTemplate != "" {
		form.Set("ostemplate", spec.OSTemplate)
	}
	if spec.Bridge != "" {
		netName := spec.NetName
		if netName == "" {
			netName = "net0"
		}
		form.Set(netName, fmt.Sprintf("name=eth0,bridge=%s,ip=dhcp", spec.Bridge))
		// Note: net0,gw is NOT a valid property — IP config goes in ipconfig0.
	}
	if spec.IPConfig != "" {
		ipName := spec.NetName
		if ipName == "" {
			ipName = "net0"
		}
		form.Set("ipconfig"+ipName[len("net"):], spec.IPConfig)
	}
	if spec.Unprivileged {
		form.Set("unprivileged", "1")
	}
	if spec.Description != "" {
		form.Set("description", spec.Description)
	}

	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}

// GetLXCConfig returns the current configuration of an LXC container.
func (c *Client) GetLXCConfig(ctx context.Context, node string, vmid int) (map[string]any, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/config",
		url.PathEscape(node), vmid)
	cfg := map[string]any{}
	if err := c.get(ctx, path, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// UpdateLXCConfig updates editable fields on an LXC container.
func (c *Client) UpdateLXCConfig(ctx context.Context, node string, vmid int, spec LXCSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d/config",
		url.PathEscape(node), vmid)
	form := url.Values{}
	if spec.Hostname != "" {
		form.Set("hostname", spec.Hostname)
	}
	if spec.MemoryMB > 0 {
		form.Set("memory", strconv.Itoa(spec.MemoryMB))
	}
	if spec.Cores > 0 {
		form.Set("cores", strconv.Itoa(spec.Cores))
	}
	if spec.Description != "" {
		form.Set("description", spec.Description)
	}
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "PUT", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}

// DeleteLXC removes an LXC container.
func (c *Client) DeleteLXC(ctx context.Context, node string, vmid int, force bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/lxc/%d", url.PathEscape(node), vmid)
	u := c.baseURL + "/api2/json" + path
	_ = force
	u += "?purge=1"
	req, err := http.NewRequestWithContext(ctx, "DELETE", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")
	return c.doTask(req)
}

// VMSpec describes inputs for VM creation/update.
type VMSpec struct {
	VMID     int
	Name     string
	MemoryMB int
	Cores    int
	Sockets  int
	DiskGB   int
	Storage  string
	Bridge   string
	ISO      string
	Boot     string
}

// CreateVM creates a new QEMU VM. Returns the task ID (Proxmox creates async).
//
// POST /api2/json/nodes/{node}/qemu
func (c *Client) CreateVM(ctx context.Context, node string, spec VMSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu", url.PathEscape(node))
	form := url.Values{}
	form.Set("vmid", strconv.Itoa(spec.VMID))
	if spec.Name != "" {
		form.Set("name", spec.Name)
	}
	if spec.MemoryMB > 0 {
		form.Set("memory", strconv.Itoa(spec.MemoryMB))
	}
	if spec.Cores > 0 {
		form.Set("cores", strconv.Itoa(spec.Cores))
	}
	if spec.Sockets > 0 {
		form.Set("sockets", strconv.Itoa(spec.Sockets))
	}
	if spec.DiskGB > 0 {
		size := fmt.Sprintf("%d", spec.DiskGB)
		if spec.Storage != "" {
			form.Set("scsi0", spec.Storage+":"+size)
		} else {
			form.Set("scsi0", size+",format=raw")
		}
	}
	if spec.Storage != "" {
		form.Set("storage", spec.Storage)
	}
	if spec.Bridge != "" {
		form.Set("net0", fmt.Sprintf("virtio,bridge=%s", spec.Bridge))
	}
	if spec.ISO != "" {
		form.Set("ide2", spec.ISO+",media=cdrom")
	}
	if spec.Boot != "" {
		form.Set("boot", spec.Boot)
	}
	form.Set("ostype", "l26") // Linux 2.6+

	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}
// DeleteVM removes a VM (returns task ID; operation is async).
//
// DELETE /api2/json/nodes/{node}/qemu/{vmid}?purge=1
func (c *Client) DeleteVM(ctx context.Context, node string, vmid int, force bool) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d", url.PathEscape(node), vmid)
	u := c.baseURL + "/api2/json" + path
	// 'force' is NOT a valid param for DELETE on /qemu/:vmid. The 'purge'
	// flag handles cleanup. We accept force for API compatibility but
	// silently drop it.
	_ = force
	u += "?purge=1"
	// DELETE on /qemu/:vmid doesn't accept a body; just DELETE with no form
	req, err := http.NewRequestWithContext(ctx, "DELETE", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")
	return c.doTask(req)
}

// VMConfig represents the editable config of a VM.
// Proxmox returns numeric fields as JSON numbers in some versions and
// strings in others — use json.Number to accept both.
type VMConfig struct {
	Name    string `json:"name"`
	Memory  any    `json:"memory"` // MB — number or string depending on Proxmox version
	Cores   any    `json:"cores"`
	Sockets any    `json:"sockets"`
	OSType  string `json:"ostype"`
	Boot    string `json:"boot"`
	Net0    string `json:"net0"`
	SCSI0   string `json:"scsi0"`
	IDE2    string `json:"ide2,omitempty"`
}

// GetVMConfig returns the current config of a VM.
func (c *Client) GetVMConfig(ctx context.Context, node string, vmid int) (*VMConfig, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config",
		url.PathEscape(node), vmid)
	cfg := &VMConfig{}
	if err := c.get(ctx, path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// UpdateVMConfig updates fields on a VM. Pass only changed fields via spec.
func (c *Client) UpdateVMConfig(ctx context.Context, node string, vmid int, spec VMSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config",
		url.PathEscape(node), vmid)
	form := url.Values{}
	if spec.Name != "" {
		form.Set("name", spec.Name)
	}
	if spec.MemoryMB > 0 {
		form.Set("memory", strconv.Itoa(spec.MemoryMB))
	}
	if spec.Cores > 0 {
		form.Set("cores", strconv.Itoa(spec.Cores))
	}
	if spec.Boot != "" {
		form.Set("boot", spec.Boot)
	}
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "PUT", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}
