// Package proxmox — Tier 14 fill-in: firewall aliases + IPSet CIDRs.
package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// Alias (firewall) entry at /cluster/firewall/aliases.
type AliasEntry struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
	CIDR    string `json:"cidr"`
	Digest  string `json:"digest,omitempty"`
}

// IPSetCidr is one CIDR inside an IPSet.
type IPSetCidr struct {
	CIDR    string `json:"cidr"`
	Comment string `json:"comment,omitempty"`
	NoMatch bool   `json:"nomatch,omitempty"`
}

// ListAliases lists datacenter aliases.
func (c *Client) ListAliases(ctx context.Context) ([]AliasEntry, error) {
	var out []AliasEntry
	err := c.get(ctx, "/cluster/firewall/aliases", &out)
	return out, err
}

// CreateAlias creates an alias entry.
func (c *Client) CreateAlias(ctx context.Context, name, cidr, comment string) error {
	form := url.Values{"name": {name}, "cidr": {cidr}}
	if comment != "" {
		form.Set("comment", comment)
	}
	return c.postForm(ctx, "/cluster/firewall/aliases", form, nil)
}

// UpdateAlias updates an existing alias.
func (c *Client) UpdateAlias(ctx context.Context, name, cidr, comment string) error {
	form := url.Values{"cidr": {cidr}}
	if comment != "" {
		form.Set("comment", comment)
	}
	return c.putForm(ctx, fmt.Sprintf("/cluster/firewall/aliases/%s", name), form, nil)
}

// DeleteAlias deletes an alias.
func (c *Client) DeleteAlias(ctx context.Context, name string) error {
	return c.deleteForm(ctx, fmt.Sprintf("/cluster/firewall/aliases/%s", name))
}

// ListIPsetCidrs lists CIDR entries in a named IPSet.
func (c *Client) ListIPsetCidrs(ctx context.Context, name string) ([]IPSetCidr, error) {
	var out []IPSetCidr
	err := c.get(ctx, fmt.Sprintf("/cluster/firewall/ipset/%s", name), &out)
	return out, err
}

// AddIPsetCidr adds a CIDR entry to an IPSet.
func (c *Client) AddIPsetCidr(ctx context.Context, name, cidr, comment string, nomatch bool) error {
	form := url.Values{"cidr": {cidr}}
	if comment != "" {
		form.Set("comment", comment)
	}
	if nomatch {
		form.Set("nomatch", "1")
	}
	return c.postForm(ctx, fmt.Sprintf("/cluster/firewall/ipset/%s", name), form, nil)
}

// DeleteIPsetCidr removes one CIDR from an IPSet.
func (c *Client) DeleteIPsetCidr(ctx context.Context, name, cidr string) error {
	return c.deleteForm(ctx, fmt.Sprintf("/cluster/firewall/ipset/%s?cidr=%s", name, url.QueryEscape(cidr)))
}
