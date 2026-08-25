# Tier 12 Tasks — Docs & GTM

## Phase 1 — Doc polish (12.1 + 12.2 + 12.3)

- [ ] 1.1 Expand `docs/COMPARISON.md` to 200+ LOC — add 12-tool comparison table + "When NOT to use" + 2 migration guides
- [ ] 1.2 Expand `docs/INSTALL.md` to 300+ LOC — verify Docker + K8s + macOS + systemd + TTFW claim
- [ ] 1.3 Verify `docs/USER-GUIDE.md` (347 LOC) — confirm per-page tour covers all current pages
- [ ] 1.4 Expand `docs/ARCHITECTURE.md` to 350+ LOC — verify all 12 service types + diagrams
- [ ] 1.5 Expand `docs/TESTING.md` to 300+ LOC — verify PASS/FAIL/expected per test
- [ ] 1.6 **MAJOR REWRITE** `docs/PRICING.md` to 250+ LOC — final post-Tier-11 pricing with all 4 plans
- [ ] 1.7 Expand `docs/FAQ.md` to 200+ LOC — add 12 more questions
- [ ] 1.8 Verify all docs have "Last updated" header
- [ ] 1.9 Verify all docs cross-link to related docs (INSTALL → USER-GUIDE → ARCHITECTURE)
- [ ] 1.10 Commit: `docs(tier12): polish customer docs (Phase 1)`

## Phase 2 — Marketing site (12.4)

- [ ] 2.1 Create `web/public/marketing/index.html` (~350 LOC)
- [ ] 2.2 Create `web/public/marketing/styles.css` (~250 LOC)
- [ ] 2.3 Create `web/public/marketing/script.js` (~50 LOC)
- [ ] 2.4 Create `scripts/deploy-marketing.sh` (~80 LOC)
- [ ] 2.5 Update `web/src/App.tsx` to add `/marketing` route
- [ ] 2.6 Update `web/src/components/AppSidebar.tsx` to add "Marketing" link
- [ ] 2.7 Verify marketing page loads at HTTP 200 + <1s
- [ ] 2.8 Verify npm run type-check + npm run build pass
- [ ] 2.9 Deploy to `.115` via `scripts/deploy-marketing.sh`
- [ ] 2.10 Take Playwright screenshot for visual verification
- [ ] 2.11 Commit: `feat(tier12): Marketing site (Phase 2)`

## Phase 3 — Finalize (TIER 12 COMPLETE)

- [ ] 3.1 Update `docs/INDEX.md` — list all 7 polished docs + marketing site
- [ ] 3.2 Write `journal-tier12-final.md` — full Phase 1-2 summary
- [ ] 3.3 Archive `.hermes/changes/010-tier12-docs-gtm/`
- [ ] 3.4 Update memory with Tier 12 status
- [ ] 3.5 Commit: `docs(tier12): TIER 12 COMPLETE marker + journal`

## Out-of-scope (Tier 13)

- React Native mobile app (MO1)
- Push notifications backend (MO2)
