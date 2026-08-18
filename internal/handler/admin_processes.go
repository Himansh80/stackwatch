// Tier 4 S4: Processes.
//
//   GET /api/v1/admin/processes?connection_id=X[&sort=cpu|mem][&limit=N][&user=root]
//
// Uses `ps -eo pid,user,pcpu,pmem,comm,args --sort=-pcpu` for portability.
package handler

import (
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Process is one row from `ps`.
type Process struct {
	PID  int     `json:"pid"`
	User string  `json:"user"`
	CPU  float64 `json:"cpu_pct"`
	MEM  float64 `json:"mem_pct"`
	Comm string  `json:"comm"`
	Args string  `json:"args"`
}

// ListProcesses returns running processes on the target.
//
// Query params:
//   - connection_id (required)
//   - sort: cpu (default) or mem
//   - limit: max rows (default 50, capped at 500)
//   - user: filter by user (optional)
func (h *AdminHandler) ListProcesses(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	sortKey := c.DefaultQuery("sort", "cpu")
	if sortKey != "cpu" && sortKey != "mem" {
		sortKey = "cpu"
	}
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	user := c.Query("user")

	// Build ps command. `ps -eo` is the portable form.
	// We fetch a wider set (200 rows) then trim+sort in Go to avoid
	// depending on the target's ps flags.
	cmd := "ps -eo pid,user,pcpu,pmem,comm,args --no-headers 2>/dev/null | head -200"
	resp, runErr := h.SSHExec(c, cid, cmd, 15000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "processes": []Process{}})
		return
	}

	procs := parsePsOutput(resp.Stdout)

	// Filter by user
	if user != "" {
		filtered := procs[:0]
		for _, p := range procs {
			if p.User == user {
				filtered = append(filtered, p)
			}
		}
		procs = filtered
	}

	// Sort
	if sortKey == "mem" {
		sort.Slice(procs, func(i, j int) bool { return procs[i].MEM > procs[j].MEM })
	} else {
		sort.Slice(procs, func(i, j int) bool { return procs[i].CPU > procs[j].CPU })
	}

	// Limit
	if len(procs) > limit {
		procs = procs[:limit]
	}

	c.JSON(200, gin.H{
		"processes": procs,
		"total":     len(procs),
		"sort":      sortKey,
	})
}

// parsePsOutput parses the text output of `ps -eo pid,user,pcpu,pmem,comm,args`.
//
// ps output is whitespace-separated with args potentially containing spaces.
// We split the first 5 fields then treat the rest as args.
func parsePsOutput(s string) []Process {
	out := []Process{}
	for _, line := range splitLines(s) {
		if line == "" {
			continue
		}
		// split into first 5 fields, rest is args
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		pid, _ := strconv.Atoi(fields[0])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)
		comm := fields[4]
		var args string
		if len(fields) > 5 {
			args = strings.Join(fields[5:], " ")
		}
		out = append(out, Process{
			PID:  pid,
			User: fields[1],
			CPU:  cpu,
			MEM:  mem,
			Comm: comm,
			Args: args,
		})
	}
	return out
}
