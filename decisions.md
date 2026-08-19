# StackWatch — Architecture Decisions

> Append-only. Every non-trivial choice + rationale.

---

## 2026-08-16 — Initial architecture decisions

### D1: Project name = stackwatch
- **Decision:** Use existing name `stackwatch` (not `ios-platform` or `stackwatch-platform`)
- **Why:** Domain `stackwatch.smarthomelab.fun` already exists from previous project
- **Trade-off:** None — name is brand-neutral

### D2: Path = `C:\Users\himan\HermesProjects\stackwatch\`
- **Decision:** Use existing workspace folder pattern
- **Why:** Consistent with other projects (book handbook, smarthomelab-iot, etc.)
- **Why not `C:\Users\himan\Projects\stackwatch\`:** Stick to established workspace

### D3: Architecture = modular monolith
- **Decision:** Single Go module, multiple services, strict modularity rules
- **Why:** User explicitly said "if I add a feature and code is not modular, syntax error breaks everything"
- **Why not microservices:** Too much operational complexity for solo founder
- **Why not single binary:** User needs to deploy individual services

### D4: Stack = Go + React + TypeScript + Postgres + Loki + Prometheus
- **Decision:** Go for backend, React+TS for frontend, Postgres for primary store, embed Prometheus + Loki for time-series
- **Why:** Go = fastest single-binary deployment, React+TS = type safety on frontend, Postgres = mature + RLS support
- **Why embed Prom + Loki:** User explicitly said "we have free open source options like Prom and Loki"
- **Why not Elasticsearch:** Too heavy

### D5: No file > 500 lines
- **Decision:** CI-enforced limit. PR blocked if violated.
- **Why:** Stripe uses 500-line rule. Datadog uses 300-line rule. User wants "change one file = only one feature breaks"
- **Why not 300:** Stripe (which serves more code than Datadog) uses 500. More lenient = less friction.

### D6: Service layer pattern
- **Decision:** handlers → services → repositories, no skip levels
- **Why:** Standard modular monolith pattern. Handlers do HTTP only. Services do business logic. Repositories do DB.
- **Why not direct handler→DB:** Untestable, business logic in HTTP layer

### D7: Per-feature folders
- **Decision:** `internal/handler/proxmox/{vm,lxc,storage,network}_test.go` not `internal/handler/proxmox.go`
- **Why:** One file per feature = one failure mode per file
- **Pattern from GitHub:** Every feature has its own folder

### D8: Cache layer = Redis
- **Decision:** Use Redis for caching, later for queue
- **Why:** Mature, available, user already has it
- **Why not in-memory:** Lose state on restart

### D9: GitHub as source of truth
- **Decision:** All code on GitHub private repo
- **Why:** User explicitly said "we push everything in our journey to the git hub for safety"
- **Repo name:** TBD (will ask user)

### D10: Deployment pattern (multiple)
- **Decision:** First release = systemd services + binaries. Docker + Kubernetes as later option.
- **Why:** User's current setup is systemd. Lower friction.
- **Why not Docker-first:** User said "I want my system to manage everything"

### D11: Frontend build = Vite + esbuild
- **Decision:** Vite for dev, esbuild for prod, ship pre-built
- **Why:** Fastest dev loop, single static bundle for production
- **Why not webpack:** Slower, more config

### D12: Build pattern = `go build ./cmd/<service>`
- **Decision:** One binary per service
- **Why:** User wanted "single binary per service" (their existing pattern)
- **Why not one big binary:** Different services need different lifecycles

### D13: Tier 0 first
- **Decision:** Build foundation before any feature
- **Why:** User wants modularity. Foundation locks the structure.
- **Includes:** git repo, CI, Go base, React base, DB schema, auth, single dummy endpoint

### D14: Proxmox error contract is normalized at the boundary
- **Decision:** The shared Proxmox client and handler responder translate legacy PVE behavior into stable StackWatch semantics.
- **Mapping:** 400 invalid input, 403 permission denied, 404 absent resource, 501 unsupported capability represented as HTTP 200 with `supported:false` where safe.
- **Why:** PVE versions return different status codes and message shapes; leaking them makes the frontend unpredictable.
- **Trade-off:** The client contains compatibility phrases for older PVE, but this is safer than duplicating version logic in every handler.

### D16: Frontend foundation = command center before feature pages
- **Decision:** Make `/` and `/dashboard` a live command center, keep Proxmox and TrueNAS as separate workspaces, and add domain pages incrementally.
- **Why:** A dashboard needs a stable shell, hierarchy, navigation, live status, loading/error/empty states, and responsive behavior before adding dozens of feature screens.
- **Visual direction:** Datadog/Grafana/Better Stack information density and operational clarity, with StackWatch cyan-indigo identity and no copied vendor branding.
- **Data rule:** Every visible metric on the command center comes from an existing API response; no hardcoded sample telemetry.
- **Trade-off:** Some sidebar destinations remain intentionally routed to the command center until their real API-backed pages are built. This avoids fake functionality.

### D15: Release verifier uses one JWT per pass
- **Decision:** A complete verifier pass authenticates once and reuses the token for every endpoint.
- **Why:** Repeated login attempts trigger the intentional in-memory brute-force limiter and create false failures.
- **Trade-off:** Four-pass orchestration restarts the gateway between passes until the limiter is moved to Redis.
