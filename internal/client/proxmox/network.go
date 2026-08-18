package proxmox

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// NetworkIface is a single Linux network interface on a Proxmox node
// (bridge, bond, VLAN, alias, or physical NIC).
//
// Proxmox returns bridge_stp / bridge_fd / bridge_vlan_aware as strings
// ("on"/"off", "0", "0"/"1"), not numbers. We use strings to match.
type NetworkIface struct {
	Iface           string `json:"iface"`                       // name: vmbr0, bond0, eno1, eth1.100, etc.
	Type            string `json:"type"`                        // bridge, bond, vlan, eth, alias, none
	Active          int    `json:"active"`                      // 1 = up, 0 = down
	Autostart       int    `json:"autostart"`                   // 1 = start on boot
	BridgePorts     string `json:"bridge_ports,omitempty"`      // comma-separated port members
	BridgeVlanAware string `json:"bridge_vlan_aware,omitempty"` // "1" if vlan-aware
	BridgeSTP       string `json:"bridge_stp,omitempty"`        // "on" / "off"
	BridgeFD        string `json:"bridge_fd,omitempty"`         // forwarding delay seconds
	BondSlaves      string `json:"bond_slaves,omitempty"`       // comma-separated NIC members
	BondMode        string `json:"bond_mode,omitempty"`         // balance-rr, active-backup, 802.3ad, etc.
	IP              string `json:"ip,omitempty"`                // CIDR like 192.168.0.107/24
	Gateway         string `json:"gateway,omitempty"`
	Netmask         string `json:"netmask,omitempty"`
	CIDR            string `json:"cidr,omitempty"`
	MTU             int    `json:"mtu,omitempty"`
	VlanID          int    `json:"vlan_id,omitempty"`         // VLAN tag for vlan type (Proxmox uses hyphen in URL form, JSON keeps underscore)
	VlanRawDevice   string `json:"vlan_raw_device,omitempty"` // parent interface
	Comments        string `json:"comments,omitempty"`
	Address         string `json:"address,omitempty"`
	Netmask6        string `json:"netmask6,omitempty"`
	Gateway6        string `json:"gateway6,omitempty"`
	IPv6            string `json:"ip6,omitempty"`
}

// NetworkSpec describes a network interface to create on a Proxmox node.
type NetworkSpec struct {
	Iface           string `json:"iface"`                       // required: name of new interface
	Type            string `json:"type"`                        // required: bridge, bond, vlan, eth
	Autostart       int    `json:"autostart"`                   // 0/1
	BridgePorts     string `json:"bridge_ports,omitempty"`      // for bridge: vmbr0 ports
	BridgeVlanAware int    `json:"bridge_vlan_aware,omitempty"` // 0/1
	BridgeSTP       int    `json:"bridge_stp,omitempty"`
	BondSlaves      string `json:"bond_slaves,omitempty"` // for bond: NIC members
	BondMode        string `json:"bond_mode,omitempty"`   // bond mode
	IP              string `json:"ip,omitempty"`
	Gateway         string `json:"gateway,omitempty"`
	Netmask         string `json:"netmask,omitempty"`
	CIDR            string `json:"cidr,omitempty"`
	MTU             int    `json:"mtu,omitempty"`
	VlanID          int    `json:"vlan_id,omitempty"`         // for vlan
	VlanRawDevice   string `json:"vlan_raw_device,omitempty"` // for vlan: parent
	Comments        string `json:"comments,omitempty"`
	IPv6            string `json:"ip6,omitempty"`
	Gateway6        string `json:"gateway6,omitempty"`
}

// ListNetwork returns all network interfaces on a node.
func (c *Client) ListNetwork(ctx context.Context, node string) ([]NetworkIface, error) {
	path := fmt.Sprintf("/nodes/%s/network", url.PathEscape(node))
	var out []NetworkIface
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateNetwork adds a new network interface (sync, no task).
// Note: changes only apply after /nodes/{node}/network reload via Proxmox UI
// or "ifreload -a" command.
func (c *Client) CreateNetwork(ctx context.Context, node string, spec NetworkSpec) (string, error) {
	path := fmt.Sprintf("/nodes/%s/network", url.PathEscape(node))
	form := url.Values{}
	form.Set("iface", spec.Iface)
	form.Set("type", spec.Type)
	if spec.Autostart > 0 {
		form.Set("autostart", "1")
	}
	if spec.BridgePorts != "" {
		form.Set("bridge_ports", spec.BridgePorts)
	}
	if spec.BridgeVlanAware > 0 {
		form.Set("bridge_vlan_aware", "1")
	}
	if spec.BridgeSTP > 0 {
		form.Set("bridge_stp", "1")
	}
	if spec.BondSlaves != "" {
		form.Set("bond_slaves", spec.BondSlaves)
	}
	if spec.BondMode != "" {
		form.Set("bond_mode", spec.BondMode)
	}
	if spec.IP != "" {
		form.Set("ip", spec.IP)
	}
	if spec.Gateway != "" {
		form.Set("gateway", spec.Gateway)
	}
	if spec.Netmask != "" {
		form.Set("netmask", spec.Netmask)
	}
	if spec.CIDR != "" {
		form.Set("cidr", spec.CIDR)
	}
	if spec.MTU > 0 {
		form.Set("mtu", fmt.Sprintf("%d", spec.MTU))
	}
	if spec.VlanID > 0 {
		// Proxmox uses hyphen, not underscore, for vlan-id field name
		form.Set("vlan-id", fmt.Sprintf("%d", spec.VlanID))
	}
	if spec.VlanRawDevice != "" {
		// Proxmox uses hyphen, not underscore, for vlan-raw-device field name
		form.Set("vlan-raw-device", spec.VlanRawDevice)
	}
	if spec.Comments != "" {
		form.Set("comments", spec.Comments)
	}
	if spec.IPv6 != "" {
		form.Set("ip6", spec.IPv6)
	}
	if spec.Gateway6 != "" {
		form.Set("gateway6", spec.Gateway6)
	}

	if err := c.postForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// UpdateNetwork modifies an existing interface (sync, no task).
// Proxmox PUT /network/{iface} REQUIRES the `type` field to be passed
// alongside any updates. Caller must include "type" in the fields map.
func (c *Client) UpdateNetwork(ctx context.Context, node, iface string, fields map[string]string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/network/%s",
		url.PathEscape(node), url.PathEscape(iface))
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	// PUT /network/{iface} requires `type` to be re-sent (Proxmox schema requirement).
	if _, ok := form["type"]; !ok {
		// Caller didn't include type — default to a reasonable value based on iface name.
		// Most common case: iface is a bridge or vlan. Caller should include type if known.
		// If iface looks like an alias (parent.N), default to vlan.
		if strings.Contains(iface, ".") {
			form.Set("type", "vlan")
		} else {
			form.Set("type", "bridge")
		}
	}
	if err := c.putForm(ctx, path, form, nil); err != nil {
		return "", err
	}
	return "", nil
}

// DeleteNetwork removes an interface (sync, no task).
func (c *Client) DeleteNetwork(ctx context.Context, node, iface string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/network/%s",
		url.PathEscape(node), url.PathEscape(iface))
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}

// RevertNetwork rolls back unapplied network changes (sync).
func (c *Client) RevertNetwork(ctx context.Context, node string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/network", url.PathEscape(node))
	if err := c.deleteForm(ctx, path); err != nil {
		return "", err
	}
	return "", nil
}
