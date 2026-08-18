# Design: Tier 5 — Container Management (Portainer + Watchtower)

## Approach (SSH-based, like Tier 4)

Per Tier 4 lesson: **reuse existing SSH bridge**. NO new binary. NO new
sidecar. Add 13 endpoint groups to api-gateway that call the existing
Tier 4 web-terminal `/api/v1/admin/exec` endpoint with `docker` CLI
commands.

**Architecture:**
```
browser → api-gateway /api/v1/containers/*
        → POST web-terminal:8085/api/v1/admin/exec
        → SSH session → `docker <subcommand> --format '{{json ...}}'`
        → parse JSON line(s) → return structured response
```

**Why docker CLI not Docker REST API:**
- Reuses Tier 3 SSH bridge (auth, key mgmt, tenant scoping all done)
- No need to expose Docker socket or REST API on the host
- One connection per host — keeps attack surface minimal
- `docker --format '{{json ...}}'` gives parseable JSON for every command
- Action commands (start/stop/restart) work the same way

**Limitations vs Portainer:**
- No interactive container console streaming (would need PTY + WS)
- No live container stats streaming (would need websocat or HTTP stream)
- No docker-compose stack deploy UI (we support stacks as files only)
- No Swarm/Kubernetes cluster mode (out of scope)

For Tier 5 v1 we return REST-shaped responses with arrays. Streaming
console/stats go in Tier 5.5 if needed.

## Architecture decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| Reuse SSH bridge (Tier 4 path) | Zero new infrastructure | Slightly slower than REST socket (one SSH per call, ~30ms) |
| docker --format '{{json .}}' | Native JSON from docker itself | Some versions differ; we detect + fallback |
| `connection_id` query param (not `server_id`) | Consistent with Tier 4 admin endpoints | Different from Proxmox tier (which uses `host_id`) |
| Audit every action in audit_log | Trace of every container start/stop/etc | Extra DB write per action |
| Read-only endpoints return 200 + empty arrays on docker missing | Tools may be missing on minimal hosts | Same warning-as-data pattern as Tier 4 |

## Implementation details

### `cmd/api-gateway/admin_containers.go` (new, ~200 LOC)
Contains the `ContainerHandler` struct with `SSHExec` from Tier 4.
Endpoints: C1-C11 + Watchtower.

### Endpoints (all under `/api/v1/containers` prefix unless noted)

| Code | Method | Path | docker command | Response |
|------|--------|------|----------------|----------|
| C1 | GET | /?connection_id=X | `docker ps -a --format '{{json .}}'` | `{containers: [...]}` |
| C1 | GET | /:id?connection_id=X | `docker inspect <id>` | `{container: {...}}` |
| C2 | POST | /:id/:action?connection_id=X | `docker <action> <id>` | `{ok: true, action, id}` |
| C3 | GET | /:id/logs?connection_id=X&tail=100 | `docker logs --tail=N <id>` | `{logs: [...]}` |
| C5 | GET | /:id/stats?connection_id=X | `docker stats --no-stream --format '{{json .}}' <id>` | `{stats: {...}}` |
| C6 | POST | /?connection_id=X | `docker run ...` (body has image, name, env, ports, etc.) | `{id, status}` |
| C7 | GET | /images?connection_id=X | `docker images --format '{{json .}}'` | `{images: [...]}` |
| C7 | POST | /images/pull?connection_id=X | `docker pull <image>` | `{ok: true}` |
| C7 | DELETE | /images/:id?connection_id=X | `docker rmi <id>` | `{ok: true}` |
| C8 | GET | /volumes?connection_id=X | `docker volume ls --format '{{json .}}'` | `{volumes: [...]}` |
| C8 | POST | /volumes?connection_id=X | `docker volume create <name>` | `{name}` |
| C8 | DELETE | /volumes/:name?connection_id=X | `docker volume rm <name>` | `{ok: true}` |
| C9 | GET | /networks?connection_id=X | `docker network ls --format '{{json .}}'` | `{networks: [...]}` |
| C9 | POST | /networks?connection_id=X | `docker network create <name>` | `{id, name}` |
| C9 | DELETE | /networks/:id?connection_id=X | `docker network rm <id>` | `{ok: true}` |
| C10 | GET | /stacks?connection_id=X | `docker stack ls --format '{{json .}}'` (if Swarm) or `ls /opt/stacks/` | `{stacks: [...]}` |
| C10 | POST | /stacks?connection_id=X | `docker stack deploy -c <compose>` | `{name}` |
| C11 | GET | /templates | hardcoded list of common compose templates | `{templates: [...]}` |
| Watchtower | GET | /watchtower?connection_id=X | `docker ps --filter label=watchtower` | `{watched: [...]}` |
| Watchtower | POST | /watchtower/update?connection_id=X | `docker run --rm watchtower` | `{updated: [...]}` |

### Routes (`cmd/api-gateway/routes.go`)
Add 13+ new routes under `/api/v1/containers/*`.

### Migration
None needed.

## Risks
- **R1: docker CLI version differs** (older: `--format '{{json .}}'` may not exist). Mitigation: detect + use `--format='{{json}}'` or fall back to text parsing.
- **R2: Action endpoints can break running services**. Mitigation: confirm with body param; audit log every action.
- **R3: No streaming console/stats** — v1 returns only REST snapshots. Document as known limitation.
- **R4: `docker run` body parsing is fragile**. Mitigation: use `--env KEY=VAL` quoting helper, similar to Tier 4 `shellQuote`.
