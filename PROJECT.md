# StackWatch — Unified Infrastructure + AI Platform

**Mission:** Build a single platform that replaces Datadog, Proxmox UI, TrueNAS UI, Portainer, Termix, Cockpit, Netdata, Grafana, Prometheus, Loki, Chrome Remote Desktop, and Homarr — strictly modular, fully open, self-hostable.

**Owner:** Himan Shukla (Founder/CTO, SmartHomeLab, Bareilly India)  
**Started:** Sunday, August 16, 2026  
**Status:** Tier 0 ✅ complete and live-verified; Tier 1 Proxmox is next

---

## The 5 LOCKED Decisions

| # | Decision | Choice |
|---|----------|--------|
| 1 | Scope | **Full vision** — all features of all 12 tools |
| 2 | Deployment | **Both** — self-hosted (central + agents) + cloud (later) |
| 3 | Architecture | **Hybrid** — own core, embed Prometheus + Loki |
| 4 | OS support | **Linux + Windows + macOS + mobile** |
| 5 | Time/money | **No budget** — time investment, iterative with user testing |

**Project name:** stackwatch (domain: stackwatch.smarthomelab.fun)  
**Path:** `C:\Users\himan\HermesProjects\stackwatch\`  
**GitHub:** TBD (will set up this session)

---

## The 12 TOOLS We Replace (Source of Truth)

| # | Tool | What it does | Tier |
|---|------|--------------|------|
| 1 | **Datadog** | Full monitoring + APM + Logs + RUM + Synthetics + Security | 7 |
| 2 | **Portainer** | Docker container management | 5 |
| 3 | **Watchtower** | Auto-update containers | 5 |
| 4 | **Termix** | Web SSH/RDP/VNC terminal | 3 |
| 5 | **Netdata** | Per-second metrics + ML anomaly | 6 |
| 6 | **Prometheus** | TSDB + PromQL | 6 |
| 7 | **Grafana** | Dashboards + data sources | 6 |
| 8 | **Cockpit** | Linux server admin UI | 4 |
| 9 | **Grafana Loki** | Log aggregation + LogQL | 6 |
| 10 | **Chrome Remote Desktop** | RDP into Windows/Mac/Linux | 3 |
| 11 | **Proxmox VE** | KVM/LXC/cluster/backup | 1 |
| 12 | **TrueNAS Scale** | ZFS/datasets/NFS/SMB/iSCSI | 2 |
| 13 | **Homarr** | Homelab dashboard | 10 |

---

## Modular Architecture (THE non-negotiable rule)

**Rule: Every file under 500 lines. Every feature in its own module. Every module < 2,000 lines.**

### Inspired by giants

| Giant | File size rule | Module boundary |
|-------|---------------|-----------------|
| Stripe | Max 500 lines per file | One business domain per package |
| Datadog | Max 300 lines per file | Domain-driven design |
| Grafana | Service per package | Hexagonal architecture |
| GitHub | Per-feature folder | Layered: transport → service → repo |

### Our folder structure

```
stackwatch/
├── cmd/                          # executables (one per service)
│   ├── api-gateway/              # HTTP gateway (:8080)
│   ├── agent/                    # host agent (Linux/Win/Mac)
│   ├── ingest/                   # metric ingestion (:8081)
│   ├── alert-engine/             # rule evaluator (:8082)
│   ├── ai-engine/                # LLM triage (:8083)
│   ├── web-terminal/             # SSH/RDP bridge (:8085)
│   ├── proxmox-connector/        # Proxmox API client
│   ├── truenas-connector/        # TrueNAS API client
│   ├── remote-access/            # Guacamole + RDP/VNC
│   ├── server-admin/             # Cockpit parity
│   ├── container-mgmt/           # Portainer parity
│   ├── analytics/                # Prometheus + Loki
│   ├── rmm/                      # Chrome RD + mesh
│   ├── auth/                     # SSO + 2FA + OAuth
│   ├── synthetics/               # HTTP/TCP/ping checks
│   ├── incident/                 # on-call + escalation
│   ├── notification/             # 15+ channels
│   ├── audit/                    # audit log central
│   └── billing/                  # Stripe + Razorpay
├── internal/                     # shared libraries
│   ├── auth/                     # JWT, 2FA, OAuth
│   ├── db/                       # pgx pool + tx
│   ├── cache/                    # Redis
│   ├── telemetry/               # OTEL + slog
│   ├── validation/              # go-playground/validator
│   ├── middleware/              # gin middleware
│   ├── errors/                  # error types
│   ├── client/                  # HTTP clients (Proxmox, TrueNAS, Guacamole)
│   ├── proto/                   # wire types
│   └── kernel/                  # shared types (Tenant, User, Pagination)
├── pkg/                         # public reusable libs
│   ├── notifier/                # interface + 15 channel impls
│   ├── parser/                  # log parser, metric extractor
│   ├── ml/                      # anomaly detection
│   ├── git/                     # GitHub/GitLab client
│   └── k8s/                     # K8s client helpers
├── web/                         # SPA frontend
│   ├── src/
│   │   ├── pages/               # one file per page
│   │   ├── components/          # UI components
│   │   ├── lib/                 # API client, formatters
│   │   ├── router/              # SPA routing
│   │   ├── state/               # state mgmt
│   │   └── styles/              # CSS modules
│   ├── dist/                    # gitignored
│   ├── tests/
│   ├── package.json
│   ├── tsconfig.json
│   └── build.mjs                # esbuild script
├── migrations/                  # numbered SQL migrations
├── deploy/                      # systemd, docker, helm
│   ├── systemd/
│   ├── docker/
│   ├── helm/
│   └── scripts/
├── tests/                       # integration + e2e
├── docs/                        # user-facing docs
│   ├── INSTALL.md
│   ├── USER-GUIDE.md
│   ├── ARCHITECTURE.md
│   ├── TESTING.md
│   ├── COMPARISON.md
│   ├── SELL-IT.md
│   ├── LICENSE.md
│   └── API.md
└── .github/workflows/           # CI
```

### One file = ONE feature

Examples:
- `pkg/notifier/slack/slack.go` — only Slack notifications
- `internal/handler/notifier.go` — lists/creates notification channels
- `cmd/alert-engine/evaluator.go` — only the rule evaluation loop
- `web/src/pages/servers.js` — only the servers list page
- `internal/handler/server/proxmox_vm.go` — only Proxmox VM endpoints

No file > 500 lines. No "misc" or "utils" or "helpers" files. If a file is growing, **split**.

### Module rules (enforced by CI)

```yaml
# .golangci.yml
linters:
  - gocyclo   # max cyclomatic complexity 15
  - dupl      # no duplicate code blocks > 100 lines
  - gocognit  # cognitive complexity 20
  - lll       # line length 120
  - wsl       # whitespace
  - funlen    # any function > 50 lines
  - nlreturn  # consistent newlines
  - nestif    # max if nesting 5
```

```bash
# CI step
go build ./...
go test ./...
golangci-lint run
gofmt -l . | grep -v vendor/    # must be empty
find . -name "*.go" -exec wc -l {} \; | awk '$1 > 500 { print }'   # must be empty
```

---

## The 13 TIERS (build order)

| Tier | What | Effort | Replaces |
|------|------|--------|----------|
| **0** | **Foundation** | 1 session | - |
| **1** | Proxmox full replacement | 3-4 sessions | Proxmox UI |
| **2** | TrueNAS full replacement | 3-4 sessions | TrueNAS UI |
| **3** | Remote access (RDP/VNC/SSH) | 1-2 sessions | Chrome RD + Termix |
| **4** | Server admin (Cockpit parity) | 2-3 sessions | Cockpit |
| **5** | Container management (Portainer + Watchtower) | 2 sessions | Portainer + Watchtower |
| **6** | Monitoring depth (Prometheus + Grafana + Loki + Netdata) | 3-4 sessions | Netdata + Prom + Grafana + Loki |
| **7** | Datadog full platform (APM/RUM/Synthetics/Security) | 4-5 sessions | Datadog |
| **8** | Intelligence & alerting (composite/anomaly/ML/on-call) | 2-3 sessions | PagerDuty |
| **9** | Security & enterprise (SSO/RBAC/2FA/SOC2) | 2 sessions | - |
| **10** | Homelab dashboard (Homarr parity) | 1 session | Homarr |
| **11** | Platform & commerce (billing/tenants/quotas) | 1-2 sessions | - |
| **12** | Docs & GTM | 1 session | - |
| **13** | Mobile apps (iOS/Android) | 1 session | - |

**Total: 26-35 sessions.**

---

## The VERIFICATION Contract (strict)

For every feature I build:

1. **Code complete** — written, compiles, all tests pass
2. **Live verify** — I run live tests against a running system, capture results in `tests/output/`
3. **TESTING.md** — I write a step-by-step guide for YOU to test
4. **You test** — you run the steps, report errors
5. **I fix** — I fix only what you reported
6. **You re-test** — you confirm fix works
7. **Move on** — only after you say "next"

I never skip step 2. I never claim "done" without fresh test output in the same turn.

---

## The WORKFLOW Rules

### Before any deploy
- I write the exact command + destination + blast radius
- I wait for your "go" before executing
- I never auto-push to GitHub without permission

### Before any commit
- `go build ./...` passes
- `go test ./...` passes
- `gofmt -l .` is empty
- `golangci-lint run` passes
- All checks captured in `tests/CI-LOG.md`

### Per module
- One module per commit
- Commit message format: `feat(tier-5): add container start/stop handler`
- Never bundle fixes + features in one commit

### Per session
- MORNING-REPORT.md at end (if overnight)
- Update PROJECT.md with status + next steps
- Update journal.md with what happened

---

## DOCS Structure (deliverables)

```
docs/
├── INDEX.md                  # user-facing TOC
├── INSTALL.md                # fresh install (dev + prod)
├── USER-GUIDE.md            # how to use every feature
├── ARCHITECTURE.md          # how it works (modular diagram)
├── TESTING.md               # what's tested, how to test yourself
├── COMPARISON.md            # vs Datadog/Proxmox/TrueNAS/Portainer/etc.
├── SELL-IT.md               # reseller guide
├── LICENSE.md               # AGPL-3
├── FAQ.md
├── TROUBLESHOOTING.md
├── SELF-HOST-GUIDE.md       # how to self-host
├── SECURITY.md              # security model + runbook
├── FEATURES.md              # every feature listed
├── API.md                   # REST API reference
└── PRICING.md               # pricing tiers
```

---

## CURRENT STATE

**Stage:** Tier 1 Proxmox backend implemented; live A-to-Z audit **not complete** (2026-08-18)  
**Verified now:** 37/38 read-endpoint checks passed in each of 3 consecutive fresh runs against `.115:8080`  
**Last live verification:** Tier 0 — 82/82 PASS, repeated four consecutive runs (2026-08-18)
**Known defect:** Missing ACME account detail returns HTTP 500 instead of HTTP 404  
**Missing for true end-to-end completion:** Tier 1 frontend screens; complete safe CRUD/write-endpoint verification; four clean full runs without rate-limit interruption  
**Next:** Fix the ACME not-found mapping, build the Tier 1 UI, then verify every Tier 1 endpoint and safe lifecycle operation live  
**Blocked on:** Nothing technical; the fourth fresh-login pass was temporarily blocked by the in-memory login rate limiter  

---

## RELATED DOCS

- `MASTER_BUILD_PLAN.md` — full tier-by-tier plan with sub-features
- `RESEARCH.md` — features of each tool we replace (from research)
- `MODULARITY_RULES.md` — strict modularity rules
- `VERIFICATION_CONTRACT.md` — what "done" means
- `tiers/` — folder containing per-tier break-down
- `journal.md` — daily session log
- `errors.md` — every error encountered + fix
- `decisions.md` — architecture decisions + rationale
- `state/` — runtime state snapshots
