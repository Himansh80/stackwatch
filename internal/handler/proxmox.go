package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stackwatch/platform/internal/client/proxmox"
	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// ProxmoxHandler exposes Proxmox management endpoints.
type ProxmoxHandler struct {
	pool *db.Pool
}

// NewProxmoxHandler constructs the handler.
func NewProxmoxHandler(pool *db.Pool) *ProxmoxHandler {
	return &ProxmoxHandler{pool: pool}
}

// CreateHostRequest is the JSON body for POST /proxmox/hosts.
type CreateHostRequest struct {
	Name      string `json:"name" binding:"required"`
	BaseURL   string `json:"base_url" binding:"required,url"`
	APIToken  string `json:"api_token" binding:"required"`
	VerifyTLS bool   `json:"verify_tls"`
}

// CreateHost registers a new Proxmox host for this tenant.
func (h *ProxmoxHandler) CreateHost(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	var req CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}

	// Test connection before persisting
	cli := proxmox.NewClient(req.BaseURL, req.APIToken, req.VerifyTLS)
	if err := cli.Ping(c.Request.Context()); err != nil {
		// re-export as 200 with error so caller can see the real reason
		c.JSON(http.StatusBadRequest, gin.H{
			"ok":    false,
			"error": "proxmox ping failed: " + err.Error() + " (check API token format: USER@REALM!TOKENID=UUID)",
		})
		return
	}

	id := uuid.New()
	now := time.Now().UTC()
	if _, err := h.pool.Pgx().Exec(c.Request.Context(), `
		INSERT INTO proxmox_hosts (id, tenant_id, name, base_url, api_token, verify_tls, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'online', $7, $7)
	`, id, tenantID, req.Name, req.BaseURL, req.APIToken, req.VerifyTLS, now); err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondCreated(c, ProxmoxHost{
		ID: id, TenantID: tenantID, Name: req.Name,
		BaseURL: req.BaseURL, VerifyTLS: req.VerifyTLS,
		Status: "online", CreatedAt: now.Format(time.RFC3339),
	})
}

// ListHosts returns all Proxmox hosts for this tenant.
func (h *ProxmoxHandler) ListHosts(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	rows, err := h.pool.Pgx().Query(c.Request.Context(), `
		SELECT id, tenant_id, name, base_url, verify_tls, COALESCE(node_name, ''), status,
		       COALESCE(last_check_at, created_at), COALESCE(last_error, ''), created_at
		FROM proxmox_hosts WHERE tenant_id = $1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer rows.Close()
	hosts := []ProxmoxHost{}
	for rows.Next() {
		var p ProxmoxHost
		var lastCheck, created time.Time
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.BaseURL, &p.VerifyTLS,
			&p.NodeName, &p.Status, &lastCheck, &p.LastError, &created); err != nil {
			kernel.RespondError(c, err)
			return
		}
		p.LastCheckAt = lastCheck.Format(time.RFC3339)
		p.CreatedAt = created.Format(time.RFC3339)
		hosts = append(hosts, p)
	}
	kernel.RespondOK(c, gin.H{"hosts": hosts, "total": len(hosts)})
}

// DeleteHost removes a Proxmox host.
func (h *ProxmoxHandler) DeleteHost(c *gin.Context) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return
	}
	hostID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	tag, err := h.pool.Pgx().Exec(c.Request.Context(),
		`DELETE FROM proxmox_hosts WHERE id = $1 AND tenant_id = $2`, hostID, tenantID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	if tag.RowsAffected() == 0 {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	kernel.RespondOK(c, gin.H{"deleted": true})
}

// TestHost checks connectivity to a registered host.
func (h *ProxmoxHandler) TestHost(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	if err := cli.Ping(c.Request.Context()); err != nil {
		// update status to offline
		_, _ = h.pool.Pgx().Exec(c.Request.Context(),
			`UPDATE proxmox_hosts SET status='offline', last_check_at=NOW(), last_error=$1 WHERE id=$2`,
			err.Error(), host.ID)
		kernel.RespondOK(c, gin.H{"ok": false, "error": err.Error()})
		return
	}
	_, _ = h.pool.Pgx().Exec(c.Request.Context(),
		`UPDATE proxmox_hosts SET status='online', last_check_at=NOW(), last_error='' WHERE id=$1`, host.ID)
	kernel.RespondOK(c, gin.H{"ok": true})
}

// fetchHostCreds reads the host row including the secret token.
func (h *ProxmoxHandler) fetchHostCreds(ctx context.Context, hostID uuid.UUID) (*ProxmoxHost, bool) {
	var p ProxmoxHost
	var lastCheck, created time.Time
	err := h.pool.Pgx().QueryRow(ctx, `
		SELECT id, tenant_id, name, base_url, api_token, verify_tls,
		       COALESCE(node_name, ''), status, COALESCE(last_check_at, created_at),
		       COALESCE(last_error, ''), created_at
		FROM proxmox_hosts WHERE id = $1`, hostID).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.BaseURL, &p.APIToken, &p.VerifyTLS,
		&p.NodeName, &p.Status, &lastCheck, &p.LastError, &created,
	)
	if err != nil {
		return nil, false
	}
	p.LastCheckAt = lastCheck.Format(time.RFC3339)
	p.CreatedAt = created.Format(time.RFC3339)
	return &p, true
}

// lookupHost fetches the host and verifies the requesting tenant owns it.
// On error, responds to c and returns (host, false).
func (h *ProxmoxHandler) lookupHost(c *gin.Context, includeCreds bool) (*ProxmoxHost, bool) {
	tenantID, ok := tenantIDFromContext(c)
	if !ok {
		kernel.RespondError(c, kernel.ErrUnauthorized)
		return nil, false
	}
	hostID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return nil, false
	}
	var p ProxmoxHost
	var lastCheck, created time.Time
	q := `SELECT id, tenant_id, name, base_url`
	if includeCreds {
		q += `, api_token`
	}
	q += `, verify_tls, COALESCE(node_name, ''), status, COALESCE(last_check_at, created_at), COALESCE(last_error, ''), created_at FROM proxmox_hosts WHERE id = $1 AND tenant_id = $2`
	args := []any{hostID, tenantID}
	if !includeCreds {
		q = `SELECT id, tenant_id, name, base_url, verify_tls, COALESCE(node_name, ''), status, COALESCE(last_check_at, created_at), COALESCE(last_error, ''), created_at FROM proxmox_hosts WHERE id = $1 AND tenant_id = $2`
		_ = args
	}
	err = h.pool.Pgx().QueryRow(c.Request.Context(), q, args...).Scan(
		&p.ID, &p.TenantID, &p.Name, &p.BaseURL,
		&p.VerifyTLS, &p.NodeName, &p.Status, &lastCheck, &p.LastError, &created,
	)
	if err != nil {
		kernel.RespondError(c, kernel.ErrNotFound)
		return nil, false
	}
	p.LastCheckAt = lastCheck.Format(time.RFC3339)
	p.CreatedAt = created.Format(time.RFC3339)
	return &p, true
}

// ListNodes returns cluster nodes for a host (live, no DB cache).
func (h *ProxmoxHandler) ListNodes(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	nodes, err := cli.ListNodes(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	out := make([]ProxmoxNode, len(nodes))
	for i, n := range nodes {
		out[i] = ProxmoxNode{
			Name: n.Node, Status: n.Status,
			UptimeSeconds: n.Uptime,
			CPUCount:      n.MaxCPU, CPUUsage: n.CPU,
			MemTotal: n.MaxMem, MemUsed: n.Mem,
			DiskTotal: n.MaxDisk, DiskUsed: n.Disk,
		}
	}
	kernel.RespondOK(c, gin.H{"nodes": out, "total": len(out)})
}

// ListVMs returns all VMs/LXC for a host (live).
func (h *ProxmoxHandler) ListVMs(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	vms, err := cli.ListVMs(c.Request.Context())
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	out := make([]ProxmoxVM, len(vms))
	for i, v := range vms {
		out[i] = ProxmoxVM{
			Node: v.Node, VMID: v.VMID, Name: v.Name,
			Kind: v.Type, Status: v.Status,
			CPUCount: v.MaxCPU, CPUUsage: v.CPU,
			MemTotal: v.MaxMem, MemUsed: v.Mem,
			DiskTotal: v.MaxDisk, DiskUsed: v.Disk,
			NetIn: v.NetIn, NetOut: v.NetOut,
			UptimeSeconds: v.Uptime, Tags: v.Tags,
		}
	}
	kernel.RespondOK(c, gin.H{"vms": out, "total": len(out)})
}

// ListStorage returns storage pools for a host (live).
func (h *ProxmoxHandler) ListStorage(c *gin.Context) {
	host, ok := h.fetchHostCreds(c, lookupID(c))
	if !ok {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	node := c.Query("node")
	if node == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	cli := proxmox.NewClient(host.BaseURL, host.APIToken, host.VerifyTLS)
	storages, err := cli.ListStorage(c.Request.Context(), node)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	out := make([]ProxmoxStorage, len(storages))
	for i, s := range storages {
		usage := 0.0
		if s.Total > 0 {
			usage = float64(s.Used) / float64(s.Total) * 100
		}
		out[i] = ProxmoxStorage{
			Name: s.Storage, Kind: s.Type, Status: s.Status,
			TotalBytes: s.Total, UsedBytes: s.Used, AvailBytes: s.Avail,
			UsagePct: usage,
		}
	}
	kernel.RespondOK(c, gin.H{"storage": out, "total": len(out)})
}

// lookupID is a tiny helper to extract the :id path param.
func lookupID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.Param("id"))
	return id
}
