# Tier 14 Phase 14.3 Tasks: Console + Snapshot/Firewall CRUD + Migrate

## Backend
- [ ] B1. Create `internal/client/proxmox/snapshots.go` (List/Create/Delete/Rollback)
- [ ] B2. Create `internal/handler/proxmox_snapshots.go` (4 handlers)
- [ ] B3. Register 4 snapshot routes
- [ ] B4. Add VM-level methods to `internal/client/proxmox/firewall.go` (ListVM/CreateVM/UpdateVM/DeleteVM)
- [ ] B5. Add 4 VM-firewall handlers
- [ ] B6. Register 4 VM-firewall routes
- [ ] B7. Create `internal/handler/proxmox_vnc.go` (VNCProxy + VNCTicket)
- [ ] B8. Register VNC ticket + VNC WS proxy routes
- [ ] B9. `go build` clean
- [ ] B10. `go vet` clean
- [ ] B11. Deploy to .115
- [ ] B12. Live curl: all new endpoints return 401 unauth

## Frontend
- [ ] F1. Install `novnc-next` npm package
- [ ] F2. Replace `ProxmoxDetailConsole.tsx` with real noVNC impl
- [ ] F3. Add ProxmoxSnapshotCreateDialog component
- [ ] F4. Replace `ProxmoxDetailSnapshots.tsx` with CRUD + dialog
- [ ] F5. Add ProxmoxFirewallRuleDialog component (add + edit)
- [ ] F6. Replace `ProxmoxDetailFirewall.tsx` with CRUD + dialog
- [ ] F7. Add ProxmoxDetailMigrateDialog component
- [ ] F8. Update ProxmoxDetailHeader to wire migrate action
- [ ] F9. Update ProxmoxVmDetailPage to pass handlers
- [ ] F10. Create `proxmox-console.css`
- [ ] F11. Create `proxmox-detail-dialogs.css`
- [ ] F12. Import new CSS files
- [ ] F13. tsc clean
- [ ] F14. npm run build clean
- [ ] F15. Deploy to .115
- [ ] F16. Live verify all new files in bundle

## Final deliverables
- [ ] D1. Feature commit
- [ ] D2. Archive speckit artifacts
- [ ] D3. Journal entry
- [ ] D4. Memory update
- [ ] D5. End-of-phase A-to-Z test plan (40 cases)

## User-acceptance (test plan will include)
- [ ] Console connects + renders VNC screen
- [ ] Snapshot CRUD round-trip works
- [ ] Firewall rule CRUD round-trip works
- [ ] Migrate dialog opens + executes