# Tier 14 Phase 14.3 Plan

## Backend (additive — no new files unless needed)

### Snapshot CRUD (1 client file, 1 handler file)
- `internal/client/proxmox/snapshots.go` (NEW) — List/Create/Delete/RollbackSnapshots
- `internal/handler/proxmox_snapshots.go` (NEW) — 4 handlers
- `cmd/api-gateway/routes_protected.go` — 4 new routes

### Firewall CRUD (VM-level)
- Add 4 methods to `internal/client/proxmox/firewall.go`
- Add 4 handlers to `internal/handler/proxmox_firewall.go`
- Add 4 routes to routes_protected.go

### VNC proxy
- `internal/handler/proxmox_vnc.go` (NEW) — 2 handlers
  - `VNCProxy` — GET /vnc-ws upgrades WebSocket connection, proxies to Proxmox VNC websocket
  - `VNCTicket` — POST /vnc-ticket returns `{ticket: api_token, port: 8006, node}`
- Routes:
  - `POST /proxmox/hosts/:id/vnc-ticket`
  - `GET /proxmox/hosts/:id/vnc-ws`

### Migrate (already exists, just wire frontend)
- No backend changes

## Frontend (8 new/modified files + 2 new CSS files)

### Modify existing
- `ProxmoxDetailConsole.tsx` — replace placeholder with real noVNC
- `ProxmoxDetailSnapshots.tsx` — replace placeholder with CRUD + modal
- `ProxmoxDetailFirewall.tsx` — replace placeholder with CRUD + modal

### New components
- `ProxmoxDetailMigrateDialog.tsx` — node picker dialog
- `ProxmoxSnapshotCreateDialog.tsx` — snapshot creation modal
- `ProxmoxFirewallRuleDialog.tsx` — rule form modal (used for add + edit)

### New CSS
- `proxmox-console.css` — black canvas + toolbar
- `proxmox-detail-dialogs.css` — modal styles (shared with other dialogs)

### Update VmDetailPage
- Replace placeholder dialogs with real ones
- Wire migrate action to dialog
- Wire snapshot create action to dialog

## Dependencies

- Add npm package `novnc-next` (latest stable) — ~1 MB
- No new backend deps (use existing gorilla/g, gorilla/websocket already in repo)

## Order of implementation

1. Backend: snapshot CRUD client + handler + routes
2. Backend: firewall VM-level methods + handler + routes
3. Backend: VNC ticket + proxy handler + routes
4. Frontend: install noVNC package
5. Frontend: replace ProxmoxDetailConsole with real impl
6. Frontend: replace ProxmoxDetailSnapshots with CRUD + create modal
7. Frontend: replace ProxmoxDetailFirewall with CRUD + add modal
8. Frontend: new ProxmoxDetailMigrateDialog + wire to header
9. Frontend: new CSS files
10. Update VmDetailPage to use new components
11. tsc + go build
12. Deploy + verify
13. Commit

## Risk mitigations

- VNC over WSS with self-signed cert: noVNC accepts `ignoreTLS=true`
- noVNC bundle size: 1 MB acceptable (existing bundle is 756 KB)
- Slow snapshot operations: progress polling via task endpoint
- Firewall rule create format quirks: Proxmox requires form-encoded, not JSON

## Modularity discipline

- Every TSX < 400 LOC
- Every CSS < 400 LOC
- New dialogs as separate components
- Reuse existing modal/confirm patterns from Phase 14.2