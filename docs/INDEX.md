# StackWatch Documentation

**Last updated:** 2026-08-17

User-facing documentation. Shipped with the platform.

---

## Quick links

### For first-time users

- [INSTALL.md](INSTALL.md) — fresh install (dev + production)
- [USER-GUIDE.md](USER-GUIDE.md) — how to use every feature
- [FAQ.md](FAQ.md) — frequently asked questions
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) — common issues + fixes

### Reference

- [ARCHITECTURE.md](ARCHITECTURE.md) — how it works (modular diagram)
- [API.md](API.md) — REST API reference
- [TESTING.md](TESTING.md) — what's tested, how to test yourself
- [FEATURES.md](FEATURES.md) — every feature listed (Tier 0-13)
- [COMPARISON.md](COMPARISON.md) — vs Datadog / Proxmox / TrueNAS / Portainer / etc.

### For operators

- [SECURITY.md](SECURITY.md) — security model + runbook
- [SELF-HOST-GUIDE.md](SELF-HOST-GUIDE.md) — how to self-host
- [PRICING.md](PRICING.md) — pricing tiers

### For business

- [SELL-IT.md](SELL-IT.md) — reseller guide
- [LICENSE](LICENSE) — license model (AGPL-3.0 + commercial)

---

## Status

**Live (verified end-to-end):** Tier 0, Tier 1, Tier 2, Tier 3

| Tier | Description | Live |
|------|-------------|------|
| 0 | Auth, Tenants, Users, API Keys | ✅ |
| 1 | Proxmox full replacement | ✅ |
| 2 | TrueNAS SCALE 25.10 sidecar | ✅ |
| 3 | Remote Access (Termix-style) | ✅ |
| 4 | Server Admin (Cockpit parity) | ⏳ |
| 5 | Containers (Portainer) | ⏳ |
| 6 | Monitoring Depth (Netdata/Prom/Grafana/Loki) | ⏳ |
| 7 | Datadog full platform | ⏳ |
| 8 | Intelligence & Alerting | 🟡 partial |
| 9 | Security & Enterprise (SSO/RLS/SOC 2) | ⏳ |
| 10 | Homelab Dashboard (Homarr-style) | ⏳ |
| 11 | Platform & Commerce (Stripe/Razorpay) | ⏳ |
| 12 | Docs & GTM (this document) | 🟡 in progress |
| 13 | Mobile (Flutter) | ⏳ |

See [FEATURES.md](FEATURES.md) for what's actually shipped per tier.

---

## Verifier scripts

End-to-end live verification (run from your dev workstation against a
freshly deployed binary):

| Script | Tests | Tier |
|--------|-------|------|
| `hermes-verify-tier0-complete-2026-08-17.py` | 17 essential Tier 0 endpoint checks | 0 |
| `hermes-verify-tier0-1-2-live-2026-08-17.py` | T0+T1+T2 cross-check (27 checks) | 0-2 |

Both live in `C:\Users\himan\AppData\Local\Temp\` on the maintainer's
workstation.

---

## Repository layout

```
stackwatch/
  cmd/
    api-gateway/         # main HTTP gateway (port 8080)
    truenas-connector/   # TrueNAS JSON-RPC over WebSocket sidecar (port 8088)
    web-terminal/        # xterm.js PTY bridge (port 8085)
    agent/               # Tier 4 — installed on monitored hosts
  internal/
    auth/                # JWT + bcrypt + claims context
    client/
      proxmox/          # Proxmox API client (auto-prefixes PVEAPIToken=)
      truenas/          # TrueNAS WS client (login + per-call JSON-RPC)
    handler/             # 45 files, <500 lines each (enforced by CI)
    service/             # service layer (currently TrueNAS only)
    kernel/              # Page, Pagination, Error types
    db/                  # pgx pool + helpers
  migrations/            # 6 schema migrations
  scripts/
    file-size-check.sh   # CI gate
    probe-truenas/       # dev-only probes
  docs/                  # this directory
  .github/workflows/     # backend-ci + frontend-ci
  .golangci.yml          # lint config (gosec, staticcheck, ineffassign, ...)
```

---

## Verification contract

Every PR must pass:

1. `go build ./...` (zero output)
2. `go vet ./...` (zero output)
3. `gofmt -l .` (zero output)
4. `golangci-lint run ./...` (in CI, with the rule set)
5. `bash scripts/file-size-check.sh` (no file > 500 lines)
6. `go test ./internal/... -count=1 -race` (all tests pass)

---

## License

[AGPL-3.0](../LICENSE) for self-hosted use; commercial license available
via `sales@stackwatch.io`. See [LICENSE](LICENSE).
