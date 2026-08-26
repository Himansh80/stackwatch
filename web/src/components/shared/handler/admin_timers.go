// Tier 4 S8: Timers (systemd timers + cron jobs).
//
//	GET /api/v1/admin/timers?connection_id=X
//
// Commands:
//   - systemctl list-timers --all --no-pager
//   - ls /etc/cron.d/ 2>/dev/null
//   - for user in /var/spool/cron/crontabs/* ; cat
package handler

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Timer is one row from `systemctl list-timers`.
type Timer struct {
	Unit      string `json:"unit"`
	Next      string `json:"next"`      // human readable, e.g. "Mon 2026-08-18 12:00:00 IST"
	Left      string `json:"left"`      // human readable, e.g. "5h 23min"
	Last      string `json:"last"`      // when last fired
	Activates string `json:"activates"` // unit that fires
	Schedule  string `json:"schedule"`  // cron-style spec, may be empty
}

// CronJob is one entry in /etc/cron.d/ or a per-user crontab.
type CronJob struct {
	Filename string `json:"filename"`
	User     string `json:"user,omitempty"`
	Schedule string `json:"schedule"` // 5-field cron: "min hour dom mon dow"
	Command  string `json:"command"`
}

// ListTimers returns systemd timers + cron jobs.
func (h *AdminHandler) ListTimers(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	// 1. systemd timers — parse output (no --output=json in older systemd)
	timersRaw, _ := h.SSHExec(c, cid,
		"systemctl list-timers --all --no-pager --legend=false 2>/dev/null", 10000)
	if timersRaw == nil {
		timersRaw = &sshExecResponse{}
	}
	timers := parseSystemdTimers(timersRaw.Stdout)

	// 2. /etc/cron.d/ files
	cronRaw, _ := h.SSHExec(c, cid, "ls /etc/cron.d/ 2>/dev/null", 5000)
	if cronRaw == nil {
		cronRaw = &sshExecResponse{}
	}
	cronJobs := []CronJob{}
	for _, fn := range splitLines(cronRaw.Stdout) {
		if fn == "" || strings.HasPrefix(fn, "#") {
			continue
		}
		fileRaw, _ := h.SSHExec(c, cid, "cat /etc/cron.d/"+shellQuote(fn)+" 2>/dev/null", 5000)
		if fileRaw == nil {
			fileRaw = &sshExecResponse{}
		}
		for _, line := range splitLines(fileRaw.Stdout) {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "SHELL") ||
				strings.HasPrefix(line, "PATH") || strings.HasPrefix(line, "MAILTO") {
				continue
			}
			job := parseCronLine(line)
			if job != nil {
				job.Filename = fn
				cronJobs = append(cronJobs, *job)
			}
		}
	}

	// 3. per-user crontabs (root + others)
	userCrons, _ := h.SSHExec(c, cid,
		"ls /var/spool/cron/crontabs/ 2>/dev/null", 5000)
	if userCrons == nil {
		userCrons = &sshExecResponse{}
	}
	for _, uname := range splitLines(userCrons.Stdout) {
		if uname == "" {
			continue
		}
		tab, _ := h.SSHExec(c, cid, "crontab -u "+shellQuote(uname)+" -l 2>/dev/null", 5000)
		if tab == nil {
			tab = &sshExecResponse{}
		}
		for _, line := range splitLines(tab.Stdout) {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			job := parseCronLine(line)
			if job != nil {
				job.User = uname
				cronJobs = append(cronJobs, *job)
			}
		}
	}

	c.JSON(200, gin.H{
		"timers":    timers,
		"cron_jobs": cronJobs,
	})
}

// parseSystemdTimers parses the table output of `systemctl list-timers`.
//
// Columns: NEXT  LEFT  LAST  PASSED  UNIT  ACTIVATES
func parseSystemdTimers(s string) []Timer {
	out := []Timer{}
	// split into rows by newline, then split by 2+ spaces (column separator)
	rowRe := regexp.MustCompile(`\s{2,}`)
	for _, line := range splitLines(s) {
		if line == "" || strings.HasPrefix(line, "NEXT") || strings.HasPrefix(line, "0 timers") {
			continue
		}
		cols := rowRe.Split(line, -1)
		if len(cols) < 5 {
			continue
		}
		out = append(out, Timer{
			Next:      cols[0],
			Left:      cols[1],
			Last:      cols[2],
			Activates: cols[4],
			Unit:      cols[5] + ".timer", // systemd uses short name
		})
	}
	return out
}

// cronFieldRe matches a cron line with 5+ space-separated fields.
var cronFieldRe = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(.+)$`)

// parseCronLine parses one cron line. Returns nil if it doesn't look like a cron entry.
func parseCronLine(line string) *CronJob {
	m := cronFieldRe.FindStringSubmatch(line)
	if len(m) < 7 {
		return nil
	}
	return &CronJob{
		Schedule: m[1] + " " + m[2] + " " + m[3] + " " + m[4] + " " + m[5],
		Command:  m[6],
	}
}
