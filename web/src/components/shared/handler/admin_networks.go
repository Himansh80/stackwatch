// Tier 5 — Networks (C9).
//
//	GET    /api/v1/containers/networks?connection_id=X
//	POST   /api/v1/containers/networks?connection_id=X  body: {"name": "frontend"}
//	DELETE /api/v1/containers/networks/:id?connection_id=X
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// networkJSON is one row of `docker network ls --format '{{json .}}'`.
type networkJSON struct {
	CreatedAt string `json:"CreatedAt"`
	Driver    string `json:"Driver"`
	ID        string `json:"ID"`
	Name      string `json:"Name"`
	Scope     string `json:"Scope"`
	IPv6      string `json:"IPv6"`
	Internal  string `json:"Internal"`
	Labels    string `json:"Labels"`
}

// ListNetworks returns all docker networks.
func (h *ContainerHandler) ListNetworks(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	cmd := "docker network ls --format '{{json .}}' 2>/dev/null"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 10000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "networks": []networkJSON{}})
		return
	}
	rows := parseDockerJSONLinesAs(resp.Stdout, func() networkJSON { return networkJSON{} })
	c.JSON(200, gin.H{
		"networks": rows,
		"total":    len(rows),
	})
}

// CreateNetwork creates a new network.
func (h *ContainerHandler) CreateNetwork(c *gin.Context) {
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
	cmd := "docker network create " + shellQuote(body.Name)
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
			"error":     "docker network create exited " + itoa(resp.ExitCode),
		})
		return
	}
	// stdout is the new network ID
	c.JSON(201, gin.H{
		"name":   body.Name,
		"id":     strings.TrimSpace(resp.Stdout),
		"status": "created",
	})
}

// RemoveNetwork removes a network by id or name.
func (h *ContainerHandler) RemoveNetwork(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing network id"})
		return
	}
	cmd := "docker network rm " + shellQuote(id) + " 2>&1"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 10000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(500, gin.H{
			"id":        id,
			"exit_code": resp.ExitCode,
			"stderr":    strings.TrimSpace(resp.Stderr),
			"error":     "docker network rm exited " + itoa(resp.ExitCode),
		})
		return
	}
	c.JSON(200, gin.H{
		"id":     id,
		"status": "removed",
	})
}
