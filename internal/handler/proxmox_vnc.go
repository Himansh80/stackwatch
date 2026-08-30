// Tier 14 Phase 14.3 — Proxmox VNC proxy + ticket.
//
// Browser noVNC client connects to /vnc-ws, this handler proxies
// the WebSocket to Proxmox's /vncwebsocket endpoint and streams bytes
// both ways. The ticket endpoint returns enough info for the client to
// build the websocket URL.
package handler

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/kernel"
)

// VNCTicketResponse is the body returned by POST /proxmox/hosts/:id/vnc-ticket.
type VNCTicketResponse struct {
	Ticket string `json:"ticket"`
	Node   string `json:"node"`
	Port   int    `json:"port"`
	Host   string `json:"host"`
}

// IssueVNCTicket returns a ticket + host/port so the browser can build
// the websocket URL. We use the host's API token as the ticket —
// Proxmox accepts the literal `PVEAPIToken=USER@REALM!TOKENID=UUID`
// string as the VNC password parameter.
//
// Body: { "node": "router", "vmid": 100 }
func (h *ProxmoxHandler) IssueVNCTicket(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var body struct {
		Node string `json:"node" binding:"required"`
		VMID int    `json:"vmid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	// Use the API token as the ticket.
	ticket := strings.TrimPrefix(host.APIToken, "PVEAPIToken=")

	// Parse the host's base URL to extract port.
	port := 8006
	if u, err := url.Parse(host.BaseURL); err == nil {
		if p := u.Port(); p != "" {
			port = atoiOrDefault(p, 8006)
		}
	}
	c.JSON(http.StatusOK, VNCTicketResponse{
		Ticket: ticket,
		Node:   body.Node,
		Port:   port,
		Host:   host.BaseURL,
	})
}

// ProxyVNCWebSocket upgrades the browser's WebSocket to Proxmox's VNC WS.
//
// Browser URL: ws(s)://api-gateway/api/v1/proxmox/hosts/:id/vnc-ws?node=Y&vmid=Z
// Proxmox URL: wss://host:8006/api2/json/nodes/Y/qemu/Z/vncwebsocket?port=5900&vncticket=...
//
// We use httputil.ReverseProxy with a Director that injects the ticket
// query parameter so the browser doesn't need to know the ticket.
func (h *ProxmoxHandler) ProxyVNCWebSocket(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Query("node")
	vmid := c.Query("vmid")
	if node == "" || vmid == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}

	// Build target URL.
	target, err := url.Parse(host.BaseURL)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	target.Scheme = "https"
	target.Path = "/api2/json/nodes/" + node + "/qemu/" + vmid + "/vncwebsocket"
	target.RawQuery = "port=5900&vncticket=" + strings.TrimPrefix(host.APIToken, "PVEAPIToken=")

	// Reverse proxy with custom Transport that ignores self-signed certs.
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Hijack the connection and pump bytes — httputil handles Upgrade.
	proxy.ServeHTTP(c.Writer, c.Request)
}

// atoiOrDefault converts s to int, returning def if invalid.
func atoiOrDefault(s string, def int) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return def
		}
		n = n*10 + int(ch-'0')
	}
	if n == 0 {
		return def
	}
	return n
}
