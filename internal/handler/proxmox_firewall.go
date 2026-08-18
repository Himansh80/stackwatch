package handler

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// CreateFirewallRuleRequest is the JSON body for POST firewall/rules.
type CreateFirewallRuleRequest struct {
	Type       string `json:"type" binding:"required,oneof=in out group forward"`
	Action     string `json:"action" binding:"required,oneof=ACCEPT DROP REJECT LOG NFLOG"`
	Enable     int    `json:"enable"`
	Source     string `json:"source"`
	Dest       string `json:"dest"`
	Proto      string `json:"proto"`
	DestPort   string `json:"dport"`
	SourcePort string `json:"sport"`
	Iface      string `json:"iface"`
	Comment    string `json:"comment"`
	Macro      string `json:"macro"`
	Log        string `json:"log"`
}

// UpdateFirewallRuleRequest is the JSON body for PUT firewall/rules/{pos}.
type UpdateFirewallRuleRequest struct {
	Type       string `json:"type,omitempty"`
	Action     string `json:"action,omitempty"`
	Enable     *int   `json:"enable,omitempty"`
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

// ListFirewallRules returns all host firewall rules on a node.
func (h *ProxmoxHandler) ListFirewallRules(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	rules, err := cli.ListFirewallRules(c.Request.Context(), node)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"rules": rules, "total": len(rules)})
}

// CreateFirewallRule appends a new rule.
func (h *ProxmoxHandler) CreateFirewallRule(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	var req CreateFirewallRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	spec := proxmox.FirewallRuleSpec{
		Type:       req.Type,
		Action:     req.Action,
		Enable:     req.Enable,
		Source:     req.Source,
		Dest:       req.Dest,
		Proto:      req.Proto,
		DestPort:   req.DestPort,
		SourcePort: req.SourcePort,
		Iface:      req.Iface,
		Comment:    req.Comment,
		Macro:      req.Macro,
		Log:        req.Log,
	}
	if _, err := cli.CreateFirewallRule(c.Request.Context(), node, spec); err != nil {
		kernel.RespondError(c, err)
		return
	}
	// Get position of newly created rule (last in list)
	updated, _ := cli.ListFirewallRules(c.Request.Context(), node)
	pos := 0
	if len(updated) > 0 {
		pos = updated[len(updated)-1].Pos
	}
	kernel.RespondCreated(c, gin.H{
		"created": true, "pos": pos, "action": req.Action,
	})
}

// UpdateFirewallRule modifies a rule at the given position.
func (h *ProxmoxHandler) UpdateFirewallRule(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	posStr := c.Param("pos")
	pos, err := strconv.Atoi(posStr)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req UpdateFirewallRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	fields := map[string]string{}
	if req.Type != "" {
		fields["type"] = req.Type
	}
	if req.Action != "" {
		fields["action"] = req.Action
	}
	if req.Enable != nil {
		if *req.Enable > 0 {
			fields["enable"] = "1"
		} else {
			fields["enable"] = "0"
		}
	}
	if req.Source != "" {
		fields["source"] = req.Source
	}
	if req.Dest != "" {
		fields["dest"] = req.Dest
	}
	if req.Proto != "" {
		fields["proto"] = req.Proto
	}
	if req.DestPort != "" {
		fields["dport"] = req.DestPort
	}
	if req.SourcePort != "" {
		fields["sport"] = req.SourcePort
	}
	if req.Iface != "" {
		fields["iface"] = req.Iface
	}
	if req.Comment != "" {
		fields["comment"] = req.Comment
	}
	if req.Macro != "" {
		fields["macro"] = req.Macro
	}
	if req.Log != "" {
		fields["log"] = req.Log
	}

	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.UpdateFirewallRule(c.Request.Context(), node, pos, fields); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"updated": true, "pos": pos})
}

// DeleteFirewallRule removes a rule at the given position.
func (h *ProxmoxHandler) DeleteFirewallRule(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	posStr := c.Param("pos")
	pos, err := strconv.Atoi(posStr)
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteFirewallRule(c.Request.Context(), node, pos); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"deleted": true, "pos": pos})
}

// CreateIPsetRequest is the JSON body for POST /firewall/ipsets.
type CreateIPsetRequest struct {
	Name    string `json:"name" binding:"required,min=1,max=64"`
	Comment string `json:"comment"`
	CIDR    string `json:"cidr"` // optional initial CIDR to add
}

// ListIPsets returns all IPsets defined on this Proxmox cluster.
func (h *ProxmoxHandler) ListIPsets(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	ipsets, err := cli.ListIPsets(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"ipsets": ipsets, "total": len(ipsets)})
}

// CreateIPset creates a new IPset.
func (h *ProxmoxHandler) CreateIPset(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req CreateIPsetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.CreateIPset(c.Request.Context(), req.Name, req.Comment); err != nil {
		kernel.RespondError(c, err)
		return
	}
	// Optionally add initial CIDR
	if req.CIDR != "" {
		if _, err := cli.AddIPsetEntry(c.Request.Context(), req.Name, req.CIDR, "", 0); err != nil {
			kernel.RespondError(c, fmt.Errorf("ipset created but cidr add failed: %w", err))
			return
		}
	}
	kernel.RespondCreated(c, gin.H{"created": true, "name": req.Name})
}

// DeleteIPset removes an IPset.
func (h *ProxmoxHandler) DeleteIPset(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteIPset(c.Request.Context(), name); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"deleted": true, "name": name})
}
