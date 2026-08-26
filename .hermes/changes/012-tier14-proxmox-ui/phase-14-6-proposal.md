# Tier 14 Phase 14.6 — VM Wizard Enhancements: Multi-NIC + Cloud-Init + UEFI Disk

## Why this phase

The Phase 14.5 wizard creates basic VMs (1 NIC, no Cloud-Init, single SCSI disk). Phase 14.6 adds the three most-requested features for production VMs:

1. **Multi-NIC** — add up to 4 NICs, each with its own bridge/VLAN/model
2. **Cloud-Init drive** — attach a Cloud-Init CD-ROM for first-boot config (user/password/ssh keys/network)
3. **UEFI disk config** — EFI storage + TPM for Windows 11 / modern Linux installs

## Goals

1. VM wizard supports up to 4 NICs (add/remove)
2. Cloud-Init drive optional toggle on step 2
3. EFI disk + TPM toggle on step 2
4. Submit builds correct Proxmox API body for all three

## Backend

ZERO new endpoints — all use Tier 1 `POST /qemu`.

## UI changes

### Modified files
- `lib/vm-types.ts` — add `nics: Nic[]` and `cloudInit: boolean` and `efiDisk: boolean` and `tpm: boolean`
- `ProxmoxVmCreateStep2Hardware.tsx` — add Cloud-Init toggle, EFI disk toggle, TPM toggle
- `ProxmoxVmCreateStep3Network.tsx` — replace single NIC with NICS list + add/remove
- `ProxmoxVmCreateStep4Confirm.tsx` — show multi-NIC summary
- `ProxmoxVmCreatePage.tsx` — update submit() to build net0/net1/net2/net3 + cloud-init drive + EFI disk

### New components
- `ProxmoxNicRow.tsx` (~100 LOC) — single NIC editor row (bridge/VLAN/model)

## Modularity discipline

- Every TSX < 400 LOC
- Reuse ProxmoxNicRow across add/remove
- Keep all NICs in a single state array