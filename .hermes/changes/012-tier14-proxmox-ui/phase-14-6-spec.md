# Tier 14 Phase 14.6 — Specification: Multi-NIC + Cloud-Init + EFI

## 1. URL pattern
Same `/proxmox-vms/new` route. State in orchestrator.

## 2. Step 2: Hardware (extended)

Add three toggles to step 2:
- **Cloud-Init drive** (checkbox) — adds `ide2: none,media=cdrom` later + `cicustom: user=...` for Cloud-Init
- **EFI disk** (checkbox) — adds `efidisk0: <storage>:1,efitype=4m,pre-enrolled-keys=1` if UEFI
- **TPM state** (checkbox, only if EFI) — adds `tpmstate0: <storage>:1,version=v2.0`

## 3. Step 3: Network (rewritten)

- NICS list (1-4 entries)
- Each NIC has: bridge, VLAN, model
- "+ Add NIC" button (disabled when 4)
- Each row: bridge input + model select + VLAN input + Remove button
- "No NICs" empty state (default: 1 NIC)

## 4. Step 4: Confirm (extended)

Show:
- EFI disk / TPM toggles (if on)
- Cloud-Init (if on)
- All NICs summary (e.g. "nic0: vmbr0 / virtio, nic1: vmbr1 / e1000 tag=10")

## 5. Submit body

`POST /qemu` body:
- net0..net3: `<model>,bridge=<bridge>,tag=<vlan>` (omit if no VLAN)
- scsi0: `<storage>:<disk>` (always)
- ide2: `none,media=cdrom` (for Cloud-Init)
- efidisk0: `<storage>:1,efitype=4m,pre-enrolled-keys=1` (if EFI)
- tpmstate0: `<storage>:1,version=v2.0` (if TPM)
- cicustom: `user=<storage>:snippets/<vmid>-user.yaml` (if Cloud-Init, deferred)
- ostype: `l26` (Linux 2.6+) or `win11` (if EFI + Win)

## 6. Acceptance criteria

1. Multi-NIC add/remove works
2. Cloud-Init toggle shows new state
3. EFI + TPM toggles work
4. Submit succeeds with all 3 features combined
5. Step 4 reflects all selections
6. Mobile-responsive
