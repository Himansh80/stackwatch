// Tier 5 — Container Management (Portainer + Watchtower parity).
//
// All endpoints under /api/v1/containers/* take a `connection_id` query
// param (the Tier 3 SSH connection pointing at the docker host) and run
// `docker` CLI commands via the Tier 4 web-terminal SSH bridge.
//
// Architecture: api-gateway → web-terminal:8085/admin/exec → SSH → docker CLI
package handler

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
)

// containerFields are the docker inspect fields we surface.
type containerJSON struct {
	ID            string   `json:"ID"`
	Name          string   `json:"Names"` // list endpoint uses "Names", inspect uses "Name"
	Image         string   `json:"Image"`
	ImageID       string   `json:"ImageID"`
	Command       string   `json:"Command"`
	CreatedAt     string   `json:"CreatedAt"`
	State         string   `json:"State"`
	Status        string   `json:"Status"`
	Ports         string   `json:"Ports"`
	Labels        string   `json:"Labels"`
	Networks      string   `json:"Networks"`
	Mounts        string   `json:"Mounts"`
	HealthStatus  string   `json:"HealthStatus"`
	// Inspect fields (extra)
	Env           []string `json:"Env,omitempty"`
	Cmd           []string `json:"Cmd,omitempty"`
	ArgsEscaped   bool     `json:"ArgsEscaped,omitempty"`
	ImageName     string   `json:"ImageName,omitempty"`
}

// normalizedContainer is the response shape — merged list + inspect fields
// so the frontend doesn't need two calls per row.
type normalizedContainer struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Image        string `json:"image"`
	ImageID      string `json:"image_id"`
	State        string `json:"state"`
	Status       string `json:"status"`
	Health       string `json:"health"`
	CreatedAt    string `json:"created_at"`
	Ports        string `json:"ports"`
	Networks     string `json:"networks"`
	Labels       string `json:"labels"`
}

// ContainerHandler bundles all Tier 5 container endpoints. It reuses the
// Tier 4 AdminHandler's SSH bridge (AdminHandler.SSHExec).
type ContainerHandler struct {
	admin *AdminHandler
}

// NewContainerHandler constructs the handler.
func NewContainerHandler(admin *AdminHandler) *ContainerHandler {
	return &ContainerHandler{admin: admin}
}

// ListContainers (C1 list)
//
// GET /api/v1/containers?connection_id=X[&all=true]
//
// `all=true` (default) returns running + stopped. `all=false` returns only running.
func (h *ContainerHandler) ListContainers(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	all := c.DefaultQuery("all", "true")
	flag := "-a"
	if all == "false" {
		flag = ""
	}
	// docker ps uses "Names" not "Name" — single JSON per line.
	cmd := "docker ps " + flag + " --no-trunc --format '{{json .}}' 2>/dev/null"

	resp, err := h.admin.SSHExec(c, cid, cmd, 15000)
	if err != nil {
		c.JSON(502, gin.H{"error": "ssh: " + err.Error(), "containers": []normalizedContainer{}})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(200, gin.H{
			"containers": []normalizedContainer{},
			"total":      0,
			"warning":    "docker exited " + itoa(resp.ExitCode) + ": " + strings.TrimSpace(resp.Stderr),
		})
		return
	}

	rows := parseDockerJSONLines(resp.Stdout)
	out := make([]normalizedContainer, 0, len(rows))
	for _, raw := range rows {
		// docker ps uses "Names" (comma-separated for multiple), inspect uses "Name"
		name := raw.Name
		if name == "" {
			// Names field is comma-separated — take first
			if raw.Name == "" {
				// try unmarshaling the raw map to get "Names"
				var m map[string]interface{}
				_ = json.Unmarshal([]byte(rawAsString(raw)), &m)
				if v, ok := m["Names"].(string); ok {
					name = strings.SplitN(v, ",", 2)[0]
				}
			}
		}
		out = append(out, normalizedContainer{
			ID:        raw.ID,
			Name:      name,
			Image:     raw.Image,
			ImageID:   raw.ImageID,
			State:     raw.State,
			Status:    raw.Status,
			Health:    raw.HealthStatus,
			CreatedAt: raw.CreatedAt,
			Ports:     raw.Ports,
			Networks:  raw.Networks,
			Labels:    raw.Labels,
		})
	}

	c.JSON(200, gin.H{
		"containers": out,
		"total":      len(out),
	})
}

// GetContainer (C1 inspect)
//
// GET /api/v1/containers/:id?connection_id=X
func (h *ContainerHandler) GetContainer(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing container id"})
		return
	}

	cmd := "docker inspect " + shellQuote(id) + " 2>/dev/null"
	resp, err := h.admin.SSHExec(c, cid, cmd, 10000)
	if err != nil {
		c.JSON(502, gin.H{"error": "ssh: " + err.Error()})
		return
	}
	if resp.ExitCode != 0 {
		c.JSON(200, gin.H{"container": nil, "warning": strings.TrimSpace(resp.Stderr)})
		return
	}

	// docker inspect returns a JSON array; we return the first element
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Stdout), &arr); err != nil || len(arr) == 0 {
		c.JSON(200, gin.H{"container": nil, "warning": "no inspect data"})
		return
	}
	c.JSON(200, gin.H{"container": arr[0]})
}

// rawAsString is a no-op formatter used to keep parseDockerJSONLines simple.
func rawAsString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

// parseDockerJSONLines parses the output of `docker --format '{{json .}}'`,
// which emits one JSON object per line.
func parseDockerJSONLines(s string) []containerJSON {
	out := []containerJSON{}
	for _, line := range splitLines(s) {
		if line == "" {
			continue
		}
		var c containerJSON
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}
		if c.ID == "" {
			continue
		}
		out = append(out, c)
	}
	return out
}
