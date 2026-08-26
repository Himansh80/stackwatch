# Tier 14 Phase 14.5 Plan

## Approach

Mirror Phase 14.4 (LXC wizard). Reuse the wizard shell + ISO picker pattern, swap step content for VM specifics.

## Backend
ZERO — all endpoints already exist.

## Frontend

### Step 1: ISO picker
- `web/src/components/proxmox/ProxmoxIsoPicker.tsx` (~220 LOC)
  - Reuses pattern from ProxmoxLxcTemplatePicker
  - Filters to ISO files in `content=iso` storages
  - Special "No media" card (for VMs that boot from disk)

### Step 2: Hardware
- `web/src/components/proxmox/ProxmoxVmCreateStep2Hardware.tsx` (~250 LOC)
  - Form: VMID, name, cores, memory, disk, storage, BIOS, machine, CPU type
  - Storage dropdown: only `images` or content storages

### Step 3: Network
- `web/src/components/proxmox/ProxmoxVmCreateStep3Network.tsx` (~130 LOC)
  - Bridge + network model + VLAN

### Step 4: Confirm
- `web/src/components/proxmox/ProxmoxVmCreateStep4Confirm.tsx` (~80 LOC)
  - Review + submit

### Orchestrator
- `web/src/components/proxmox/ProxmoxVmCreatePage.tsx` (~350 LOC)
  - 4-step flow with progress bar
  - Mirrors ProxmoxLxcCreatePage but for VMs

### Shared types
- `web/src/components/proxmox/lib/vm-types.ts` (~80 LOC)
  - VmSpec interface

### Styling
- Reuse `proxmox-wizard.css` + `proxmox-detail-dialogs.css`

### Routes
- `/proxmox-vms/new` mounted in App.tsx

### Sidebar
- Add "Create VM" link under Proxmox sub-nav

## Order

1. VmSpec type
2. ISO picker
3. Step 1 wrapper
4. Step 2 hardware
5. Step 3 network
6. Step 4 confirm
7. Orchestrator
8. Wire route
9. Sidebar nav item
11. tsc + build + deploy
12. Commit

## Modularity discipline

- Every TSX < 400 LOC
- Reuse CSS from proxmox-wizard.css

## Risks

- Proxmox qemu POST endpoint takes many fields — need to map wizard spec → Proxmox API correctly
- ISO list may be empty (no ISOs uploaded) — show empty state
- Storage detection may need iteration (storages with content=iso OR content=images)