# Tier 14 Phase 14.7 Requirements Checklist

## Node KPI Strip
- [ ] **K1** CPU% card
- [ ] **K2** RAM used/total
- [ ] **K3** Disk used/total
- [ ] **K4** Load avg
- [ ] **K5** Uptime
- [ ] **K6** Animated count-up

## Node Dashboard
- [ ] **D1** Breadcrumb
- [ ] **D2** Heading with status badge
- [ ] **D3** Disks table
- [ ] **D4** ZFS pools table (or empty)
- [ ] **D5** Network table
- [ ] **D6** Services table
- [ ] **D7** Auto-refresh 30s
- [ ] **D8** Each section error handles its own failure
- [ ] **D9** Mobile-responsive (cards stack on <768px)

## Storage Content
- [ ] **S1** Breadcrumb
- [ ] **S2** Storage info header (content types)
- [ ] **S3** File list table (volid, format, size, parent, date)
- [ ] **S4** Type icons
- [ ] **S5** Delete per row + confirm
- [ ] **S6** Upload form (URL field)
- [ ] **S7** Empty state
- [ ] **S8** Mobile-responsive

## Quality
- [ ] **Q1** Every TSX < 400 LOC
- [ ] **Q2** tsc --noEmit: 0 errors
- [ ] **Q3** npm run build: clean
- [ ] **Q4** Bundle deployed
- [ ] **Q5** Mobile-responsive
- [ ] **Q6** Dark theme + Datadog style
