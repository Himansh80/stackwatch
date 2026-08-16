# StackWatch — Journal

> Append-only log of every session. Newest at bottom.

---

## Session 1 — 2026-08-16 (planning)

### Goals
- Build full proof plan for replacing Datadog + 12 other tools
- Establish strict modularity rules
- Lock verification contract
- Set up project structure

### Done
- ✅ Researched all 12 tools via web search
- ✅ Read existing plans (MASTER-BUILD-PLAN, ENTERPRISE-ROADMAP, IOS-PLATFORM-MISSING-ENDPOINTS, FEATURE-PLAN, IOS-PLATFORM-vs-DATADOG-COMPARISON, IOS-PLATFORM-ROADMAP-2026-2027)
- ✅ Created project folder `C:\Users\himan\HermesProjects\stackwatch\`
- ✅ Initialized git repo (local)
- ✅ Wrote PROJECT.md
- ✅ Wrote MASTER_BUILD_PLAN.md (13 tiers, 26-35 sessions)
- ✅ Wrote MODULARITY_RULES.md
- ✅ Wrote VERIFICATION_CONTRACT.md
- ✅ Wrote RESEARCH.md
- ✅ Wrote README.md
- ✅ Wrote .gitignore

### Decisions
- Project name: stackwatch
- Module size rule: No file > 500 lines (Datadog/Stripe style)
- Verify before claim: Every feature must have live test output in same turn
- GitHub: deferred (user said "later")
- First build: Tier 0 — Foundation

---

## Session 2 — 2026-08-16 (Tier 0 implementation)

### Goals
- Build Tier 0: auth, DB schema, CI, frontend, docs
- Verify 4 times consecutively
- Commit everything

### Done
- ✅ `go.mod` with gin, pgx, jwt, bcrypt, uuid
- ✅ `internal/kernel/` — Tenant, User, Page, Error, Responder (5 files, max 45 lines each)
- ✅ `internal/db/db.go` — pgx pool wrapper
- ✅ `internal/middleware/` — Recover (panic), RequestID, CORS, SecurityHeaders, Logging (5 files)
- ✅ `internal/auth/` — HashPassword (bcrypt cost 12), JWT issuer/verifier (HS256, 24h TTL)
- ✅ `internal/handler/` — AuthHandler split across 8 files (login, signup, me, change-password, logout, forgot, reset, context middleware)
- ✅ `cmd/api-gateway/` — main.go, config.go, db.go, routes.go (4 files)
- ✅ `migrations/000_init_schema.sql` — tenants, users, api_keys, audit_log, triggers
- ✅ Applied migration to .116 (4 tables created)
- ✅ `web/` — React + TypeScript + Vite, Login + Dashboard pages, 164KB bundle
- ✅ `.golangci.yml` — lint config
- ✅ `scripts/file-size-check.sh` — blocks files > 500 lines
- ✅ `.github/workflows/backend-ci.yml` + `frontend-ci.yml`
- ✅ Docs: INSTALL.md, USER-GUIDE.md, ARCHITECTURE.md, TESTING.md, COMPARISON.md
- ✅ Built `api-gateway-linux` (17MB, md5: 8ff63d53)
- ✅ Deployed to .115, running on :8080
- ✅ Committed (11da342 + fec6f22)

### Bugs caught + fixed during verification
1. **JWT TTL bug** — `24*60*60` was being interpreted as **seconds**, but the lib expected `time.Duration`. Token expired in 11 seconds. Fixed to `24*time.Hour`.
2. **Bearer prefix mismatch** — middleware used `h[:7]` which broke with edge cases. Switched to `strings.HasPrefix` + `strings.TrimPrefix`.
3. **ChangePassword used empty PasswordHash** — JWT claims don't include `password_hash`, so `userFromContext` returned a user with empty hash. `checkHash("", old)` always returns false. Fixed: fetch full user from DB by email.
4. **Duplicate signup returned 500** — unique constraint violation wasn't classified. Added `isUniqueViolation()` helper using `pgconn.PgError.Code == "23505"`.
5. **Security header typo** — `DENI` instead of `DENY`. Fixed.

### Verification (4 consecutive runs, all PASS)
```
Run 1: 18/18 PASS
Run 2: 18/18 PASS
Run 3: 18/18 PASS
Run 4: 18/18 PASS
```

Static checks:
- `go build ./...` ✅
- `go vet ./...` ✅
- `gofmt -l .` ✅
- `go test ./internal/...` ✅
- `scripts/file-size-check.sh` ✅
- `npm run type-check` ✅
- `npm run build` ✅ (164KB / 53KB gzipped)

### Decisions added
- JWT TTL: 24 hours
- bcrypt cost: 12 (NIST recommended)
- File size cap: 500 lines per source file (Stripe-style)
- Frontend: Vite + React 18 + TypeScript (vs Create React App / Next.js)
- DB driver: pgx (vs database/sql + lib/pq)

### Next (Tier 1 — Proxmox full replacement)
- 1.1 Proxmox connector service (cmd/proxmox-connector)
- 1.2 VM lifecycle
- 1.3 LXC lifecycle
- 1.4 Storage
- 1.5 Networking
- 1.6 Backup
- 1.7 Templates
- 1.8 HA + cluster
- 1.9 Monitoring

Estimated 3-4 sessions.

### User actions
- Test Tier 0 by following `docs/TESTING.md`
- Report errors or say "next" to start Tier 1
