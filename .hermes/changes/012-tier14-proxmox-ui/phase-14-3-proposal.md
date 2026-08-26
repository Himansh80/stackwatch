# Tier 14 Phase 14.3 — VM Detail Completion: Console + Snapshot/Firewall CRUD + Migrate

## Why this phase

Phase 14.2 shipped the VM detail page with placeholders for:
- Console (no real impl)
- Snapshots (list view only)
- Firewall (list view only)
- Migrate (toast only)

Phase 14.3 fills in the real implementations + adds CRUD operations.

## Goals

1. **Real noVNC console** — browser-based graphical console via noVNC client + WebSocket bridge
2. **Snapshot CRUD** — list, create (with VM state option), delete, rollback
3. **Firewall CRUD** — list VM-level rules, add rule, edit, delete
4. **Migrate dialog** — node picker + online/offline toggle + execute

## Why this scope

These are the four features Proxmox web UI users hit most often after VM list/detail:
- Open console (debug boot issues, install OS)
- Snapshot before risky change (rollback safety)
- Firewall for security isolation
- Migrate for HA / load balancing

## Backend dependencies (NEW endpoints)

### 1. Snapshot CRUD — 5 endpoints
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot` — list
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot` — create
- `DELETE /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/snapshot?snapname=X` — delete
- `POST .../snapshot/:snapname/rollback` — rollback (POST because destructive)

### 2. Firewall CRUD (VM-level) — 4 endpoints
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/firewall/rules` — list VM rules
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/firewall/rules` — add
- `PUT /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/firewall/rules/:pos` — edit
- `DELETE /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/firewall/rules/:pos` — delete

### 3. Migrate dialog — already exists
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/:kind/:vmid/migrate` — already wired (Tier 1)

### 4. Console — browser-side only (noVNC connects directly to Proxmox)
Proxmox VNC WebSocket: `wss://host:8006/api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket`
- We need a per-user auth ticket for VNC. Proxmox supports `POST /access/ticket` to create one.
- New backend endpoint: `POST /api/v1/proxmox/hosts/:id/vnc-ticket` returns `{ticket, port, node}`
- Frontend uses noVNC client (npm package `novnc-next`) connecting via WSS

## Out of scope (Phase 14.4+)

- Console recording
- Snapshot scheduling
- Firewall aliases + IPSets (already exists at node level; VM-level deferred)
- Bulk operations

## UI components (planned, all ≤400 LOC)

### Frontend — 8 new + 4 modified

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxDetailConsole.tsx` | +180 → 200 | Replace placeholder with real noVNC client + ticket fetch |
| `ProxmoxDetailSnapshots.tsx` | +120 → 200 | Replace placeholder with CRUD + create modal |
| `ProxmoxDetailFirewall.tsx` | +200 → 280 | Replace placeholder with CRUD + add modal |
| `ProxmoxDetailMigrateDialog.tsx` | 150 | NEW: node picker + online toggle + execute |
| `ProxmoxSnapshotCreateDialog.tsx` | 100 | NEW: snapshot name + VM state toggle |
| `ProxmoxFirewallRuleDialog.tsx` | 180 | NEW: rule form (action/source/dest/proto/dport/comment) |
| `proxmox-console.css` | 200 | Console-specific styles |
| `proxmox-detail-dialogs.css` | 250 | Dialog styles |

### Backend — 11 new client methods + 9 new handlers + 9 new routes
- All in existing files (no new files; just additions)

## Console architecture

```
Browser (noVNC)
  ↕ WebSocket (wss://stackwatch.smarthomelab.fun/api/v1/proxmox/hosts/{id}/vnc-ws?hostId=X&node=Y&vmid=Z)
API gateway (proxy /vnc-ws)
  ↕ VNC WebSocket (wss://proxmox-host:8006/api2/json/nodes/{node}/qemu/{vmid}/vncwebsocket)
Proxmox
```

The browser needs a VNC password (ticket). Proxmox supports `POST /access/ticket`:
- Body: `username@realm + password` OR `PVEAPIToken=USER@REALM!TOKENID=UUID`
- Response: `{ticket: "...", CSRFPreventionToken: "...", username: "..."}`
- The ticket is used as the VNC password for the websocket connection

If we use the **API token** (PVEAPIToken format) the "password" field becomes the literal `PVEAPIToken=...` value. Proxmox accepts this as a ticket for VNC.

Simpler approach: **proxy the VNC through the api-gateway** so the browser only needs to talk to our backend (which has the the API token).

## Risks

1. **noVNC package size** — `novnc-next` is ~1 MB. Acceptable.
2. **VNC over WSS** — requires valid TLS on Proxmox host. `.107` has self-signed cert. We can pass `ignoreTLS=true` to noVNC.
3. **Firewall rule create** — Proxmox returns 200 with empty UPID; success.
4. **Snapshot with RAM** — can be slow on big VMs (several minutes); need progress indicator.

## Acceptance criteria

1. Click "Console" tab → noVNC client loads → VNC screen visible
2. Type + click + keystrokes work in VNC
3. Reconnect button works when VNC disconnects
4. Click "Snapshots" tab → list loads from API
5. Click "+ Create snapshot" → modal opens → submit creates snapshot → list refreshes
6. Click delete on a snapshot → confirm modal → snapshot removed
7. Click rollback on a snapshot → confirm modal → rollback task starts
8. Click "Firewall" tab → rules load from API
9. Click "+ Add rule" → form modal → submit creates rule → list refreshes
10. Edit existing rule → form prefilled → save updates rule
11. Delete rule → confirm → rule removed
12. Click "Migrate" → dialog opens with node dropdown (other nodes in cluster) + online toggle
13. Submit migrate → task starts → notification appears
14. Empty states for all (no snapshots, no rules)
15. Mobile-responsive (dialogs become sheets on <768px)

## Test plan (delivered at end of phase)

40-case A-to-Z test plan for all 4 features (console + snapshots + firewall + migrate).

## Modularity discipline

- Every TSX < 400 LOC
- Every CSS < 400 LOC
- Backend methods added to existing files (no new files unless necessary)
- Each new component has single responsibility