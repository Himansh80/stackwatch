// Tier 4 S5: Updates (apt or dnf list-upgradable).
//
//	GET /api/v1/admin/updates?connection_id=X
//
// Detects package manager by checking /etc/os-release.
//   - Debian/Ubuntu: `apt list --upgradable 2>/dev/null`
//   - RHEL/Fedora:   `dnf check-update 2>/dev/null` (exit 100 = updates available)
//
// Both formats are parsed into a unified Package struct.
package handler

import (
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// Package is one row in the available-updates list.
type Package struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	NewVersion     string `json:"new_version"`
	Repository     string `json:"repository"`
	Security       bool   `json:"security"`
}

// ListUpdates returns the list of available package updates on the target.
func (h *AdminHandler) ListUpdates(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	// 1. Detect package manager via /etc/os-release ID
	osInfo, _ := h.SSHExec(c, cid, "cat /etc/os-release 2>/dev/null", 5000)
	if osInfo == nil {
		osInfo = &sshExecResponse{}
	}
	osID := parseOSID(osInfo.Stdout)

	var (
		packages []Package
		cmd      string
		mgr      string
	)
	switch osID {
	case "ubuntu", "debian", "pop", "linuxmint", "elementary":
		mgr = "apt"
		cmd = "apt list --upgradable 2>/dev/null"
	case "rhel", "centos", "fedora", "rocky", "almalinux", "ol":
		mgr = "dnf"
		cmd = "dnf -q check-update 2>/dev/null || true"
	default:
		// Try apt first, then dnf
		mgr = "unknown"
		cmd = "(apt list --upgradable 2>/dev/null || dnf -q check-update 2>/dev/null || true)"
	}

	resp, runErr := h.SSHExec(c, cid, cmd, 30000)
	if runErr != nil {
		c.JSON(502, gin.H{
			"error":   "ssh: " + runErr.Error(),
			"manager": mgr,
			"updates": []Package{},
		})
		return
	}

	switch mgr {
	case "apt":
		packages = parseAptUpgradable(resp.Stdout)
	case "dnf":
		packages = parseDnfCheckUpdate(resp.Stdout)
	default:
		// heuristic: look for "Listing..." or "Last metadata"
		if strings.Contains(resp.Stdout, "Listing") || strings.Contains(resp.Stdout, "upgradable") {
			packages = parseAptUpgradable(resp.Stdout)
		} else {
			packages = parseDnfCheckUpdate(resp.Stdout)
		}
	}

	// Mark security updates if apt output contains "-security" or "/security"
	securityMarker := regexp.MustCompile(`(?i)(security|/security|-security)`)

	// For dnf, mark security using CVE heuristics
	cvePattern := regexp.MustCompile(`(?i)CVE-`)

	count := 0
	for i, p := range packages {
		// For apt, the repo field is in the same line; for dnf we need extra parsing
		// (handled in the per-format parsers)
		if securityMarker.MatchString(p.Repository) || cvePattern.MatchString(p.NewVersion) {
			packages[i].Security = true
		}
		if p.Security {
			count++
		}
	}

	c.JSON(200, gin.H{
		"manager":        mgr,
		"updates":        packages,
		"total":          len(packages),
		"security_count": count,
	})
}

// parseOSID extracts ID=ubuntu / ID=debian / ID=rhel / etc from os-release.
func parseOSID(s string) string {
	for _, line := range splitLines(s) {
		if strings.HasPrefix(line, "ID=") {
			v := strings.TrimPrefix(line, "ID=")
			v = strings.Trim(v, `"`)
			return strings.ToLower(v)
		}
	}
	return ""
}

// parseAptUpgradable parses `apt list --upgradable`.
//
// Example line:
//
//	libssl3/jammy-updates 3.0.2-0ubuntu1.10 amd64 [upgradable from: 3.0.2-0ubuntu1.9]
func parseAptUpgradable(s string) []Package {
	out := []Package{}
	re := regexp.MustCompile(`^(\S+?)/(\S+)\s+(\S+)\s+(\S+)(?:\s+\[upgradable from: ([^\]]+)\])?`)
	for _, line := range splitLines(s) {
		if line == "" || strings.HasPrefix(line, "Listing") || strings.HasPrefix(line, "WARNING") {
			continue
		}
		m := re.FindStringSubmatch(line)
		if len(m) < 5 {
			continue
		}
		p := Package{
			Name:           m[1],
			Repository:     m[2],
			NewVersion:     m[3],
			CurrentVersion: m[5], // may be empty
		}
		if p.CurrentVersion == "" {
			p.CurrentVersion = "(unknown)"
		}
		out = append(out, p)
	}
	return out
}

// parseDnfCheckUpdate parses `dnf -q check-update`.
//
// Example lines:
//
//	libxml2.x86_64    2.10.4-1.fc38    updates
//	kernel.x86_64     6.4.7-200.fc38   updates
//
// Lines starting with whitespace are advisories (security). We skip them
// since we don't have full advisory parsing here.
func parseDnfCheckUpdate(s string) []Package {
	out := []Package{}
	for _, line := range splitLines(s) {
		if line == "" || strings.HasPrefix(line, "Last metadata") ||
			strings.HasPrefix(line, "Obsoleting") || strings.HasPrefix(line, " ") ||
			strings.HasPrefix(line, "Loaded") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Format: name.arch  new_version  repository [other...]
		nameArch := fields[0]
		// split on dot to get name
		dotIdx := strings.LastIndex(nameArch, ".")
		name := nameArch
		if dotIdx > 0 {
			name = nameArch[:dotIdx]
		}
		out = append(out, Package{
			Name:           name,
			NewVersion:     fields[1],
			CurrentVersion: "(installed)",
			Repository:     fields[2],
		})
	}
	return out
}
