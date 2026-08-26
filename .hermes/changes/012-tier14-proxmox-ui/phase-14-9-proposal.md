# Tier 14 Phase 14.9 — Users + API Tokens + Permissions + Pools

## Why this phase

Proxmox's access control has 4 layers: users, API tokens, permissions, pools. Phase 14.9 adds UI for all 4 — needed for multi-user production deployments.

## Goals

1. **Users page** at `/proxmox/users` — list + create + edit + delete
2. **API tokens page** at `/proxmox/users/:userid/tokens` — per-user token list + create + revoke
3. **Permissions page** at `/proxmox/permissions` — list + grant + revoke
4. **Pools page** at `/proxmox/pools` — list + create + delete + members

## Backend

ZERO new endpoints — all use Tier 1:
- `GET /proxmox/users` — list users
- `POST /proxmox/users` — create
- `PUT /proxmox/users/:userid` — update
- `DELETE /proxmox/users/:userid` — delete
- `GET /proxmox/users/:userid/token` — list tokens
- `POST /proxmox/users/:userid/token` — create
- `DELETE /proxmox/users/:userid/token/:tokenid` — revoke
- `GET /proxmox/permissions` — list
- `PUT /proxmox/permissions` — grant
- `DELETE /proxmox/permissions` — revoke
- `GET /proxmox/pools` — list
- `POST /proxmox/pools` — create
- `DELETE /proxmox/pools/:poolid` — delete

## UI components (planned, all ≤400 LOC)

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxUsers.tsx` | 280 | Users list + create/edit/delete |
| `ProxmoxUserTokens.tsx` | 200 | Per-user API token list + create/revoke |
| `ProxmoxPermissions.tsx` | 250 | Permissions list + grant/revoke |
| `ProxmoxPools.tsx` | 220 | Pools list + create/delete/members |

## Routes

- `/proxmox/users`
- `/proxmox/users/:userid/tokens`
- `/proxmox/permissions`
- `/proxmox/pools`

## Acceptance criteria

1. Users: list + create (with password) + edit (email, role, enable) + delete
2. API tokens: list per user + create (show token once) + revoke
3. Permissions: list + grant role on path + revoke
4. Pools: list + create + delete + show members
5. Empty states everywhere
6. Mobile-responsive
7. Dark theme + Datadog style
