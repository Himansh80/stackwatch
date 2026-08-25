# Tier 12 Specification — Docs & GTM

## 1. Goals (measurable)

| Metric | Target |
|--------|--------|
| COMPARISON.md LOC | ≥ 200 (currently 159) |
| INSTALL.md LOC | ≥ 300 (currently 223) |
| USER-GUIDE.md LOC | ≥ 350 (currently 347 ✓) |
| ARCHITECTURE.md LOC | ≥ 350 (currently 324 — needs polish) |
| TESTING.md LOC | ≥ 300 (currently 273 — needs polish) |
| PRICING.md LOC | ≥ 250 (currently 85 — needs major expansion) |
| FAQ.md LOC | ≥ 200 (currently 148 — needs polish) |
| NEW: `marketing/index.html` | ≥ 300 LOC with hero + features + pricing + testimonials + CTA |
| NEW: `marketing/styles.css` | ≥ 200 LOC, Datadog-quality (dark theme + gradient + KPI cards) |
| Marketing page load time | < 1s |
| All docs link to live API | Yes (or mark "manual update needed") |
| All 4 docs ship README badge (build passing) | Yes |

## 2. Sub-features

### 12.1 — COMPARISON.md polish (already shipped, expand)

Currently 159 LOC. Need to expand to ≥ 200 LOC with:
- 12 alternatives covered (currently may be fewer)
- Each comparison: price / on-prem vs SaaS / language / feature coverage / lock-in
- Use a Markdown table for at-a-glance comparison
- Honest about StackWatch gaps (don't pretend to be better at everything)
- Add 2 new sections: "When NOT to use StackWatch" + "Migration from competitor X"

### 12.2 — Customer docs polish (already shipped, verify quality)

Already shipped 7 files totaling 1,665 LOC. Polish where needed:
- INSTALL.md (223 LOC): verify Docker + K8s install paths + systemd unit example + 15-min TTFW claim
- USER-GUIDE.md (347 LOC): verify per-page tour + screenshots placeholders
- ARCHITECTURE.md (324 LOC): verify all 12 service types covered + multi-tenancy + data flow diagrams
- TESTING.md (273 LOC): verify PASS/FAIL/expected criteria per test

### 12.3 — PRICING.md finalization (expand from 85 → 250 LOC)

Post-Tier-11 plans:
| Plan | Free | Starter | Pro | Enterprise |
|------|------|---------|-----|------------|
| Price | $0/mo | $8/mo | $24/mo | Custom |
| Servers | 3 | 10 | 50 | Unlimited |
| Retention | 7d | 30d | 90d | 1y |
| Alerts | 10 | 50 | 500 | Unlimited |
| Team | 1 | 3 | 15 | Unlimited |
| Storage | 1GB | 10GB | 100GB | Unlimited |
| API/min | 10 | 60 | 100 | 1000 |
| Support | Community | Email | Priority | SLA |

Add sections:
- "Why we're 10x cheaper than Datadog" (no per-host pricing, no per-feature gates)
- "Self-host is free, forever" (AGPL-3.0 source available)
- "ROI calculator" (rough estimate of savings vs Datadog/Grafana)
- "Frequently asked pricing questions" (annual discount, custom plans, refunds)

### 12.4 — Marketing site (NEW)

Create `web/public/marketing/` with these files:

**`web/public/marketing/index.html`** (~350 LOC):
- Hero section: "Self-Hosted Monitoring That You Own" + 2 CTAs (Try Free / Read Docs)
- Stats section: 4 KPIs (servers supported, features, tier coverage, OSS license)
- Features section: 6 cards (Datadog parity / Self-host / Multi-tenant / Open source / Datadog-style / All-in-one)
- Comparison section: table comparing StackWatch vs Datadog vs Grafana vs New Relic on 5 axes
- Pricing teaser: 4 tier cards (Free / Starter / Pro / Enterprise) with "Get Started" CTA
- Testimonials: 3-5 placeholder quotes from "users" (use fictional company names — mark as placeholder)
- FAQ accordion: 6 questions
- Footer: links to docs, github, contact, status page

**`web/public/marketing/styles.css`** (~250 LOC):
- Dark theme (matching the existing dashboard) using the same color tokens (`#0a0e1a` bg, `#22d3ee` accent, `#10b981` green, `#f59e0b` amber, `#ef4444` red)
- 1200px max-width container
- Responsive (mobile-first, breakpoint at 768px + 1024px)
- Smooth scroll, gradient hero, animated KPI numbers
- Datadog-style cards with subtle hover lift + colored top borders
- FAQ accordion with JS toggle
- Hero background: subtle gradient mesh animation (CSS-only, no JS framework)

**`web/public/marketing/script.js`** (~50 LOC):
- FAQ accordion toggle
- Smooth scroll for anchor links
- (optional) animated KPI counter on scroll into view

**`scripts/deploy-marketing.sh`** (~80 LOC):
- Rsync `web/public/marketing/` to `.115:/opt/stackwatch/web/public/marketing/`
- Restart serve.py on `.115`
- Verify HTTP 200 on the landing page

**`web/src/App.tsx`** + **`web/src/components/AppSidebar.tsx`** updates:
- Add "Marketing" link in AppSidebar (visible to anonymous + signed-in users)
- Register `/marketing` route → render MarketingPage (or redirect to /marketing/index.html if static)

## 3. Non-functional

- All docs use Datadog-style Markdown headers (H1 = title, H2 = section, H3 = subsection)
- All docs include "Last updated: YYYY-MM-DD" header
- All docs cross-link to related docs (INSTALL → USER-GUIDE → ARCHITECTURE)
- All docs include a "Troubleshooting" section linking to TROUBLESHOOTING.md
- Marketing page uses existing tokens (no hardcoded hex)
- Marketing page is mobile-responsive
- Marketing page load time < 1s (no heavy framework)

## 4. Verification gates

- `wc -l docs/*.md` shows all target LOC met
- Marketing page opens at http://192.168.0.115:8090/marketing/ → HTTP 200
- Marketing page renders correctly (Playwright screenshot for visual check)
- All cross-doc links valid (no 404)
- `scripts/deploy-marketing.sh` runs cleanly end-to-end

## 5. Deployment

- Marketing files copied to `.115:/opt/stackwatch/web/public/marketing/`
- serve.py already serves `web/public/` on port 8090 (no config change)
- Optional: add a redirect rule in serve.py so `/marketing` → `/marketing/`

## 6. Out-of-scope gates

- NO Tier 13 features (mobile, push notifications)
- NO new backend routes (Tier 12 is frontend + docs only)
- NO Stripe/Razorpay live integration (deferred)
- NO Razorpay webhook HMAC verification (deferred)
- NO email service production config (deferred)
- NO professional copywriter review (use factual founder copy)
