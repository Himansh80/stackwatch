// Package handler — known_hosts (Tier 3.2 part 2).
//
// SSH host fingerprint store + helper to capture host keys.
package handler

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/ssh"

	"github.com/stackwatch/platform/internal/kernel"
)

// KnownHost is a stored fingerprint.
type KnownHost struct {
	ID          uuid.UUID `json:"id"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	KeyType     string    `json:"key_type"`
	Fingerprint string    `json:"fingerprint"`
	Comment     string    `json:"comment"`
	AddedAt     time.Time `json:"added_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListKnownHosts returns all fingerprints for the tenant.
func (h *TerminalHandler) ListKnownHosts(c *gin.Context) {
	tenantID := h.tenantID(c)
	host := c.Query("host")
	var rows pgx.Rows
	var err error
	if host != "" {
		rows, err = h.pool.Pgx().Query(c.Request.Context(),
			`SELECT id, host, port, key_type, fingerprint, comment, added_at, updated_at
             FROM known_hosts WHERE tenant_id = $1 AND host = $2 ORDER BY port`,
			tenantID, host)
	} else {
		rows, err = h.pool.Pgx().Query(c.Request.Context(),
			`SELECT id, host, port, key_type, fingerprint, comment, added_at, updated_at
             FROM known_hosts WHERE tenant_id = $1 ORDER BY host, port`, tenantID)
	}
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer rows.Close()
	hosts := []KnownHost{}
	for rows.Next() {
		var kh KnownHost
		if err := rows.Scan(&kh.ID, &kh.Host, &kh.Port, &kh.KeyType,
			&kh.Fingerprint, &kh.Comment, &kh.AddedAt, &kh.UpdatedAt); err != nil {
			kernel.RespondError(c, err)
			return
		}
		hosts = append(hosts, kh)
	}
	kernel.RespondOK(c, gin.H{"hosts": hosts, "total": len(hosts)})
}

// GetKnownHost returns one.
func (h *TerminalHandler) GetKnownHost(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var kh KnownHost
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`SELECT id, host, port, key_type, fingerprint, comment, added_at, updated_at
         FROM known_hosts WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&kh.ID, &kh.Host, &kh.Port, &kh.KeyType, &kh.Fingerprint,
			&kh.Comment, &kh.AddedAt, &kh.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			kernel.RespondError(c, kernel.ErrNotFound)
			return
		}
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, kh)
}

// TrustHost connects to host:port, captures the host key, and stores it.
// This is the "first-time-trust" flow: server admin says "yes I trust this".
func (h *TerminalHandler) TrustHost(c *gin.Context) {
	tenantID := h.tenantID(c)
	var req struct {
		Host    string `json:"host" binding:"required"`
		Port    int    `json:"port"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if req.Port == 0 {
		req.Port = 22
	}
	keyType, fp, err := captureHostKey(req.Host, req.Port)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("capture host key: %w", err))
		return
	}
	var id uuid.UUID
	err = h.pool.Pgx().QueryRow(c.Request.Context(),
		`INSERT INTO known_hosts (tenant_id, host, port, key_type, fingerprint, comment)
         VALUES ($1, $2, $3, $4, $5, $6)
         ON CONFLICT (tenant_id, host, port) DO UPDATE
            SET key_type = EXCLUDED.key_type,
                fingerprint = EXCLUDED.fingerprint,
                comment = EXCLUDED.comment,
                updated_at = NOW()
         RETURNING id`,
		tenantID, req.Host, req.Port, keyType, fp, req.Comment).Scan(&id)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	kernel.RespondOK(c, gin.H{
		"id":          id,
		"host":        req.Host,
		"port":        req.Port,
		"key_type":    keyType,
		"fingerprint": fp,
		"comment":     req.Comment,
		"status":      "trusted",
	})
}

// DeleteKnownHost removes a fingerprint (host will need re-trust).
func (h *TerminalHandler) DeleteKnownHost(c *gin.Context) {
	tenantID := h.tenantID(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	tag, err := h.pool.Pgx().Exec(c.Request.Context(),
		`DELETE FROM known_hosts WHERE tenant_id = $1 AND id = $2`, tenantID, id)
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

// captureHostKey connects to host:port and returns the SSH server's host key
// type + SHA256 fingerprint. Uses HostKeyCallback to intercept the key
// before auth — this is the canonical Go-SSH pattern.
func captureHostKey(host string, port int) (string, string, error) {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return "", "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Captured via closure — the callback receives the host key during handshake
	var capturedKey ssh.PublicKey
	captured := make(chan error, 1)
	callback := func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		capturedKey = key
		// Return an error to abort auth AFTER capturing the key.
		// Many SSH servers keep the connection alive long enough for the
		// handshake to complete and the callback to fire before the close.
		captured <- nil
		return errors.New("capture-only")
	}
	go func() {
		_, _, _, _ = ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
			User:            "capture",
			HostKeyCallback: callback,
			Auth:            []ssh.AuthMethod{ssh.Password("none")},
			Timeout:         5 * time.Second,
		})
		select {
		case captured <- nil:
		default:
		}
	}()
	// Wait for callback to fire (or 5s timeout)
	select {
	case <-captured:
	case <-time.After(5 * time.Second):
	}
	if capturedKey == nil {
		return "", "", errors.New("host key callback never invoked")
	}
	return capturedKey.Type(), ssh.FingerprintSHA256(capturedKey), nil
}
