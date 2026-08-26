# Tier 14 Phase 14.2 Requirements Checklist

## Backend
- [ ] **B1** `GetVMNetworkInterfaces(ctx, node, vmid)` method added to client
- [ ] **B2** HTTP handler in proxmox_vm_agent.go
- [ ] **B3** Proxmox 500 → 200 `{interfaces: null, agent_running: false}` mapping
- [ ] **B4** Route registered
- [ ] **B5** `go build ./cmd/api-gateway` exit 0
- [ ] **B6** `go vet` exit 0
- [ ] **B7** Binary deployed to .115
- [ ] **B8** Live curl: 401 unauth, 200 auth

## Frontend - Page Shell
- [ ] **F1** Route `/proxmox-vms/:hostId/:node/:vmid` mounted in App.tsx
- [ ] **F2** useParams parses hostId/node/vmid
- [ ] **F3** useLocation parses hash for tab
- [ ] **F4** Default tab = #summary
- [ ] **F5** Loading state (skeleton)
- [ ] **F6** Error state (red banner)

## Frontend - Header
- [ ] **F7** Breadcrumb: Proxmox > {node} > VM {vmid}
- [ ] **F8** VM name (large)
- [ ] **F9** Status pill (color matches running/stopped/paused)
- [ ] **F10** ID/Type/Node/Uptime row
- [ ] **F11** Auto-refresh every 5s

## Frontend - Actions
- [ ] **F12** Start button + confirm + action
- [ ] **F13** Shutdown button + confirm + action
- [ ] **F14** Reboot button + confirm + action
- [ ] **F15** Migrate button → "coming in v2.3" toast (deferred)
- [ ] **F16** Delete button → type-DELETE confirm → redirects to /proxmox-vms
- [ ] **F17** Disabled state for irrelevant actions (e.g. Start on running VM)

## Frontend - Tabs Nav
- [ ] **F18** 6 tab buttons (Summary/Hardware/Network/Console/Snapshots/Firewall)
- [ ] **F19** Active tab styling
- [ ] **F20** URL hash updates on click
- [ ] **F21** Deep-linkable (refresh on hash works)
- [ ] **F22** Mobile: tabs collapse to dropdown on <768px

## Frontend - Summary tab
- [ ] **F23** Status + Uptime
- [ ] **F24** CPU % + cores
- [ ] **F25** Memory used / total
- [ ] **F26** Disk used / total
- [ ] **F27** Tags (chips)
- [ ] **F28** Pool
- [ ] **F29** HA state (if configured)

## Frontend - Hardware tab
- [ ] **F30** Cores (sockets × cores)
- [ ] **F31** Memory MB
- [ ] **F32** Disk size GB
- [ ] **F33** BIOS type (seabios/ovmf)
- [ ] **F34** Machine type (q35/i440fx)
- [ ] **F35** CPU type
- [ ] **F36** SCSI controller
- [ ] **F37** Network model
- [ ] **F38** Boot order

## Frontend - Network tab
- [ ] **F39** Table: Interface / MAC / Bridge / IP / MTU / Model
- [ ] **F40** QEMU agent info card (running / not running)
- [ ] **F41** Empty state when no interfaces

## Frontend - Console tab
- [ ] **F42** Placeholder card "Console opens in v2.3"
- [ ] **F43** Disabled "Open noVNC" button

## Frontend - Snapshots tab
- [ ] **F44** List: Name / Date / VM State / Size
- [ ] **F45** Empty state if no snapshots
- [ ] **F46** "Create snapshot" button (disabled with tooltip "deferred")

## Frontend - Firewall tab
- [ ] **F47** List: Pos / Action / Source / Dest / Proto / Dport / Comment
- [ ] **F48** Empty state
- [ ] **F49** "+ Add rule" (disabled, "deferred")

## Frontend - Row click navigation
- [ ] **F50** Click row in /proxmox-vms → navigate to detail page

## Frontend - Build quality
- [ ] **F51** Every TSX < 400 LOC
- [ ] **F52** CSS < 400 LOC
- [ ] **F53** tsc --noEmit: 0 errors
- [ ] **F54** npm run build: clean
- [ ] **F55** Bundle deployed to .115

## User-acceptance (test plan)
- [ ] **U1** Open detail page from VM row click
- [ ] **U2** Navigate tabs via hash
- [ ] **U3** Start/stop/reboot work
- [ ] **U4** Delete with confirmation
- [ ] **U5** Mobile responsive
- [ ] **U6** All 6 tabs render
- [ ] **U7** Empty/error states

## Final deliverables
- [ ] **D1** Feature commit
- [ ] **D2** Archive speckit artifacts
- [ ] **D3** Journal entry
- [ ] **D4** Memory update
- [ ] **D5** End-of-phase A-to-Z test plan delivered to user