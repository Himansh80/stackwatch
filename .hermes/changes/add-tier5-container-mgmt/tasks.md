# Tasks: Tier 5 — Container Management

## 1. Handler files
- [ ] 1.1 internal/handler/admin_containers.go (C1 list + C1 get)
- [ ] 1.2 internal/handler/admin_container_action.go (C2)
- [ ] 1.3 internal/handler/admin_container_logs.go (C4)
- [ ] 1.4 internal/handler/admin_container_stats.go (C5)
- [ ] 1.5 internal/handler/admin_container_create.go (C6)
- [ ] 1.6 internal/handler/admin_images.go (C7)
- [ ] 1.7 internal/handler/admin_volumes.go (C8)
- [ ] 1.8 internal/handler/admin_networks.go (C9)
- [ ] 1.9 internal/handler/admin_stacks.go (C10)
- [ ] 1.10 internal/handler/admin_templates.go (C11)
- [ ] 1.11 internal/handler/admin_watchtower.go

## 2. Routes
- [ ] 2.1 Register 20+ container routes in cmd/api-gateway/routes.go

## 3. Build + deploy
- [ ] 3.1 go build ./...
- [ ] 3.2 Cross-compile api-gateway-linux
- [ ] 3.3 scp to .115, backup old binary, install
- [ ] 3.4 Restart api-gateway (kill + nohup with env)
- [ ] 3.5 Verify /health 200 + db:ok

## 4. Verifier
- [ ] 4.1 Write hermes-verify-tier5-full-2026-08-17.py
  - Ensure SSH connection to .113 (docker host) — create if missing
  - Ensure SSH key installed on .113's authorized_keys
  - Test C1 list/get, C2 start/stop/restart, C3 logs, C5 stats
  - Test C7 images, C8 volumes, C9 networks
  - Test C10 stacks, C11 templates, Watchtower
  - Use a TEST container (e.g. nginx or alpine sleep) for actions
- [ ] 4.2 Run 5 back-to-back; verify all PASS
- [ ] 4.3 Cross-check T0+T1+T2+T3+T4 still clean

## 5. Frontend (deferred to Tier 12)
- [ ] 5.1 Containers page (list + cards)
- [ ] 5.2 ContainerDetail (logs + stats + actions)
- [ ] 5.3 ContainerCreate (form)
- [ ] 5.4 Images page
- [ ] 5.5 Volumes page
- [ ] 5.6 Networks page
- [ ] 5.7 Stacks page
- [ ] 5.8 Templates page

## Done criteria
- [ ] All 11 handler files exist, build clean
- [ ] 20+ container routes registered
- [ ] hermes-verify-tier5-full: ≥25 checks, 5/5 back-to-back PASS
- [ ] T0+T1+T2+T3+T4 cross-check: still clean
- [ ] Commit landed in main branch
- [ ] Tier 5 marked ✅ in MASTER_BUILD_PLAN.md
