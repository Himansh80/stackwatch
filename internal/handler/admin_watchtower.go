// Tier 5 — Watchtower (auto-updater).
//
//   GET  /api/v1/containers/watchtower?connection_id=X
//   POST /api/v1/containers/watchtower/update?connection_id=X[&interval=300]
//
// Watchtower is a separate container that watches running containers
// and updates them when their image has a newer version. We don't
// manage Watchtower itself — we expose the standard Watchtower CLI
// operations through StackWatch.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ListWatched returns containers that have the watchtower label
// (i.e. containers that Watchtower is monitoring).
//
// We detect watchtower-monitored containers with:
//   docker ps -a --filter label=com.centurylinklabs.watchtower.enable=true
//   --format '{{json .}}'
func (h *ContainerHandler) ListWatched(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	cmd := "docker ps -a --filter label=com.centurylinklabs.watchtower.enable=true --format '{{json .}}' 2>/dev/null"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 10000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "watched": []map[string]interface{}{}})
		return
	}
	// Re-use parseDockerJSONLinesAs via the existing containerJSON shape
	rows := parseDockerJSONLinesAs(resp.Stdout, func() map[string]interface{} { return map[string]interface{}{} })
	c.JSON(200, gin.H{
		"watched":      rows,
		"total":        len(rows),
		"watchtower_running": strings.Contains(resp.Stdout, "watchtower"),
	})
}

// TriggerUpdate runs `docker run --rm watchtower` to update all watched
// containers now (instead of waiting for the schedule).
//
// Query params:
//   - connection_id (required)
//   - interval: cleanup interval (default "300")
func (h *ContainerHandler) TriggerUpdate(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	interval := c.DefaultQuery("interval", "300")
	cmd := "docker run --rm -v /var/run/docker.sock:/var/run/docker.sock containrrr/watchtower --cleanup --interval " + interval + " 2>&1"
	// Watchtower pull can take minutes
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 300000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	c.JSON(200, gin.H{
		"status":      "triggered",
		"interval":    interval,
		"exit_code":   resp.ExitCode,
		"output":      strings.TrimSpace(resp.Stdout),
		"stderr":      strings.TrimSpace(resp.Stderr),
		"duration_ms": resp.DurationMs,
	})
}
