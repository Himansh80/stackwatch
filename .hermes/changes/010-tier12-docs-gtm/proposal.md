# Tier 12 — Docs & GTM

## Why this tier

After 11 tiers of building infrastructure, monitoring, intelligence,
security, homelab, and platform features, we have a great product. But
nobody can buy what they don't understand. This tier is about
**telling the story** — the docs, the comparison, the pricing, and the
marketing surface.

This is the **go-to-market** tier. It produces 4 deliverables:

1. **COMPARISON.md** (vs all 12 tools) — already shipped, polish if needed
2. **Customer docs** (INSTALL, USER-GUIDE, ARCHITECTURE, TESTING) — already shipped, polish if needed
3. **Pricing/packaging** — PRICING.md already exists but is "provisional"; finalize it post-Tier 11
4. **Marketing site** — DOES NOT EXIST; build a static landing page at stackwatch.smarthomelab.fun

## User Stories

### US-1 — Find the right tool (12.1 COMPARISON)
**As** a prospective customer evaluating monitoring tools, **I want** a
single page that compares StackWatch against the 12 alternatives (Datadog,
New Relic, Grafana Cloud, Better Stack, SigNoz, HyperDX, Sentry, Prometheus,
Zabbix, Nagios, Dynatrace, AppDynamics) on the axes that matter to me
(price, on-prem vs SaaS, language support, feature coverage, lock-in)
**so that** I can confidently choose StackWatch.

**Priority**: HIGH. This is the single most-shared marketing asset.

### US-2 — Install StackWatch myself (12.2 INSTALL)
**As** a customer who chose StackWatch, **I want** step-by-step install
instructions for Linux, macOS, Docker, and Kubernetes, **so that** I can
get up and running in under 15 minutes without contacting support.

**Priority**: HIGH. Time-to-first-dashboard determines conversion.

### US-3 — Understand the architecture (12.2 ARCHITECTURE)
**As** a technical evaluator, **I want** a deep-dive architecture document
covering the 12 service types, multi-tenancy model, data flow, and
extension points **so that** I can answer "will StackWatch scale for us?"

**Priority**: HIGH. Required for enterprise sales.

### US-4 — Use StackWatch effectively (12.2 USER-GUIDE)
**As** a new user, **I want** a guided tour of every page and feature,
from "create my first server" to "build a custom dashboard", **so that**
I get value in my first hour.

**Priority**: HIGH. Drives activation + retention.

### US-5 — Understand pricing (12.3 PRICING)
**As** a prospect comparing tools, **I want** a clear pricing page with
tier comparison, what's free vs paid, and a "why we're different from
Datadog at 10x cheaper" pitch **so that** I can make a budget case.

**Priority**: HIGH. Pricing transparency = trust.

### US-6 — Find us on the web (12.4 MARKETING)
**As** a curious developer, **I want** a polished marketing landing page
at stackwatch.smarthomelab.fun that I can share with colleagues **so
that** they can evaluate StackWatch without me sending a 10-page PDF.

**Priority**: HIGH. The marketing page is the front door.

## Scope (what's IN)

**Tier 12 ships**:
- Final COMPARISON.md (vs 12 tools) — polish existing 159 LOC
- Final INSTALL.md (Linux + macOS + Docker + K8s) — already 223 LOC
- Final USER-GUIDE.md (per-page tour) — already 347 LOC
- Final ARCHITECTURE.md (deep dive) — already 324 LOC
- Final TESTING.md (verification procedures) — already 273 LOC
- Final PRICING.md (post-Tier-11 plans) — extend from 85 LOC → 200+ LOC
- **NEW**: `web/src/pages/MarketingPage.tsx` (or static HTML) — landing page
- **NEW**: `web/public/marketing/` directory — static assets
- **NEW**: `scripts/deploy-marketing.sh` — one-command deploy to NPM/Caddy

## Out of Scope (Tier 13+)

- React Native mobile app (Tier 13)
- Push notifications backend (Tier 13)
- Pricing-paid Stripe/Razorpay integration (Tier 11.2 covers metering;
  Tier 12 only finalizes the PRICE PAGE content)
- Razorpay webhook HMAC verification (deferred — not blocking launch)
- Email service production config (deferred)

## Risks

1. **Marketing site over-engineering** — keep it static HTML + 1 CSS
   file. No CMS, no SPA framework. Should load in <1 second.
2. **Comparison page bias** — be honest about StackWatch gaps vs
   competitors. Don't pretend to be better at everything.
3. **Pricing page commitments** — don't lock in prices we can't honor
   once Stripe is wired.
4. **Doc drift** — link from docs to live API; if API breaks docs break
   (defer auto-sync to Tier 13).
5. **Marketing copy quality** — the user is the founder; no professional
   copywriter. Keep it factual, not flowery.

## Architectural Decisions

1. **Marketing page is static HTML + CSS** — deployed via the existing
   serve.py on `port 8090`. NO React SPA framework for marketing.
2. **Single marketing directory** — `web/public/marketing/index.html`
   + `styles.css` + `script.js` (if needed for tiny interactions).
3. **Docs are markdown, served via serve.py** — already works.
4. **PRICING.md stays the source of truth** — marketing page links to it.
5. **No new dependencies** — vanilla HTML + CSS + tiny JS.
6. **Modular discipline** — split the marketing page into 5-7 files
   (hero.html, features.html, pricing.html, faq.html, footer.html, styles.css)
   but they all live in `web/public/marketing/` and get served as-is.
