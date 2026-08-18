# Proposal: Tier 4 — Server Admin (Cockpit parity)

## Why
When a user opens any server in StackWatch, they currently see only
basic metrics (cpu/mem/disk/load) and the file system browser (Tier 3).
Cockpit (the standard Linux admin UI) and TrueNAS-style UIs give 9
admin panels: Services, Storage, Network, Processes, Updates, Logs,
Users, Timers, Performance. Tier 4 adds all of these to StackWatch so
any server can be fully managed from one UI.

## What changes
- New agent binary: `cmd/agent/` (9 new files)
- New handler layer: `internal/handler/admin_*.go` (9 files)
- New routes: `cmd/api-gateway/routes.go` (30+ new registrations)
- New DB migration: `migrations/007_server_admin.sql` (3 tables)
- Frontend: 9 new pages in `web/src/pages/admin/`

## Impact
- Areas affected: api-gateway, agent, frontend, migrations
- Breaking changes: **none** — Tier 4 is purely additive
- Migration needed: **yes** — one new migration file
- New binary: yes (`cmd/agent/agent-linux`) — must be deployed to each server
- Server impact: agent binary gets 9 new endpoints, no extra load expected