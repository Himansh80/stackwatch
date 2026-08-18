// Tier 5 — Container logs (C4).
//
//   GET /api/v1/containers/:id/logs?connection_id=X&tail=100&since=1h
package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContainerLogs returns the most recent N lines of logs for a container.
//
// Query params:
//   - connection_id (required)
//   - tail: number of lines (default 100, capped at 10000)
//   - since: time spec like "1h", "30m" (optional)
//   - timestamps: bool — show timestamps (default false)
func (h *ContainerHandler) ContainerLogs(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing container id"})
		return
	}

	tailStr := c.DefaultQuery("tail", "100")
	tail, err := strconv.Atoi(tailStr)
	if err != nil || tail <= 0 {
		tail = 100
	}
	if tail > 10000 {
		tail = 10000
	}
	since := c.Query("since")
	timestamps := c.Query("timestamps") == "true"

	args := []string{"--tail", itoa(tail)}
	if since != "" {
		args = append(args, "--since", shellQuote(since))
	}
	if timestamps {
		args = append(args, "--timestamps")
	}
	args = append(args, shellQuote(id))

	cmd := "docker logs " + strings.Join(args, " ") + " 2>&1"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 30000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "logs": []string{}})
		return
	}

	// docker logs outputs stderr + stdout interleaved. Split lines.
	lines := splitLines(resp.Stdout)
	c.JSON(200, gin.H{
		"logs":       lines,
		"total":      len(lines),
		"id":         id,
		"exit_code":  resp.ExitCode,
		"stderr":     strings.TrimSpace(resp.Stderr),
	})
}
