# StackWatch

**One platform. Every feature. Replaces Datadog + Proxmox UI + TrueNAS UI + Portainer + Watchtower + Termix + Cockpit + Netdata + Grafana + Prometheus + Loki + Chrome Remote Desktop + Homarr.**

Open-source. Self-hostable. Strictly modular.

## Quick links

- [PROJECT.md](PROJECT.md) — single source of truth for the project
- [MASTER_BUILD_PLAN.md](MASTER_BUILD_PLAN.md) — 13 tiers, 26-35 sessions
- [MODULARITY_RULES.md](MODULARITY_RULES.md) — strict file/package rules
- [VERIFICATION_CONTRACT.md](VERIFICATION_CONTRACT.md) — what "done" means
- [RESEARCH.md](RESEARCH.md) — feature inventory of all 12 tools
- [docs/](docs/) — user-facing documentation
- [tests/](tests/) — verifier scripts + outputs

## Architecture

```
go-services (1 file per feature, <500 lines each)
   ↓
internal/* (kernel, db, auth, telemetry, middleware)
   ↓
pkg/* (notifier, parser, ml, k8s)
   ↓
plugins (in-tree integrations)
```

**Strict rule:** No file > 500 lines. No god packages. No "misc" files. CI blocks PRs that violate this.

## Status

**Phase:** Tier 0 ✅ complete → Tier 1 next
**Current session:** 2026-08-18 (Tier 0 live verification: 82/82 PASS)

## License

AGPL-3.0
