// Tier 5 — Images (C7).
//
//	GET    /api/v1/containers/images?connection_id=X
//	POST   /api/v1/containers/images/pull?connection_id=X  body: {"image": "nginx:latest"}
//	DELETE /api/v1/containers/images/:id?connection_id=X
package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// imageJSON is one row of `docker images --format '{{json .}}'`.
type imageJSON struct {
	Containers   string `json:"Containers"`
	CreatedAt    string `json:"CreatedAt"`
	CreatedSince string `json:"CreatedSince"`
	Digest       string `json:"Digest"`
	ID           string `json:"ID"`
	Repository   string `json:"Repository"`
	SharedSize   string `json:"SharedSize"`
	Size         string `json:"Size"`
	Tag          string `json:"Tag"`
}

// ListImages (C7 list) returns docker images on the target.
func (h *ContainerHandler) ListImages(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	cmd := "docker images --format '{{json .}}' 2>/dev/null"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 15000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "images": []imageJSON{}})
		return
	}
	rows := parseDockerJSONLinesAs(resp.Stdout, func() imageJSON { return imageJSON{} })
	c.JSON(200, gin.H{
		"images": rows,
		"total":  len(rows),
	})
}

// PullImage (C7 pull) pulls an image.
//
// body: {"image": "nginx:latest"}
func (h *ContainerHandler) PullImage(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	var body struct {
		Image string `json:"image" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"error": "image is required"})
		return
	}
	cmd := "docker pull " + shellQuote(body.Image) + " 2>&1"
	// docker pull can take a while — 5min timeout
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 300000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(500, gin.H{
			"image":     body.Image,
			"exit_code": resp.ExitCode,
			"stdout":    strings.TrimSpace(resp.Stdout),
			"stderr":    strings.TrimSpace(resp.Stderr),
			"error":     "docker pull exited " + itoa(resp.ExitCode),
		})
		return
	}
	c.JSON(200, gin.H{
		"image":  body.Image,
		"status": "pulled",
		"output": strings.TrimSpace(resp.Stdout),
	})
}

// RemoveImage (C7 rm) removes an image.
//
// DELETE /api/v1/containers/images/:id?connection_id=X
func (h *ContainerHandler) RemoveImage(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing image id"})
		return
	}
	force := c.Query("force") == "true"
	args := "rmi"
	if force {
		args = "rmi -f"
	}
	cmd := "docker " + args + " " + shellQuote(id) + " 2>&1"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 60000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(500, gin.H{
			"id":        id,
			"exit_code": resp.ExitCode,
			"stdout":    strings.TrimSpace(resp.Stdout),
			"stderr":    strings.TrimSpace(resp.Stderr),
			"error":     "docker rmi exited " + itoa(resp.ExitCode),
		})
		return
	}
	c.JSON(200, gin.H{
		"id":     id,
		"status": "removed",
	})
}
