// Package handler — Terminal connections + history + SSH dial (Tier 3.1 part 2).
package handler

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/stackwatch/platform/internal/kernel"
)

// Connection is a saved SSH connection.
type Connection struct {
	ID                  uuid.UUID  `json:"id"`
	Name                string     `json:"name"`
	Host                string     `json:"host"`
	Port                int        `json:"port"`
	User                string     `json:"user"`
	SSHKeyID            *uuid.UUID `json:"ssh_key_id,omitempty"`
	GroupName           string     `json:"group_name"`
	Tags                []string   `json:"tags"`
	Color               string     `json:"color"`
	Icon                string     `json:"icon"`
	LastConnectedAt     *time.Time `json:"last_connected_at,omitempty"`
	LastConnectedStatus *string    `json:"last_connected_status,omitempty"`
	TotalConnections    int        `json:"total_connections"`
	Notes               string     `json:"notes"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func scanConnection(rows pgx.Row, cn *Connection) error {
	return rows.Scan(&cn.ID, &cn.Name, &cn.Host, &cn.Port, &cn.User,
		&cn.SSHKeyID, &cn.GroupName, &cn.Tags, &cn.Color, &cn.Icon,
		&cn.LastConnectedAt, &cn.LastConnectedStatus,
		&cn.TotalConnections, &cn.Notes, &cn.CreatedAt, &cn.UpdatedAt)
}

// ListConnections returns all saved connections for the tenant.
func (h *TerminalHandler) ListConnections(c *gin.Context) {
	tenantID := h.tenantID(c)
	group := c.Query("group")
	var rows pgx.Rows
	var err error
	if group != "" {
		rows, err = h.pool.Pgx().Query(c.Request.Context(),
			`SELECT id, name, host, port, user_, ssh_key_id, group_name, tags,
                    color, icon, last_connected_at, last_connected_status,
                    total_connections, notes, created_at, updated_at
                    FROM connections WHERE tenant_id = $1 AND group_name = $2 ORDER BY name`,
			tenantID, group)
	} else {
		rows, err = h.pool.Pgx().Query(c.Request.Context(),
			`SELECT id, name, host, port, user_, ssh_key_id, group_name, tags,
                    color, icon, last_connected_at, last_connected_status,
                    total_connections, notes, created_at, updated_at
                    FROM connections WHERE tenant_id = $1 ORDER BY name`, tenantID)
	}
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer rows.Close()
	conns := []Connection{}
	for rows.Next() {
		var cn Connection
		if err := scanConnection(rows, &cn); err != nil {
			kernel.RespondError(c, err)
			return
		}
		conns = append(conns, cn)
	}
	kernel.RespondOK(c, gin.H{"connections": conns, "total": len(conns)})
}

// GetConnection returns one connection.
func (h *TerminalHandler) GetConnection(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var cn Connection
	err = scanConnection(h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT id, name, host, port, user_, ssh_key_id, group_name, tags,
                color, icon, last_connected_at, last_connected_status,
                total_connections, notes, created_at, updated_at
                FROM connections WHERE tenant_id = $1 AND id = $2`,
		tenantID, id), &cn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, cn)
}

// CreateConnection adds a new saved connection.
func (h *TerminalHandler) CreateConnection(c *gin.Context) {
	tenantID := h.tenantID(c)
	var req struct {
		Name      string     `json:"name" binding:"required"`
		Host      string     `json:"host" binding:"required"`
		Port      int        `json:"port"`
		User      string     `json:"user"`
		SSHKeyID  *uuid.UUID `json:"ssh_key_id"`
		GroupName string     `json:"group_name"`
		Tags      []string   `json:"tags"`
		Color     string     `json:"color"`
		Icon      string     `json:"icon"`
		Notes     string     `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}
	if req.GroupName == "" {
		req.GroupName = "default"
	}
	if req.Color == "" {
		req.Color = "#3b82f6"
	}
	if req.Icon == "" {
		req.Icon = "server"
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	var id uuid.UUID
	err := h.pool.Pgx().QueryRow(c.Request.Context(),
		`INSERT INTO connections (tenant_id, name, host, port, user_, ssh_key_id, group_name, tags, color, icon, notes)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
		tenantID, req.Name, req.Host, req.Port, req.User, req.SSHKeyID,
		req.GroupName, req.Tags, req.Color, req.Icon, req.Notes).Scan(&id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			kernel.RespondError(c, kernel.ErrConflict)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":     id,
		"name":   req.Name,
		"host":   req.Host,
		"port":   req.Port,
		"user":   req.User,
		"status": "created",
	})
}

// UpdateConnection modifies a connection.
func (h *TerminalHandler) UpdateConnection(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	// Map JSON keys to column names. user → user_.
	if v, ok := req["user"]; ok {
		req["user_"] = v
		delete(req, "user")
	}
	if v, ok := req["ssh_key_id"]; ok {
		if v == nil {
			delete(req, "ssh_key_id")
		}
	}
	if v, ok := req["group"]; ok {
		req["group_name"] = v
		delete(req, "group")
	}

	sets := []string{}
	args := []interface{}{}
	idx := 1
	for col, val := range req {
		if col == "id" || col == "tenant_id" || col == "created_at" || col == "total_connections" {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", col, idx))
		args = append(args, val)
		idx++
	}
	if len(sets) == 0 {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	sets = append(sets, "updated_at = NOW()")
	args = append(args, tenantID, id)
	q := fmt.Sprintf("UPDATE connections SET %s WHERE tenant_id = $%d AND id = $%d",
		strings.Join(sets, ", "), idx, idx+1)
	tag, err := h.pool.Pgx().Exec(c.Request.Context(), q, args...)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	if tag.RowsAffected() == 0 {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	kernel.RespondOK(c, gin.H{"id": id, "status": "updated"})
}

// DeleteConnection removes a saved connection.
func (h *TerminalHandler) DeleteConnection(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	tag, err := h.pool.Pgx().Exec(c.Request.Context(),
		`DELETE FROM connections WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	if tag.RowsAffected() == 0 {
		kernel.RespondError(c, kernel.ErrNotFound)
		return
	}
	kernel.RespondOK(c, gin.H{"id": id, "status": "deleted"})
}

// TestConnection attempts to connect to a host and reports success/failure.
func (h *TerminalHandler) TestConnection(c *gin.Context) {
	tenantID := h.tenantID(c)
	userID := h.userID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var host string
	var port int
	var user string
	var sshKeyID *uuid.UUID
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT host, port, user_, ssh_key_id FROM connections
         WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&host, &port, &user, &sshKeyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondError(c, err)
		return
	}

	// Load auth_method to report even if dial fails before authenticating
	var intendedMethod string
	_ = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT auth_method FROM connections WHERE tenant_id = $1 AND id = $2`,
		tenantID, id).Scan(&intendedMethod)

	start := time.Now()
	authRes, dialErr := h.dialConnection(c.Request.Context(), tenantID, id, 10*time.Second)
	duration := time.Since(start)
	statusStr, errorMsg := classifyDialError(dialErr, c.Request.Context())

	// Record which auth method was used (or attempted)
	authMethod := intendedMethod
	if authRes != nil {
		authMethod = authRes.AuthMethod
		authRes.Client.Close()
	}

	// Always log history
	_, _ = h.pool.Pgx().Exec(c.Request.Context(),
		`INSERT INTO connection_history (tenant_id, user_id, connection_id, host, port, user_, status, duration_ms, error_message, source_ip, user_agent)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		tenantID, userID, id, host, port, user, statusStr, duration.Milliseconds(), errorMsg,
		c.ClientIP(), c.Request.UserAgent())

	// Update connection stats
	if statusStr == "success" {
		_, _ = h.pool.Pgx().Exec(c.Request.Context(),
			`UPDATE connections SET last_connected_at = NOW(), last_connected_status = 'success',
                total_connections = total_connections + 1, updated_at = NOW()
                WHERE id = $1`, id)
	} else {
		_, _ = h.pool.Pgx().Exec(c.Request.Context(),
			`UPDATE connections SET last_connected_at = NOW(), last_connected_status = $1,
                updated_at = NOW() WHERE id = $2`, statusStr, id)
	}

	kernel.RespondOK(c, gin.H{
		"id":          id,
		"host":        host,
		"port":        port,
		"status":      statusStr,
		"duration_ms": duration.Milliseconds(),
		"success":     statusStr == "success",
		"auth_method": authMethod,
		"error":       errorMsg,
	})
}

// ListConnectionHistory returns the connection history for the tenant.
func (h *TerminalHandler) ListConnectionHistory(c *gin.Context) {
	tenantID := h.tenantID(c)
	limit := 100
	if l := c.Query("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	rows, err := h.pool.Pgx().Query(c.Request.Context(),
		`SELECT id, user_id, connection_id, host, port, user_, status,
                duration_ms, error_message, source_ip, started_at
                FROM connection_history WHERE tenant_id = $1
                ORDER BY started_at DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer rows.Close()
	type entry struct {
		ID           uuid.UUID  `json:"id"`
		UserID       uuid.UUID  `json:"user_id"`
		ConnectionID *uuid.UUID `json:"connection_id,omitempty"`
		Host         string     `json:"host"`
		Port         int        `json:"port"`
		User         string     `json:"user"`
		Status       string     `json:"status"`
		DurationMs   int        `json:"duration_ms"`
		ErrorMessage string     `json:"error_message"`
		SourceIP     string     `json:"source_ip"`
		StartedAt    time.Time  `json:"started_at"`
	}
	entries := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.UserID, &e.ConnectionID, &e.Host, &e.Port,
			&e.User, &e.Status, &e.DurationMs, &e.ErrorMessage,
			&e.SourceIP, &e.StartedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		entries = append(entries, e)
	}
	kernel.RespondOK(c, gin.H{"history": entries, "total": len(entries)})
}

// ListConnectionGroups returns distinct group names + counts.
func (h *TerminalHandler) ListConnectionGroups(c *gin.Context) {
	tenantID := h.tenantID(c)
	rows, err := h.pool.Pgx().Query(c.Request.Context(),
		`SELECT group_name, COUNT(*) FROM connections
         WHERE tenant_id = $1 GROUP BY group_name ORDER BY group_name`, tenantID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer rows.Close()
	type g struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	groups := []g{}
	for rows.Next() {
		var gp g
		if err := rows.Scan(&gp.Name, &gp.Count); err != nil {
			kernel.RespondError(c, err)
			return
		}
		groups = append(groups, gp)
	}
	kernel.RespondOK(c, gin.H{"groups": groups, "total": len(groups)})
}

// generateKey creates an SSH keypair.
func generateKey(keyType string, rsaBits int) (priv interface{}, pub interface{}, err error) {
	switch keyType {
	case "ed25519":
		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return privKey, pubKey, nil
	case "rsa":
		privKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
		if err != nil {
			return nil, nil, err
		}
		return privKey, &privKey.PublicKey, nil
	default:
		return nil, nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// classifyDialError maps an SSH dial error to one of our standard status strings.
func classifyDialError(err error, ctx context.Context) (string, string) {
	if err == nil {
		return "success", ""
	}
	if ctx.Err() == context.DeadlineExceeded {
		return "timeout", err.Error()
	}
	if strings.Contains(err.Error(), "connection refused") {
		return "refused", err.Error()
	}
	if strings.Contains(err.Error(), "no route to host") {
		return "failed", err.Error()
	}
	if strings.Contains(err.Error(), "knownhosts") || strings.Contains(err.Error(), "host key") {
		return "mitm", err.Error()
	}
	return "failed", err.Error()
}


