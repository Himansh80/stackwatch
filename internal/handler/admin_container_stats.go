// Tier 5 — Container stats (C5).
//
//   GET /api/v1/containers/:id/stats?connection_id=X[&no_stream=true]
//
// Uses `docker stats --no-stream --format '{{json .}}'` for a single
// snapshot. Returns CPU%, mem usage, net I/O, block I/O.
package handler

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
)

// dockerStatsRow is one row of `docker stats --no-stream --format '{{json .}}'`.
type dockerStatsRow struct {
	BlockIO       string `json:"BlockIO"`
	CPUPerc       string `json:"CPUPerc"`
	Container     string `json:"Container"`
	ID            string `json:"ID"`
	MemPerc       string `json:"MemPerc"`
	MemUsage      string `json:"MemUsage"`
	Name          string `json:"Name"`
	NetIO         string `json:"NetIO"`
	PIDs          string `json:"PIDs"`
}

// ContainerStats returns one snapshot of stats for a container.
func (h *ContainerHandler) ContainerStats(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	id := c.Param("id")
	if id == "" {
		c.JSON(400, gin.H{"error": "missing container id"})
		return
	}

	cmd := "docker stats --no-stream --format '{{json .}}' " + shellQuote(id) + " 2>&1"
	resp, runErr := h.admin.SSHExec(c, cid, cmd, 15000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error()})
		return
	}

	rows := parseDockerJSONLinesAs(resp.Stdout, func() dockerStatsRow { return dockerStatsRow{} })
	if len(rows) == 0 {
		c.JSON(200, gin.H{
			"stats": nil,
			"warning": "no stats output: " + strings.TrimSpace(resp.Stderr),
		})
		return
	}

	c.JSON(200, gin.H{"stats": rows[0]})
}

// parseDockerJSONLinesAs is a generic version of parseDockerJSONLines that
// parses each line into the provided type via JSON.
func parseDockerJSONLinesAs[T any](s string, _ func() T) []T {
	out := []T{}
	for _, line := range splitLines(s) {
		if line == "" {
			continue
		}
		var v T
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			continue
		}
		out = append(out, v)
	}
	return out
}
