# Tier 14 Phase 14.5 Requirements Checklist

## Wizard shell
- [ ] **W1** Route `/proxmox-vms/new` mounted
- [ ] **W2** Progress bar shows 1-4 / 4
- [ ] **W3** Next/Back buttons work
- [ ] **W4** State persists across steps
- [ ] **W5** Pre-fills hostId + node from query string

## Step 1: ISO
- [ ] **I1** ISO picker fetches ISOs from each `content=iso` storage
- [ ] **I2** Shows name + size
- [ ] **I3** Special "No media" option
- [ ] **I4** Empty state if no ISOs
- [ ] **I5** Selection persists to spec.iso

## Step 2: Hardware
- [ ] **H1** VMID input (number, >=100)
- [ ] **H2** Name input (required)
- [ ] **H3** Cores input (1-128)
- [ ] **H4** Memory input (MB)
- [ ] **H5** Disk input (GB)
- [ ] **H6** Storage dropdown (only images-capable storages)
- [ ] **H7** BIOS dropdown (seabios/ovmf)
- [ ] **H8** Machine dropdown (q35/i440fx)
- [ ] **H9** CPU type dropdown (host/kvm64/x86-64-v2-AES)
- [ ] **H10** Validation: VMID + name + storage required

## Step 3: Network
- [ ] **N1** Bridge input
- [ ] **N2** Network model dropdown (virtio/e1000/vmxnet3)
- [ ] **N3** VLAN tag input (optional)

## Step 4: Confirm
- [ ] **C1** Review all settings + ISO + storage + network
- [ ] **C2** "Create VM" button
- [ ] **C3** On success → redirect to detail page
- [ ] **C4** On error → stay with error banner

## Quality
- [ ] **Q1** Every TSX < 400 LOC
- [ ] **Q2** tsc --noEmit: 0 errors
- [ ] **Q3** npm run build: clean
- [ ] **Q4** Bundle deployed
- [ ] **Q5** Mobile-responsive
- [ ] **Q6** Dark theme + Datadog style

## Backend
- [ ] No new endpoints (uses Tier 1)
- [ ] Verifies POST .../qemu works (manual test via curl)

## Final deliverables
- [ ] Feature commit
- [ ] Archive speckit
- [ ] End-of-phase A-to-Z test plan