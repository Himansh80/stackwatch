# Tier 12 Plan — Docs & GTM

## Phase 1 — Doc polish (12.1 + 12.2 + 12.3)

**Files (mostly expansion of existing):**
- `docs/COMPARISON.md` (159 → 200+ LOC) — add 12-tool comparison table + "When NOT to use" + migration guides
- `docs/INSTALL.md` (223 → 300+ LOC) — verify Docker + K8s + macOS paths + systemd unit + 15-min TTFW claim
- `docs/USER-GUIDE.md` (347 LOC ✓) — verify per-page tour covers all current pages
- `docs/ARCHITECTURE.md` (324 → 350+ LOC) — verify all 12 service types + diagrams
- `docs/TESTING.md` (273 → 300+ LOC) — verify PASS/FAIL/expected per test
- `docs/PRICING.md` (85 → 250+ LOC) — COMPLETE rewrite post-Tier-11 with real plans
- `docs/FAQ.md` (148 → 200+ LOC) — add 12 more questions

**Outcome**: 7 polished docs, all ≥ 200 LOC, cross-linked, "Last updated" headers.

## Phase 2 — Marketing site (12.4)

**Files (all NEW):**
- `web/public/marketing/index.html` (~350 LOC)
  - Hero + stats + features + comparison + pricing teaser + testimonials + FAQ + footer
- `web/public/marketing/styles.css` (~250 LOC)
  - Dark theme (reuse dashboard tokens)
  - Responsive (mobile-first)
  - Datadog-style cards
  - FAQ accordion
  - Animated KPI counters
- `web/public/marketing/script.js` (~50 LOC)
  - FAQ toggle
  - Smooth scroll
- `scripts/deploy-marketing.sh` (~80 LOC)
  - rsync + restart + verify

**Files modified:**
- `web/src/App.tsx` — add `/marketing` route (renders static index.html or MarketingPage React component)
- `web/src/components/AppSidebar.tsx` — add "Marketing" link

**Outcome**: Datadog-quality marketing page live at stackwatch.smarthomelab.fun/marketing/.

## Phase 3 — Finalize (TIER 12 COMPLETE)

- Update `docs/INDEX.md` — list all 7 polished docs + marketing site
- Update `journal-tier12-final.md` — write summary
- Archive `.hermes/changes/010-tier12-docs-gtm/`
- Update memory with Tier 12 status

## Total (estimated)

- 7 docs polished (existing, just expanded)
- 4 new marketing files (~730 LOC)
- 2 small frontend updates (App.tsx + AppSidebar.tsx)
- 1 deploy script
- 1 journal entry

## Risks

1. **Marketing site over-engineering** — keep it static HTML + 1 CSS
   file. No CMS, no SPA framework. Should load in <1 second.
2. **Comparison page bias** — be honest about StackWatch gaps vs
   competitors. Don't pretend to be better at everything.
3. **Pricing page commitments** — don't lock in prices we can't honor
   once Stripe is wired.
4. **Doc drift** — link from docs to live API; if API breaks docs break
   (defer auto-sync to Tier 13).
5. **Marketing copy quality** — keep it factual, not flowery.

## Deployment

- Marketing files copied to `.115:/opt/stackwatch/web/public/marketing/`
- serve.py already serves `web/public/` (no config change)
- File size target: <100 KB total (HTML + CSS + JS)

## Verifier

```bash
# 1. Doc size check
wc -l docs/*.md  # all meet LOC targets

# 2. Marketing page check
curl -s -o /dev/null -w "%{http_code}\n" http://192.168.0.115:8090/marketing/

# 3. Visual check
playwright screenshot http://192.168.0.115:8090/marketing/ /tmp/marketing.png

# 4. Cross-doc links valid
grep -rE "\]\(\.\./[a-z-]+\.md\)" docs/*.md | head -20
# (manual check — make sure all links resolve)
```
