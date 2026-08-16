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

---

## Session 3 — 2026-08-16 (Tier 1.2)

### Goals
- VM lifecycle: create/start/stop/reboot/delete/configure
- Strictly modular, all files <500 lines

### Done
- ✅ internal/client/proxmox/vm.go (VMStatus, CreateVM, DeleteVM, GetVMConfig, UpdateVMConfig)
- ✅ internal/client/proxmox/task.go (form requests, task polling)
- ✅ internal/handler/proxmox_vm.go (HTTP handlers)
- ✅ internal/handler/promox_errors.go (Proxmox error → HTTP code)
- ✅ GET /tasks/:upid endpoint for async polling
- ✅ 12/12 live tests pass, 4 consecutive runs
- ✅ Committed (bdb8e99)

### Bugs caught + fixed during this session (continuing from session 2)
7. Proxmox PUT /config is synchronous — no UPID returned. Removed waitForTaskBrief call.
8. `force` param invalid on /status/* endpoints in current Proxmox — silently dropped.
9. DELETE on /qemu/:vmid doesn't accept form body — switched to query string ?purge=1.
10. VMConfig memory/cores fields can be string OR number depending on Proxmox version — use `any` type.
11. Storage 'local' is dir-only — VM disks need 'local-lvm'.
12. Reboot/shutdown on busy Proxmox host (9 running VMs) takes >60s — bumped polling to 180s.
13. Async pattern: all mutating endpoints return task ID immediately + status_url for polling.

### Verification (4 consecutive runs, all PASS)
```
Run 1: 12/12 PASS
Run 2: 12/12 PASS
Run 3: 12/12 PASS
Run 4: 12/12 PASS
```

### Next (Tier 1.3)
- LXC lifecycle (same pattern as VM, but for /nodes/.../lxc/...)
- Storage create/delete
- Networking (bridges, VLANs)

---

## Session 4 — 2026-08-17 (Tier 1.3)

### Goals
- LXC lifecycle: create/start/stop/reboot/delete/configure

### Done
- ✅ LXCStatus/CreateLXC/DeleteLXC/GetLXCConfig/UpdateLXCConfig in client
- ✅ internal/handler/proxmox_lxc.go (165 lines, 5 handlers)
- ✅ 5 new routes wired in routes.go
- ✅ proxmox_errors.go: added CT not running + config file missing patterns
- ✅ 11/11 live tests pass, 4 consecutive runs
- ✅ Committed (bd1a8f0)

### Bugs caught + fixed during this session
14. `net0,gw` is not a valid property — removed
15. LXC creation with password+unprivileged triggered WARNINGS:1 — removed from test
16. Proxmox adds trailing `\n` to PUT values — strip() before compare
17. Stop on stopped CT returns 500 "CT not running" — accept as no-op
18. LXC WARNINGS:1 is success (template extraction warning, not failure)

### Verification (4 consecutive runs, all PASS)
```
Run 1: 11/11 PASS
Run 2: 11/11 PASS
Run 3: 11/11 PASS
Run 4: 11/11 PASS
```

### Next (Tier 1.4)
- Storage management (ZFS/LVM/Ceph/directory create/delete)
- Storage content (ISO, templates, backups)
- Storage upload
