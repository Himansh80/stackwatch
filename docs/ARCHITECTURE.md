# StackWatch — Architecture

**One interface. Every feature. Strictly modular.**

## Design principles

1. **Strict modularity** — no file > 500 lines. One file per feature. CI enforces this.
2. **Service per package** — every business domain has its own folder
3. **Hexagonal / ports & adapters** — handlers → services → repositories
4. **Open-source by default** — AGPL-3.0, no proprietary lock-in
5. **Self-hosted first, cloud second** — works on bare metal, scales later
6. **Hybrid architecture** — own core, embed OSS (Prometheus, Loki) where it saves years

## Layers

```
┌─────────────────────────────────────────┐
│ Web (React + TypeScript)                 │  ← Tier 0+: Dashboard
└────────────────┬────────────────────────┘
                 │ HTTPS / WebSocket
┌────────────────▼────────────────────────┐
│ cmd/api-gateway (single binary)          │  ← Tier 0: auth + health
│   gin + slog + govalidator               │
└────────────────�────────────────────────┘
                 │ gRPC / REST
┌────────────────▼────────────────────────�
│ internal/handler/                        │
│   auth.go, proxmox_vm.go, truenas.go,    │  ← One file per feature
│   container.go, alert.go, ...            │
└────────────────┬────────────────────────┘
                 │ business logic
┌────────────────▼────────────────────────┐
│ internal/service/                        │
│   auth/, alert/, server/, billing/       │  ← Domain-driven services
│   ai/, notification/, ...                │
└────────────────┬────────────────────────┘
                 │ data access
┌────────────────▼────────────────────────┐
│ internal/repository/  +  internal/db/    │
│   pgx + transactions                     │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│ PostgreSQL 16                            │
└─────────────────────────────────────────┘

Sidecars (Tier 1+):
  cmd/agent/                — host agent (Linux/Win/Mac)
  cmd/proxmox-connector/    — Proxmox REST client
  cmd/truenas-connector/    — TrueNAS REST client
  cmd/remote-access/        — Guacamole RDP/VNC/SSH
  cmd/container-mgmt/       — Docker socket wrapper
  cmd/analytics/            — Prometheus + Loki
  cmd/synthetics/           — HTTP/TCP/ping checks
  cmd/incident/             — on-call + escalation
  cmd/ai-engine/            — LLM triage (Ollama)
  cmd/web-terminal/         — SSH bridge
```

## Tier 0 file tree (shipped)

```
stackwatch/
├── cmd/api-gateway/
│   ├── main.go           (entry, signal handling, graceful shutdown)
│   ├── config.go         (env var loading)
│   ├── db.go             (db connection + issuer factory)
│   └── routes.go         (gin route registration)
├── internal/
│   ├── kernel/           (shared types: Tenant, User, Page, errors)
│   │   ├── types.go
│   │   ├── tenant.go
│   │   ├── user.go
│   │   ├── responder.go
│   │   └── kernel_test.go
│   ├── db/
│   │   └── db.go         (pgx pool wrapper)
│   ├── middleware/
│   │   ├── recover.go    (panic recovery)
│   │   ├── cors.go       (CORS + security headers)
│   │   ├── logging.go    (structured slog)
│   │   └── middleware_test.go
│   ├── auth/
│   │   ├── hash.go       (bcrypt)
│   │   ├── jwt.go        (JWT issue + verify)
│   │   └── auth_test.go
│   └── handler/
│       ├── auth.go            (AuthHandler struct)
│       ├── auth_login.go
│       ├── auth_signup.go
│       ├── auth_me.go
│       ├── auth_helpers.go
│       ├── auth_context.go    (RequireAuth middleware)
│       └── errors.go          (PG error helpers)
├── web/
│   ├── src/
│   │   ├── main.tsx
│   │   ├── App.tsx
│   │   ├── styles.css
│   │   ├── lib/api.ts
│   │   └── pages/
│   │       ├── Login.tsx
│   │       └── Dashboard.tsx
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig.json
│   └── package.json
├── migrations/
│   └── 000_init_schema.sql
├── scripts/
│   └── file-size-check.sh
├── docs/
│   ├── INSTALL.md
│   ├── ARCHITECTURE.md
│   ├── TESTING.md
│   └── INDEX.md
├── .github/workflows/
│   ├── backend-ci.yml
│   └── frontend-ci.yml
├── .golangci.yml
├── go.mod
├── LICENSE (AGPL-3)
├── README.md
├── PROJECT.md
└── MASTER_BUILD_PLAN.md
```

## Modular rules (enforced by CI)

| Check | Tool | Threshold |
|-------|------|-----------|
| File size | `scripts/file-size-check.sh` | 500 lines max |
| Gofmt | `gofmt -l` | no files listed |
| Lint | `golangci-lint` | gocyclo < 15, funlen < 50 |
| Vet | `go vet` | clean |
| Tests | `go test ./internal/...` | all pass |

## Why this matters

> "if I add a new feature and the code is not modular, and I have done a mistake or syntax error or something else, it would break my entire system and then I have to do all the work from start"

This is what every tier protects against. Adding `internal/handler/proxmox_vm.go` cannot break `internal/handler/auth_login.go`. Each file is independent. CI prevents anyone from merging a file > 500 lines.

## Tech stack

| Layer | Choice | Why |
|-------|--------|-----|
| Backend | Go 1.23 | Single binary, fast, mature |
| HTTP framework | Gin | Fastest Go router |
| DB | PostgreSQL 16 | Mature, RLS support, JSONB |
| DB driver | pgx | Faster than database/sql |
| Logging | log/slog (stdlib) | Structured JSON, no extra deps |
| Auth | golang-jwt/jwt v5 | HS256, standard claims |
| Hash | golang.org/x/crypto/bcrypt | Battle-tested |
| Frontend | React 18 + Vite + TS | Fast dev, type safety |
| Router | react-router-dom v6 | Standard |
| CI | GitHub Actions | Free for public repos |
| Lint | golangci-lint | Industry standard |

## What we own vs embed

**Own:**
- All Go services (cmd/*, internal/*)
- All frontend code
- All SQL migrations
- All documentation

**Embed (later tiers):**
- Prometheus (TSDB) — TSDB would take 1 year to build right
- Loki (logs) — same
- Apache Guacamole (RDP/VNC) — proven, complex
- Ollama (LLM runtime) — local LLM, no API key

**Use APIs:**
- Proxmox REST API (we wrap + cache)
- TrueNAS REST API (we wrap + cache)
- Docker socket (we proxy)
- Stripe + Razorpay (billing)

## Performance targets (Tier 7+)

| Metric | Target |
|--------|--------|
| API latency p50 | < 50 ms |
| API latency p95 | < 200 ms |
| Metric ingestion | 100K points/sec/agent |
| Logs | 10K lines/sec/agent |
| Dashboard initial load | < 1.5 s |
| Agent memory | < 100 MB |
| Agent CPU | < 5% idle, < 30% under load |

## Security model

| Layer | Defense |
|-------|---------|
| Transport | TLS (Caddy/NPM front proxy) |
| API | JWT (HS256, 24h TTL) + bcrypt (cost 12) |
| Auth | Tenant isolation via JWT claims |
| Future tiers | 2FA, SAML, RBAC, RLS, IP allowlist |
| Audit | Every privileged action logged |
