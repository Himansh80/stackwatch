# Tier 11 — Platform & Commerce (Datadog/Stripe Parity)

## Why this tier

StackWatch has monitoring, intelligence, security, and homelab — but
no way to RUN THE BUSINESS. After Tier 11, the platform is a real
commercial product: customers can sign up, hit their limits, get
billed, see usage, deploy to their own infra, back up + restore, scale
horizontally, and trust the platform to stay up.

This is the **business tier**. Tiers 1-10 were product. Tier 11 is
what turns the product into a company.

## User Stories

### US-1 — Push-Button Deploy (PL1)
**As** a customer, **I want** to download a one-line install script
that turns any Linux box into a StackWatch node (sends heartbeats +
metrics to my account), **so that** I can monitor my homelab in <60s
without manually configuring anything.

**Priority**: HIGHEST. Without it, signup has nothing to land on.

### US-2 — Usage Metering (PL2)
**As** the platform owner, **I want** to meter every billable event
(servers, metrics ingested, alerts fired, alerts delivered, API
calls, GB stored, etc.) per tenant, per day, **so that** I can
charge the right amount each month.

**Priority**: HIGH. Without it, every plan is "free forever".

### US-3 — Self-Service Signup (PL3)
**As** a visitor, **I want** to sign up with email + password (no
salesman call), get a tenant + default org + API key immediately,
**so that** I can try the product in 30 seconds.

**Priority**: HIGH. Every minute of friction = X% conversion loss.

### US-4 — Tenant Limits (PL4)
**As** the platform owner, **I want** per-plan limits (servers,
metrics, alerts, seats) enforced at the API layer, **so that** free
tier doesn't bankrupt me.

**Priority**: HIGH. Enforce QUOTA_EXCEEDED before usage happens.

### US-5 — Backup/Restore (PL5)
**As** a customer (or platform owner), **I want** full database
backups (daily + on-demand) and a one-click restore, **so that** a
mistake or disaster doesn't lose data.

**Priority**: HIGH. Customers won't trust the product without it.

### US-6 — Multi-Region / HA (PL6)
**As** the platform owner, **I want** the option to run read replicas
in additional regions (US, EU, APAC) with automatic failover,
**so that** latency is low for everyone and uptime is 99.95%.

**Priority**: MEDIUM. Single-region works for v1. HA is for enterprise.

### US-7 — Rate Limiting (PL7)
**As** the platform owner, **I want** per-tenant + per-IP rate limits
on every API route, with burst allowance and 429 responses,
**so that** one runaway customer can't DoS the platform.

**Priority**: HIGH. Already partly done in Tier 0 (auth) — extend to
all routes.

### US-8 — Platform Health (PL8)
**As** the platform owner, **I want** a /admin/health dashboard that
shows every service's status, every worker's last run, every queue's
depth, every error rate, every cache hit rate, **so that** I know
the platform is healthy before customers report problems.

**Priority**: HIGH. Operators need it. Customers trust it (status page).

## Scope (what's IN)

- 8 DB tables + several ALTER TABLE statements
- ~30 protected routes + 5 public/admin routes
- 6 background workers (deploy webhook poller, backup scheduler,
  usage rollup, HA replicator, rate limit cleaner, platform health
  reporter)
- Stripe + Razorpay integration (payment providers — primary: Stripe;
  secondary: Razorpay for India)
- Email integration (Resend / SMTP) for transactional + billing emails
- 1 unified `AdminPage` with 8 sub-tabs
- 1 unified `SettingsPage` (per-tenant) with billing + limits

## Out of Scope (Tier 12+)

- Public marketing site (Tier 12.4)
- Docs (Tier 12.1-12.3)
- Mobile app (Tier 13)
- Multi-cloud (AWS/GCP/Azure deploy) — single-region only

## Risks

1. **Stripe + Razorpay + Email integration** are external services
   that can fail in production. Mitigate with retry queues + circuit
   breakers + dev-mode that logs instead of sending.
2. **Self-service signup** opens the platform to abuse (spammers,
   crypto miners, etc). Mitigate with email verification + captcha +
   per-IP rate limit on signup endpoint + DB-level rate limit.
3. **Backup/restore** with a 4 GB database is non-trivial. Use
   `pg_dump` + `pg_restore` + 30-day rotation + checksum verification.
4. **Multi-region HA** is hard. v1 ships read replicas (async) but
   NOT multi-master. Document the failover story clearly.
5. **Usage metering** must be COMPLETE before any limit enforcement.
   Otherwise limits are wrong and customers complain.

## Architectural Decisions

1. **Stripe-first, Razorpay-second**: Stripe handles 80% of world
   payments; Razorpay unlocks UPI (India) where Stripe is restricted.
2. **Single binary** (api-gateway) hosts billing + signup + admin.
   No new sidecar services.
3. **Single migration file** (`042_platform.sql`) for all Tier 11 tables.
4. **In-process rate limiter** (per-IP + per-tenant) using `sync.Map`
   + sliding window. No external Redis dependency at this scale.
5. **Backup via `pg_dump`** running on a cron-like schedule via a
   background worker. Store to local disk (with rotation) AND
   optional S3-compatible upload.
6. **Email via Resend** (default) with SMTP fallback. Dev mode
   returns the email content instead of sending.
7. **Platform health** is a JSON endpoint + a Datadog-styled UI page
   consumed by both operators and the public status page.
8. **No new tier-11-specific public endpoints** — signup is the only
   new public POST; everything else is admin-only.

## Subscription Plans (proposed)

| Plan | Price | Servers | Metrics/day | Seats | Retention |
|------|-------|--------:|-------------|------:|----------:|
| **Free** | $0 | 3 | 100k | 1 | 7d |
| **Pro** | $24/mo | 25 | 10M | 5 | 30d |
| **Business** | $99/mo | 100 | 100M | 25 | 90d |
| **Enterprise** | Custom | Unlimited | Unlimited | Unlimited | 365d |

Limits enforced server-side. Free tier requires email verification.
Pro+ requires payment method on file.
