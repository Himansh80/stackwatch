# Tier 14 — Phase 14.1 Tasks: VM List Page

## Frontend

- [ ] 1. Create `web/src/pages/proxmox/ProxmoxPage.tsx` (page shell)
- [ ] 2. Create `web/src/components/proxmox/ProxmoxHostSelector.tsx`
- [ ] 3. Create `web/src/components/proxmox/ProxmoxKpiStrip.tsx`
- [ ] 4. Create `web/src/components/proxmox/ProxmoxFilterBar.tsx`
- [ ] 5. Create `web/src/components/proxmox/ProxmoxVmTable.tsx`
- [ ] 6. Create `web/src/components/proxmox/ProxmoxVmRow.tsx`
- [ ] 7. Create `web/src/components/proxmox/ProxmoxVmActions.tsx`
- [ ] 8. Create `web/src/components/proxmox/ProxmoxEmptyState.tsx`
- [ ] 9. Wire route `/app.html/proxmox` in router
- [ ] 10. Add `web/src/styles/proxmox.css` if needed
- [ ] 11. Verify `framer-motion` installed (install if not)
- [ ] 12. Add framer-motion animations
- [ ] 13. Mobile responsive (cards on <768px)

## Backend

- [ ] (no new endpoints — using Tier 1)

## Tests

- [ ] Live curl `GET /api/v1/proxmox/hosts` returns host
- [ ] Live curl `GET /api/v1/proxmox/hosts/<id>/qemu` returns VMs
- [ ] Manual UI test: filter, search, start, stop
- [ ] Mobile test: 375px viewport

## Acceptance criteria

- [ ] Page renders with KPI strip + table
- [ ] Filters work (search, status chips)
- [ ] VM actions work (start/stop with confirm)
- [ ] Empty + error states render correctly
- [ ] Auto-refresh every 5s
- [ ] Animations smooth
- [ ] Mobile-responsive
- [ ] No new TypeScript errors
- [ ] No console errors in browser

## Final commit

- [ ] `feat(tier14): Phase 14.1 Proxmox VM list page`
- [ ] Archive speckit artifacts
- [ ] Update journal
- [ ] Update memory
