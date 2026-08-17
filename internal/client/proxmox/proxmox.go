// Package proxmox is a client for the Proxmox VE REST API.
// Uses API tokens (PVEAPIToken header), supports skip-verify for self-signed.
package proxmox

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a Proxmox API client bound to one host.
type Client struct {
	baseURL    string
	apiToken   string
	verifyTLS  bool
	httpClient *http.Client
}

// NewClient creates a client. Accepts either:
//   - "PVEAPIToken=user@realm!id=uuid" (preferred)
//   - "user@realm!id=uuid" (auto-prefixed with PVEAPIToken=)
func NewClient(baseURL, apiToken string, verifyTLS bool) *Client {
	if !strings.HasPrefix(apiToken, "PVEAPIToken=") {
		apiToken = "PVEAPIToken=" + apiToken
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !verifyTLS},
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiToken:   apiToken,
		verifyTLS:  verifyTLS,
		httpClient: &http.Client{Transport: tr, Timeout: 15 * time.Second},
	}
}

// get performs an authenticated GET. Returns parsed JSON or error.
func (c *Client) get(ctx context.Context, path string, out any) error {
	u := c.baseURL + "/api2/json" + path
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("proxmox: unauthorized (check API token)")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("proxmox: HTTP %d on %s", resp.StatusCode, path)
	}
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return err
	}
	return json.Unmarshal(wrapper.Data, out)
}

// postForm executes a POST with form-encoded body and decodes data field.
// Unlike doTask, this just returns the error if any (for sync endpoints
// like /storage that return the resource, not a UPID).
func (c *Client) postForm(ctx context.Context, path string, form url.Values, out any) error {
	return c.doForm(ctx, "POST", path, form, out)
}

// putForm executes a PUT with form-encoded body.
func (c *Client) putForm(ctx context.Context, path string, form url.Values, out any) error {
	return c.doForm(ctx, "PUT", path, form, out)
}

// doForm is the shared body for postForm/putForm.
func (c *Client) doForm(ctx context.Context, method, path string, form url.Values, out any) error {
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, method, u, form, c.apiToken)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("proxmox: unauthorized")
	}
	if resp.StatusCode == 403 {
		return fmt.Errorf("proxmox: forbidden")
	}
	if resp.StatusCode == 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("proxmox: bad request: %s", string(body))
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("proxmox: HTTP %d: %s", resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	var wrapper struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return err
	}
	return json.Unmarshal(wrapper.Data, out)
}

// deleteForm executes a DELETE (no body) and ignores the result.
// Returns error if HTTP >= 400.
func (c *Client) deleteForm(ctx context.Context, path string) error {
	u := c.baseURL + "/api2/json" + path
	req, err := http.NewRequestWithContext(ctx, "DELETE", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 401 {
		return fmt.Errorf("proxmox: unauthorized")
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("proxmox: HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// Ping checks the host is reachable and the token works.
func (c *Client) Ping(ctx context.Context) error {
	var nodes []Node
	return c.get(ctx, "/nodes", &nodes)
}

// Node is a Proxmox cluster node (hypervisor host).
type Node struct {
	Node           string  `json:"node"`
	Status         string  `json:"status"`
	Uptime         int64   `json:"uptime"`
	CPU            float64 `json:"cpu"`
	MaxCPU         int     `json:"maxcpu"`
	Mem            int64   `json:"mem"`
	MaxMem         int64   `json:"maxmem"`
	Disk           int64   `json:"disk"`
	MaxDisk        int64   `json:"maxdisk"`
	SSLFingerprint string  `json:"ssl_fingerprint"`
	Level          string  `json:"level"`
	ID             string  `json:"id"`
}

// ListNodes returns all nodes on this host.
func (c *Client) ListNodes(ctx context.Context) ([]Node, error) {
	var nodes []Node
	if err := c.get(ctx, "/nodes", &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}

// VM represents a QEMU VM or LXC container from /cluster/resources.
type VM struct {
	ID        string  `json:"id"` // e.g. "qemu/100" or "lxc/101"
	Type      string  `json:"type"`
	Node      string  `json:"node"`
	VMID      int     `json:"vmid"`
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	CPU       float64 `json:"cpu"`
	MaxCPU    int     `json:"maxcpu"`
	Mem       int64   `json:"mem"`
	MaxMem    int64   `json:"maxmem"`
	MemHost   int64   `json:"memhost,omitempty"`
	Disk      int64   `json:"disk"`
	MaxDisk   int64   `json:"maxdisk"`
	DiskRead  int64   `json:"diskread"`
	DiskWrite int64   `json:"diskwrite"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
	Uptime    int64   `json:"uptime"`
	Template  int     `json:"template"`
	Tags      string  `json:"tags"`
}

// ListVMs returns all VMs + LXC across all nodes via /cluster/resources.
func (c *Client) ListVMs(ctx context.Context) ([]VM, error) {
	var vms []VM
	if err := c.get(ctx, "/cluster/resources", &vms); err != nil {
		return nil, err
	}
	return vms, nil
}

// ListNodeVMs returns VMs/LXC on a specific node.
func (c *Client) ListNodeVMs(ctx context.Context, node string, kind string) ([]VM, error) {
	path := "/nodes/" + url.PathEscape(node) + "/" + kind
	var vms []VM
	if err := c.get(ctx, path, &vms); err != nil {
		return nil, err
	}
	return vms, nil
}

// Storage represents a Proxmox storage pool.
type Storage struct {
	Storage  string  `json:"storage"`
	Type     string  `json:"type"`
	Status   string  `json:"status"`
	Total    int64   `json:"total"`
	Used     int64   `json:"used"`
	Avail    int64   `json:"avail"`
	Content  string  `json:"content"`
	Active   int     `json:"active"`
	UsedFrac float64 `json:"used_fraction"`
}

// ListStorage returns all storage on a node.
func (c *Client) ListStorage(ctx context.Context, node string) ([]Storage, error) {
	var storages []Storage
	if err := c.get(ctx, "/nodes/"+url.PathEscape(node)+"/storage", &storages); err != nil {
		return nil, err
	}
	return storages, nil
}
