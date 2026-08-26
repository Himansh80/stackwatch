# Tier 14 Phase 14.2 Plan

## Backend (1 new endpoint)

1. **`internal/handler/proxmox_vm_agent.go`** — VM QEMU agent endpoints
   - `GetVMNetworkInterfaces` — `GET /proxmox/hosts/:id/nodes/:node/qemu/:vmid/agent/network-get-interfaces`
   - Maps Proxmox 500 (agent not running) → HTTP 200 with `{interfaces: null, agent_running: false}`

2. **`internal/client/proxmox/proxmox.go`** — add `GetVMNetworkInterfaces()` method
   - Calls `/nodes/{node}/qemu/{vmid}/agent/network-get-interfaces`

3. **`cmd/api-gateway/routes_protected.go`** — register the route
   - `protected.GET("/proxmox/hosts/:id/nodes/:node/qemu/:vmid/agent/network-get-interfaces", proxmoxH.GetVMNetworkInterfaces)`

## Frontend (8 components + CSS)

### Step 1: Page shell + routing
- `web/src/components/proxmox/ProxmoxVmDetailPage.tsx` (~200 LOC)
- Reads `:hostId`, `:node`, `:vmid` from useParams
- Reads `tab` from `useLocation().hash`
- Mounts in `App.tsx` at `/proxmox-vms/:hostId/:node/:vmid`

### Step 2: Header + actions
- `ProxmoxDetailHeader.tsx` (~150 LOC) — breadcrumb + name + status pill + action buttons
- `ProxmoxDetailActions.tsx` (~150 LOC) — Start/Stop/Reboot/Migrate/Delete buttons with confirm

### Step 3: Tab content
- `ProxmoxDetailSummary.tsx` (~250 LOC) — Summary tab content
- `ProxmoxDetailHardware.tsx` (~200 LOC) — Hardware tab content
- `ProxmoxDetailNetwork.tsx` (~200 LOC) — Network tab content
- `ProxmoxDetailConsole.tsx` (~100 LOC) — Console placeholder
- `ProxmoxDetailSnapshots.tsx` (~150 LOC) — Snapshots list
- `ProxmoxDetailFirewall.tsx` (~150 LOC) — Firewall list

### Step 4: Tab router component
- `ProxmoxDetailTabs.tsx` (~80 LOC) — tab buttons + hash navigation

### Step 5: Styling
- `web/src/styles/proxmox-detail.css` (~400 LOC) — all detail styles
- Reuse tokens from proxmox.css

### Step 6: Wire to existing list page
- Update `ProxmoxVmTable.tsx` — `onRowClick` navigates to `/proxmox-vms/:hostId/:node/:vmid`

### Step 7: Build + verify
- tsc clean
- npm run build
- Deploy to .115
- curl-verify all new endpoints return 200 with auth

### Step 8: Final commit
- `feat(tier14): Phase 14.2 Proxmox VM detail page`

## Order of implementation

1. Backend (1 endpoint + client method + route)
2. Page shell + route
3. Header + actions
4. Tabs nav
5. Summary tab
6. Hardware tab
7. Network tab (uses new endpoint)
8. Console placeholder
9. Snapshots list
10. Firewall list
11. Wire row click → detail nav
12. CSS
13. Build + deploy + verify
14. Final commit + archive

## Modular discipline

- Every TSX < 400 LOC
- Every CSS < 400 LOC
- Each tab is its own file
- No business logic in page shell

## Risks

- QEMU agent not running on guest → graceful fallback (Phase 14.2 spec already handles)
- /qemu/:vmid may return large config → render only key fields
- URL hash navigation needs React Router `useNavigate` carefully