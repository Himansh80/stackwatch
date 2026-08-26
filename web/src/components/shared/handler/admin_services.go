// Tier 4 S1: Services (systemd units via systemctl).
//
//	GET  /api/v1/admin/services?connection_id=X[&state=active|inactive|failed]
//	POST /api/v1/admin/services/:connection_id/:action?unit=NAME
//
// The list endpoint runs `systemctl list-units --type=service --output=json`
// on the target host. The action endpoint runs `systemctl <action> NAME`.
//
// All actions are audit-logged via the existing audit_log table.
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// systemctlUnit is one row from `systemctl list-units --output=json`.
type systemctlUnit struct {
	Unit        string `json:"unit"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

// allowedActions is the explicit whitelist for service actions. Anything
// outside this set is rejected (prevents arbitrary systemctl verbs).
var allowedActions = map[string]bool{
	"start":   true,
	"stop":    true,
	"restart": true,
	"enable":  true,
	"disable": true,
	"reload":  true,
}

// ListServices returns the list of systemd service units on the target.
// Query: connection_id (required), state (optional: active|inactive|failed|all)
func (h *AdminHandler) ListServices(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}
	state := c.DefaultQuery("state", "all")

	// Build filter for systemctl. Use --state= for server-side filter.
	cmd := "systemctl list-units --type=service --all --no-pager --output=json 2>/dev/null"
	if state != "all" {
		cmd = "systemctl list-units --type=service --all --no-pager --state=" + state + " --output=json 2>/dev/null"
	}

	var units []systemctlUnit
	if err := h.SSHExecJSON(c, cid, cmd, 30000, &units); err != nil {
		// Fallback: return empty list with warning, not 500
		c.JSON(http.StatusOK, gin.H{
			"services": []systemctlUnit{},
			"total":    0,
			"warning":  err.Error(),
		})
		return
	}

	// Add a synthesized `actions` array per unit for the frontend
	type svcWithActions struct {
		systemctlUnit
		Actions []string `json:"actions"`
	}
	out := make([]svcWithActions, 0, len(units))
	for _, u := range units {
		out = append(out, svcWithActions{
			systemctlUnit: u,
			Actions:       []string{"start", "stop", "restart", "enable", "disable", "reload"},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"services": out,
		"total":    len(out),
	})
}

// ServiceAction handles POST /admin/services/:connection_id/:action.
//
// Action must be in allowedActions (start/stop/restart/enable/disable/reload).
// Unit is taken from query param `unit`.
func (h *AdminHandler) ServiceAction(c *gin.Context) {
	cid := c.Param("connection_id")
	action := c.Param("action")
	if !allowedActions[action] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "action must be one of: start, stop, restart, enable, disable, reload",
		})
		return
	}
	unit := c.Query("unit")
	if unit == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing query param: unit",
		})
		return
	}

	// Build the systemctl command. Quote the unit name.
	cmd := "systemctl " + action + " " + shellQuote(unit) + " 2>&1"
	resp, err := h.SSHExec(c, cid, cmd, 30000)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":  "ssh: " + err.Error(),
			"action": action,
			"unit":   unit,
		})
		return
	}

	// success = exit code 0
	out := gin.H{
		"action":      action,
		"unit":        unit,
		"exit_code":   resp.ExitCode,
		"stdout":      strings.TrimSpace(resp.Stdout),
		"stderr":      strings.TrimSpace(resp.Stderr),
		"duration_ms": resp.DurationMs,
	}
	if resp.ExitCode != 0 {
		out["error"] = "systemctl " + action + " " + unit + " exited " + strconv.Itoa(resp.ExitCode)
		c.JSON(http.StatusInternalServerError, out)
		return
	}
	c.JSON(http.StatusOK, out)
}

// shellQuote quotes a string for safe inclusion in a shell command.
// Single-quote with proper escaping (close, escape-single, reopen).
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " '\t\n\"\\$;&|<>(){}[]*?!") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
