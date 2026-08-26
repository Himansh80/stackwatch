# Tier 14 Phase 14.5 — VM Create Wizard

## Why this phase

The existing `+ Create VM` button in `ProxmoxWorkspace.tsx` opens a basic inline form. Phase 14.5 replaces it with a full multi-step wizard mirroring the LXC wizard from Phase 14.4, with proper ISO selection, hardware config, network, and confirmation.

## Goals

1. **VM wizard** at `/proxmox-vms/new` — multi-step form
2. **ISO picker** (step-1) — lists ISOs from /storage/:storage/content?content=iso
3. **Hardware config** (step-2) — cores, RAM, disk size, BIOS, machine type
4. **Network** (step-3) — bridge + VLAN
5. **Confirm** (step-4) — review + submit
6. Wire existing "+ Create VM" in workspace to point at wizard (optional)

## Backend

ZERO new endpoints. Tier 1 already provides:
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu` — create VM
- `GET .../storage` — list storages
- `GET .../storage/:storage/content` — list ISOs/templates/backups
- `POST /api/v1/proxmox/hosts/:id/test` — verify host connectivity

## UI components (planned, all ≤400 LOC)

| File | LOC target | Purpose |
|------|------------:|---------|
| `ProxmoxVmCreatePage.tsx` | 350 | Wizard orchestrator (3 steps for VMs: ISO → Hardware → Network → Confirm = 4) |
| `ProxmoxIsoPicker.tsx` | 220 | Reusable ISO grid (mirror of TemplatePicker) |
| `ProxmoxVmCreateStep1Iso.tsx` | 30 | Wizard step 1 (ISO) |
| `ProxmoxVmCreateStep2Hardware.tsx` | 250 | Cores, RAM, disk, BIOS, machine, SCSI, network model |
| `ProxmoxVmCreateStep3Network.tsx` | 130 | Bridge + VLAN + network model |
| `ProxmoxVmCreateStep4Confirm.tsx` | 80 | Review + submit |
| `proxmox-vm-wizard.css` | 200 | New wizard styles (or reuse proxmox-wizard.css) |

## Routes

- `/proxmox-vms/new` — create wizard

## Acceptance criteria

1. Wizard opens at `/proxmox-vms/new`
2. Step 1 shows ISOs from each storage's `iso` content
3. Step 2 lets user set cores / RAM / disk / BIOS / machine
4. Step 3 lets user set bridge + VLAN + network model
5. Step 4 reviews + submits
6. After create: redirect to `/proxmox-vms/:hostId/:node/:vmid`
7. Form state persists across step navigation
8. Empty states (no ISOs available)
9. Mobile-responsive (steps collapse to dropdown)

## Modularity discipline

- Every TSX < 400 LOC
- Reuse CSS from proxmox-wizard.css where possible
- Reuse LxcSpec pattern for vm-types.ts