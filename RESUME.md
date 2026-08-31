# RESUME — StackWatch Rebuild Session 2026-08-31+

> **This document is the chat-loss-recovery anchor.** If the conversation is
> ever lost, open this file first. It tells you everything: what StackWatch
> is, what the user wants, where the plans live, what's done, what's next.

---

## What StackWatch is

The user (Himan, founder/CTO SmartHomeLab, Bareilly India) is building
StackWatch — a single open-source platform that **replaces**:

| Tool | Replaces |
|---|---|
| Tier 5 + Tier 16 | Portainer (container mgmt) |
| Tier 3 | Termix (terminal) |
| Tier 6 | Loki (logs) |
| Tier 4 | Cockpit (server admin) |
| Tier 6 + Tier 17 | Prometheus + Grafana + Netdata (monitoring) |
| Tier 7 + 15 + 18 + 19 | Datadog + Grafana Cloud + Better Stack + SigNoz + HyperDX + New Relic + Sentry |
| Tier 1 + Tier 14 | Proxmox UI |
| Tier 2 | TrueNAS Scale UI |

**No third-party lock-in after this. Every feature of every referenced tool.**

---

## What the user locked in (this conversation)

| Decision | Value | Why |
|---|---|---|
| Tier order | **T20 first**, then T21, then T15, T17, T16, T18, T19, T22, T23, T24 | Foundation work before features |
| Plan-then-build rhythm | **A1** — one tier fully planned, then next | Slow + careful; nothing slips |
| When to start building | **B3** — after Tier 20 alone | Start with what user explicitly called out (UI consistency) |
| Save everything | Yes — disk artifacts + RESUME.md | So chat loss can't kill progress |

---

## Tier roadmap (10 tiers to plan + ship)

| # | Tier | Scope | Routes | Tables | Est. plan lines |
|---|---|---|---|---|---|
| **1** | **T20** ← CURRENT | UI Consistency Polish | ~0 | 0 | ~1,800 |
| 2 | T21 | Modularity Cleanup | 0 | 0 | ~1,200 |
| 3 | T15 | On-Call & SLOs | ~25 | 6 | ~1,500 |
| 4 | T17 | Observability depth | ~25 | 5 | ~1,400 |
| 5 | T16 | K8s + Edge + RBAC | ~40 | 10 | ~1,800 |
| 6 | T18 | Case Mgmt + Runbooks + Integrations | ~50 | 15 | ~2,000 |
| 7 | T19 | Cost + Cloud SIEM + LLM Obs | ~25 | 10 | ~1,600 |
| 8 | T22 | Service Catalog + Feature Flags + SAST | ~30 | 8 | ~1,600 |
| 9 | T23 | Security Hardening | 0 | 0 | ~1,400 |
| 10 | T24 | Marketing + Docs refresh | 0 | 0 | ~800 |

---

## What lives where

| Path | What |
|---|---|
| `C:\Users\himan\HermesProjects\stackwatch\PROJECT.md` | Master plan (vision, current state, gaps, execution plan) |
| `C:\Users\himan\HermesProjects\stackwatch\RESUME.md` | **THIS FILE** — chat-loss recovery anchor |
| `C:\Users\himan\HermesProjects\stackwatch\.hermes\changes\NNN-tierXX-name\` | Speckit artifacts for active tier |
| `C:\Users\himan\HermesProjects\stackwatch\.hermes\changes\archive\YYYY-MM-DD-NNN-tierXX-name\` | Archived (completed) tiers |
| `C:\Users\himan\HermesProjects\stackwatch\docs\` | Customer-facing docs (live) |
| `C:\Users\himan\HermesProjects\stackwatch\web\src\` | Frontend (React + TypeScript) |
| `C:\Users\himan\HermesProjects\stackwatch\cmd\` | Backend (Go) |
| `C:\Users\himan\HermesProjects\stackwatch\migrations\` | DB migrations (SQL) |

---

## Per-tier speckit workflow (5 artifacts)

For each tier, write these in order:

1. **`proposal.md`** — Problem + scope + goals + risks + open questions
2. **`spec.md`** — Functional + non-functional requirements + user stories + acceptance criteria
3. **`plan.md`** — Architecture + file-by-file changes + migration + rollback
4. **`tasks.md`** — Numbered steps with verification gates
5. **`checklist.md`** — Per-feature acceptance (✅ / ❌)

User approves each one before next is written. After all 5 approved → **build phase** (code → deploy → verify).

---

## Current status

### T20 — UI Consistency Polish (in progress)

**Status:** PLANNING — 2 of 5 artifacts done.

- [x] **proposal.md** written → `.hermes/changes/020-ui-consistency-polish/proposal.md`
- [x] **spec.md** written → `.hermes/changes/020-ui-consistency-polish/spec.md` (8 components spec'd, 29 AC)
- [x] **plan.md** written → `.hermes/changes/020-ui-consistency-polish/plan.md` (6 phases A–F, file-by-file changes, ~31-33h effort)
- [x] **tasks.md** written → `.hermes/changes/020-ui-consistency-polish/tasks.md` (33 numbered tasks A.1 → F.7)
- [x] **checklist.md** written → `.hermes/changes/020-ui-consistency-polish/checklist.md` (29 ACs + 27 page checklists)
- [ ] **BUILD PHASE** — Phase A.1 in progress (audit dispatched), Phase B.1 done (Button)
  - [x] **Phase B.1** — Button.tsx + test + CSS
  - [ ] Phase B.2 — Modal
  - [ ] Phase B.3 — DataTable
  - [ ] Phase B.4 — Input + Select + Textarea
  - [ ] Phase B.5 — SkeletonCard
  - [ ] Phase B.6 — BrandLogo
  - [ ] Phase B.7 — Verify existing shared components
  - [ ] Phase B.8 — Commit Phase B
  - [ ] Phase C — StyleGuide page
  - [ ] Phase D — Page refactor (27 pages)
  - [ ] Phase E — Cleanup + branding
  - [ ] Phase F — Verification
- [ ] VERIFY PHASE (4 back-to-back full-stack runs)

### Tier 20 quick summary (for context)

Found 8 issues from initial audit:
- U1: Old sidebar uses emoji icons (forbidden)
- U2: Old sidebar is dead code (delete)
- U3: Brand mark is letter "S" (use SVG logo)
- U4: 8 topbar sub-components (verify clean composition)
- U5: 31 `.bak-*` binaries + 20+ `tier*-linux-build` clutter repo
- U6: Working tree dirty (2 modified + 1 untracked + planning files deleted)
- U7: `.specify/memory/constitution.md` is blank template
- U8: `ProxmoxWorkspace.tsx` 668 LOC (over 400 cap)

---

## How to resume after chat loss

1. **Open `RESUME.md`** (this file) — you are reading it.
2. **Open `PROJECT.md`** — master plan with vision + execution phases.
3. **Check `.hermes/changes/`** — see what tiers are in-progress (active) vs archived (done).
4. **For the active tier** — open its speckit artifacts in order:
   - `proposal.md` (read first)
   - `spec.md` (read second)
   - `plan.md` (read third)
   - `tasks.md` (read fourth)
   - `checklist.md` (read fifth)
5. **Determine where you left off**: which artifact is the latest. If only proposal exists, the next step is to write spec.md. If all 5 exist, the next step is the build phase.
6. **Tell the user**: "Resuming Tier X. Currently [proposal/spec/plan/tasks/checklist/build/verify] is done. Next is to [write next artifact / start building / etc]."

---

## Production state (live, `.115`)

- **api-gateway-linux** md5 `e39b4bcd50692a82466ea5cf6bc227c3` (PID 1517, active)
- **truenas-connector-linux** PID 1007 (active)
- **DB**: Postgres on `.116` (database `ios`, user `ios`)
- **Web terminal**: web-terminal-linux on `:8085`
- **Dashboard URL**: https://stackwatch.smarthomelab.fun/app.html
- **Credentials**:
  - super_admin: `himanshukhandelwal944@gmail.com` / `@Himanshu&8363`
  - consumer demo: `demo@ios-platform.io` / `NewSecret!2026BlueCube`
- **SSH**: `ssh root@192.168.0.115` (direct)
- **DB shell**: `PGPASSWORD=3d5cb43fba1f82283a2ba02c79e116cf psql -h 192.168.0.116 -U ios -d ios`

---

## Build/deploy rules (from stackwatch-build-deploy skill)

1. **Windows git-bash build gotcha**: `export GOOS=linux; export GOARCH=amd64; export CGO_ENABLED=0` then `go build -o ./<local-file>` — combined form silently fails
2. **ALWAYS** `ls -la ./<local-file>` after build to confirm it exists
3. **5-second sleep** between heavy tool calls (builds, deploys, restarts)
4. **Deploy**: `scp` binary to `.115` → `systemctl stop stackwatch-api-gateway` → `cp + chmod +x` → `systemctl start` → `sleep 7 && systemctl is-active`
5. **MUST COMMIT**: every change, no exceptions
6. **No emoji as primary UI** — use SVG icons from `web/src/components/icons.tsx`
7. **No file >400 LOC** — split by domain
8. **All colors via `var(--*)` tokens** — `web/src/styles/tokens.css` is source of truth

---

## Universal standards (locked from user)

- **Modular**: every file ≤400 LOC
- **Datadog UI/UX**: dark theme, KPI cards with stripes, status pills, framer-motion
- **Speckit workflow**: proposal → spec → plan → tasks → checklist per workstream
- **framer-motion**: every transition (whileHover, AnimatePresence, layout transitions, count-up)
- **Tests**: per-phase A-to-Z test + tier-end mega-test (4 back-to-back full-stack runs)
- **Honest verification**: 4 back-to-back, fresh cache, real data, never narrative

---

## Last updated

**2026-08-31** — Tier 20 planning started; proposal.md complete; awaiting user review before spec.md.