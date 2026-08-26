# Tier 14 Phase 14.9 Requirements Checklist

## Users
- [ ] **U1** Table (User ID/Name/Email/Role/Enabled/2FA/Expires/Actions)
- [ ] **U2** Create dialog
- [ ] **U3** Edit dialog (pre-filled)
- [ ] **U4** Delete confirm

## Tokens
- [ ] **T1** Table (Token ID/Comment/Expire/Privsep/Actions)
- [ ] **T2** Create dialog
- [ ] **T3** Show plain-text once + copy
- [ ] **T4** Revoke

## Permissions
- [ ] **P1** Table (Path/Role/Actions)
- [ ] **P2** Grant form (path, role, propagate)
- [ ] **P3** Revoke

## Pools
- [ ] **L1** Table (Pool ID/Comment/Members/Actions)
- [ ] **L2** Create form
- [ ] **L3** Delete
- [ ] **L4** Members list (read-only)

## Quality
- [ ] **Q1** Every TSX < 400 LOC
- [ ] **Q2** tsc --noEmit: 0 errors
- [ ] **Q3** npm run build: clean
- [ ] **Q4** Bundle deployed
- [ ] **Q5** Mobile-responsive
