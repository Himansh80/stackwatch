package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/kernel"
)

// ListCertificates returns available certificate slots on a node.
func (h *ProxmoxHandler) ListCertificates(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Param("node")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	certs, err := cli.ListCertificates(c.Request.Context(), node)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"certificates": certs, "total": len(certs)})
}

// ListACMEAccounts returns all ACME accounts.
func (h *ProxmoxHandler) ListACMEAccounts(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	accts, err := cli.ListACMEAccounts(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"accounts": accts, "total": len(accts)})
}

// GetACMEAccount returns one ACME account.
func (h *ProxmoxHandler) GetACMEAccount(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)

	// Older Proxmox versions return 403 ("user != root@pam") for a
	// direct lookup of an absent ACME account instead of 404. Consult the
	// list endpoint first so the StackWatch contract stays deterministic:
	// absent resource -> 404, existing resource -> detail/real error.
	accounts, err := cli.ListACMEAccounts(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	found := false
	for _, account := range accounts {
		if account.Name == name {
			found = true
			break
		}
	}
	if !found {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}

	acct, err := cli.GetACMEAccount(c.Request.Context(), name)
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, acct)
}

// CreateACMEAccount registers a new ACME account.
// Body: {"name": "...", "email": "...", "directory": "...", "tos_url": "..."}
func (h *ProxmoxHandler) CreateACMEAccount(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req struct {
		Name      string `json:"name" binding:"required"`
		Email     string `json:"email" binding:"required"`
		Directory string `json:"directory" binding:"required"`
		TosURL    string `json:"tos_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.CreateACMEAccount(c.Request.Context(), req.Name, req.Email, req.Directory, req.TosURL); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"name": req.Name, "email": req.Email, "directory": req.Directory, "status": "registered"})
}

// DeleteACMEAccount removes an ACME account.
func (h *ProxmoxHandler) DeleteACMEAccount(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteACMEAccount(c.Request.Context(), name); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"name": name, "status": "deleted"})
}

// ListACMEPlugins returns all installed ACME plugins.
func (h *ProxmoxHandler) ListACMEPlugins(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	plugins, err := cli.ListACMEPlugins(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"plugins": plugins, "total": len(plugins)})
}

// CreateACMEPlugin installs a new ACME plugin.
// Body: {"name": "...", "type": "standalone|dns", "data": "key1=val1\nkey2=val2"}
func (h *ProxmoxHandler) CreateACMEPlugin(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type" binding:"required"`
		Data string `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.CreateACMEPlugin(c.Request.Context(), req.Name, req.Type, req.Data); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"name": req.Name, "type": req.Type, "status": "installed"})
}

// DeleteACMEPlugin removes an ACME plugin.
func (h *ProxmoxHandler) DeleteACMEPlugin(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	name := c.Param("name")
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if _, err := cli.DeleteACMEPlugin(c.Request.Context(), name); err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"name": name, "status": "deleted"})
}

// ListACMEChallengeSchema returns supported ACME challenge plugins.
func (h *ProxmoxHandler) ListACMEChallengeSchema(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	schema, err := cli.ListACMEChallengeSchema(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"challenges": schema, "total": len(schema)})
}

// ListACMEDirectories returns supported ACME directories (CAs).
func (h *ProxmoxHandler) ListACMEDirectories(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	dirs, err := cli.ListACMEDirectories(c.Request.Context())
	if err != nil {
		respondProxmoxError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{"directories": dirs, "total": len(dirs)})
}

// GetACMEInfo returns current ACME config.
func (h *ProxmoxHandler) GetACMEInfo(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	handleProxmoxCall(c, func() (any, error) {
		return cli.GetACMEInfo(c.Request.Context())
	})
}
