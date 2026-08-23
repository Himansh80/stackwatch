// Tier 5 — Stacks (C10).
//
//	GET    /api/v1/containers/stacks?connection_id=X
//	POST   /api/v1/containers/stacks?connection_id=X
//
// `docker stack` requires Swarm mode (docker swarm init). For non-Swarm
// hosts we fall back to scanning `/opt/stacks/*.yaml` (common convention).
// Deploys require Swarm — non-Swarm hosts get a 4xx with a clear message.
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// stackJSON describes one stack.
type stackJSON struct {
	Name         string `json:"name"`
	Services     int    `json:"services"`
	Orchestrator string `json:"orchestrator"` // "swarm" or "compose-files"
}

// ListStacks lists stacks. Tries Swarm first, falls back to /opt/stacks.
func (h *ContainerHandler) ListStacks(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	// Try `docker stack ls` first; on non-zero exit (likely not Swarm),
	// fall back to /opt/stacks.
	cmd := "docker stack ls --format '{{json .}}' 2>/dev/null"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 10000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "stacks": []stackJSON{}})
		return
	}

	if resp.ExitCode == 0 && strings.TrimSpace(resp.Stdout) != "" {
		// Swarm mode — use docker stack ls JSON output
		type swarmStack struct {
			Name     string `json:"Name"`
			Services string `json:"Services"`
		}
		rows := parseDockerJSONLinesAs(resp.Stdout, func() swarmStack { return swarmStack{} })
		out := make([]stackJSON, 0, len(rows))
		for _, r := range rows {
			out = append(out, stackJSON{Name: r.Name, Orchestrator: "swarm", Services: len(r.Services)})
		}
		c.JSON(200, gin.H{"stacks": out, "total": len(out)})
		return
	}

	// Non-Swarm: scan /opt/stacks/*.yaml
	fbCmd := "ls /opt/stacks/*.yaml /opt/stacks/*.yml 2>/dev/null || echo ''"
	fb, _ := h.admin.SSHExec(c, cid, fbCmd, 5000)
	stacks := []stackJSON{}
	for _, f := range splitLines(fb.Stdout) {
		if f == "" {
			continue
		}
		// Extract name from filename
		parts := strings.Split(f, "/")
		name := strings.TrimSuffix(parts[len(parts)-1], ".yaml")
		name = strings.TrimSuffix(name, ".yml")
		stacks = append(stacks, stackJSON{
			Name:         name,
			Orchestrator: "compose-files",
			Services:     0, // unknown without parsing
		})
	}
	c.JSON(200, gin.H{
		"stacks":       stacks,
		"total":        len(stacks),
		"orchestrator": "compose-files",
	})
}

// DeployStack deploys a stack. Requires Swarm mode for compose files.
//
// body: {"name": "webstack", "compose_file": "base64-encoded compose YAML"}
//
// On non-Swarm hosts we save the compose file under /opt/stacks/<name>.yaml
// so the user can later `cd /opt/stacks && docker compose up -d`.
func (h *ContainerHandler) DeployStack(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	var body struct {
		Name        string `json:"name" binding:"required"`
		ComposeFile string `json:"compose_file" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "name and compose_file are required"})
		return
	}

	// Always write the compose file to /opt/stacks/<name>.yaml first
	writeCmd := "mkdir -p /opt/stacks && cat > /opt/stacks/" + shellQuote(body.Name) + ".yaml << 'STACK_EOF'\n" +
		body.ComposeFile + "\nSTACK_EOF\n"
	writeCmd += "echo saved"
	// Combine: write, then try swarm deploy
	cmd := writeCmd + " && (docker stack deploy --compose-file /opt/stacks/" + shellQuote(body.Name) + ".yaml " + shellQuote(body.Name) + " 2>&1 || echo 'NOT_SWARM')"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 60000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	out := strings.TrimSpace(resp.Stdout)
	if strings.Contains(out, "NOT_SWARM") {
		c.JSON(200, gin.H{
			"name":         body.Name,
			"status":       "saved",
			"orchestrator": "compose-files",
			"path":         "/opt/stacks/" + body.Name + ".yaml",
			"note":         "non-Swarm host — file saved, run 'docker compose up -d' manually",
		})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(500, gin.H{
			"name":      body.Name,
			"exit_code": resp.ExitCode,
			"stderr":    strings.TrimSpace(resp.Stderr),
			"error":     "docker stack deploy exited " + itoa(resp.ExitCode),
		})
		return
	}
	c.JSON(201, gin.H{
		"name":         body.Name,
		"status":       "deployed",
		"orchestrator": "swarm",
	})
}
