# Tier 14 Phase 14.2 Tasks: VM Detail Page

## Backend

- [ ] B1. Add `GetVMNetworkInterfaces(ctx, node, vmid)` to internal/client/proxmox/proxmox.go
- [ ] B2. Create `internal/handler/proxmox_vm_agent.go` with `GetVMNetworkInterfaces` handler
- [ ] B3. Map Proxmox 500 ("agent not running") → HTTP 200 `{interfaces: null, agent_running: false}`
- [ ] B4. Register route in `cmd/api-gateway/routes_protected.go`
- [ ] B5. go build clean
- [ ] B6. go vet clean
- [ ] B7. Deploy to .115
- [ ] B8. Live curl: route returns 200 without auth = 401, with auth = 200

## Frontend

- [ ] F1. Create `web/src/components/proxmox/ProxmoxVmDetailPage.tsx`
- [ ] F2. Create `web/src/components/proxmox/ProxmoxDetailHeader.tsx`
- [ ] F3. Create `web/src/components/proxmox/ProxmoxDetailActions.tsx`
- [ ] F4. Create `web/src/components/proxmox/ProxmoxDetailTabs.tsx`
- [ ] F5. Create `web/src/components/proxmox/ProxmoxDetailSummary.tsx`
- [ ] F6. Create `web/src/components/proxmox/ProxmoxDetailHardware.tsx`
- [ ] F7. Create `web/src/components/proxmox/ProxmoxDetailNetwork.tsx`
- [ ] F8. Create `web/src/components/proxmox/ProxmoxDetailConsole.tsx`
- [ ] F9. Create `web/src/components/proxmox/ProxmoxDetailSnapshots.tsx`
- [ ] F10. Create `web/src/components/proxmox/ProxmoxDetailFirewall.tsx`
- [ ] F11. Wire route `/proxmox-vms/:hostId/:node/:vmid` in App.tsx
- [ ] F12. Update `ProxmoxVmTable.tsx` row-click handler to navigate
- [ ] F13. Create `web/src/styles/proxmox-detail.css`
- [ ] F14. Import CSS in main.tsx
- [ ] F15. tsc clean
- [ ] F16. npm run build clean
- [ ] F17. Deploy to .115
- [ ] F18. Live verify all 8 new files in bundle

## Final commit

- [ ] `feat(tier14): Phase 14.2 Proxmox VM detail page`
- [ ] Archive speckit artifacts
- [ ] Update journal
- [ ] Update memory
- [ ] **Deliver end-of-phase test plan (A-to-Z)**