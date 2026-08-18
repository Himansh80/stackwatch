// Tier 4 S6: Logs (journald + /var/log fallback).
//
//   GET /api/v1/admin/logs?connection_id=X[&unit=NAME][&priority=N][&since=1h][&limit=100]
//
// Uses `journalctl --output=json` when available, falls back to
// `tail -n LIMIT /var/log/syslog` or `/var/log/messages`.
package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// JournalEntry is one parsed journald line (JSON from --output=json).
type JournalEntry struct {
	Timestamp       string `json:"timestamp"`        // RFC3339Nano
	Priority        string `json:"priority"`         // emerg/alert/crit/err/warning/notice/info/debug
	Unit            string `json:"unit"`             // systemd unit name
	Message         string `json:"message"`          // raw message
	Hostname        string `json:"hostname,omitempty"`
}

// journalRawEntry is the shape `journalctl --output=json` emits.
type journalRawEntry struct {
	Timestamp    string `json:"__REALTIME_TIMESTAMP"` // microseconds since epoch (string)
	Priority     string `json:"PRIORITY"`             // numeric, e.g. "3"
	Unit         string `json:"_SYSTEMD_UNIT"`        // unit name
	Message      string `json:"MESSAGE"`             // raw
	Hostname     string `json:"_HOSTNAME"`
}

// ListLogs returns the most recent log entries.
//
// Query params:
//   - connection_id (required)
//   - unit: filter by systemd unit (optional)
//   - priority: filter by priority (optional: emerg|alert|crit|err|warning|notice|info|debug)
//   - since: how far back to look (optional, default "1 hour ago")
//   - limit: max entries (default 100, capped at 1000)
func (h *AdminHandler) ListLogs(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	unit := c.Query("unit")
	priority := c.Query("priority")
	since := c.DefaultQuery("since", "1 hour ago")
	limit := c.DefaultQuery("limit", "100")

	// Build journalctl command. Quote args carefully.
	args := []string{"--no-pager", "-n", limit, "--output=json"}
	if unit != "" {
		args = append(args, "-u", shellQuote(unit))
	}
	if priority != "" {
		args = append(args, "-p", priority)
	}
	if since != "" {
		args = append(args, "--since", shellQuote(since))
	}
	cmd := "journalctl " + strings.Join(args, " ")

	resp, runErr := h.SSHExec(c, cid, cmd, 30000)
	if runErr != nil {
		c.JSON(502, gin.H{"error": "ssh: " + runErr.Error(), "entries": []JournalEntry{}})
		return
	}

	entries := parseJournalJSON(resp.Stdout)

	// Fallback if journalctl returns nothing useful and /var/log exists
	if len(entries) == 0 {
		fbCmd := "tail -n " + limit + " /var/log/syslog 2>/dev/null || tail -n " + limit + " /var/log/messages 2>/dev/null || echo ''"
		fb, _ := h.SSHExec(c, cid, fbCmd, 10000)
		entries = parseSyslog(fb.Stdout)
	}

	c.JSON(200, gin.H{
		"entries": entries,
		"total":   len(entries),
		"source":  "journalctl",
	})
}

// parseJournalJSON parses the JSON-lines output of `journalctl --output=json`.
func parseJournalJSON(s string) []JournalEntry {
	out := []JournalEntry{}
	for _, line := range splitLines(s) {
		if line == "" {
			continue
		}
		var raw journalRawEntry
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		out = append(out, JournalEntry{
			Timestamp: microsecToISO(raw.Timestamp),
			Priority:  priorityString(raw.Priority),
			Unit:      raw.Unit,
			Message:   raw.Message,
			Hostname:  raw.Hostname,
		})
	}
	return out
}

// microsecToISO converts journald microseconds-since-epoch (string) to RFC3339Nano.
func microsecToISO(us string) string {
	if us == "" {
		return ""
	}
	var micros int64
	for _, ch := range us {
		if ch < '0' || ch > '9' {
			return us
		}
		micros = micros*10 + int64(ch-'0')
	}
	t := time.Unix(0, micros*1000).UTC()
	return t.Format(time.RFC3339Nano)
}

// priorityString maps journald numeric priority to a string.
func priorityString(p string) string {
	switch p {
	case "0":
		return "emerg"
	case "1":
		return "alert"
	case "2":
		return "crit"
	case "3":
		return "err"
	case "4":
		return "warning"
	case "5":
		return "notice"
	case "6":
		return "info"
	case "7":
		return "debug"
	}
	return p
}

// parseSyslog parses classic syslog format.
//
// Example line: "Aug 17 22:03:01 cloud-app kernel: [1234.5] message"
// We can't recover the year, so the date gets a placeholder.
func parseSyslog(s string) []JournalEntry {
	out := []JournalEntry{}
	for _, line := range splitLines(s) {
		if line == "" {
			continue
		}
		// Best-effort split: ts(15) host(15) proc:
		if len(line) < 16 {
			out = append(out, JournalEntry{
				Timestamp: "",
				Priority:  "info",
				Message:   line,
			})
			continue
		}
		out = append(out, JournalEntry{
			Timestamp: "1970-" + line[:6] + "T" + line[7:15] + ":00Z",
			Priority:  "info",
			Unit:      "syslog",
			Message:   line,
		})
	}
	return out
}
