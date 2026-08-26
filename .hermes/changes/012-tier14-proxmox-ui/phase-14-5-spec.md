# Tier 14 Phase 14.5 — Specification: VM Create Wizard

## 1. URL pattern

- `/proxmox-vms/new` — wizard
- `?hostId=X&node=Y` — pre-select host/node

## 2. Step layout (4 steps with progress bar)

### Step 1: ISO / Media
- Host selector (pre-fill from URL query)
- Node selector (pre-fill from URL query)
- ISO picker grid:
  - Fetches ISOs from each storage's `iso` content
  - Shows: name + size
  - Optionally: "No media" option (for VMs that boot from existing disk)

### Step 2: Hardware
- VMID (next free >= 100)
- Name (required)
- Cores (1-128)
- Memory MB (128-524288)
- Disk GB (1-16384)
- Storage (root disk storage)
- BIOS: `seabios` (default) | `ovmf` (UEFI)
- Machine: `q35` (default) | `i440fx`
- CPU type: `host` (default) | `kvm64` | `x86-64-v2-AES` | ...

### Step 3: Network
- Bridge: `vmbr0` (default)
- Network model: `virtio` (default) | `e1000` | `vmxnet3`
- VLAN tag (optional)
- "Add another NIC" toggle (Phase 14.6 — for now: single NIC)

### Step 4: Confirm
- Review all settings + ISO + storage + network
- "Create VM" button → POST /qemu
- On success → redirect to detail page
- On error → stay on step 4 with error banner

## 3. State management

- Wizard state in `ProxmoxVmCreatePage` only (no URL persistence across steps)
- ISO + Hardware + Network fields collected
- "Back" buttons on steps 2-4; "Next" on 1-3
- Final submit on step 4

## 4. Backend API

All endpoints already exist (Tier 1):
- `GET /api/v1/proxmox/hosts/:id/nodes/:node/storage` — storages
- `GET .../storage/:storage/content` — list ISOs
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu` — create

## 5. Acceptance criteria

1. Wizard opens at `/proxmox-vms/new`
2. Pre-fills host/node from query string
3. Step 1 lists ISOs from all `iso`-supporting storages
4. Step 2 validates required fields (VMID, name, storage)
5. Step 3 lets user pick network model + VLAN
6. Step 4 reviews + submits
8. Empty states for each step (no host, no ISOs, no storages)
9. Mobile-responsive
10. After create: redirect to detail page

## 6. Out of scope

- Multi-NIC (Phase 14.6+)
- TPM/UEFI disk config (Phase 14.6+)
- Cloud-init drive (Phase 14.6+)