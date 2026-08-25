# Tier 12 Requirements Checklist — Docs & GTM

## Functional

- [ ] **12.1** COMPARISON.md covers 12 tools (Datadog, New Relic, Grafana Cloud, Better Stack, SigNoz, HyperDX, Sentry, Prometheus, Zabbix, Nagios, Dynatrace, AppDynamics)
- [ ] **12.1** COMPARISON.md has "When NOT to use StackWatch" section (honest)
- [ ] **12.1** COMPARISON.md has at least 2 migration guides (e.g., from Datadog, from Grafana Cloud)
- [ ] **12.2** INSTALL.md covers Linux, macOS, Docker, Kubernetes
- [ ] **12.2** INSTALL.md has 15-minute TTFW claim with verification
- [ ] **12.2** USER-GUIDE.md covers every page in the current dashboard
- [ ] **12.2** ARCHITECTURE.md covers all 12 service types (api-gateway, ingest, alert, ai, smart-switch, web-terminal, status-page, usage-meter, retention, backup-scheduler, capacity-forecast, platform health)
- [ ] **12.2** TESTING.md has PASS/FAIL/expected per test
- [ ] **12.3** PRICING.md has the 4 plans (free / starter / pro / enterprise) with all limits
- [ ] **12.3** PRICING.md has "Why we're 10x cheaper than Datadog" pitch
- [ ] **12.3** PRICING.md has "Self-host is free, forever" (AGPL-3.0 source available)
- [ ] **12.3** PRICING.md has ROI calculator + FAQ
- [ ] **12.4** Marketing site has hero + stats + features + comparison + pricing teaser + testimonials + FAQ + footer
- [ ] **12.4** Marketing site uses Datadog-quality dark theme + KPI cards + gradient hero
- [ ] **12.4** Marketing site is mobile-responsive
- [ ] **12.4** Marketing page load time < 1s

## Non-functional

- [ ] All docs use Datadog-style Markdown headers
- [ ] All docs include "Last updated: YYYY-MM-DD" header
- [ ] All docs cross-link to related docs (no 404 links)
- [ ] All docs include "Troubleshooting" section linking to TROUBLESHOOTING.md
- [ ] Marketing page uses existing color tokens (no hardcoded hex)
- [ ] All 7 polished docs meet LOC targets:
  - COMPARISON.md ≥ 200 LOC
  - INSTALL.md ≥ 300 LOC
  - USER-GUIDE.md ≥ 350 LOC
  - ARCHITECTURE.md ≥ 350 LOC
  - TESTING.md ≥ 300 LOC
  - PRICING.md ≥ 250 LOC
  - FAQ.md ≥ 200 LOC

## Verification gates

- [ ] `wc -l docs/*.md` shows all target LOC met
- [ ] Marketing page opens at http://192.168.0.115:8090/marketing/ → HTTP 200
- [ ] Marketing page renders correctly (Playwright screenshot)
- [ ] All cross-doc links valid (no 404)
- [ ] `scripts/deploy-marketing.sh` runs cleanly end-to-end
- [ ] `cd web && npm run build` exits 0
- [ ] Marketing page HTML+CSS+JS total < 100 KB

## Deployment

- [ ] Marketing files copied to `.115:/opt/stackwatch/web/public/marketing/`
- [ ] serve.py serves them on port 8090
- [ ] Verified via curl

## Out-of-scope gates

- [ ] NO Tier 13 features (mobile, push notifications)
- [ ] NO new backend routes (Tier 12 is frontend + docs only)
- [ ] NO Stripe/Razorpay live integration (deferred)
- [ ] NO Razorpay webhook HMAC verification (deferred)
- [ ] NO email service production config (deferred)
- [ ] NO professional copywriter review (use factual founder copy)
