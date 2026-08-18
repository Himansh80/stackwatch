// Tier 5 — Container create (C6).
//
//   POST /api/v1/containers?connection_id=X
//   body: {
//     "image": "nginx:latest",       (required)
//     "name": "webserver",           (optional, --name)
//     "ports": ["8080:80"],          (optional, -p)
//     "env": ["KEY=VAL"],            (optional, -e)
//     "volumes": ["/host/path:/container/path"], (optional, -v)
//     "network": "bridge",           (optional, --network)
//     "detach": true,                (default true; false = --interactive --tty)
//     "rm": false,                   (default false; true = --rm)
//     "restart": "no",               (optional, --restart)
//     "command": ["arg1", "arg2"]    (optional, overrides image CMD)
//   }
package handler

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// createRequest is the body for POST /containers.
type createRequest struct {
	Image   string   `json:"image" binding:"required"`
	Name    string   `json:"name"`
	Ports   []string `json:"ports"`
	Env     []string `json:"env"`
	Volumes []string `json:"volumes"`
	Network string   `json:"network"`
	Detach  *bool    `json:"detach"`  // pointer so we can default to true
	RM      bool     `json:"rm"`
	Restart string   `json:"restart"`
	Command []string `json:"command"`
}

// CreateContainer runs `docker run ...` on the target.
func (h *ContainerHandler) CreateContainer(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "bad request body: " + err.Error()})
		return
	}
	if req.Image == "" {
		c.JSON(400, gin.H{"error": "image is required"})
		return
	}

	detach := true
	if req.Detach != nil {
		detach = *req.Detach
	}

	args := []string{}
	if detach {
		args = append(args, "-d")
	} else {
		args = append(args, "-it")
	}
	if req.Name != "" {
		args = append(args, "--name", shellQuote(req.Name))
	}
	for _, p := range req.Ports {
		args = append(args, "-p", shellQuote(p))
	}
	for _, e := range req.Env {
		args = append(args, "-e", shellQuote(e))
	}
	for _, v := range req.Volumes {
		args = append(args, "-v", shellQuote(v))
	}
	if req.Network != "" {
		args = append(args, "--network", shellQuote(req.Network))
	}
	if req.RM {
		args = append(args, "--rm")
	}
	if req.Restart != "" {
		args = append(args, "--restart", shellQuote(req.Restart))
	}
	args = append(args, shellQuote(req.Image))
	for _, c := range req.Command {
		args = append(args, shellQuote(c))
	}

	cmd := "docker run " + strings.Join(args, " ") + " 2>&1"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 60000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(500, gin.H{
			"image": req.Image,
			"exit_code": resp.ExitCode,
			"stdout": strings.TrimSpace(resp.Stdout),
			"stderr": strings.TrimSpace(resp.Stderr),
			"error": "docker run exited " + fmt.Sprint(resp.ExitCode),
		})
		return
	}
	// stdout is the new container ID
	id := strings.TrimSpace(resp.Stdout)
	c.JSON(201, gin.H{
		"id":     id,
		"image":  req.Image,
		"name":   req.Name,
		"status": "created",
	})
}
