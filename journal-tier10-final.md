---

## Session 2026-08-25 — TIER 10 COMPLETE (Homelab Dashboard)

# TIER 10 COMPLETE — Homelab Dashboard (Homarr Parity)

**Speckit change 008-tier10-homelab-dashboard** shipped end-to-end with full
speckit workflow (proposal → spec → plan → tasks → checklist → 9 phases →
archive + journal).

### 9 Phases (53 routes + 17 tables + 6 background workers)

| Phase | Sub-tier | Commit | Routes | Tables | Workers |
|-------|----------|--------|-------:|-------:|---------|
| 0+1 | routes split + Widget Framework + Page shell | `6174873` | 6 | 2 | — |
| 2 | Service Status (H2) | `eccd734` | 8 | 2 | ServiceHealth (60s) |
| 3 | Notes + Todos (H3) | `7590148` | 12 | 2 | — |
| 4 | Calendar (H4) | `18757c4` | 5 | 2 | Calendar (30min) |
| 5 | Download Stats (H5) | `a6e905a` | 4 | 2 | Downloads (60s) |
| 6 | Media Server (H6) | `a3c88ac` | 4 | 3 | Media (60s) |
| 7 | Search (H7) | `3049851` | 3 | 0 | — |
| 8 | RSS (H9) | `a89432c` | 5 | 2 | RSS (5min) |
| 9 | Task Scheduler (H10) | `2fd5641` | 6 | 2 | Scheduler (60s) |
| — | fix: trim routes_protected.go | `f1dede4` | — | — | — |
| **TOTAL** | | **11 commits** | **53** | **17** | **6** |

### What was built

- **Per-user layouts**: `react-grid-layout` widget framework + per-user theme/refresh prefs (separate from tenant dashboards)
- **Service Status**: HTTP/TCP/ICMP probes, 60s background polling, per-user pin cap (50), aggregate status counts
- **Notes**: markdown + tags + pinned + search (title/body/tags)
- **Todos**: priority (low/medium/high/urgent) + due_date + tags + completed_at + due-soon widget
- **Calendar**: iCal URL subscriptions + 30min worker, week-view widget with colored chips
- **Download Stats**: 6 media clients (Sonarr/Radarr/qBittorrent/SABnzbd/Lidarr/Readarr) + 60s polling
- **Media Server**: Plex/Jellyfin/Emby now-playing + recent additions + 60s polling
- **Search**: parameterized SQL UNION ALL across 6 kinds (notes/todos/services/calendars/downloads/media) + score-based ranking
- **RSS**: 5min worker using `github.com/mmcdole/gofeed` + unread partial index + mark-as-read
- **Scheduler (security-critical)**: NO shell-out, action allowlist (http_get/http_post ONLY), DNS-resolved private IP blocklist (10/8, 172.16/12, 192.168/16, 127/8, ::1, fc00::/7), header blacklist (Host/Cookie/Authorization), 4KB body limit, per-user 25-job cap, 5min rate limit, defense-in-depth URL validation at runtime

### Mod discipline

- All Go files under 400 LOC (max 393 = handlers_homelab_validation.go)
- All TSX files under 400 LOC (max 398 = MediaWidget.tsx)
- routes_protected.go untouched at 396 LOC since Phase 0
- routes_homelab.go at ~330 LOC
- 2 new deps: `github.com/arran4/golang-ical` (Phase 4), `github.com/mmcdole/gofeed` (Phase 8)
- Minimal 5-field cron parser written from scratch (avoids robfig/cron dep)

### Architecture

- New Go package `internal/homelab/` for all background workers
- DRY pattern: handler functions reuse worker poll helpers (`PollClientAndInsertSnapshot`, `PollFeedAndUpsertItems`, `RunSchedulerJobAndInsertRun`) for immediate poll on add
- Per-user (NOT per-tenant) on every query
- Static-path siblings BEFORE `:id` patterns in routes_homelab.go (Gin radix-tree safe)

### Verification

- All 53 Tier 10 routes live on .115 (return 401 without auth, verified)
- All 17 Tier 10 tables on .116
- 6 background workers started without panic
- /health = 200
- Cross-tier regression: Tier 0-9 routes still return 401/200
- routes_protected.go untouched at 396 LOC
- Archive: `.hermes/changes/archive/2026-08-25-008-tier10-homelab-dashboard/`

### Cumulative stackwatch status (2026-08-25)
- Tier 0 (Foundation): DONE
- Tier 1 (Proxmox): DONE
- Tier 2 (TrueNAS): DONE
- Tier 3 (Remote Access): DONE
- Tier 4 (Server Admin): DONE
- Tier 5 (Container Mgmt): DONE
- Tier 6 (Monitoring Depth): DONE
- Tier 7 (Datadog Full Platform): DONE
- Tier 8 (Intelligence & Alerting): DONE
- Tier 9 (Security & Enterprise): DONE
- **Tier 10 (Homelab Dashboard): DONE** — TIER 10 COMPLETE 2026-08-25
- Tier 11 (Platform & Commerce): NOT STARTED
- Tier 12 (Docs & GTM): NOT STARTED
- Tier 13 (Mobile): NOT STARTED

### KEY LESSONS (saved to skill stackwatch-build-deploy)

1. **Build env gotcha**: `export GOOS=linux; export GOARCH=amd64; go build -o ./<local>` — combined form silently fails on Windows
2. **MUST COMMIT**: Prior subagents forgot this 3+ times. Commit immediately after live verify with the file list.
3. **Speckit workflow validated**: Proposal → Spec → Plan → Tasks → Checklist → Build → Archive works end-to-end across 10 tiers.
4. **routes_protected.go modular split is critical**: Phase 0 split (routes_homelab.go) unblocked all 53 routes across 9 phases.
5. **Frontend 2-3 file section pattern works**: Main + Modals + FormViews keeps each under 400 LOC.
6. **DRY pattern for background workers**: Handler exports a poll helper, worker reuses it. Tested pattern across Phases 4/5/6/8/9.
7. **SSRF defense in depth**: Validate URL at add-time AND at runtime (DNS could resolve differently). Block private IP ranges. No shell-out ever.
8. **Subagents commit reliably when MANDATORY is explicit** in the goal body.
9. **Cron parser from scratch** is fine for 5-field — avoids adding `robfig/cron` dep just for one feature.
10. **routes_homelab.go can hold 50+ routes** if split into phases; routes_protected.go stays untouched.
