# Design: Tier 4 — Server Admin (Cockpit parity)

## Approach (revised — SSH-based)

**Decision:** Use Tier 3's existing SSH infrastructure. NO new binary. NO new agent.

Tier 3 already wires:
- `ssh_keys` table (private keys per tenant)
- `connections` table (host:port + ssh_key_id)
- Web-terminal sidecar (running SSH sessions)

For Tier 4, the same SSH bridge opens a non-PTY session and runs
commands, parses output (JSON, lines, etc.) and returns structured
data. This avoids:

- New binary to deploy to every server
- New auth mechanism (bridge keys, agent ports)
- Two-layer architecture (api-gateway → agent)

**Architecture:**

```
Browser (Tier 4 page)
  ↓ HTTPS
api-gateway (/api/v1/admin/*)
  ↓ HTTP POST
web-terminal sidecar (:8085)
  ↓ SSH session (golang.org/x/crypto/ssh)
  ↓ cmd.Run (no PTY)
Target server (e.g. .107, .116)
  ↓ returns stdout/stderr as JSON
api-gateway parses + returns to browser
```

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Reuse Tier 3 SSH, no new binary | Zero new deploys; consistent auth | Slower (SSH session per call) vs persistent agent |
| SSH sessions opened per request | Simpler code | ~50ms overhead per request; acceptable for admin endpoints (not hot path) |
| Web-terminal sidecar does the SSH work | Already has openSSHPty + key loading | Adds load to :8085; can scale by running multiple instances |
| No persistent state cached | Always fresh data | Slower; can cache later if needed |
| Parse commands return JSON (e.g. `lsblk -J`) | Reliable, standard format | Requires tools installed on target |
| Action endpoints (service start/stop) audit_logged | Trace of every mutation | Extra DB write per action |

## Implementation details

### New endpoints on web-terminal (`cmd/web-terminal/main.go`)
- `POST /api/v1/admin/exec` — body: `{connection_id, command, timeout_ms}`
- Returns `{stdout, stderr, exit_code, duration_ms}`

### New handlers in api-gateway (`internal/handler/admin_*.go`)
Each handler:
1. JWT-auth → tenant_id
2. Look up connection (must belong to tenant)
3. Format command(s) for the data type (e.g. systemctl list-units --output=json)
4. POST to web-terminal `/api/v1/admin/exec`
5. Parse output → return structured JSON

### Routes (`cmd/api-gateway/routes.go`)
- `protected.GET("/admin/services", adminH.ListServices)`
- `protected.POST("/admin/services/:conn_id/:action", adminH.ServiceAction)`
- `protected.GET("/admin/storage", adminH.ListStorage)`
- `protected.GET("/admin/network", adminH.ListNetwork)`
- `protected.GET("/admin/processes", adminH.ListProcesses)`
- `protected.GET("/admin/updates", adminH.ListUpdates)`
- `protected.GET("/admin/logs", adminH.ListLogs)`
- `protected.GET("/admin/users", adminH.ListUsers)`
- `protected.GET("/admin/timers", adminH.ListTimers)`
- `protected.GET("/admin/performance", adminH.ListPerformance)`

### Migration
- None needed (Tier 3 already has the SSH infrastructure)

## Risks
- **R1: Tools missing on minimal containers** — `systemctl`, `lsblk`, `journalctl`
  might not exist. Mitigation: detect + return empty + warning.
- **R2: Parse format varies between distros** — apt vs dnf, /var/log/messages vs journald.
  Mitigation: detect distro on first call, cache result on connection.
- **R3: SSH session per request is slow** (~50-200ms overhead).
  Mitigation: batch calls when possible (e.g. one ssh session returns all storage data).
- **R4: Action endpoints can break servers** — services stop.
  Mitigation: require explicit confirmation in frontend; audit every action.