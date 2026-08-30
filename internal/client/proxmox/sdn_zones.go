// Package proxmox — Tier 14 fill-in: SDN zones CRUD.
package proxmox

import (
	"context"
	"fmt"
	"net/url"
)

// SdnZone is one SDN zone.
type SdnZone struct {
	Zone   string `json:"zone"`
	Type   string `json:"type,omitempty"` // simple, vlan, vxlan, qinq, evpn
	Bridge string `json:"bridge,omitempty"`
	IPAM   string `json:"ipam,omitempty"`
	DHCP   string `json:"dhcp,omitempty"`
	State  string `json:"state,omitempty"`
}

// ListSDNZones returns all zones under /cluster/sdn/zones.
func (c *Client) ListSDNZones(ctx context.Context) ([]SdnZone, error) {
	var out []SdnZone
	err := c.get(ctx, "/cluster/sdn/zones", &out)
	return out, err
}

// CreateSDNZone creates a zone. `typ` is simple/vlan/vxlan/qinq/evpn.
func (c *Client) CreateSDNZone(ctx context.Context, zone, typ, bridge string) error {
	form := url.Values{"zone": {zone}, "type": {typ}}
	if bridge != "" {
		form.Set("bridge", bridge)
	}
	return c.postForm(ctx, "/cluster/sdn/zones", form, nil)
}

// DeleteSDNZone removes a zone.
func (c *Client) DeleteSDNZone(ctx context.Context, zone string) error {
	return c.deleteForm(ctx, fmt.Sprintf("/cluster/sdn/zones/%s", zone))
}
