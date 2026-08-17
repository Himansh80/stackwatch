# StackWatch Product Plan — v1.0 (CLOUD + SELF-HOSTED)
*Last updated: 2026-08-17*

## Product Vision (from user, 2026-08-17)

**"Two install modes, same backend, different onboarding UX."**

### Mode 1 — Cloud (Datadog-like)
- Public website: `https://stackwatch.io` (future — currently just `https://stackwatch.smarthomelab.fun`)
- User signs up: email + password
- Picks plan: **Free forever (3 servers max)** — no card, no trial
- Gets dashboard + one-liner install command
- Runs that command on their servers (sends data to cloud)
- Manages everything from `https://app.stackwatch.io`

### Mode 2 — Self-hosted (Portainer-like)
- User runs `docker compose up` on one machine (the central)
- Central server **also monitors itself** (the agent runs locally)
- Goes to `https://<central-ip>:9443` (configurable)
- **Setup wizard** appears: pick admin password + set domain + set TLS
- Creates first user, can invite more (multi-tenant allowed locally)
- Gets agent install command for OTHER machines they want to monitor
- All agents connect back to the central server
- **No internet required**, no payment, no signup

## Architecture (one binary, two deploy modes)

```
+---------------------------------------------------------------+
|                   SAME GO BINARY (api-gateway)                |
|        - Multi-tenant (tenant_id on every table)              |
|        - Per-agent API key auth (Datadog-style)                |
|        - Same 50+ endpoints work in both modes                |
+-----------------------------+---------------------------------+
                              |
              +---------------+----------------+
              |                                |
+-----------------------------+   +-----------------------------+
|     CLOUD DEPLOYMENT        |   |   SELF-HOSTED DEPLOYMENT   |
| - Postgres (RDS / managed)  |   | - Postgres (Docker)         |
| - Caddy + TLS (Let's Encrypt)|  | - Caddy + TLS (self-signed) |
| - Stripe/Razorpay billing   |   | - No billing                |
| - Public signup page        |   | - Setup wizard (first run)  |
| - Marketing site            |   | - Admin UI (no marketing)   |
| - /api/v1/.../signup        |   | - No /signup (local only)   |
| - /api/v1/.../billing       |   | - No /billing               |
| - /api/v1/.../plan          |   | - Plan = unlimited by deflt |
+-----------------------------+   +-----------------------------+
```

## Frontend strategy (user's call: TWO separate frontends)
- `web-public/` — marketing site + cloud signup/login (separate frontend)
- `web-selfhosted/` — self-hosted admin UI (separate frontend)
- `web-shared/` — dashboard, server list, alerts, etc. (shared code)

Same backend serves all three. Different HTML entry points.

## Key technical decisions (locked from Q&A)

| Decision | Choice |
|----------|--------|
| Free tier for cloud | Free forever (3 servers max), no card, no trial |
| Self-hosted dashboard UX | Separate frontend (admin UI, no marketing) |
| Self-hosted DB | Docker Compose spins up Postgres + api-gateway together |
| Self-hosted agent auth | Per-agent API key (current approach) |
| Cloud agent auth | Per-account API key (one per account, like Datadog) |
| Self-hosted multi-user | YES — multi-tenant, can create multiple local users |
| First-run experience | Setup wizard (password + domain + TLS + first user) |
| Migration strategy | Don't break Tier 0-3. Add new on TOP. Save plan. |

## Phased build plan

### Phase 0 (re-confirm & add the missing pieces) — TIER 0.5
**Goal:** Make the existing Tier 0 work for BOTH modes without breaking anything.

1. Add `INSTALL_MODE` env var: `cloud` | `self-hosted` (default: detect from env)
2. If `self-hosted`:
   - First run: show setup wizard (no login required until setup complete)
   - Generate self-signed cert automatically
   - Create system tenant + admin user
3. If `cloud`:
   - Public `/api/v1/auth/signup` (already exists — verify)
   - Email verification
   - Plan assignment (free = 3 server limit, enforced in `installAgent`)
4. Add `mode` field to `/health` and `/api/v1/auth/me` for transparency

### Phase 1 (Docker + 1-line install) — TIER 4.0 (was Tier 4)
1. Create `Dockerfile` (multi-stage: build Go binary in golang:1.23, copy to debian-slim)
2. Create `docker-compose.yml` (api-gateway + postgres + caddy)
3. Create `install.sh` — one-liner that:
   - Detects OS
   - Pulls Docker image
   - Asks for domain/admin password
   - Generates self-signed cert (or uses Let's Encrypt if domain set)
   - Starts the stack
   - Prints dashboard URL + first admin credentials
4. Create `agent-install.sh` (one-liner for agents — works for both cloud and self-hosted):
   - Reads `BACKEND` + `INGEST_KEY` from env
   - Auto-detects hostname/IP
   - Registers with backend
   - Installs as systemd service
5. Test full flow: install central on VM1, install agent on VM2, see VM2 in VM1's dashboard

### Phase 2 (Plan enforcement) — TIER 4.1
1. Server limit already enforced for super-admin (from earlier work)
2. Make it per-tenant: each tenant has `plan` (`free` | `pro` | `enterprise`)
3. `free` = 3 servers max
4. `pro` = 50 servers max
5. `enterprise` = unlimited
6. Server count check on `POST /servers` and `POST /agents/register`

### Phase 3 (Cloud signup + email) — TIER 4.2
1. `/api/v1/auth/signup` already exists — verify
2. Email verification flow (send link, user clicks, account activated)
3. Welcome email with first-install command
4. Forgot password (already exists)
5. Stripe + Razorpay integration (cloud billing)
6. Plan upgrade/downgrade endpoints

### Phase 4 (Frontend split) — TIER 4.3
1. Extract current SPA into `web-shared/` (dashboard, servers, alerts, terminal, etc.)
2. Create `web-public/` (marketing site + cloud signup/login at `stackwatch.io`)
3. Create `web-selfhosted/` (admin-only UI, served by central server on first install)
4. Each frontend talks to same `/api/v1/`

### Phase 5+ (continue Tier 5+)
Continue building features (storage, container mgmt, networking, etc.) — they work in BOTH modes automatically.

## File layout (target)

```
stackwatch/
├── cmd/
│   ├── api-gateway/         # main backend (Go)
│   ├── web-terminal/        # WebSocket bridge
│   ├── alert-engine/        # (future)
│   └── install-agent/       # agent binary
├── internal/
│   ├── handler/             # 50+ endpoints
│   ├── service/             # business logic
│   ├── repository/          # DB layer
│   ├── kernel/              # shared types
│   ├── auth/                # JWT, password hash
│   └── middleware/          # auth, rate limit, etc.
├── migrations/              # SQL migrations
├── web/
│   ├── public/              # CURRENT (will be deprecated, moved to web-shared)
│   ├── shared/              # NEW: shared dashboard code
│   ├── public-cloud/        # NEW: marketing + cloud signup
│   └── public-selfhosted/   # NEW: setup wizard + admin UI
├── docker/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   ├── Caddyfile
│   └── entrypoint.sh        # setup wizard on first run
├── scripts/
│   ├── install.sh           # one-liner for self-hosted install
│   ├── install-agent.sh     # one-liner for agent install
│   └── file-size-check.sh   # CI gate
├── docs/
│   ├── INSTALL.md           # self-hosted install guide
│   ├── USER-GUIDE.md
│   └── ARCHITECTURE.md
├── MASTER_BUILD_PLAN.md     # existing — keep, add reference to this plan
├── PLAN-CLOUD-SELFHOSTED.md # THIS FILE
└── README.md
```

## What this means for the existing Tier 0-3 work

**Everything stays.** The 50 endpoints, 13 tables, multi-tenant model, per-agent API key auth — all of it is already compatible with both modes. We're only adding the **onboarding layer** on top.

## Open questions (will ask before coding)

1. Domain name for cloud (currently `stackwatch.smarthomelab.fun`)? Or different?
2. Email service for cloud (Resend, SendGrid, SES, Postmark, SMTP)?
3. Payment provider priority: Stripe first, Razorpay second? Or both at once?
4. Should the self-hosted setup wizard support Let's Encrypt (needs public domain) or only self-signed?

## Build order (proposed)

| Order | Tier | What | Why first |
|-------|------|------|-----------|
| 1 | 0.5 | Install mode detection + setup wizard | Foundation for both modes |
| 2 | 4.0 | Docker + docker-compose + install.sh | User can actually install self-hosted |
| 3 | 4.0 | agent-install.sh | User can add agents |
| 4 | 4.1 | Plan enforcement | Gate cloud revenue |
| 5 | 4.2 | Cloud signup + email | Cloud can actually get users |
| 6 | 4.2 | Stripe + Razorpay | Cloud can actually make money |
| 7 | 4.3 | Frontend split (web-public, web-selfhosted, web-shared) | UX polish |
| 8+ | 5+ | New features (storage, networking, etc.) | Both modes benefit |

## Success criteria

When this is "done":
- ✅ Random person can `curl -fsSL https://stackwatch.io/install.sh | bash` and have a working self-hosted instance in 5 minutes
- ✅ Random person can sign up at `https://app.stackwatch.io`, get an install command, run it on their server, see their server in the dashboard in 2 minutes
- ✅ Same backend binary serves both
- ✅ No code path is "cloud only" or "self-hosted only" — same endpoints, gated by mode
- ✅ Modular (every file < 500 lines)
- ✅ 100% verified by tests (each tier: build clean, file-size-check pass, go vet pass, live tests 4× PASS)

---
*This plan is locked. No changes without explicit user approval.*
