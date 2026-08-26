# Tier 14 Phase 14.3 Requirements Checklist

## Backend
- [ ] **B1** Snapshot list client method + handler + route
- [ ] **B2** Snapshot create client + handler + route
- [ ] **B3** Snapshot delete client + handler + route
- [ ] **B4** Snapshot rollback client + handler + route
- [ ] **B5** VM firewall rules list
- [ ] **B6** VM firewall rule create
- [ ] **B7** VM firewall rule update
- [ ] **B8** VM firewall rule delete
- [ ] **B9** VNC ticket endpoint (POST)
- [ ] **B10** VNC WS proxy endpoint (GET, upgrades to WSS)
- [ ] **B11** `go build` exit 0
- [ ] **B12** `go vet` exit 0
- [ ] **B13** Deployed + all new endpoints 401 unauth

## Frontend - Console
- [ ] **F1** novnc-next installed
- [ ] **F2** Console fetches VNC ticket from /vnc-ticket
- [ ] **F3** noVNC client connects to WSS proxy
- [ ] **F4** Renders VNC screen in canvas
- [ ] **F5** Reconnect button works
- [ ] **F6** Ctrl+Alt+Del button works
- [ ] **F7** Fullscreen button works
- [ ] **F8** Status indicator (Connected/Disconnected/Connecting/Error)
- [ ] **F9** Loading state
- [ ] **F10** Disconnects on tab switch (cleanup)

## Frontend - Snapshots
- [ ] **F11** List loads from API
- [ ] **F12** Create dialog opens
- [ ] **F13** Create dialog submits POST
- [ ] **F14** Created snapshot appears in list
- [ ] **F15** Delete confirmation
- [ ] **F16** Delete removes from list
- [ ] **F17** Rollback confirmation
- [ ] **F18** Rollback starts task
- [ ] **F19** Empty state
- [ ] **F20** Error states

## Frontend - Firewall
- [ ] **F21** List loads from API
- [ ] **F22** Add rule dialog opens
- [ ] **F23** Form: type/action/source/dest/proto/dport/comment
- [ ] **F24** Submit creates rule
- [ ] **F25** Edit rule prefills form
- [ ] **F26** Edit save updates rule
- [ ] **F27** Delete confirmation
- [ ] **F28** Delete removes rule
- [ ] **F29** Action badge color (ACCEPT=green, REJECT=amber, DROP=red)
- [ ] **F30** Empty state

## Frontend - Migrate
- [ ] **F31** Dialog opens with node dropdown
- [ ] **F32** Only OTHER nodes in dropdown (not current)
- [ ] **F33** Online/offline toggle
- [ ] **F34** Disabled when only one node
- [ ] **F35** Online disabled if VM not running
- [ ] **F36** Execute calls migrate API
- [ ] **F37** Success closes dialog + notification
- [ ] **F38** With-local-storage checkbox

## Frontend - Build quality
- [ ] **F39** Every TSX < 400 LOC
- [ ] **F40** Every CSS < 400 LOC
- [ ] **F41** tsc --noEmit: 0 errors
- [ ] **F42** npm run build: clean
- [ ] **F43** Bundle deployed

## User-acceptance (test plan will cover)
- Console connects + VNC visible
- Snapshot CRUD round-trip
- Firewall CRUD round-trip
- Migrate dialog executes

## Final deliverables
- [ ] Feature commit
- [ ] Archive speckit
- [ ] Journal entry
- [ ] Memory update
- [ ] End-of-phase A-to-Z test plan (40 cases)