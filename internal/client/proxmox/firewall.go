package proxmox

import (
	"context"
	"fmt"
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
	Proto      string `json:"proto,omitempty"`      // tcp, udp, icmp, all
	DestPort   string `json:"dport,omitempty"`     // destination port/range
	SourcePort string `json:"sport,omitempty"`     // source port/range
	Iface      string `json:"iface,omitempty"`      // network interface
	Comment    string `json:"comment,omitempty"`
	Macro      string `json:"macro,omitempty"`      // shortcut: NO, DHCP, HTTP, HTTPS, etc.
	Log        string `json:"log,omitempty"`       // emerg, alert, crit, err, warning, notice, info, debug, none
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
	Name    string         `json:"name"`           // e.g. "my-blocklist"
	Comment string         `json:"comment,omitempty"`
	Entries []IPsetEntry   `json:"entries,omitempty"`
}

// IPsetEntry is a single IP/CIDR within an IPset.
type IPsetEntry struct {
	CIDR    string `json:"cidr"`           // e.g. "192.168.1.5" or "10.0.0.0/24"
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