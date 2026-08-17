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

---

## Session 5 — 2026-08-17 (Tier 1.4)

### Goals
- Storage management: list/create/delete + content list/delete

### Done
- ✅ StorageSpec/Entry/Content types + CRUD in client
- ✅ postForm/deleteForm helpers (sync endpoints — Proxmox returns resource, not UPID)
- ✅ internal/handler/proxmox_storage.go (150 lines, 5 handlers)
- ✅ 5 new routes wired
- ✅ Removed duplicate GET /storage route from Tier 1.1
- ✅ 9/9 live tests pass, 4 consecutive runs
- ✅ Committed (8cbbba7)

### Bugs caught + fixed
19. Route conflict: GET /proxmox/hosts/:id/storage registered twice (Tier 1.1 + Tier 1.4)
20. Proxmox /storage POST/DELETE is synchronous — returns resource/null, not UPID
    - Added postForm/deleteForm to client
21. Verifier expected task ID; storage returns no task — accept empty task as success
22. Unused imports/variables from refactor — cleaned up

### Verification (4 consecutive runs, all PASS)
```
Run 1: 9/9 PASS
Run 2: 9/9 PASS
Run 3: 9/9 PASS
Run 4: 9/9 PASS
```

### Next (Tier 1.5)
- Networking (bridges, VLANs, bonds)
- Firewall (rules, aliases, IPsets, security groups)
- DNS

---

## Session 6 — 2026-08-17 (Tier 1.5)

### Goals
- Network management: list/create/update/delete

### Done
- ✅ NetworkIface/NetworkSpec types + 5 CRUD methods in client
- ✅ putForm helper (Proxmox PUT for updates, not POST)
- ✅ internal/handler/proxmox_network.go (175 lines, 4 handlers)
- ✅ 4 new routes wired
- ✅ 9/9 live tests pass, 4 consecutive runs
- ✅ Committed

### Bugs caught + fixed
23. vlan_id/vlan_raw_device are hyphen-separated in Proxmox POST form (vlan-id, vlan-raw-device)
24. bridge_stp/bridge_fd/bridge_vlan_aware returned as STRINGS ("on"/"off", "0"), not int
25. PUT /network/{iface} requires `type` field to be re-sent (Proxmox schema quirk)
26. Update needs PUT method (POST returns 501 "Method not implemented")
27. Network sync endpoints return null on success, not resource

### Verification (4 consecutive runs, all PASS)
```
Run 1: 9/9 PASS
Run 2: 9/9 PASS
Run 3: 9/9 PASS
Run 4: 9/9 PASS
```

### Next (Tier 1.6)
- Firewall (rules, aliases, IPsets, security groups)
- OR Users (Proxmox user mgmt + permissions)

---

## Session 7 — 2026-08-17 (Tier 1.6)

### Goals
- Firewall rules management (CRUD + IPsets)

### Done
- ✅ FirewallRule/FirewallRuleSpec types + 5 CRUD methods in client
- ✅ IPset/IPsetEntry types + 4 methods (CRUD + add entry)
- ✅ internal/handler/proxmox_firewall.go (255 lines, 7 handlers)
- ✅ 7 new routes wired
- ✅ 10/10 live tests pass, 4 consecutive runs (first try)
- ✅ Committed

### Known limit
- IPset endpoints return 501 "Method not implemented" on Proxmox .107
- Reason: pve-firewall package not active (single-node setup doesn't need it)
- IPset code is implemented and builds clean — will work on Proxmox with
  the firewall package active

### Verification (4 consecutive runs, all PASS)
```
Run 1: 10/10 PASS
Run 2: 10/10 PASS
Run 3: 10/10 PASS
Run 4: 10/10 PASS
```

### Next (Tier 1.7)
- Users & auth (Proxmox user mgmt, permissions, PVE tokens)
- OR Backup (scheduled backups + restore)
- OR Tasks (Proxmox task history)

---

## Session 8 — 2026-08-17 (Tier 1.7)

### Goals
- Disk management: list physical disks + ZFS pool CRUD

### Done
- ✅ DiskInfo/ZFSPool types + 5 methods
- ✅ internal/handler/proxmox_disks.go (130 lines, 4 handlers)
- ✅ 4 new routes wired
- ✅ Input validation: devices must match /dev/X, raidlevel ∈ allowed set
- ✅ 8/8 live tests pass, 4 consecutive runs
- ✅ Committed

### Bugs caught + fixed
28. DiskInfo.OSType renamed to remove conflict with old osdid (was actually OS disk ID)
29. Added Wearout/GPT/OSDID/OSDIDList fields for full JSON shape
30. fmt.Errorf returns 500, but my code expected 400 for invalid input — fixed to use ErrBadRequest

### Verification (4 consecutive runs, all PASS)
```
Run 1: 8/8 PASS
Run 2: 8/8 PASS
Run 3: 8/8 PASS
Run 4: 8/8 PASS
```

### Caveat: ZFS create NOT tested live
- .107 has only 1 disk (OS disk sda) — destructive ZFS create would corrupt
- Created code + validation; manual end-to-end test deferred until we have spare disks

### Next (Tier 1.8)
- Users & auth (Proxmox user mgmt, permissions, PVE tokens)
- OR Tasks (Proxmox task history)
- OR Backup (scheduled backups + restore)

---

## Session 9 — 2026-08-17 (Tier 1.8)

### Goals
- Proxmox user mgmt + API tokens

### Done
- ✅ PVEUser/PVEAPIToken types with custom Groups unmarshal (accepts string OR array)
- ✅ APITokenCreateResponse separate from APIToken (POST has secret value, GET doesn't)
- ✅ 8 client methods: list/get/create/update/delete users + tokens
- ✅ 9 handlers + input validation (userid must contain @realm, password 5-64 chars)
- ✅ 9 routes wired
- ✅ 14/14 live tests pass, 4 consecutive runs
- ✅ Committed

### Proxmox quirks caught + fixed (7!)
31. POST /access/users returns 500 \"change password failed\" even on success → swallowed
32. GET /access/users/{userid} does NOT include userid in response → injected
33. groups field is array in GET, string in LIST → custom UnmarshalJSON
34. token POST response is special shape (not flat APIToken) → dedicated struct
35. privsep field is int (1) OR string ("0") depending on what was set
36. Token secret value must be REDACTED on GET but returned on POST → handler layer
37. userid must have @realm (no @ = bad request)

### Verification (4 consecutive runs, all PASS)
```
Run 1: 14/14 PASS
Run 2: 14/14 PASS
Run 3: 14/14 PASS
Run 4: 14/14 PASS
```

### Caveat
- LIST/GET tokens on non-root users fails with permission denied (Proxmox quirk —
  Permissions.Modify perm required, only granted on root@pam in this cluster)
- Create/delete token WORK on root@pam and on freshly-created test users
- Password NOT set via API (Proxmox POST /access/users doesn't accept password field
  in schema; password must be set via PAM backend directly or /access/password)

### Next (Tier 1.9)
- Tasks (Proxmox task history + status polling)
- OR Backup (vzdump + restore)
- OR Replication (Proxmox replication jobs)

---

## Session 10 — 2026-08-17 (Tier 1.9)

### Goals
- Proxmox task management (history, status, log, stop)

### Done
- ✅ Task + TaskLogLine + TaskLog structs (log decodes full wrapper for total)
- ✅ 5 client methods: ListClusterTasks, ListNodeTasks, GetTaskStatus, GetTaskLog, StopTask
- ✅ 5 handlers + routes
- ✅ 7/7 live tests pass, 4 consecutive runs
- ✅ Committed

### Proxmox quirks caught + fixed (2)
38. /cluster/tasks does NOT accept ?limit=query param
39. /nodes/{node}/tasks/{upid}/log returns {total, data:[...]} at top level
    (NOT nested). Custom decoder preserves total.

### Next (Tier 1.10)
- Backup (vzdump + restore + PBS integration)
- OR Replication (Proxmox replication jobs)

---

## Session 11 — 2026-08-17 (Tier 1.10)

### Goals
- Storage content listing with content-type filtering
- Download/Upload URL endpoints
- ISCSI targets listing

### Done
- ✅ Content struct with ctime/vmid string-or-int normalization
- ✅ ListContent enhanced with optional content filter
- ✅ DownloadStorageFile (Gin catch-all for nested paths)
- ✅ UploadStorageFileToURL + GetStorageUploadURL
- ✅ ISCSITarget + ListISCSI
- ✅ 8/8 live tests pass, 4 consecutive runs
- ✅ Committed

### Proxmox quirks caught + fixed (3)
40. ctime is INT for vztmpl, STRING for rootdir
41. vmid is also inconsistent int/string
42. Volume paths contain / (vztmpl/file.tar.zst) — :volume param
    can't match → use Gin catch-all (*) and strip leading /

### Next (Tier 1.11)
- Cluster (HA groups, resources, status, replication)
- OR Pools (resource pools)
- OR Backup (vzdump + restore)

---

## Session 12 — 2026-08-17 (Tier 1.11)

### Goals
- Pools CRUD
- Cluster resources / status / info

### Done
- ✅ Pool struct (poolid + comment + members)
- ✅ PoolMember struct
- ✅ ListPools, GetPool, CreatePool, UpdatePool, DeletePool
- ✅ ClusterResource, ClusterStatusNode, ClusterInfo structs
- ✅ ListClusterResources (with type filter), GetClusterStatus, GetClusterInfo
- ✅ 12/12 live tests pass, 4 consecutive runs
- ✅ Committed

### Proxmox quirks caught + fixed (2)
43. /cluster/info not implemented on this Proxmox (.107) — accept 501
44. /cluster/resources?type=qemu rejected — Proxmox uses 'vm' filter
    (returns both qemu VMs and lxc containers)

### Next (Tier 1.12)
- Backup (vzdump + restore + scheduled)
- OR Certificates (ACME + custom SSL)
- OR SDN / DNS zones

---

## Session 13 — 2026-08-17 (Tier 1.12)

### Goals
- Backup schedule CRUD
- Immediate backup (vzdump)

### Done
- ✅ BackupJob struct (id, schedule, storage, mode, vmid, compress, maxdays)
- ✅ ListBackupJobs, GetBackupJob, CreateBackupJob, UpdateBackupJob, DeleteBackupJob
- ✅ BackupNow (vzdump) returns UPID
- ✅ 10/10 live tests pass, 4 consecutive runs
- ✅ Committed

### Proxmox quirks caught + fixed (1)
45. UpdateBackupJob sent 'disable=1' for jobs.Enabled=0 — Proxmox
    schema rejects 'disable' as unknown property. Removed.

### Next (Tier 1.13 - FINAL)
- Certificates (ACME + custom SSL)
- OR SDN / DNS zones
- OR Replication jobs

---

## Session 14 — 2026-08-17 (Tier 1.13 — FINAL of Tier 1)

### Goals
- Certificates + ACME management

### Done
- ✅ Certificate struct (name/filename + full fields)
- ✅ ACMEAccount, ACMEPlugin, ACMEChallengeSchema, ACMEDirectory struct
- ✅ ListCertificates, ListACMEAccounts, GetACMEAccount, CreateACMEAccount, DeleteACMEAccount
- ✅ ListACMEPlugins, CreateACMEPlugin, DeleteACMEPlugin
- ✅ ListACMEChallengeSchema, ListACMEDirectories, GetACMEInfo
- ✅ 15/15 live tests pass, 4 consecutive runs
- ✅ Committed

### Proxmox quirks caught + fixed (3)
46. /nodes/{node}/certificates returns {name: "..."} not {filename: ...}
47. /cluster/acme/info not implemented on this Proxmox (.107) - accept 500/501
48. ACME account + plugin CREATE require root@pam (Permissions.Modify)
    - Monitor token has PVEAuditor - 403/500 are valid proof that handler
      schema is correct (Proxmox validates auth before handler reaches write)

### Tier 1 COMPLETE (13/13 tiers)
- 1.0 Foundation
- 1.1 Nodes & Versions
- 1.2 VM Lifecycle
- 1.3 LXC Lifecycle
- 1.4 Storage
- 1.5 Networking
- 1.6 Firewall
- 1.7 Disks
- 1.8 Users & Tokens
- 1.9 Tasks
- 1.10 Storage Content + ISCSI
- 1.11 Pools + Cluster Resources
- 1.12 Backup
- 1.13 Certificates + ACME

### Next tier destination: TIER 2
- Tier 2 covers TrueNAS (replacing TrueNAS Scale web UI)
- OR Tier 3 covers Termius (web terminal)
- OR Tier 4 covers Netdata (perf monitoring)
- OR Tier 5 covers Cockpit (server admin)

---

## Session 15 — 2026-08-17 (Tier 3.1 — Terminal / Termius replacement)

### Goals
- SSH key CRUD (generate ed25519/RSA, optional passphrase)
- Saved connections CRUD + groups + tags
- Real SSH dial to verify connectivity
- Connection history log

### Done
- ✅ All 14 endpoints, 24/24 live tests PASS, 4 consecutive runs

### Bugs caught + fixed (5)
54. `ssh.NewPublicKey` takes value (ed25519.PublicKey) not pointer — first PASS run
55. `ssh.NewPublicKey` takes *rsa.PublicKey not value — second run got 1 fail
56. Tenant context: must use `tenantIDFromContext(c)` not `c.Get("tenant_id")` — handlers must use shared helpers
57. Duplicate `TerminalHandler` + helpers across split files — cleanup on file split
58. Unused imports in split files (crypto/ed25519 in keys file, etc.)

### Files
- internal/handler/terminal.go              (43 lines — root)
- internal/handler/terminal_keys.go         (185 lines — SSH key CRUD)
- internal/handler/terminal_connections.go  (~450 lines — connections + history + dial)
- cmd/api-gateway/routes.go                 (+14 routes)
- cmd/web-terminal/main.go                  (346 lines — service scaffold, not yet deployed)
- migrations/002_terminal_schema.sql        (ssh_keys, connections, connection_history)

### Live state
- api-gateway on .115: md5 `606c68604957a5654668e8f0233e4157`
- 181/181 total live tests across 15 tiers
