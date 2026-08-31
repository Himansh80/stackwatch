# StackWatch — Master Plan (Session 2026-08-31)

> **Source of truth for the entire rebuild.** All work tracked here. Replace, never
> delete — every version is a strict cumulative superset.

---

## 0. Vision (user-stated 2026-08-31)

StackWatch replaces the entire toolchain below. **No third-party lock-in after this.**
End-to-end feature parity. Full Datadog-style UI/UX. Strict modularity. Every feature
of every referenced platform — user must feel like they're using the original.

| Replaces | Tier |
|---|---|
| Portainer (container mgmt) | T5 (extend) + T16 (K8s) |
| Termix (terminal) | T3 (extend) |
| Loki (logs) | T6 (extend) |
| Cockpit (server admin) | T4 (extend) |
| Prometheus / Grafana / Netdata (monitoring) | T6 (extend) + T17 (depth) |
| Datadog / Grafana Cloud / Better Stack / SigNoz / HyperDX / New Relic | T7 + T15 + T18 + T19 |
| Proxmox UI | T1 + T14 (extend) |
| TrueNAS Scale UI | T2 (extend) |

**Universal standards (per user directive, locked):**
- Modular: every file ≤400 LOC
- Datadog-quality UI (dark theme, KPI cards with stripes, status pills, framer-motion)
- Speckit workflow per workstream (proposal → spec → plan → tasks → checklist → build → archive)
- framer-motion for animation (whileHover, AnimatePresence, layout transitions, count-up)
- Test workflow: during-phase build/test/fix + per-phase A-to-Z test plan + tier-end mega-test
- Honest verification: 4 back-to-back runs, fresh cache, real data

---

## 1. Current state (audit 2026-08-31)

### What's built (working on `.115`)
- Tier 0 — Foundation (auth + tenants + users + api-keys) — ~17 routes
- Tier 0.5 — Core Monitoring (servers, agents, metrics, logs) — ~11 routes
- Tier 1 — Proxmox VE (1.1–1.13 +14.x) — ~125 routes
- Tier 2 — TrueNAS SCALE sidecar — 1 proxy route + 11 handler files
- Tier 3 — Terminal/SSH/SFTP — ~30 routes
- Tier 4 — Server Admin (Cockpit parity) — 10 routes
- Tier 5 — Containers (Portainer + Watchtower) — ~20 routes
- Tier 6 — Prom/Loki/ML/Dashboards/RUM/Tracing — ~30 routes
- Tier 7 — Datadog parity D2-D12 — ~80 routes
- Tier 8 — Intelligence & Alerting — 25 routes
- Tier 9 — Enterprise SSO/SCIM/RBAC/Audit/Compliance/Orgs — 31 routes
- Tier 10 — Homelab Dashboard — 53 routes
- Tier 11 — Platform & Commerce — 34 routes
- Tier 12 — Docs & GTM — 16 docs files (~150 KB)
- Tier 13 — Mobile + Push — 9 routes + RN app
- Tier 14 — Proxmox UI frontend (17 pages)
- Phase 1.x — Modularity + Observability (in-progress, WIP)

### Live binary on `.115`
- `api-gateway-linux` md5 `e39b4bcd50692a82466ea5cf6bc227c3` (PID 1517, active)
- `truenas-connector-linux` PID 1007
- All routes mounted, healthy

### Codebase metrics
- Go: `cmd/api-gateway` (10 files) + `cmd/truenas-connector` + `cmd/web-terminal`
- Go: `internal/` — 233 handler files + 14 sub-packages
- Go: 41 SQL migrations
- TS: 27 pages, 78 components, 14 CSS files (~52K LOC)
- TS: 30+ frontend routes registered in App.tsx

### UI design system
- `web/src/styles/tokens.css` — color/spacing/type/radius/motion tokens
- `web/src/styles/DESIGN-SYSTEM.md` — documented philosophy + anti-slop checklist
- `web/src/tokens.ts` — TypeScript mirror (untracked in git)
- Shared components: StatusPill, KpiCard, EmptyState, TimeSeriesChart, ErrorGroupCard,
  IncidentCard, ThreatCard, SsoProviderCard, FlameGraph, etc. (~34 files in shared/)

---

## 2. Gaps found (must address)

### Backend gaps
- Tier 15 — **On-Call & SLOs**: schedules, rotations, escalations, SLO/SLI tracking
- Tier 16 — **K8s + Edge + RBAC**: K8s management, edge agents, per-resource RBAC
- Tier 17 — **Observability depth**: per-second native, recording rules, federation, Pyroscope, auto-discovery
- Tier 18 — **Case Mgmt + Runbooks + Integrations marketplace**
- Tier 19 — **Cost + Cloud SIEM + LLM Obs**
- Tier 22 — **Service Catalog + Feature Flags + SAST**
- Tier 23 — **Security hardening**: SQL injection sweep, JWT rotation, audit completeness, RLS enable

### Frontend / UI gaps (from initial audit)
- **U1** Old sidebar (`AppSidebar.tsx`) uses emoji chars — already dead code, delete
- **U4** Topbar has 8 sub-components — verify clean composition
- **T20** UI consistency pass: every page's button/card/modal/table consistent

### Repo hygiene
- **U5** 31 `.bak-*` binaries in `/opt/stackwatch/bin/` + 20+ `tier*-linux-build` in repo root
- **U6** Working tree dirty (modified, untracked, deleted planning files)
- **U7** `.specify/memory/constitution.md` is blank template
- **U8** `ProxmoxWorkspace.tsx` 668 LOC — over 400 cap
- **U9** 233 handler files — likely many duplicate patterns

---

## 3. Execution plan

### Phase A — Foundation (1-2 sessions)
| Step | Action | Output | Done |
|---|---|---|---|
| A.1 | Full UI consistency audit (every page + component) | `docs/UI-AUDIT-REPORT.md` | ☐ |
| A.2 | Backend health audit (SQL injection, JWT, audit log, tenant scope) | `docs/BACKEND-AUDIT-REPORT.md` | ☐ |
| A.3 | Fill `.specify/memory/constitution.md` with quality gates | constitution.md ratified | ☐ |
| A.4 | Delete `web/src/components/AppSidebar.tsx` (dead code) | file deleted | ☐ |
| A.5 | Archive `.bak-*` binaries to `archive/binaries-2026-08-31/` | moved | ☐ |
| A.6 | Commit working tree (modified + untracked + deleted-planning-files decision) | clean HEAD | ☐ |

### Phase B — Tier gap-fill (speckit per tier)

Each tier below = one `.hermes/changes/NNN-tierXX-name/` with proposal → spec → plan →
tasks → checklist → build → archive workflow.

| Order | Tier | Subagent? | Priority |
|---|---|---|---|
| 1 | **T20 — UI Consistency Polish** | Yes | High (foundation for everything else) |
| 2 | **T21 — Modularity Cleanup** | Yes | High (foundation) |
| 3 | **T15 — On-Call & SLOs** | Yes | Medium |
| 4 | **T17 — Observability Depth** | Yes | Medium |
| 5 | **T16 — K8s + Edge + RBAC** | Yes | Medium |
| 6 | **T18 — Case Mgmt + Runbooks + Integrations** | Yes | Medium |
| 7 | **T19 — Cost + Cloud SIEM + LLM Obs** | Yes | Low |
| 8 | **T22 — Service Catalog + Feature Flags + Code Sec** | Yes | Low |
| 9 | **T23 — Security Hardening** | Yes | Critical (before prod promotion) |
| 10 | **T24 — Marketing site + Docs refresh** | Yes | Final |

### Phase C — Verification (live, every tier)
- 4 back-to-back full-stack verifier runs (fresh JWT per pass)
- UI: Playwright screenshots vs Datadog/Stripe reference, hard refresh (Ctrl+Shift+R)
- Backend: every endpoint probed, rate-limiter cycled
- Browser cache verified cleared before each visual claim

---

## 4. Skill stack (per phase)

| Phase | Skills |
|---|---|
| Audit | `ui-ux-pro-max` (not installed — fall back to DESIGN-SYSTEM.md), `advanced-debugging-methodology`, `static-verification`, `simplify-code` |
| Speckit | `spec-driven-development`, `subagent-driven-development` (2-stage review) |
| Build | `framer-motion`, `popular-web-designs`, `agent-design-intelligence`, `21st.dev` (via tool_search), `tdd-enforcement` (where applicable) |
| Verify | `verification-first-completion`, `requesting-code-review`, `static-verification` |
| Custom | `stackwatch-build-deploy`, `ios-platform-backend` |

---

## 5. Universal rules (locked from user directive)

1. **No emoji as primary UI.** Use SVG icons from `web/src/components/icons.tsx`.
2. **No hardcoded hex in components.** Use `var(--*)` tokens.
3. **No file >400 LOC.** Split by domain.
4. **Every interactive element** has hover + focus + active states.
5. **KPI cards** use 3px left stripe + tabular-nums on values.
6. **Status pills** use `StatusPill` shared component (never inline).
7. **Framer-motion** on every transition (200ms ease-out default).
8. **Skip-link** at top of every authenticated page.
9. **Empty states** guide action ("Add your first server" not "No data").
10. **Loading states** with spinners, not blank space.
11. **Mobile responsive** at 320px and 1920px.
12. **Tabular-nums** on all KPI values.
13. **WCAG AA contrast** (4.5:1 body text).

---

## 6. Deployment discipline (locked from stackwatch-build-deploy skill)

- Build env gotcha: `export GOOS=linux; export GOARCH=amd64; export CGO_ENABLED=0; go build -o ./<local>` — combined form silently fails on Windows
- ALWAYS `ls -la ./<local>` to confirm binary exists
- 5-second sleep between heavy tool calls
- `chmod +x` after every `cp` to /opt/stackwatch/bin/
- No restart of api-gateway without env sourced (`set -a && . .env && set +a`)
- Verify with `md5sum /proc/$(pgrep -f 'api-gateway-linux$' | head -1)/exe` to confirm running binary
- Rate-limit clears on restart (in-memory) — wait 60s between verifier passes

---

## 7. Locked decisions (2026-08-31)

User chose:

| Decision | Value | Meaning |
|---|---|---|
| **Tier order** | T20 first | UI consistency polish before anything else |
| **Plan rhythm** | A1 — one tier fully planned, then next | Slow, careful; user approves each step |
| **Build trigger** | B3 — after Tier 20 alone | Start building the moment Tier 20 plans are complete |

**Tier sequence (locked):** T20 → T21 → T15 → T17 → T16 → T18 → T19 → T22 → T23 → T24

---

## 8. Persistence + resume

To survive chat loss:

- `PROJECT.md` — this file (master plan)
- `RESUME.md` — chat-loss-recovery anchor (what's done, what's next, how to resume)
- `.hermes/changes/NNN-tierXX-name/` — speckit artifacts (proposal/spec/plan/tasks/checklist) per active tier
- `.hermes/changes/archive/` — completed tiers

If conversation is lost, open `RESUME.md` first.

---

## 9. Speckit workflow (per tier)

For each tier, 5 artifacts in this order:

1. `proposal.md` — Problem + scope + goals + risks + open questions
2. `spec.md` — FR + NFR + user stories + acceptance criteria
3. `plan.md` — Architecture + file-by-file + rollback
4. `tasks.md` — Numbered steps with verification gates
5. `checklist.md` — Per-feature acceptance ✅/❌

User reviews + approves each before next is written. After all 5 approved
→ build phase (code → commit → deploy → verify) → archive.

---

**Last updated:** 2026-08-31 (Tier 20 planning started; proposal.md done)
**Owner:** StackWatch Engineering