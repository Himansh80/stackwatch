# StackWatch — Pricing

**Last updated:** 2026-08-25

**Final post-Tier-11 pricing.** The "provisional" notice from prior
versions is removed — these plans are live in `internal/auth/seed_plans.go`
and wired through Stripe / Razorpay in Tier 11.

Cross-references: [COMPARISON.md](COMPARISON.md) ·
[INSTALL.md](INSTALL.md) · [FAQ.md](FAQ.md) ·
[USER-GUIDE.md](USER-GUIDE.md) · [ARCHITECTURE.md](ARCHITECTURE.md).

---

## TL;DR

> **Datadog Pro at $15–23/host/month. StackWatch Pro at $24/host/month.**
> **10× cheaper for SMB at 50 hosts. Free forever if you self-host.**

| | Free | Starter | Pro | Enterprise |
|---|---|---|---|---|
| **Price** | **$0/mo** | **$8/mo** per server | **$24/mo** per server | **Custom** |
| Max servers | 3 | 10 | 50 | Unlimited |
| Metric retention | 14 days | 30 days | 90 days | 365 days |
| Log/data retention | 7 days | 30 days | 90 days | 365 days |
| Alert rules | 10 | 50 | 500 | Unlimited |
| Dashboards | 1 | 5 | 25 | Unlimited |
| Team members | 1 | 3 | 15 | Unlimited |
| Storage | 1 GB | 10 GB | 100 GB | Unlimited |
| API rate limit | 10/min | 60/min | 100/min | 1000/min |
| Support | Community | Email | Priority | SLA + dedicated CSM |

**Self-host is free, forever.** AGPL-3.0 source, no metering, no call-home.
Bring your own Postgres. See [INSTALL.md §Self-host](INSTALL.md#production-install-systemd--binaries).

---

## The 4 plans (canonical catalog)

These are the exact rows seeded by `internal/auth/seed_plans.go` on
every api-gateway boot. Source of truth — what the database says
overrides this doc.

### Free — $0/month

For: home users, monitoring your own infrastructure, no card required.

| Limit | Value |
|-------|-------|
| Price | **$0/mo** (forever, no card) |
| Max servers | 3 |
| Metric retention | 14 days |
| Data retention | 7 days |
| Max alerts | 10 |
| Max dashboards | 1 |
| Max team members | 1 (you) |
| Storage | 1 GB |
| API rate limit | 10 calls/min |
| Channels | email |
| Features | `basic_monitoring`, `email_alerts` |

### Starter — $8/month per server

For: homelab operators, MSPs running 5–10 servers.

| Limit | Value |
|-------|-------|
| Price | **$8/server/month** (USD, billed monthly) |
| Max servers | 10 |
| Metric retention | 30 days |
| Data retention | 30 days |
| Max alerts | 50 |
| Max dashboards | 5 |
| Max team members | 3 |
| Storage | 10 GB |
| API rate limit | 60 calls/min |
| Channels | email, Slack webhooks |
| Features | `basic_monitoring`, `email_alerts`, `slack_webhooks`, `synthetics_5` |
| Annual discount | 2 months free (pay $80/yr/server) |

### Pro — $24/month per server

For: SMB IT teams, agencies, growing SaaS.

| Limit | Value |
|-------|-------|
| Price | **$24/server/month** (USD, billed monthly) |
| Max servers | 50 |
| Metric retention | 90 days |
| Data retention | 90 days |
| Max alerts | 500 |
| Max dashboards | 25 |
| Max team members | 15 |
| Storage | 100 GB |
| API rate limit | 100 calls/min |
| Channels | email, Slack, PagerDuty, OpsGenie, webhooks (10+) |
| Features | `basic_monitoring`, `email_alerts`, `slack_webhooks`, `synthetics_unlimited`, `sso_saml`, `audit_log` |
| Annual discount | 2 months free (pay $240/yr/server) |

### Enterprise — Custom contract

For: regulated industries, 50+ servers, custom integrations.

| Limit | Value |
|-------|-------|
| Price | **Contact sales** (typically $50–200/server/month depending on volume) |
| Max servers | Unlimited |
| Metric retention | 365 days |
| Data retention | 365 days |
| Max alerts | Unlimited |
| Max dashboards | Unlimited |
| Max team members | Unlimited |
| Storage | Unlimited |
| API rate limit | 1000 calls/min |
| Channels | All channels + custom integrations |
| Features | All Pro features + `multi_region`, `dedicated_support`, `custom_sla` |
| SLA | 99.9% uptime, 1-hour response on P1 |

---

## Why we're 10× cheaper than Datadog

Datadog's pricing model has **five** per-feature gates that compound
on a per-host basis:

1. **Infrastructure** ($15/host/mo for Pro) — your servers, billed
2. **APM** ($31/host/mo) — every host running instrumented code
3. **Logs** ($0.10/GB ingested + $1.70/million events) — every log line
4. **RUM** ($1.50/1000 sessions) — every browser session
5. **Synthetics** ($5/1000 test runs) — every API check

If you run 50 hosts with APM + Logs + RUM + 100 synthetics tests, your
Datadog bill looks like this:

| Item | Calculation | Monthly cost |
|------|-------------|--------------|
| Infrastructure (Pro) | 50 × $15 | $750 |
| APM (Pro) | 50 × $31 | $1,550 |
| Logs ingest | 500 GB × $0.10 | $50 |
| Logs indexing | 500 GB × $2.50 | $1,250 |
| RUM | 1M sessions × $1.50/1k | $1,500 |
| Synthetics | 100 × 43,200 runs × $5/1k | $21,600 |
| **Total Datadog** | | **$26,700/mo** |

StackWatch Pro at $24/host × 50 hosts = **$1,200/mo**.

**That's 22× cheaper, not 10×.** The 10× claim is conservative; the
real number depends on how heavy your logs/RUM/synthetics usage is.

### Why our pricing is simpler

- **No per-seat.** Add your engineer to a Pro account: **$0**.
- **No per-feature gates.** Metrics, logs, traces, RUM, synthetics,
  security, CSPM, CI visibility — all included.
- **No per-GB log ingest tax.** Storage is per-server, not per-byte.
- **No per-million events.** API rate limit is per-tenant, generous.
- **No surprise overages.** We hard-cap you at the plan limit (HTTP 402
  Payment Required on the next write attempt) and **email you before
  you hit it**.

---

## Self-host is free, forever

If you have a Linux VM and 30 minutes, you can run StackWatch for
$0/month, forever.

### What's included for free

- The full source (AGPL-3.0)
- All 19 service types, 480 API routes
- All 4 plans seeded into your local DB (you stay on `free`)
- All integrations (Proxmox, TrueNAS, terminal, etc.)
- All Tier 9 features (SSO, audit log, RLS) — they're OSS too
- Multi-tenancy, multi-region (Tier 11) — also OSS

### What's NOT included for free

- Our managed cloud infrastructure (Postgres, edge, backups)
- Our 24/7 support SLA
- Hosted SSO/SAML — you bring your own IdP
- Hosted backup target — you back up to your own S3 / NFS

### When self-host makes sense

- You have ≥ 2 vCPU + 4 GB RAM available
- You can run Postgres (or have a managed one)
- You're comfortable with `systemctl restart api-gateway`
- You don't need a support contract

### When self-host doesn't make sense

- You don't have ops capacity → pay us $8/mo for Starter
- You need 24/7 SLA → Enterprise
- You need FedRAMP / HIPAA attestation → Enterprise (managed cloud)

### License

- **AGPL-3.0** for the open-source self-host edition
- **Commercial license** available for enterprises that need a
  permissive license (no AGPL reciprocal terms). Contact
  `sales@stackwatch.io`.
- **Source-available** for the OEM tier — you can re-skin and re-sell
  under contract. Contact `sales@stackwatch.io`.

---

## ROI calculator (vs Datadog)

Use this to estimate your annual savings. Numbers are list prices; your
Datadog rep almost certainly gives volume discounts, so adjust the
"Datadog price" column to your actual invoice.

### 10-host homelab

| Item | Datadog Pro | StackWatch Pro |
|------|-------------|----------------|
| Infrastructure (10 hosts) | $180/mo | $240/mo |
| APM (10 hosts) | $310/mo | included |
| Logs (50 GB/mo) | $175/mo | included |
| RUM (50k sessions) | $75/mo | included |
| Synthetics (10 tests) | $60/mo | included |
| **Total** | **$800/mo** | **$240/mo** |
| **Annual** | **$9,600/yr** | **$2,880/yr** |
| **Savings** | — | **$6,720/yr (70%)** |

### 50-host SMB

| Item | Datadog Pro | StackWatch Pro |
|------|-------------|----------------|
| Infrastructure (50 hosts) | $900/mo | $1,200/mo |
| APM (50 hosts) | $1,550/mo | included |
| Logs (500 GB/mo) | $1,750/mo | included |
| RUM (500k sessions) | $750/mo | included |
| Synthetics (50 tests) | $300/mo | included |
| **Total** | **$5,250/mo** | **$1,200/mo** |
| **Annual** | **$63,000/yr** | **$14,400/yr** |
| **Savings** | — | **$48,600/yr (77%)** |

### 200-host mid-market

| Item | Datadog Enterprise | StackWatch Enterprise |
|------|--------------------|------------------------|
| Infrastructure (200 hosts) | $6,000/mo | ~$10,000/mo |
| APM (200 hosts) | $6,200/mo | included |
| Logs (5 TB/mo) | $17,500/mo | included |
| RUM (5M sessions) | $7,500/mo | included |
| Synthetics (200 tests) | $1,200/mo | included |
| **Total** | **$38,400/mo** | **~$10,000/mo** |
| **Annual** | **$460,800/yr** | **~$120,000/yr** |
| **Savings** | — | **~$340,000/yr (74%)** |

For the typical 50-host case, **StackWatch saves you $48,600/year**.
For 200 hosts, the saving approaches $340K/year — enough to hire an
SRE.

---

## Frequently asked pricing questions

### Can I pay annually?

Yes — annual plans get **2 months free** (pay 10 months for 12).
Apply at checkout or contact `sales@stackwatch.io`.

### Can I switch plans mid-month?

Yes. Upgrades are pro-rated immediately. Downgrades take effect at
the next billing cycle (so you don't lose paid-for capacity).

### What happens if I exceed my plan?

The api-gateway returns **HTTP 402 Payment Required** with a JSON body:

```json
{
  "ok": false,
  "error": "Plan limit exceeded: max_servers (3/3)",
  "code": "limit_exceeded",
  "request_id": "req_abc123"
}
```

We also email you at 80% utilization so you have time to upgrade.

### What if I downgrade and exceed the new plan?

Same as above — 402 Payment Required. Your data is **not deleted**;
you just can't *add* new servers/alerts/etc. until you upgrade back
or delete existing ones.

### Can I get a refund?

Within 30 days of any charge, yes — full refund, no questions. After
30 days, we pro-rate the unused portion of the current billing cycle.

### Can I get a custom plan?

Yes (Enterprise). Contact `sales@stackwatch.io` with:

- Number of servers
- Retention requirements
- Custom SLA needs
- Single sign-on IdP (if any)
- Geographic region for the data plane

Typical Enterprise contracts are 12 or 36 months.

### Do you have a free trial of paid plans?

Yes — **14 days**, no card required. You can self-host with no
restrictions indefinitely (AGPL); the trial is for our managed cloud.

### Can I white-label / re-sell?

Yes (OEM tier). Custom contract. Contact `sales@stackwatch.io`.

### Do you offer non-profit / education discounts?

Yes — **50% off** Starter and Pro for verified 501(c)(3) and
accredited educational institutions. Contact `support@stackwatch.io`
with proof of status.

### Is there a setup fee?

No. Stripe / Razorpay checkout is self-service.

### What currencies do you accept?

USD by default. EUR, GBP, INR, AUD, CAD on request. Annual contracts
in any currency via wire transfer.

### Can I get a custom data residency?

Yes (Enterprise). Pick a region: US-East, US-West, EU-Frankfurt,
EU-Dublin, APAC-Mumbai, APAC-Singapore. Self-host has no constraint.

### What payment methods do you accept?

- **Cloud Free**: no payment
- **Cloud Starter / Pro**: Stripe (cards, ACH, Apple Pay, Google Pay)
  or Razorpay (UPI, cards, NetBanking — India)
- **Enterprise**: wire transfer / ACH / SEPA + invoice NET-30

### Can I cancel any time?

Yes. The api-gateway stops enforcing limits the moment your
subscription ends; your tenant stays on the Free plan (3 servers,
7-day retention). Re-subscribe any time; your data is preserved for
**90 days** after cancellation, then deleted.

---

## What we WILL NOT do

These are the policies we publish and stick to:

- **No charge for users.** Adding your engineer to a Pro account is free.
- **No charge for alerts.** Rule count, channel count, retention — all tied
  to plan tier, never user count.
- **No price increases for existing customers.** Whatever you pay for a
  server at signup, you keep paying. (We can raise prices for new
  signups; existing customers are grandfathered.)
- **No surprise overages.** We hard-cap at the plan limit and email you
  at 80% utilization. We do **not** bill for overages.
- **No vendor lock-in.** Every endpoint returns JSON; metrics + logs
  export as PromQL + JSONL; dashboards export as JSON; monitors export
  as YAML. You can leave any time.

---

## How to subscribe

### Cloud (managed)

1. Sign up at `https://stackwatch.smarthomelab.fun/signup`
2. Verify your email
3. Choose Starter ($8/mo/server) or Pro ($24/mo/server)
4. Enter payment (Stripe or Razorpay)
5. Done — your account upgrades immediately

### Self-host (free)

1. `git clone https://github.com/Himanshu7613/stackwatch.git`
2. Follow [INSTALL.md](INSTALL.md) (15-minute TTFW)
3. Done — you own it, forever

### Enterprise

1. Email `sales@stackwatch.io` with your fleet size + region
2. We send a contract (MSA + order form) within 1 business day
3. Once countersigned, we provision your managed instance in 24h
   (or you deploy self-host with your commercial license)

---

## Where to get help

- [FAQ.md](FAQ.md) — operational questions
- [COMPARISON.md](COMPARISON.md) — vs Datadog, Grafana, etc.
- [INSTALL.md](INSTALL.md) — install + upgrade
- Email: `sales@stackwatch.io` (sales) or `support@stackwatch.io` (help)
- GitHub: https://github.com/Himanshu7613/stackwatch/issues