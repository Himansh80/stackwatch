// Tier 4 S7: Users (system users, sudoers, SSH keys).
//
//	GET /api/v1/admin/users?connection_id=X
//
// Commands:
//   - getent passwd                  → system users
//   - ls /etc/sudoers.d/             → sudoers.d snippets
//   - getent group                   → group lookup for GID → name
//   - ls /home/*/.ssh/authorized_keys 2>/dev/null  → SSH keys per user
package handler

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// SystemUser is one row from `getent passwd`.
type SystemUser struct {
	Username string `json:"username"`
	UID      int    `json:"uid"`
	GID      int    `json:"gid"`
	Home     string `json:"home"`
	Shell    string `json:"shell"`
}

// SudoersEntry is one snippet from /etc/sudoers.d/.
type SudoersEntry struct {
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// SSHKeyEntry is one key from a user's authorized_keys file.
type SSHKeyEntry struct {
	Username string `json:"username"`
	Key      string `json:"key"`
}

// ListUsers returns system users + sudoers + SSH authorized keys.
func (h *AdminHandler) ListUsers(c *gin.Context) {
	cid, ok := requireConnectionID(c)
	if !ok {
		return
	}

	// 1. System users
	usersRaw, _ := h.SSHExec(c, cid, "getent passwd 2>/dev/null", 10000)
	if usersRaw == nil {
		usersRaw = &sshExecResponse{}
	}
	users := parsePasswd(usersRaw.Stdout)

	// 2. Sudoers.d
	sudoersRaw, _ := h.SSHExec(c, cid, "ls /etc/sudoers.d/ 2>/dev/null", 5000)
	if sudoersRaw == nil {
		sudoersRaw = &sshExecResponse{}
	}
	sudoers := []SudoersEntry{}
	for _, fn := range splitLines(sudoersRaw.Stdout) {
		if fn == "" || strings.HasSuffix(fn, "~") || strings.HasSuffix(fn, ".bak") {
			continue
		}
		contentRaw, _ := h.SSHExec(c, cid, "cat /etc/sudoers.d/"+shellQuote(fn)+" 2>/dev/null", 5000)
		if contentRaw == nil {
			contentRaw = &sshExecResponse{}
		}
		sudoers = append(sudoers, SudoersEntry{
			Filename: fn,
			Content:  contentRaw.Stdout,
		})
	}

	// 3. SSH authorized_keys (one file per user, lines: "type key comment")
	keysRaw, _ := h.SSHExec(c, cid,
		"find /home /root -maxdepth 3 -name authorized_keys 2>/dev/null", 10000)
	if keysRaw == nil {
		keysRaw = &sshExecResponse{}
	}
	keys := []SSHKeyEntry{}
	for _, ak := range splitLines(keysRaw.Stdout) {
		if ak == "" {
			continue
		}
		// infer username from path
		uname := inferUsername(ak)
		contents, _ := h.SSHExec(c, cid, "cat "+shellQuote(ak)+" 2>/dev/null", 5000)
		if contents == nil {
			contents = &sshExecResponse{}
		}
		for _, line := range splitLines(contents.Stdout) {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			keys = append(keys, SSHKeyEntry{
				Username: uname,
				Key:      line,
			})
		}
	}

	c.JSON(200, gin.H{
		"users":    users,
		"sudoers":  sudoers,
		"ssh_keys": keys,
	})
}

// parsePasswd parses `getent passwd`.
//
// Example: "root:x:0:0:root:/root:/bin/bash"
//
//	fields: name : passwd : uid : gid : gecos : home : shell
func parsePasswd(s string) []SystemUser {
	out := []SystemUser{}
	for _, line := range splitLines(s) {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}
		uid, _ := strconv.Atoi(fields[2])
		gid, _ := strconv.Atoi(fields[3])
		out = append(out, SystemUser{
			Username: fields[0],
			UID:      uid,
			GID:      gid,
			Home:     fields[5],
			Shell:    fields[6],
		})
	}
	return out
}

// inferUsername pulls a username out of /home/<name>/... or /root/...
// If nothing matches, returns the basename of the parent.
var userRe = regexp.MustCompile(`/(home|root)/([^/]+)/`)

func inferUsername(path string) string {
	m := userRe.FindStringSubmatch(path)
	if len(m) >= 3 {
		return m[2]
	}
	return ""
}
