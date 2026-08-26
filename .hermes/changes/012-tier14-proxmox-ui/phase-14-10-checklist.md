# Tier 14 Phase 14.10 Requirements Checklist

## Tasks Panel
- [ ] **T1** Table (UPID/Node/Type/Status/Started/Duration/Actions)
- [ ] **T2** Filter: node
- [ ] **T3** Filter: status
- [ ] **T4** Filter: type
- [ ] **T5** Auto-refresh 5s
- [ ] **T6** Stop task action
- [ ] **T7** Click → task log

## Backup Jobs
- [ ] **B1** Table (ID/Schedule/Storage/Keep/Enabled)
- [ ] **B2** Create form
- [ ] **B3** Run-now button
- [ ] **B4** Delete

## Replication
- [ ] **R1** Table (ID/Job/Target/Schedule/Enabled)
- [ ] **R2** Create form
- [ ] **R3** Delete

## HA Groups
- [ ] **H1** Table (ID/Resources/State/Comment)
- [ ] **H2** Create form
- [ ] **H3** Delete

## Certificates
- [ ] **C1** List (filename/subject/issuer/notAfter)
- [ ] **C2** View detail
- [ ] **C3** Renew button
- [ ] **C4** Delete

## SDN (upgrade from 14.8)
- [ ] **S1** + Create zone button
- [ ] **S2** Create form
- [ ] **S3** Delete per row
- [ ] **S4** Read-only state removed (now full CRUD)

## Quality
- [ ] **Q1** Every TSX < 400 LOC
- [ ] **Q2** tsc --noEmit: 0 errors
- [ ] **Q3** npm run build: clean
- [ ] **Q4** Bundle deployed
- [ ] **Q5** Mobile-responsive

## End of Tier 14
- [ ] **E1** All 10 phases shipped
- [ ] **E2** Mega-test delivered
- [ ] **E3** Archive
- [ ] **E4** Memory + journal update
