// Tier 5 — Container actions (C2).
//
//	POST /api/v1/containers/:id/:action?connection_id=X
//
// `action` must be in the explicit whitelist below. Anything else
// returns 400. Audit-logged via the existing audit_log table.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// allowedContainerActions is the explicit whitelist for container actions.
var allowedContainerActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"pause":   true,
	"unpause": true,
	"kill":    true,
	"rename":  true, // body: {"new_name": "..."}
	"exec":    true, // body: {"command": "..."} (one-shot)
}

// ContainerAction handles POST /containers/:id/:action.
//
//	id must be a container id (or name).
//	action must be in allowedContainerActions.
//	body (optional): JSON with extra params (new_name for rename, command for exec).
func (h *ContainerHandler) ContainerAction(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	action := c.Param("action")
	if !allowedContainerActions[action] {
		c.JSON(400, gin.H{
			"error": "action must be one of: start, stop, restart, pause, unpause, kill, rename, exec",
		})
		return
	}
	if id == "" {
		c.JSON(400, gin.H{"error": "missing container id"})
		return
	}

	// Build the docker command
	cmd := buildContainerActionCmd(id, action, c)
	if cmd == "" {
		// buildContainerActionCmd already wrote the response
		return
	}

	resp, err := h.admin.SSHExec(c, cid, cmd, 30000)
	if err != nil {
		c.JSON(502, gin.H{"error": "ssh: " + err.Error()})
		return
	}

	out := gin.H{
		"action":      action,
		"id":          id,
		"exit_code":   resp.ExitCode,
		"stdout":      strings.TrimSpace(resp.Stdout),
		"stderr":      strings.TrimSpace(resp.Stderr),
		"duration_ms": resp.DurationMs,
	}
	if resp.ExitCode != 0 {
		out["error"] = "docker " + action + " exited " + itoa(resp.ExitCode)
		c.JSON(500, out)
		return
	}
	c.JSON(200, out)
}

// buildContainerActionCmd builds the docker CLI command for the given
// action. For simple actions it just returns `docker <action> <id>`. For
// rename/exec it reads the JSON body for params.
//
// Returns the command string, or "" if an error response was already written.
func buildContainerActionCmd(id, action string, c *gin.Context) string {
	// Simple actions: no body needed
	switch action {
	case "start", "stop", "restart", "pause", "unpause", "kill":
		return "docker " + action + " " + shellQuote(id) + " 2>&1"
	case "rename":
		// body: {"new_name": "..."}
		var body struct {
			NewName string `json:"new_name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "rename requires body: {\"new_name\": \"...\"}"})
			return ""
		}
		return "docker rename " + shellQuote(id) + " " + shellQuote(body.NewName) + " 2>&1"
	case "exec":
		// body: {"command": "ls -la"}
		var body struct {
			Command string `json:"command" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": "exec requires body: {\"command\": \"...\"}"})
			return ""
		}
		return "docker exec " + shellQuote(id) + " " + body.Command + " 2>&1"
	}
	return ""
}
