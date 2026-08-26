// Package handler — SFTP file browser (Tier 3.3).
//
// Browse, read, write, mkdir, rename, delete files on remote hosts via SSH.
// Uses pkg/sftp (golang.org/x/crypto/ssh underneath).
package handler

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/stackwatch/platform/internal/kernel"
)

// FSEntry describes one file/directory in a remote listing.
type FSEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
	Owner   string `json:"owner,omitempty"`
	Group   string `json:"group,omitempty"`
}

// ListFS returns directory contents for a connection.
func (h *TerminalHandler) ListFS(c *gin.Context) {
	tenantID := h.tenantID(c)
	connID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	path := c.DefaultQuery("path", "/")
	if !filepath.IsAbs(path) {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path must be absolute"))
		return
	}

	client, sftpClient, err := h.dialSFTP(c.Request.Context(), tenantID, connID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer sftpClient.Close()
	defer client.Close()

	files, err := sftpClient.ReadDir(path)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("read dir: %w", err))
		return
	}

	entries := make([]FSEntry, 0, len(files))
	for _, f := range files {
		entries = append(entries, FSEntry{
			Name:    f.Name(),
			Path:    filepath.Join(path, f.Name()),
			Size:    f.Size(),
			Mode:    f.Mode().String(),
			IsDir:   f.IsDir(),
			ModTime: f.ModTime().UTC().Format(time.RFC3339),
		})
	}
	kernel.RespondOK(c, gin.H{
		"path":    path,
		"entries": entries,
		"total":   len(entries),
	})
}

// ReadFS reads a file from the remote host.
func (h *TerminalHandler) ReadFS(c *gin.Context) {
	tenantID := h.tenantID(c)
	connID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	path := c.Query("path")
	if path == "" {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path required"))
		return
	}
	if !filepath.IsAbs(path) {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path must be absolute"))
		return
	}
	maxBytes := int64(65536)
	if mb := c.Query("max_bytes"); mb != "" {
		fmt.Sscanf(mb, "%d", &maxBytes)
	}
	if maxBytes > 1048576 {
		maxBytes = 1048576 // cap at 1MB
	}

	client, sftpClient, err := h.dialSFTP(c.Request.Context(), tenantID, connID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer sftpClient.Close()
	defer client.Close()

	f, err := sftpClient.Open(path)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("open: %w", err))
		return
	}
	defer f.Close()
	// Get file info to know total size
	fi, err := f.Stat()
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("stat: %w", err))
		return
	}
	// LimitReader: only read up to maxBytes
	buf := make([]byte, maxBytes)
	n, _ := io.ReadFull(io.LimitReader(f, maxBytes), buf)
	truncated := int64(n) < fi.Size()
	kernel.RespondOK(c, gin.H{
		"path":        path,
		"size":        fi.Size(),
		"max_bytes":   maxBytes,
		"bytes_read":  n,
		"truncated":   truncated,
		"content_b64": base64Encode(buf[:n]),
		"mod_time":    fi.ModTime().UTC().Format(time.RFC3339),
		"mode":        fi.Mode().String(),
	})
}

// WriteFS writes a file (or appends) to the remote host.
type WriteFSRequest struct {
	Path string `json:"path" binding:"required"`
	// ContentB64 is the base64-encoded file content. Required unless
	// Content is provided (UTF-8 text convenience field).
	ContentB64 string `json:"content_b64"`
	// Content is UTF-8 text. Auto-encoded to base64 if ContentB64 is empty.
	// Useful for "create a small config file" flows where base64 is annoying.
	Content string `json:"content"`
	Append  bool   `json:"append"`
	Mode    string `json:"mode"` // e.g. "0644" — applied on create
}

func (h *TerminalHandler) WriteFS(c *gin.Context) {
	tenantID := h.tenantID(c)
	connID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req WriteFSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if !filepath.IsAbs(req.Path) {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path must be absolute"))
		return
	}
	if req.Path == "" {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	// Resolve body → bytes. Prefer content_b64; fall back to content (UTF-8).
	var content []byte
	switch {
	case req.ContentB64 != "":
		content, err = base64Decode(req.ContentB64)
		if err != nil {
			kernel.RespondError(c, fmt.Errorf("decode b64: %w", err))
			return
		}
	case req.Content != "":
		content = []byte(req.Content)
	default:
		kernel.RespondError(c, kernel.ErrBadRequest) // need content or content_b64
		return
	}

	client, sftpClient, err := h.dialSFTP(c.Request.Context(), tenantID, connID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer sftpClient.Close()
	defer client.Close()

	flags := os.O_WRONLY | os.O_CREATE
	if req.Append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := sftpClient.OpenFile(req.Path, flags)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("open for write: %w", err))
		return
	}
	defer f.Close()
	written, err := f.Write(content)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("write: %w", err))
		return
	}
	// Apply mode if specified
	if req.Mode != "" {
		var mode uint32
		fmt.Sscanf(req.Mode, "%o", &mode)
		_ = sftpClient.Chmod(req.Path, os.FileMode(mode))
	}
	kernel.RespondOK(c, gin.H{
		"path":   req.Path,
		"bytes":  written,
		"status": "written",
	})
}

// MkdirFS creates a directory.
type MkdirFSRequest struct {
	Path string `json:"path" binding:"required"`
	Mode string `json:"mode"` // optional, default 0755
}

func (h *TerminalHandler) MkdirFS(c *gin.Context) {
	tenantID := h.tenantID(c)
	connID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req MkdirFSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if !filepath.IsAbs(req.Path) {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path must be absolute"))
		return
	}

	client, sftpClient, err := h.dialSFTP(c.Request.Context(), tenantID, connID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer sftpClient.Close()
	defer client.Close()

	mode := os.FileMode(0o755)
	if req.Mode != "" {
		var m uint32
		fmt.Sscanf(req.Mode, "%o", &m)
		mode = os.FileMode(m)
	}
	if err := sftpClient.Mkdir(req.Path); err != nil {
		kernel.RespondError(c, fmt.Errorf("mkdir: %w", err))
		return
	}
	_ = sftpClient.Chmod(req.Path, mode)
	kernel.RespondOK(c, gin.H{
		"path":   req.Path,
		"mode":   mode.String(),
		"status": "created",
	})
}

// DeleteFS removes a file or directory.
func (h *TerminalHandler) DeleteFS(c *gin.Context) {
	tenantID := h.tenantID(c)
	connID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	path := c.Query("path")
	if path == "" {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path required"))
		return
	}
	if !filepath.IsAbs(path) {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path must be absolute"))
		return
	}

	client, sftpClient, err := h.dialSFTP(c.Request.Context(), tenantID, connID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer sftpClient.Close()
	defer client.Close()

	// Try Remove first (file), then RemoveDirectory
	if err := sftpClient.Remove(path); err != nil {
		// Fall back to RemoveDirectory
		if err2 := sftpClient.RemoveDirectory(path); err2 != nil {
			kernel.RespondError(c, fmt.Errorf("delete: file=%v dir=%v", err, err2))
			return
		}
	}
	kernel.RespondOK(c, gin.H{
		"path":   path,
		"status": "deleted",
	})
}

// StatFS returns info about a file/dir.
func (h *TerminalHandler) StatFS(c *gin.Context) {
	tenantID := h.tenantID(c)
	connID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	path := c.Query("path")
	if path == "" {
		path = "/"
	}
	if !filepath.IsAbs(path) {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("path must be absolute"))
		return
	}

	client, sftpClient, err := h.dialSFTP(c.Request.Context(), tenantID, connID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer sftpClient.Close()
	defer client.Close()

	fi, err := sftpClient.Stat(path)
	if err != nil {
		kernel.RespondError(c, fmt.Errorf("stat: %w", err))
		return
	}
	kernel.RespondOK(c, gin.H{
		"path":     path,
		"name":     fi.Name(),
		"size":     fi.Size(),
		"mode":     fi.Mode().String(),
		"is_dir":   fi.IsDir(),
		"mod_time": fi.ModTime().UTC().Format(time.RFC3339),
	})
}

// RenameFS renames/moves a file or directory.
type RenameFSRequest struct {
	OldPath string `json:"old_path" binding:"required"`
	NewPath string `json:"new_path" binding:"required"`
}

func (h *TerminalHandler) RenameFS(c *gin.Context) {
	tenantID := h.tenantID(c)
	connID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	var req RenameFSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		kernel.RespondError(c, kernel.ErrBadRequest)
		return
	}
	if !filepath.IsAbs(req.OldPath) || !filepath.IsAbs(req.NewPath) {
		kernel.RespondError(c, kernel.ErrBadRequest) // was: kernel.NewErrBadRequest("paths must be absolute"))
		return
	}

	client, sftpClient, err := h.dialSFTP(c.Request.Context(), tenantID, connID)
	if err != nil {
		kernel.RespondError(c, err)
		return
	}
	defer sftpClient.Close()
	defer client.Close()

	if err := sftpClient.Rename(req.OldPath, req.NewPath); err != nil {
		kernel.RespondError(c, fmt.Errorf("rename: %w", err))
		return
	}
	kernel.RespondOK(c, gin.H{
		"old_path": req.OldPath,
		"new_path": req.NewPath,
		"status":   "renamed",
	})
}

// dialSFTP opens an SSH + SFTP client for a saved connection.
// Uses the unified auth helper (Tier 3.6) so password auth works too.
func (h *TerminalHandler) dialSFTP(ctx context.Context, tenantID, connID uuid.UUID) (*ssh.Client, *sftp.Client, error) {
	authRes, err := h.dialConnection(ctx, tenantID, connID, 15*time.Second)
	if err != nil {
		return nil, nil, err
	}
	sftpClient, err := sftp.NewClient(authRes.Client)
	if err != nil {
		authRes.Client.Close()
		return nil, nil, fmt.Errorf("sftp new: %w", err)
	}
	return authRes.Client, sftpClient, nil
}

// base64Encode / Decode wrappers (avoid importing encoding/base64 in every helper)
func base64Encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
func base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// ensure imports referenced
var (
	_ = strings.HasPrefix
	_ = io.Copy
)
