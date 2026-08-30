package proxmox

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// FirewallRule is a single iptables rule in the Proxmox firewall.
// Rules are ordered by `pos` (position). Lower = applied first.
type FirewallRule struct {
	Pos        int    `json:"pos"`              // position (0-based)
	Type       string `json:"type"`             // in, out, forward (group direction)
	Action     string `json:"action"`           // ACCEPT, DROP, REJECT, LOG, NFLOG
	Enable     int    `json:"enable"`           // 1 = enabled
	Source     string `json:"source,omitempty"` // CIDR or IPset name
	Dest       string `json:"dest,omitempty"`
	Proto      string `json:"proto,omitempty"` // tcp, udp, icmp, all
	DestPort   string `json:"dport,omitempty"` // destination port/range
	SourcePort string `json:"sport,omitempty"` // source port/range
	Iface      string `json:"iface,omitempty"` // network interface
	Comment    string `json:"comment,omitempty"`
	Macro      string `json:"macro,omitempty"` // shortcut: NO, DHCP, HTTP, HTTPS, etc.
	Log        string `json:"log,omitempty"`   // emerg, alert, crit, err, warning, notice, info, debug, none
}

// FirewallRuleSpec is the body for POST/PUT /firewall/rules.
// Field names match Proxmox API exactly.
type FirewallRuleSpec struct {
	Type       string `json:"type,omitempty"`
	Action     string `json:"action" binding:"required"`
	Enable     int    `json:"enable,omitempty"`
	Source     string `json:"source,omitempty"`
	Dest       string `json:"dest,omitempty"`
	Proto      string `json:"proto,omitempty"`
	DestPort   string `json:"dport,omitempty"`
	SourcePort string `json:"sport,omitempty"`
	Iface      string `json:"iface,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Macro      string `json:"macro,omitempty"`
	Log        string `json:"log,omitempty"`
}

// ListFirewallRules returns all firewall rules on a node (host firewall).
// For VM-level rules, use /nodes/{node}/qemu/{vmid}/firewall/rules.
func (c *Client) ListFirewallRules(ctx context.Context, node string) ([]FirewallRule, error) {
	path := fmt.Sprintf("/nodes/%s/firewall/rules", url.PathEscape(node))
	var out []FirewallRule
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateFirewallRule appends a rule to the end of the firewall chain.
func (c *Client) CreateFirewallRule(ctx context.Context, node string, spec FirewallRuleSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/firewall/rules", url.PathEscape(node))
	form := url.Values{}
	if spec.Type != "" {
		form.Set("type", spec.Type)
	}
	form.Set("action", spec.Action)
	if spec.Enable > 0 {
		form.Set("enable", "1")
	} else {
		form.Set("enable", "0")
	}
	if spec.Source != "" {
		form.Set("source", spec.Source)
	}
	if spec.Dest != "" {
		form.Set("dest", spec.Dest)
	}
	if spec.Proto != "" {
		form.Set("proto", spec.Proto)
	}
	if spec.DestPort != "" {
		form.Set("dport", spec.DestPort)
	}
	if spec.SourcePort != "" {
		form.Set("sport", spec.SourcePort)
	}
	if spec.Iface != "" {
		form.Set("iface", spec.Iface)
	}
	if spec.Comment != "" {
		form.Set("comment", spec.Comment)
	}
	if spec.Macro != "" {
		form.Set("macro", spec.Macro)
	}
	if spec.Log != "" {
		form.Set("log", spec.Log)
	}
	if err := c.postForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// UpdateFirewallRule modifies a rule at the given position.
func (c *Client) UpdateFirewallRule(ctx context.Context, node string, pos int, fields map[string]string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/firewall/rules/%d",
		url.PathEscape(node), pos)
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	if err := c.putForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteFirewallRule removes a rule at the given position.
func (c *Client) DeleteFirewallRule(ctx context.Context, node string, pos int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/firewall/rules/%d",
		url.PathEscape(node), pos)
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

// IPset is a named set of IPs/CIDRs used by firewall rules.
type IPset struct {
	Name    string       `json:"name"` // e.g. "my-blocklist"
	Comment string       `json:"comment,omitempty"`
	Entries []IPsetEntry `json:"entries,omitempty"`
}

// IPsetEntry is a single IP/CIDR within an IPset.
type IPsetEntry struct {
	CIDR    string `json:"cidr"` // e.g. "192.168.1.5" or "10.0.0.0/24"
	Comment string `json:"comment,omitempty"`
	NoMatch int    `json:"nomatch,omitempty"` // 1 = exclude this CIDR
}

// ListIPsets returns all IPset definitions.
func (c *Client) ListIPsets(ctx context.Context) ([]IPset, error) {
	var out []IPset
	if err := c.get(ctx, "/firewall/ipsets", &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateIPset creates a new IPset (initially empty).
func (c *Client) CreateIPset(ctx context.Context, name, comment string) (string, error) {
	path := "/firewall/ipsets"
	form := url.Values{}
	form.Set("name", name)
	if comment != "" {
		form.Set("comment", comment)
	}
	if err := c.postForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteIPset removes an IPset.
func (c *Client) DeleteIPset(ctx context.Context, name string) (string, error) {
	path := "/firewall/ipsets/" + url.PathEscape(name)
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

// AddIPsetEntry adds an IP/CIDR to an existing IPset.
func (c *Client) AddIPsetEntry(ctx context.Context, name, cidr, comment string, nomatch int) (string, error) {
	path := "/firewall/ipsets/" + url.PathEscape(name)
	form := url.Values{}
	form.Set("cidr", cidr)
	if comment != "" {
		form.Set("comment", comment)
	}
	if nomatch > 0 {
		form.Set("nomatch", "1")
	}
	if err := c.postForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// VMSnapshot rule helpers are in snapshots.go (same Tier 14.3 phase).
// Below: VM-level firewall CRUD (Proxmox path: /nodes/{node}/qemu/{vmid}/firewall/rules).

// ListVMFirewallRules returns the firewall rules for a specific VM.
func (c *Client) ListVMFirewallRules(ctx context.Context, node string, vmid int) ([]FirewallRule, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", url.PathEscape(node), vmid)
	var out []FirewallRule
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateVMFirewallRule appends a rule to the VM's firewall chain.
func (c *Client) CreateVMFirewallRule(ctx context.Context, node string, vmid int, spec FirewallRuleSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules", url.PathEscape(node), vmid)
	form := buildFirewallForm(spec)
	return c.postFormPath(ctx, path, form)
}

// UpdateVMFirewallRule replaces a rule at the given position.
func (c *Client) UpdateVMFirewallRule(ctx context.Context, node string, vmid, pos int, fields map[string]string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", url.PathEscape(node), vmid, pos)
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	return c.postFormPath(ctx, path, form)
}

// DeleteVMFirewallRule removes a rule at the given position.
func (c *Client) DeleteVMFirewallRule(ctx context.Context, node string, vmid, pos int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/firewall/rules/%d", url.PathEscape(node), vmid, pos)
	u := c.baseURL + "/api2/json" + path
	req, err := http.NewRequestWithContext(ctx, "DELETE", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.apiToken)
	req.Header.Set("Accept", "application/json")
	return c.doTask(req)
}

// postFormPath is a tiny shim that mirrors postForm but returns a string UPID
// rather than erroring. Used for endpoints that always return a task.
func (c *Client) postFormPath(ctx context.Context, path string, form url.Values) (string, error) {
	u := c.baseURL + "/api2/json" + path
	req, err := newFormRequestWithContext(ctx, "POST", u, form, c.apiToken)
	if err != nil {
		return "", err
	}
	return c.doTask(req)
}

// buildFirewallForm converts a FirewallRuleSpec into url.Values, omitting empty fields.
func buildFirewallForm(spec FirewallRuleSpec) url.Values {
	form := url.Values{}
	if spec.Type != "" {
		form.Set("type", spec.Type)
	}
	if spec.Action != "" {
		form.Set("action", spec.Action)
	}
	form.Set("enable", "1")
	if spec.Source != "" {
		form.Set("source", spec.Source)
	}
	if spec.Dest != "" {
		form.Set("dest", spec.Dest)
	}
	if spec.Proto != "" {
		form.Set("proto", spec.Proto)
	}
	if spec.DestPort != "" {
		form.Set("dport", spec.DestPort)
	}
	if spec.SourcePort != "" {
		form.Set("sport", spec.SourcePort)
	}
	if spec.Iface != "" {
		form.Set("iface", spec.Iface)
	}
	if spec.Comment != "" {
		form.Set("comment", spec.Comment)
	}
	if spec.Macro != "" {
		form.Set("macro", spec.Macro)
	}
	if spec.Log != "" {
		form.Set("log", spec.Log)
	}
	return form
}
