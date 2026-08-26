// Tier 5 — Volumes (C8).
//
//	GET    /api/v1/containers/volumes?connection_id=X
//	POST   /api/v1/containers/volumes?connection_id=X  body: {"name": "data"}
//	DELETE /api/v1/containers/volumes/:name?connection_id=X[&force=true]
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// volumeJSON is one row of `docker volume ls --format '{{json .}}'`.
type volumeJSON struct {
	Driver     string `json:"Driver"`
	Labels     string `json:"Labels"`
	Mountpoint string `json:"Mountpoint"`
	Name       string `json:"Name"`
	Scope      string `json:"Scope"`
	Size       string `json:"Size"`
}

// ListVolumes returns all docker volumes.
func (h *ContainerHandler) ListVolumes(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	cmd := "docker volume ls --format '{{json .}}' 2>/dev/null"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 10000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "volumes": []volumeJSON{}})
		return
	}
	rows := parseDockerJSONLinesAs(resp.Stdout, func() volumeJSON { return volumeJSON{} })
	c.JSON(200, gin.H{
		"volumes": rows,
		"total":   len(rows),
	})
}

// CreateVolume creates a new named volume.
func (h *ContainerHandler) CreateVolume(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	var body struct {
		Name   string `json:"name" binding:"required"`
		Driver string `json:"driver"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "name is required"})
		return
	}
	cmd := "docker volume create " + shellQuote(body.Name)
	if body.Driver != "" {
		cmd += " --driver " + shellQuote(body.Driver)
	}
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 10000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(500, gin.H{
			"name":      body.Name,
			"exit_code": resp.ExitCode,
			"stderr":    strings.TrimSpace(resp.Stderr),
			"error":     "docker volume create exited " + itoa(resp.ExitCode),
		})
		return
	}
	c.JSON(201, gin.H{
		"name":   body.Name,
		"status": "created",
	})
}

// RemoveVolume removes a volume.
func (h *ContainerHandler) RemoveVolume(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	name := c.Param("name")
	if name == "" {
		c.JSON(400, gin.H{"error": "missing volume name"})
		return
	}
	force := c.Query("force") == "true"
	args := "rm"
	if force {
		args = "rm -f"
	}
	cmd := "docker volume " + args + " " + shellQuote(name) + " 2>&1"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 10000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(500, gin.H{
			"name":      name,
			"exit_code": resp.ExitCode,
			"stderr":    strings.TrimSpace(resp.Stderr),
			"error":     "docker volume rm exited " + itoa(resp.ExitCode),
		})
		return
	}
	c.JSON(200, gin.H{
		"name":   name,
		"status": "removed",
	})
}
