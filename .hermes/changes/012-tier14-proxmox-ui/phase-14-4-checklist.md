# Tier 14 Phase 14.4 Requirements Checklist

## List page
- [ ] **L1** Route `/proxmox-lxc` mounted
- [ ] **L2** Host selector works
- [ ] **L3** KPI strip shows LXC counts
- [ ] **L4** Filter chips (All/Running/Stopped/Paused)
- [ ] **L5** Search bar
- [ ] **L6** Table / mobile cards with: Status / VMID / Type / Name / Node / IP / RAM / CPU / Uptime / Actions
- [ ] **L7** Empty state if no LXC
- [ ] **L8** Auto-refresh every 5s
- [ ] **L9** Row click → detail page

## Detail page
- [ ] **D1** Route `/proxmox-lxc/:hostId/:node/:vmid` mounted
- [ ] **D2** Breadcrumb: Proxmox > {node} > CT {ctid}
- [ ] **D3** Header with name + status pill + meta
- [ ] **D4** Action bar: Start / Shutdown / Reboot / Migrate / Delete
- [ ] **D5** Tab: Summary (status + CPU + RAM + Disk + Tags)
- [ ] **D6** Tab: Resources (cores + memory + arch + ostype + boot)
- [ ] **D7** Tab: Network (QEMU agent interfaces)
- [ ] **D8** Tab: Snapshots (CRUD)
- [ ] **D9** Tab: Firewall (CRUD)
- [ ] **D10** Tab: Console (placeholder, real is Phase 14.7)

## Create wizard
- [ ] **W1** Route `/proxmox-lxc/new` mounted
- [ ] **W2** Step 1: Template picker (list from /storage/:storage/content)
- [ ] **W3** Step 2: Resources (VMID, hostname, cores, RAM, swap, disk)
- [ ] **W4** Step 3: Network (bridge, IPv4 mode, IPv6 mode, VLAN)
- [ ] **W5** Step 4: Confirm (review + submit)
- [ ] **W6** Next/Back buttons work
- [ ] **W7** State persists across steps
- [ ] **W8** Submit creates LXC + navigates to detail
- [ ] **W9** Error display on submit failure
- [ ] **W10** Progress bar shows 1-4 / 4

## Quality
- [ ] **Q1** Every TSX < 400 LOC
- [ ] **Q2** tsc --noEmit: 0 errors
- [ ] **Q3** npm run build: clean
- [ ] **Q4** Bundle deployed to .115
- [ ] **Q5** Mobile-responsive (cards <768px)
- [ ] **Q6** Dark theme + Datadog style

## User-acceptance (delivered at end of phase)
- List page works
- Detail page works
- Wizard creates an LXC end-to-end