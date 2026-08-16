# StackWatch — Modular Architecture Rules (STRICT)

**Date:** 2026-08-16  
**Goal:** Match Datadog / Stripe / GitHub code quality. Change one file = only that feature breaks.

---

## Why this matters

The user explicitly said: "if I add a new feature and the code is not modular and I have done a mistake or syntax error or something else it would break my entire system and then I have to do all the work from start."

**This is the #1 priority. Period.**

---

## The 7 HARD RULES

### Rule 1: No file > 500 lines

This is enforced by CI. If a file is 501 lines, **PR is blocked**.

Exemptions (require explicit comment + approval):
- Generated code (auto-generated mocks, protobuf)
- `*_test.go` files for integration tests
- Migration files (SQL — long blobs are OK)

### Rule 2: No god packages

A package should have ONE clear purpose. Examples:
- ✅ `internal/handler/proxmox_vm.go` — only Proxmox VM endpoints
- ✅ `internal/handler/notifier.go` — only notification channel CRUD
- ❌ `internal/handler/all.go` — VMs + alerts + billing + everything
- ❌ `internal/utils/helpers.go` — random collection of unrelated helpers

### Rule 3: One file per business domain

Each business domain gets its own folder:
- `internal/handler/proxmox/` — all Proxmox handlers
- `internal/handler/truenas/` — all TrueNAS handlers
- `internal/handler/remote/` — all RDP/VNC/SSH handlers
- `internal/handler/admin/` — all server admin handlers

Each file in that folder holds ONE feature:
- `internal/handler/proxmox/vm.go`
- `internal/handler/proxmox/lxc.go`
- `internal/handler/proxmox/storage.go`

### Rule 4: Service layer for all business logic

Handlers are PURE HTTP. No DB queries in handlers. No business logic in handlers.

```go
// BAD — handler does both
func (s *Server) listAlerts(c *gin.Context) {
  rows, _ := s.db.Query(ctx, "SELECT * FROM alerts WHERE tenant_id = $1", tenantID)
  // 50 lines of scanning
  c.JSON(200, alerts)
}

// GOOD — handler delegates
func (s *Server) listAlerts(c *gin.Context) {
  alerts, err := s.services.Alerts.List(ctx, tenantID)
  if err != nil { respondError(c, err); return }
  c.JSON(200, alerts)
}
```

### Rule 5: Repository layer for all DB access

Services don't use SQL. Repositories do.

```go
// internal/repository/alert_repo.go
type AlertRepository interface {
  ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]Alert, error)
  Get(ctx context.Context, id uuid.UUID) (*Alert, error)
  Create(ctx context.Context, alert *Alert) error
  Update(ctx context.Context, alert *Alert) error
  Delete(ctx context.Context, id uuid.UUID) error
}
```

### Rule 6: Dependency injection via constructor

```go
// GOOD
func NewAlertService(repo repository.AlertRepository, notifier notifier.Notifier) *AlertService {
  return &AlertService{repo: repo, notifier: notifier}
}

// BAD — global state
var globalDB *pgxpool.Pool
```

### Rule 7: Interface-first dependencies

Define interfaces in the consumer, not the producer.

```go
// In internal/handler/notifier.go:
type Notifier interface {
  Send(ctx context.Context, channel string, msg Message) error
}

// In pkg/notifier/registry.go:
type SlackNotifier struct{...}
func (s *SlackNotifier) Send(...) error {...}
```

The handler depends on `Notifier` interface. Test with mock. Real implementation is wired at startup.

---

## File layout conventions

### Backend (Go)

```
cmd/<service>/
  main.go              # entry point
  routes.go            # route registration
  config.go            # config struct
  
internal/<domain>/
  <noun>.go            # one noun per file
  <noun>_test.go
  service.go           # business logic
  repository.go        # DB access
  types.go             # domain types
  
internal/handler/<domain>/
  <verb>_<noun>.go     # one handler per file (verb_noun)
  types.go
  
internal/middleware/
  recover.go
  requestid.go
  cors.go
  logging.go
  auth.go
  ratelimit.go
```

### Frontend (React + TypeScript)

```
web/src/
  pages/              # one file per page
    Servers.tsx
    ServerDetail.tsx
    Alerts.tsx
    
  components/         # one file per component
    Button.tsx
    Card.tsx
    VMActions.tsx     # domain-specific
    ServerTable.tsx
    
  lib/                # utilities
    api.ts
    format.ts
    constants.ts
    
  router/
    index.tsx
    
  state/
    auth.ts
    theme.ts
    
  styles/
    variables.css
    base.css
    components.css
```

### NO file > 500 lines. NO "misc" / "utils" / "helpers" files.

---

## Linting (enforced by CI)

`.golangci.yml`:

```yaml
linters:
  disable-all: true
  enable:
    - gofmt
    - goimports
    - gocyclo         # max cyclomatic 15
    - dupl            # no duplicate blocks > 100
    - gosec           # security
    - gocognit        # cognitive complexity 20
    - lll             # line length 120
    - wsl             # whitespace
    - funlen          # function > 50 lines
    - nestif          # if nesting > 5
    - errcheck
    - staticcheck
    - unused
    - ineffassign
    - govet
    - bodyclose
    - sqlclosecheck   # all DB rows closed
    - errorlint
    - contextcheck
```

`.eslintrc.json` (frontend):

```json
{
  "rules": {
    "max-len": ["error", { "code": 120 }],
    "max-lines": ["error", { "max": 500, "skipBlankLines": true, "skipComments": true }],
    "complexity": ["error", 15],
    "max-lines-per-function": ["error", { "max": 80, "skipBlankLines": true }],
    "no-duplicate-imports": "error"
  }
}
```

---

## CI Step (file-size check)

```bash
# File size check
find . -name "*.go" -not -path "*/vendor/*" -not -name "*_test.go" -not -name "*.pb.go" \
  -exec wc -l {} \; | awk '$1 > 500 { print FILENAME ": " $1 " lines" }' | grep . && exit 1 || echo "OK"

# Frontend
find web/src -name "*.ts" -o -name "*.tsx" -o -name "*.css" \
  -exec wc -l {} \; | awk '$1 > 500 && !/.bak/ && !/node_modules/ { print }' | grep . && exit 1 || echo "OK"
```

---

## Test coverage targets

| Layer | Target |
|-------|--------|
| `internal/auth/*` | 90% |
| `internal/handler/*` (all) | 80% |
| `internal/service/*` | 80% |
| `internal/repository/*` | 90% |
| `cmd/*` | 50% (only critical paths) |
| `web/src/*` | 70% (critical paths only) |

Coverage gates:
- < 60% total = PR blocked
- < 80% on auth/handler = PR blocked

---

## Documentation rules

Every exported function in Go has a godoc comment:
```go
// ListAlerts returns all alerts for the given tenant ordered by created_at DESC.
// Returns ErrNotFound if tenant doesn't exist.
func (s *AlertService) ListAlerts(ctx context.Context, tenantID uuid.UUID) ([]Alert, error) {
```

Every exported TS component has a JSDoc:
```typescript
/**
 * Displays a single server card with status indicator and quick actions.
 * @param server - Server entity from API
 * @param onAction - Callback for action buttons
 */
export function ServerCard({ server, onAction }: ServerCardProps) { ... }
```

---

## When in doubt — SPLIT

If a function is > 50 lines → split into helpers.
If a file is > 400 lines → split into multiple files.
If a package is doing 2 things → split into 2 packages.
If you can't find a clear name for a function → it's doing too much.

---

## Enforcement summary

| Check | Tool | When |
|-------|------|------|
| File size | `.file-size-check.sh` | CI |
| Lint | `golangci-lint`, `eslint` | CI |
| Tests | `go test`, `vitest` | CI |
| Coverage | `go test -cover`, `vitest run --coverage` | CI |
| Build | `go build`, `npm run build` | CI |
| Lint comments | `go vet`, `tsc --noEmit` | CI |

If ANY check fails, **PR is blocked**. No exceptions.

---

## Architecture decision records (ADRs)

For every non-trivial architectural choice, write an ADR in `docs/adr/`:

```
docs/adr/
  0001-record-architecture-decisions.md
  0002-use-postgres-as-primary-store.md
  0003-use-pgx-as-pg-driver.md
  0004-modular-package-layout.md
  0005-no-file-over-500-lines-rule.md
```

Format: Context → Decision → Consequences.
