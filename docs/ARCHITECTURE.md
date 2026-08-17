# StackWatch — Architecture

**Last updated:** 2026-08-17

This document describes the **shipped** architecture: Tier 0, 1, 2, 3.

---

## Design principles

1. **One concern per binary.** Three services today (`api-gateway`,
   `web-terminal`, `truenas-connector`) — each independently deployable,
   each with its own port.

2. **Multi-tenant from day one.** Every business table has
   `tenant_id`; every API handler checks the JWT claim's tenant against
   the row's tenant. Super-admin is the only bypass.

3. **Stateless services.** No file-backed state. All state in Postgres
   or in upstream systems (Proxmox, TrueNAS). Horizontal scaling is
   "add another binary".

4. **Modular-by-line-count.** No `.go` file > 500 lines (CI-enforced).
   Readability wins over reuse.

5. **SQL is also a contract.** All SQL lives in `internal/service/*`
   or `internal/handler/*` clearly labeled; raw SQL in `routes.go` is
   forbidden.

6. **Honest errors.** Error responses include request ID; error
   envelope: `{ "ok": false, "error": "...", "code": "...", "request_id": "..." }`.

---

## Components (Tier 0-3)

```
┌─────────────────────────────────────────────────────────────────────┐
│                              Browser                                 │
│              Static dashboard (web/public/, future React)           │
└──────┬────────────────────┬──────────────────────┬───────────────────┘
       │ HTTPS :443         │ WS :8085             │ HTTPS :443
       ▼                    ▼                      ▼
┌──────────────────┐ ┌──────────────────┐ ┌────────────────────────────────┐
│  api-gateway     │ │ web-terminal    │ │  truenas-connector (sidecar)   │
│  :8080           │ │ :8085            │ │  :8088                         │
│                  │ │                  │ │                                 │
│ REST API         │ │ WebSocket ↔      │ │ JSON-RPC over WebSocket ↔     │
│ JWT auth         │ │ SSH PTY bridge   │ │ TrueNAS middleware             │
│ Tenant scoping   │ │ (real creack/    │ │                                 │
│ Host registry    │ │  pty)            │ │ Wraps wss://<host>/api/current │
│                  │ │                  │ │                                 │
└──────┬───────────┘ └──────┬───────────┘ └──────────┬─────────────────────┘
       │ pgx                  │ pgx                   │ wss
       ▼                      ▼                       ▼
┌────────────────────────┐              ┌──────────────────────────┐
│      Postgres          │              │     TrueNAS SCALE       │
│  Users / Tenants /     │              │      (any host)         │
│  API Keys / Audit /    │              └──────────────────────────┘
│  Proxmox Hosts /       │
│  TrueNAS Hosts /       │ ◄─────── HTTPS :8006 ──── Proxmox host
│  Terminal Connections  │
│  + Logs / Metrics      │
└────────────────────────┘
```

---

## Layers (within api-gateway)

```
HTTP request
   ↓
[gin middleware]
   ├── request ID      — middleware/requestid.go
   ├── panic recovery  — middleware/recover.go
   ├── CORS allowlist  — middleware/cors.go
   ├── structured log  — middleware/logging.go
   └── security headers — middleware/security.go (HSTS, CSP)
   ↓
[protected group /api/v1/*]
   └── JWT verify (auth.ClaimsFromContext) + tenant inject
   ↓
[handler/proxmox.go etc.]
   → thin: parse + delegate
   ↓
[service/*]
   → thick: business logic, SQL, transactions
   ↓
[repository.DB interface]
   → pgxpool for prod, mock for tests
   ↓
Postgres
```

---

## Tier 0 — auth + tenants + users + API keys

| File | Lines | Purpose |
|------|-------|---------|
| `internal/auth/jwt.go` | ~80 | JWT issue + verify (HS256) |
| `internal/auth/hash.go` | ~30 | bcrypt cost-12 |
| `internal/auth/context.go` | ~30 | `ClaimsFromContext`, `ClaimsCtxKey` |
| `internal/handler/auth_*.go` | ~400 | login / signup / me / logout / change-password / forgot / reset / refresh |
| `internal/handler/handlers_tenant.go` | ~150 | tenant CRUD |
| `internal/handler/handlers_users.go` | ~200 | user CRUD |
| `internal/handler/handlers_api_keys.go` | ~190 | API key CRUD + sha256 storage |

All sizes intentionally < 500 lines. CI enforces this.

---

## Tier 1 — Proxmox client (78 endpoints)

```
Browser/dashboard
   ↓ HTTPS
api-gateway (proxmox handler)
   ↓ internal/client/proxmox/proxmox.NewClient(baseURL, "PVEAPIToken=<token>")
   ↓ auto-prefixes PVEAPIToken= if missing
   ↓ HTTP basic-over-custom-header
Proxmox VE REST API at :8006
```

Key files:
- `internal/client/proxmox/proxmox.go` — HTTP transport, auto-prefix
- `internal/client/proxmox/{vm,lxc,storage,...}.go` — typed APIs
- `internal/handler/proxmox*.go` — 16 handler files mapping URL → client call

Lessons learned from the field (in journal.md):
- `PVEAPIToken=` prefix: client auto-injects (fixed in commit `4bfb97c`)
- Task log returns flat array, not wrapped (struct field shape)
- ISCSI returns 501 on older Proxmox (handled as no-op)
- `/cluster/acme/info` doesn't exist on older Proxmox (warn, don't 500)

---

## Tier 2 — TrueNAS sidecar

TrueNAS SCALE 25.10 **removed** REST API v2.0. We use JSON-RPC over
WebSocket at `wss://<host>/api/current` instead.

```
Browser/handlers (truenas_pools.go, etc.)
   ↓ POST /pools/list with {host_id, filters}
api-gateway handler
   ↓ service/truenas_service.go: resolveCachedClient(tenantID, hostID)
truenas-connector (separate binary, port 8088)
   ↓ JSON-RPC {"method": "pool.query", "params": []}
   ↓ WS upgrade → cookie session (login)
TrueNAS SCALE middleware
   ↓ response
（cached to avoid the EBUSY rate-limit on re-login）
```

Key files:
- `internal/client/truenas/truenas.go` — WS transport + login + per-call invoke
- `internal/service/truenas_service.go` — caches `*truenas.Client` per host_id
- `cmd/truenas-connector/main.go` — separate binary (so the gateway isn't
  itself a privileged executor of arbitrary JSON-RPC)

Why a sidecar?
- Isolate the WS + rate-limit dance from the gateway
- If the TrueNAS middleware rate-limits, the gateway tier stays responsive
- Easy to deploy extra sidecar replicas behind a small LB

---

## Tier 3 — web-terminal (PTY bridge)

```
Browser (xterm.js, future frontend)
   ↓ ws://host:8085/api/v1/ws?connection=<uuid>
web-terminal (binary, port 8085)
   ├── validate connection exists + JWT
   ├── load credentials from internal store
   ├── SSH dial via golang.org/x/crypto/ssh
   ├── allocate PTY via github.com/creack/pty
   ├── stream input from WS ↔ stdin of PTY (gorilla/websocket)
   └── stream stdout of PTY ↔ WS (binary frames)
```

Key files:
- `cmd/web-terminal/main.go` — entry point, WS upgrade
- `internal/handler/terminal.go` — connections / keys / credentials / known_hosts CRUD
- The actual PTY is allocated inside `cmd/web-terminal/main.go`'s
  `handleWS` function

---

## File layout (canonical)

```
stackwatch/
├── cmd/
│   ├── api-gateway/                  # main HTTP API (port 8080)
│   │   ├── main.go                   # ~80 lines — wires everything
│   │   ├── db.go                     # pgxpool setup
│   │   ├── config.go                 # env var loading
│   │   └── routes.go                 # ALL HTTP routes in ONE place
│   ├── truenas-connector/            # JSON-RPC sidecar (port 8088)
│   │   └── main.go                   # ~200 lines
│   └── web-terminal/                 # PTY bridge (port 8085)
│       └── main.go                   # ~250 lines
├── internal/
│   ├── auth/
│   │   ├── jwt.go                    # JWT issue + verify
│   │   ├── hash.go                   # bcrypt
│   │   └── context.go                # ClaimsCtxKey, ClaimsFromContext
│   ├── client/
│   │   ├── proxmox/                  # Proxmox client
│   │   │   └── proxmox.go            # auto-prefixes PVEAPIToken=
│   │   └── truenas/                  # TrueNAS JSON-RPC client
│   │       └── truenas.go            # login + per-call invoke
│   ├── handler/                       # 45 files, <500 lines each
│   │   ├── auth*.go
│   │   ├── handlers_tenant.go
│   │   ├── handlers_users.go
│   │   ├── handlers_api_keys.go
│   │   ├── proxmox*.go               # 16 files
│   │   ├── truenas*.go               # 10 files
│   │   ├── terminal*.go              # Tier 3
│   │   └── setup.go
│   ├── service/                       # thick business logic
│   │   └── truenas_service.go        # caches client per host_id
│   ├── db/db.go                       # pgxpool + helpers
│   ├── kernel/kernel.go              # Page, Pagination, Error types
│   └── middleware/                    # request id, recover, CORS, log, security
├── migrations/                        # 6 schema migrations (000-006)
├── .github/workflows/
│   ├── backend-ci.yml                # go build + vet + lint + test + file-size check
│   └── frontend-ci.yml               # npm install + build + file-size check
├── .golangci.yml
├── scripts/
│   └── file-size-check.sh            # CI gate: no file > 500 lines
├── docs/                              # 16 documentation files (this is one of them)
├── go.mod                              # Go 1.23 module: github.com/stackwatch/platform
├── go.sum
├── LICENSE                            # AGPL-3.0
└── README.md
```

---

## Modular rules (CI-enforced)

1. **No file > 500 lines** (`scripts/file-size-check.sh`, also
   `golangci-lint`'s `funlen` config)
2. **No file > 50 exported symbols** (`golangci-lint`'s `gomnd`)
3. **No raw SQL in handlers** — all SQL lives in `internal/service/*`
   or in labeled functions in handler
4. **No package-level mutable state** — pass the `*pgxpool.Pool` via
   function arguments

These rules keep files at < 500 lines. As new tiers ship (Tier 4-13),
new `internal/handler/*` files are added; if any would exceed 500, it
is split into `proxmox_<subfeature>.go` etc.

---

## Why this matters

Engineering principle: **change one thing = only that thing breaks**.

If you accidentally break a line in `proxmox_storage.go`, the **storage
endpoints** break. They don't take down auth, they don't take down the
terminal, they don't take down TrueNAS. Compare to a god-file where
one syntax error kills the entire api-gateway process (this happened in
prior sessions; the cost was ~30 minutes of recovery).

---

## Tech stack

- **Language:** Go 1.23 (api-gateway, sidecars, agent, all binaries)
- **Database:** PostgreSQL 14+
- **Web framework:** `gin-gonic/gin`
- **JWT:** `golang-jwt/jwt/v5`
- **DB driver:** `jackc/pgx/v5` (pgxpool)
- **Proxmox client:** custom (thin wrapper over `net/http`)
- **TrueNAS:** custom (JSON-RPC over WS)
- **SSH:** `golang.org/x/crypto/ssh`
- **PTY:** `github.com/creack/pty`
- **WS:** `github.com/gorilla/websocket`

---

## What we own vs embed

| Embed (use as-is) | Own |
|-------------------|-----|
| Postgres | API gateway |
| JWT library | Multi-tenant scoping |
| pgx | Proxmox client (78 endpoints) |
| creack/pty | TrueNAS sidecar + per-host WS cache |
| gorilla/websocket | SSH bridge (WebSocket ↔ PTY) |
| | SFTP file browser (over SSH) |

---

## Performance targets

- GET endpoints: p95 < 50ms (median 5-10ms)
- POST endpoints: p95 < 200ms
- WS frames: streamed at line rate (no buffering)
- DB pool: 50 connections max

---

## Security model

See [SECURITY.md](SECURITY.md).

---

## What changes in Tier 4+

- `cmd/agent/` directory ships, contains the agent that runs on each
  monitored server
- Tier 4 will add the `internal/handler/admin_*.go` files
- Tier 6 introduces a TSDB (likely Prometheus) — Postgres stays for
  metadata only
- Tier 9 introduces Row-Level Security with a per-tenant `ios_rls` role
