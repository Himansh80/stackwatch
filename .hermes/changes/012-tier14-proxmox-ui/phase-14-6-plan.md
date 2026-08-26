# Tier 14 Phase 14.6 Plan

## Approach
Extend VmSpec + the 3 relevant step components. Keep file sizes small.

## Frontend (5 modified + 1 new)

### Step 2 (Hardware)
- Add: Cloud-Init checkbox
- Add: EFI disk checkbox
- Add: TPM checkbox (enabled only if EFI)
- Reuse existing form layout

### Step 3 (Network)
- Replace single NIC fields with `nics: Nic[]`
- Add ProxmoxNicRow component (reusable)
- + Add NIC button (max 4)
- Each row: bridge + model + VLAN + remove

### Step 4 (Confirm)
- Add rows for: Cloud-Init, EFI disk, TPM, multi-NIC summary
- Mirror pattern of existing rows

### New component: ProxmoxNicRow (~100 LOC)
- Props: nic, index, onChange, onRemove
- Single NIC editor with bridge input + model select + VLAN input + remove button

### vm-types.ts
- Add `nics: Nic[]` (replaces net0/bridge/vlanTag fields)
- Add `cloudInit: boolean`
- Add `efiDisk: boolean`
- Add `tpmState: boolean`
- Add `osType: 'l26' | 'win11' | 'other'`

### Orchestrator
- Update submit() to build net0..net3 strings
- Add efidisk0/tpmstate0/ide2/cicustom per toggles
- Map osType to ostype field

## Order
1. Update vm-types.ts
2. Create ProxmoxNicRow
3. Update Step 2 (add toggles)
4. Update Step 3 (multi-NIC)
5. Update Step 4 (extended summary)
6. Update orchestrator submit
7. tsc + build + deploy
8. Commit

## Modularity discipline
- Every TSX < 400 LOC
- Reuse existing form field patterns
