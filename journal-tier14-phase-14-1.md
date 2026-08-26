---

## Session 2026-08-25/26 — Tier 14 begins + Phase 14.1 shipped

# PHASE 14.1 COMPLETE — Proxmox VM List Page

**Speckit change 012-tier14-proxmox-ui shipped end-to-end. Phase 14.1 of Tier 14.**

## What this phase delivers

A polished, Datadog-style **VM list page** at `/proxmox-vms` that lives alongside the existing `/proxmox` workspace. Replaces the daily-use screens of the Proxmox web UI: see VMs, start/stop them, search, filter.

## Why a new page (not just improve the workspace)

- `/proxmox` is a multi-section workspace with 10 tabs (Compute/Storage/Network/etc.). It's the power-user view.
- `/proxmox-vms` is **focused**: one job — show me my VMs and let me act on them. Opens fast, looks Datadog-quality, mobile-responsive.
- Two paths, same backend, no duplication of API logic.

## Spec ↔ Live

| Promise | Actual |
|---------|--------|
| 7 components | ProxmoxVmListPage (196) + ProxmoxHostSelector (98) + ProxmoxKpiStrip (128) + ProxmoxFilterBar (57) + ProxmoxVmTable (160) + ProxmoxVmActions (114) + ProxmoxEmptyState (46) |
| 1 CSS file | proxmox.css (357 LOC) |
| 1 utility helper | formatUptime + readString added to lib/proxmox.ts |
| Framer-motion animations | whileHover + while tap + + AnimatePresence + useCountUp |
| Mobile-responsive | Table → cards on <768px (CSS-only) |
| Auto-refresh | 5s + paused on visibilitychange |
| Modular discipline | Max 196 LOC, all files < 400 ✓ |

## Backend

Zero new endpoints. Uses existing Tier 1:
- `GET /api/v1/proxmox/hosts`
- `GET /api/v1/proxmox/hosts/:id/qemu`
- `POST /api/v1/proxmox/hosts/:id/nodes/:node/qemu/:vmid/status/:action`

## Commits

- `3aac103` — speckit artifacts (proposal, spec, plan, tasks, checklist)
- `58dfefc` — feature (7 components + CSS + route + helper)
- `1b6344a` — sidebar fix (full 21 items + Proxmox sub-nav)

## Verification

- ✓ tsc --noEmit: 0 errors
- ✓ npm run build: clean (756 KB bundle)
- ✓ HTTP serves: /proxmox-vms → 200, all assets → 200
- ✓ Bundle contains all 8 component classes (px-vm-page, px-kpi-strip, px-filterbar, etc.)
- ✓ Framer-motion features detected in bundle
- ✓ User tested live and confirmed "all good"

## Operational notes this session

### TEST WORKFLOW RULE (locked 2026-08-26, user directive)

User said: "we will do like this we test during the phase 14 build like now and at the end you will give me the full detailed test for the entire phase 14 complete test end to end from A to Z ok note it down and we strictly follow it ok"

Strict workflow for ALL of Tier 14 (going forward):
1. **During each phase** — build, user tests, fix, ship (current pattern — keep it).
2. **At the end of every phase** — deliver a **detailed A-to-Z test plan for that phase ONLY**. User runs through it manually.
3. **When ALL of Tier 14 is complete** — deliver the **FULL end-to-end A-to-Z test for the entire Tier 14** (every feature across every phase). This is the final acceptance gate.

No exceptions. NEVER skip the per-phase test plan at the end of each phase. NEVER combine all phases into one mega-test (do per-phase as built, mega-test only when Tier 14 is 100% done).

Test plan format (mandatory):
- Numbered test cases
- Expected vs actual
- "What to tell me" outcome (all good / fix X / still broken)
- Browser refresh instructions (Ctrl+Shift+R) if needed
- Mobile + edge cases
- Empty/error states

### Operational lessons

1. **In-memory rate limiter requires service restart to clear.** `systemctl restart stackwatch-api-gateway.service`. Don't reset users; just restart. (User hit "too many requests" from my failed login probes earlier.)
2. **Dashboard.tsx was filtering sidebar to 4 items.** `show={['dashboard','billing','proxmox','truenas']}` was a leftover from earlier UX iteration. Removed to show all 21.
3. **No `rsync` on .115 host.** Deploy frontend via `scp` instead of `rsync`. Per-file, not bulk.
4. **Frontend deploy pattern:** `scp web/dist/index.html root@.115:/opt/stackwatch/web/public/index.html` + `scp web/dist/assets/* root@.115:/opt/stackwatch/web/public/assets/`.
5. **speckit methodology validated for tier-level work.** Proposal → spec → plan → tasks → checklist → build → archive. Each phase = one feature = full vertical slice.

## What Phase 14.2 looks like

When the user says "next", I'll start Phase 14.2: **VM detail page** at `/proxmox-vms/:vmid` with tabs:
- **Summary** — CPU/RAM/Disk/Uptime live metrics, host info
- **Config** — hardware, BIOS, machine type, SCSI/virtio settings (read + edit)
- **Network** — interfaces, bridges, IPs, MAC, MTU
- **Console** — xterm.js + noVNC iframe (re-use existing web-terminal WS bridge)
- **Firewall** — rules, aliases, IPSets, security groups
- **Snapshots** — list + create + delete + rollback
- **Backup** — schedule + run-now + restore

Estimated ~6-8 more sessions for all of Tier 14 (per MASTER-PLAN).

---