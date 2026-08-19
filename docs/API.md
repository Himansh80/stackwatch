# StackWatch — REST API Reference

**Last updated:** 2026-08-17

All endpoints live under `/api/v1/`. All protected endpoints require
`Authorization: Bearer <jwt>`.

**Authentication**: login via `POST /auth/login`, get a JWT, pass it
on subsequent calls.

---

## Tier 0 — Auth, Tenants, Users, API Keys

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | Email + password → JWT |
| POST | `/api/v1/auth/signup` | New tenant + admin (gated by env) |
| POST | `/api/v1/auth/forgot` | Issue password-reset token |
| POST | `/api/v1/auth/reset` | Apply new password with token |
| POST | `/api/v1/auth/magic-link` | Issue one-time passwordless login token |
| GET | `/api/v1/auth/magic-link/consume` | Consume magic-link token and return JWT |
| POST | `/api/v1/auth/accept-invite` | Consume invite token, set password, activate user |
| GET | `/api/v1/auth/me` | Current user profile |
| PATCH | `/api/v1/auth/profile` | Update current user's profile |
| POST | `/api/v1/auth/logout` | Client discards stateless JWT |
| POST | `/api/v1/auth/change-password` | Change own password |
| POST | `/api/v1/auth/refresh` | Re-issue JWT for an unexpired one |
| GET | `/api/v1/tenants` | List tenants in scope |
| GET | `/api/v1/tenants/me` | Current tenant |
| GET | `/api/v1/tenants/:id` | Tenant by ID |
| PATCH | `/api/v1/tenants/:id` | Update tenant name/plan |
| GET | `/api/v1/users` | List users in scope |
| GET | `/api/v1/users/me` | Current user |
| GET | `/api/v1/users/:id` | User by ID |
| POST | `/api/v1/users` | Create user (super_admin only) |
| PATCH | `/api/v1/users/:id` | Update user (full_name, role, is_active) |
| DELETE | `/api/v1/users/:id` | Soft-delete user |
| GET | `/api/v1/api-keys` | List API keys for current tenant |
| POST | `/api/v1/api-keys` | Create API key (returns plaintext ONCE) |
| POST | `/api/v1/api-keys/:id/revoke` | Revoke API key |
| DELETE | `/api/v1/api-keys/:id` | Permanently delete API key |

---

## Tier 1 — Proxmox

78 endpoints under `/api/v1/proxmox/hosts/:id/...`

Sub-resources:
- **Hosts**: register, list, update, delete, test
- **Nodes**: list, status
- **VMs** (`:nodes/:node/qemu`): list, create, config, status, start, stop, reboot, suspend, resume, migrate, snapshot, backup, delete
- **LXC** (`:nodes/:node/lxc`): same as VMs
- **Storage** (`:nodes/:node/storage`): list, create, delete, content, upload, download
- **Network** (`:nodes/:node/network`): list, create, update, delete, apply
- **Firewall** (`:nodes/:node/firewall`): rules, groups, IPsets, aliases, options
- **Disks** (`:nodes/:node/disks`): list, init, wipe, ZFS create/destroy
- **Users / Tokens** (`:access/users`): list, create, update, delete; `:token` for API tokens
- **Tasks** (`:cluster/tasks` and `:nodes/:node/tasks`): list, status, log, stop
- **ISCSI** (`:storage/:storage/content`): list, download URLs
- **Pools**: list, create, update, delete
- **Backup** (`:cluster/backup`): list, create, update, delete, run
- **Certificates + ACME** (`:cluster/acme`): list/create/dir/plugins

See code: `internal/handler/proxmox_*.go` for the route table.

---

## Tier 2 — TrueNAS SCALE

The authenticated gateway exposes `/api/v1/truenas/*` and proxies to the local
connector on `127.0.0.1:8088`. The connector authenticates to TrueNAS SCALE
25.10+ through JSON-RPC over WebSocket (`/api/current`). Browser clients must
never call port 8088 directly.

Host management:

```
GET    /api/v1/truenas/hosts
POST   /api/v1/truenas/hosts
POST   /api/v1/truenas/hosts/:id/test
DELETE /api/v1/truenas/hosts/:id
```

Read and mutation routes are forwarded under the same prefix:

```
POST /api/v1/truenas/pools/{list,get,create,extend,import,export,destroy,scrub,scrub_state}
POST /api/v1/truenas/datasets/{list,get,create,update,delete}
POST /api/v1/truenas/nfs/{list,create,update,delete}
POST /api/v1/truenas/smb/{list,create,update,delete}
POST /api/v1/truenas/iscsi/{extents,targets,associated}/{list,create,delete}
POST /api/v1/truenas/snapshots/{list,create,delete,rollback}
POST /api/v1/truenas/replications/{list,create,delete,run}
POST /api/v1/truenas/disks/{list,smart_run,smart_history,smart_results,replace,wipe}
POST /api/v1/truenas/users/{list,get,create,update,delete}
POST /api/v1/truenas/groups/{list,create,delete}
POST /api/v1/truenas/acl/{get,set}
POST /api/v1/truenas/system/info
POST /api/v1/truenas/system/{bootenv,services}/{list,activate,delete,action,autostart}
POST /api/v1/truenas/system/update/{check,apply}
POST /api/v1/truenas/cloud/{credentials,sync}/{list,create,update,delete,run,dry_run}
```

All resource requests use JSON body `{"host_id":"<uuid>", ...}`. Stored API
keys and passwords are never returned by host-list or host-get responses.

---

## Tier 3 — Remote Access / Terminal

33 endpoints under `/api/v1/terminal/...`

- **connections** (`/terminal/connections`) — list / get / create / update / delete / test / groups / history
- **verification** (`/terminal/connections/:id/verification`) — get / update per-connection verification policy
- **auth** (`/terminal/connections/:id/auth`) — connection-level credentials config
- **SFTP fs** (`/terminal/connections/:id/fs`) — list, read, write, mkdir, delete, rename, stat
- **credentials** (`/terminal/credentials`) — encrypted vault for SSH passwords / keys
- **known-hosts** (`/terminal/known-hosts`) — trust list (MITM protection)
- **keys** (`/terminal/keys`) — named user SSH keys
- **WebSocket** (`GET /api/v1/ws?connection=<uuid>`) — PTY streaming

---

## Tier 4-13 — Coming

See `MASTER_BUILD_PLAN.md` for the complete roadmap.

---

## Status codes

| Code | Meaning |
|------|---------|
| 200 | OK |
| 201 | Created |
| 400 | Bad Request (malformed JSON, missing field) |
| 401 | Unauthorized (missing or invalid JWT) |
| 403 | Forbidden (role / scope check) |
| 404 | Not Found (also: 401 ambiguously — body has `error: "not_found"`) |
| 409 | Conflict (duplicate unique field) |
| 429 | Too Many Requests (5 logins / 5min) |
| 500 | Internal Error |
| 503 | Service Unavailable (DB unreachable, sidecar down) |

---

## Error envelope

```json
{
  "ok": false,
  "error": "human-readable message",
  "code": "machine_readable",
  "request_id": "uuid"
}
```

Successful responses follow the same shape:
```json
{ "ok": true, "data": {...} }
```
