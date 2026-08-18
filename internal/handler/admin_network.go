// Tier 4 S3: Network (interfaces, routes, DNS).
//
//   GET /api/v1/admin/network?connection_id=X
//
// Commands:
//   - ip -j addr           → interfaces + IPs
//   - ip -j route          → routing table
//   - cat /etc/resolv.conf → DNS nameservers
package handler

import (
	"github.com/gin-gonic/gin"
)

// ipAddrOutput is the JSON shape of `ip -j addr`.
type ipAddrOutput []ipInterface

type ipInterface struct {
	Ifindex   int      `json:"ifindex"`
	Ifname    string   `json:"ifname"`
	Flags     []string `json:"flags"`
	MTU       int      `json:"mtu"`
	Operstate string   `json:"operstate"`
	Link      string   `json:"link"`     // MAC
	Address   string   `json:"address"`  // sometimes populated
	AddrInfo  []ipAddr `json:"addr_info"`
}
type ipAddr struct {
	Family string `json:"family"` // "inet" / "inet6"
	Local  string `json:"local"`
	Prefix int    `json:"prefixlen"`
}

// ipRouteOutput is the JSON shape of `ip -j route`.
type ipRouteOutput []ipRoute

type ipRoute struct {
	Dst      string `json:"dst"`
	Gateway  string `json:"gateway"`
	Dev      string `json:"dev"`
	Metric   int    `json:"metric"`
	Protocol string `json:"protocol"`
	Scope    string `json:"scope"`
}

// resolvConfOutput is parsed from `cat /etc/resolv.conf`.
// Lines: `nameserver 1.2.3.4`, `search example.com`, etc.
type resolvConfOutput struct {
	Nameservers []string `json:"nameservers"`
	Search      []string `json:"search"`
}

// ListNetwork returns network info: interfaces + routes + DNS.
func (h *AdminHandler) ListNetwork(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	// 1. Interfaces
	var ifaces ipAddrOutput
	ifErr := h.SSHExecJSON(c, cid, "ip -j addr 2>/dev/null", 10000, &ifaces)

	// 2. Routes
	var routes ipRouteOutput
	rtErr := h.SSHExecJSON(c, cid, "ip -j route 2>/dev/null", 10000, &routes)

	// 3. DNS
	var dns resolvConfOutput
	dnsRaw, dnsErr := h.SSHExec(c, cid, "cat /etc/resolv.conf 2>/dev/null", 5000)
	if dnsErr == nil {
		dns = parseResolvConf(dnsRaw.Stdout)
	}

	resp := gin.H{
		"interfaces": []ipInterface{},
		"routes":     []ipRoute{},
		"dns":        resolvConfOutput{},
	}
	if ifErr == nil {
		resp["interfaces"] = ifaces
	} else {
		resp["warning_interfaces"] = ifErr.Error()
	}
	if rtErr == nil {
		resp["routes"] = routes
	} else {
		resp["warning_routes"] = rtErr.Error()
	}
	if dnsErr == nil {
		resp["dns"] = dns
	}

	c.JSON(200, resp)
}

// parseResolvConf extracts nameservers and search domains from
// /etc/resolv.conf format. Comment lines start with `#` or `;`.
func parseResolvConf(s string) resolvConfOutput {
	out := resolvConfOutput{
		Nameservers: []string{},
		Search:      []string{},
	}
	for _, line := range splitLines(s) {
		// strip leading whitespace
		for len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			line = line[1:]
		}
		if len(line) == 0 || line[0] == '#' || line[0] == ';' {
			continue
		}
		// find first whitespace between keyword and value
		sp := indexWhitespace(line)
		if sp < 0 {
			continue
		}
		keyword := line[:sp]
		value := line[sp+1:]
		switch keyword {
		case "nameserver":
			out.Nameservers = append(out.Nameservers, value)
		case "search":
			out.Search = append(out.Search, value)
		}
	}
	return out
}

// indexWhitespace returns the index of the first whitespace char in s, or -1.
func indexWhitespace(s string) int {
	for i, ch := range s {
		if ch == ' ' || ch == '\t' {
			return i
		}
	}
	return -1
}
