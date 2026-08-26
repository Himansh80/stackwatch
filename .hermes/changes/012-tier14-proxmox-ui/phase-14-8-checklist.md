# Tier 14 Phase 14.8 Requirements Checklist

## Node Firewall
- [ ] **F1** Rules table (Pos/Action/Type/Source/Dest/Proto/Dport/Comment/Enable)
- [ ] **F2** "+ Add rule" button
- [ ] **F3** Edit per row (dialog pre-filled)
- [ ] **F4** Delete per row (confirm)
- [ ] **F5** Empty state
- [ ] **F6** Reuse ProxmoxFirewallRuleDialog

## IPSets
- [ ] **I1** Table (Name/Comment/#CIDRs/Actions)
- [ ] **I2** "+ Create IPset"
- [ ] **I3** Expand to show CIDRs
- [ ] **I4** "+ Add CIDR" inline
- [ ] **I5** Delete CIDR
- [ ] **I6** Delete IPset
- [ ] **I7** Empty state

## Aliases
- [ ] **A1** Table (Name/CIDR/Comment/Actions)
- [ ] **A2** "+ Create alias"
- [ ] **A3** Delete
- [ ] **A4** Empty state

## SDN Zones
- [ ] **S1** Table (Name/Type/Bridge/DHCP/State)
- [ ] **S2** Read-only for Phase 14.8
- [ ] **S3** Empty state
- [ ] **S4** Phase 14.10 note ("Full CRUD in v2.10")

## Quality
- [ ] **Q1** Every TSX < 400 LOC
- [ ] **Q2** tsc --noEmit: 0 errors
- [ ] **Q3** npm run build: clean
- [ ] **Q4** Bundle deployed
- [ ] **Q5** Mobile-responsive
