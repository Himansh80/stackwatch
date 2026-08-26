# Tier 14 Phase 14.6 Requirements Checklist

## VmSpec
- [ ] **S1** nics: Nic[] added
- [ ] **S2** cloudInit: boolean added
- [ ] **S3** efiDisk: boolean added
- [ ] **S4** tpmState: boolean added
- [ ] **S5** osType: 'l26' | 'win11' | 'other' added
- [ ] **S6** DEFAULT_VM_SPEC updated

## NicRow
- [ ] **N1** Bridge input
- [ ] **N2** Model select (virtio/e1000/vmxnet3)
- [ ] **N3** VLAN input
- [ ] **N4** Remove button
- [ ] **N5** Index label (nic0, nic1, ...)

## Step 2 (Hardware)
- [ ] **H1** Cloud-Init checkbox
- [ ] **H2** EFI disk checkbox
- [ ] **H3** TPM checkbox (enabled only if EFI)
- [ ] **H4** OS type select (l26/win11/other)

## Step 3 (Network)
- [ ] **N6** 1 NIC by default
- [ ] **N7** "+ Add NIC" button (max 4)
- [ ] **N8** Remove button per NIC
- [ ] **N9** NIC count badge (e.g. "1 of 4")

## Step 4 (Confirm)
- [ ] **C1** Multi-NIC summary
- [ ] **C2** Cloud-Init row (if on)
- [ ] **C3** EFI disk row (if on)
- [ ] **C4** TPM row (if on)
- [ ] **C5** OS type row

## Submit
- [ ] **O1** Body includes net0..netN
- [ ] **O2** Body includes ide2=none if Cloud-Init
- [ ] **O3** Body includes efidisk0 if EFI
- [ ] **O4** Body includes tpmstate0 if TPM
- [ ] **O5** Body includes ostype from spec

## Quality
- [ ] **Q1** Every TSX < 400 LOC
- [ ] **Q2** tsc --noEmit: 0 errors
- [ ] **Q3** npm run build: clean
- [ ] **Q4** Bundle deployed
- [ ] **Q5** Mobile-responsive
- [ ] **Q6** Dark theme + Datadog style
