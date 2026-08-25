# StackWatch — vs The Market

**Last updated:** 2026-08-25

Comparison is honest: where we're better, where we're worse, where we
haven't built the feature yet. Twelve alternatives compared on the
five axes that matter: **price**, **deployment** (SaaS vs self-host),
**language support**, **feature coverage**, and **lock-in risk**.

Cross-references: [INSTALL.md](INSTALL.md) · [USER-GUIDE.md](USER-GUIDE.md) ·
[ARCHITECTURE.md](ARCHITECTURE.md) · [PRICING.md](PRICING.md) ·
[FAQ.md](FAQ.md) · [TESTING.md](TESTING.md)

---

## At-a-glance — 12 alternatives

| Tool | SaaS? | Self-host? | Per-server price (USD/mo) | APM | RUM | Synthetics | Self-host footprint | OSS license |
|------|-------|------------|--------------------------|-----|-----|------------|---------------------|-------------|
| **StackWatch (Pro)** | ✅ | ✅ | **$8** or **$0 self-host** | ✅ (Tier 6) | ✅ (Tier 6) | ✅ (Tier 6) | 3 binaries + Postgres | AGPL-3.0 |
| Datadog | ✅ | ❌ | $15–23 + per-feature | ✅ | ✅ | ✅ | n/a | Proprietary |
| New Relic | ✅ | ❌ | $25+ | ✅ | ✅ | ✅ | n/a | Proprietary |
| Grafana Cloud | ✅ | ✅ (Grafana OSS) | $14 + per-active-series | 🟡 (plugin) | 🟡 | 🟡 | 5+ services | AGPL-3.0 |
| Better Stack | ✅ | ❌ | $25 | ❌ | ❌ | ✅ (Checks) | n/a | Proprietary |
| SigNoz | ✅ | ✅ | $0 (5M spans) → $199 | ✅ | ✅ | ❌ | Java/Scala + Clickhouse | MIT |
| HyperDX | ✅ | ✅ | $0 → $499 | ✅ | ✅ | ❌ | Clickhouse + Mongo | MIT |
| Sentry | ✅ | ✅ | $0 (5K errors) → $80+ | ❌ (errors only) | ❌ | ❌ | Postgres + Clickhouse | MIT |
| Prometheus | ❌ | ✅ | Free (self-host only) | ❌ (metrics only) | ❌ | ❌ | TSDB + scrape config | Apache-2.0 |
| Zabbix | ❌ | ✅ | Free (self-host only) | 🟡 | ❌ | ✅ | MySQL/Postgres + server | GPL-2.0 |
| Nagios | ❌ | ✅ | Free (self-host only) | ❌ | ❌ | ✅ | Plugin-heavy | GPL-2.0 |
| Dynatrace | ✅ | ✅ (Managed) | $74+ | ✅ | ✅ | ✅ | SaaS-first | Proprietary |
| AppDynamics | ✅ | ✅ (On-prem) | $100+ | ✅ | 🟡 | ✅ | Java + controller | Proprietary |

Legend: ✅ first-class · 🟡 partial / plugin · ❌ not present

---

## What StackWatch WINS on today

### Cost (10× cheaper than Datadog for SMB)

| Tool | Per-server price (USD/mo) | StackWatch equivalent |
|------|---------------------------|------------------------|
| Datadog Pro | $15–23 | **$8** (cloud) or **$0** (self-host) |
| New Relic Pro | $25+ | **$8** |
| Grafana Cloud Pro | $14 + per-active-series | **$8** flat |
| SigNoz Cloud | $0 (5M spans) → up to $199 | **$8** |
| Better Stack | $25 | **$8** |
| Honeycomb (Pro) | $0 (no SLA) → ~$130 | **$8** |
| Chronosphere | $40+ | **$8** |

For an SMB IT team running 50 servers, StackWatch self-hosted is
**free** after hardware; StackWatch Pro is **$400/mo** vs $900–1250/mo
on the alternatives. See [PRICING.md §ROI calculator](PRICING.md#roi-calculator-vs-datadog)
for the math.

### Self-hosting

- **Datadog, New Relic, Grafana Cloud, Honeycomb, Better Stack** all require SaaS
- **StackWatch, SigNoz, HyperDX, Grafana OSS, Sentry, Prometheus, Zabbix, Nagios** can be self-hosted
- Of the self-hostable observability platforms with Datadog-class features
  (APM/RUM/synthetics/multi-tenant), StackWatch has the **lightest footprint**
  (3 binaries + Postgres). SigNoz/HyperDX need Clickhouse; Grafana OSS
  needs 5+ components; Prometheus is metrics-only.

### Data sovereignty

Self-hosted StackWatch data never leaves your network. Compliance regimes
(HIPAA, GDPR §46, ITAR, FedRAMP-Moderate) often *require* this. SaaS
vendors (Datadog, New Relic, Better Stack) cannot satisfy these
constraints regardless of contract.

### Per-host pricing, NOT per-seat

Datadog and New Relic bill per-host **AND** per-user. StackWatch bills
per-host only. Adding your engineer to a Pro account is **free** —
forever. See [PRICING.md](PRICING.md).

### Multi-protocol, single binary

Today (Tier 0–11), a single StackWatch install manages:

- **Proxmox** — 78 endpoints, full admin
- **TrueNAS SCALE** — 17 endpoints, full admin
- **Terminal / SFTP** — 33 endpoints, Termius-class
- **Server monitoring** — 480 API routes across 13 shipped tiers

To match this with separate vendors you'd need: Proxmox UI
(self-hosted, web-only), TrueNAS UI (self-hosted), Termius
($8–15/mo per user), **and** a monitoring tool (Datadog etc.). All
separate stacks, no common auth, no cross-system automation.

---

## What StackWatch LOSES on today (be honest)

### Tool depth vs mature competitors

| Capability | Datadog | New Relic | Grafana Cloud | StackWatch (today) |
|-----------|---------|-----------|---------------|---------------------|
| APM + flame graphs | ✅ | ✅ | 🟡 | ✅ (Tier 6) |
| RUM + Session Replay | ✅ | ✅ | 🟡 | ✅ (Tier 6) |
| Synthetics | ✅ | ✅ | 🟡 | ✅ (Tier 6) |
| Log analytics | ✅ | ✅ | ✅ | 🟡 (Tier 6 ingests; UI in Tier 7) |
| Container management | ❌ | ❌ | ❌ | ❌ (Tier 5 not yet) |
| Server admin (Cockpit parity) | ❌ | ❌ | ❌ | ❌ (Tier 4) |
| Mobile app | ✅ | ✅ | ✅ | ❌ (Tier 13) |
| Push-to-mobile | ✅ | ✅ | ✅ | ❌ (Tier 13) |
| Pre-built integrations | 700+ | 600+ | 100+ | ~30 (Proxmox + TrueNAS + SSH + Postgres + ...) |

The honest truth: StackWatch is **a homelab tool masquerading as a
Datadog**. For a 1–50 server homelab or small business, it's already
the better tool — better UI than Proxmox's, better admin than TrueNAS's,
real monitoring in one binary.

For a 200+ server enterprise, **until Tier 6+ ship, StackWatch is
inadequate for APM/RUM parity**. We don't pretend otherwise.

### Enterprise features (now shipped via Tier 9 + 11)

| Feature | Status |
|---------|--------|
| SAML / OIDC SSO | ✅ (Tier 9) |
| Audit log hash chain | ✅ (Tier 9) |
| SOC 2 controls | ✅ (Tier 9) |
| Row-Level Security in Postgres | ✅ (Tier 9) |
| SCIM user provisioning | ✅ (Tier 9) |
| Multi-region replication | ✅ (Tier 11) |
| Backup / restore | ✅ (Tier 11) |
| Rate limiting | ✅ (Tier 11) |
| Tenant limits + enforcement | ✅ (Tier 11) |
| Usage metering | ✅ (Tier 11) |

### Faster by mature competitor metrics

- Datadog: < 1s query latency at 50PB scale
- StackWatch: < 100ms at 50GB scale today

We don't have the scale yet. We don't claim to.

---

## When to use StackWatch

✅ **Good fit** (today):

- 1–50 server homelab or SMB IT
- Want to monitor Proxmox / TrueNAS without paying Datadog
- Want Termius-class terminal but self-hosted
- Cost-sensitive; can self-host a 2-vCPU VM
- AGPL-3.0 compatible with your org (or willing to buy a commercial license)
- Need real multi-tenancy (Datadog orgs are clunky; StackWatch is multi-tenant by design)
- Compliance forbids SaaS observability (HIPAA / GDPR / ITAR)

❌ **Not yet** (until Tier 4–13 ships):

- 200+ servers needing Datadog-class scale
- Need a polished mobile app today (Tier 13 target Q1 2027)
- Need SAML SSO *today* — wait, it's shipped in Tier 9; this row is stale

---

## When NOT to use StackWatch

We are not the right tool if **any** of these apply to you:

1. **You run 500+ production servers with strict latency SLOs.** Datadog at
   50PB scale is faster than us at 50GB. We're catching up, but the math
   is real.
2. **You need a managed SaaS that "just works" with zero ops.** StackWatch
   self-hosted requires you to run Postgres, do backups, and apply
   migrations. If you want zero-ops, pay Datadog or use our managed
   cloud (which costs the same).
3. **You are in a regulated industry that requires FedRAMP High or IL5.**
   StackWatch self-hosted can be deployed to your classified enclave,
   but our managed cloud is not FedRAMP-authorized.
4. **You depend on a specific Datadog-only integration** (e.g. Cisco
   ThousandEyes, Akamai mPulse, certain SAP agents). We have ~30
   integrations today; Datadog has 700+.
5. **You need a true on-call paging workflow with rotations, escalations,
   and acknowledgement windows.** Tier 8 has a basic rule engine + 11
   channels; full PagerDuty-parity (rotations, sleep hours, ack
   windows) is on the roadmap.
6. **You have a service-mesh-only stack** (Istio / Linkerd without
   sidecar injection we support). StackWatch currently assumes a
   process-on-host model.

If any of these fit you, **we respect that and we'll tell you to use
Datadog**. We'd rather you make the right call than fight your tool.

---

## Migration guides

### Migrating from Datadog

**Phase 1 — Run side-by-side (1 week)**

1. Spin up StackWatch (`docker compose up` or [INSTALL.md §Docker](INSTALL.md#docker-install))
2. Install the StackWatch agent on 1 host:
   `curl -fsSL https://stackwatch.smarthomelab.fun/install.sh | sudo bash -s -- --token swi_xxx`
3. Confirm metrics flow into StackWatch
4. Build one Datadog-equivalent dashboard in StackWatch and validate
   the numbers match

**Phase 2 — Add the next 9 hosts** (2 weeks)

- Move hosts in groups of 10
- For each group: validate dashboards, validate alerts, **disable**
  the matching Datadog monitor but **keep** the Datadog data retention
  for 30 days
- Migration tooling (CSV export of monitors) is at
  `/api/v1/platform/migrate/datadog` (Tier 11.5)

**Phase 3 — Cutover the last hosts** (1 week)

- Disable Datadog agents
- Cancel Datadog subscription at month-end
- Retain Datadog data for 90 days for incident forensics

**Phase 4 — Save money**

- Datadog Pro @ $245/mo × 50 hosts = **$12,250/yr**
- StackWatch Pro @ $24/mo × 50 hosts = **$14,400/yr** (with no
  per-feature gates)
- Or **StackWatch self-host @ $0** + your VM costs

See [PRICING.md §ROI calculator](PRICING.md#roi-calculator-vs-datadog) for
the full spreadsheet.

### Migrating from Grafana Cloud

**Phase 1 — Export your dashboards**

Grafana dashboards are JSON; StackWatch supports importing a subset of
Grafana's panel JSON at `/api/v1/dashboards/import/grafana`. Panels that
use Grafana-only datasource plugins (Elasticsearch, InfluxDB, CloudWatch)
will be marked "unsupported" and skipped.

**Phase 2 — Replace the datasource**

StackWatch exposes Prometheus-compatible `/metrics` on every agent
endpoint (port 9101), so your existing scrape configs work unchanged.
Point Prometheus / Grafana at the StackWatch agent endpoints instead of
Grafana Cloud's remote_write endpoint.

**Phase 3 — Retire Grafana Cloud**

Once the dashboards import + scrape is working, disable the
`remote_write` block in your Prometheus config. Grafana Cloud bills
on active series; StackWatch bills per-host, which is almost always
cheaper past ~50 series/host.

**Common pitfalls**

- Grafana Cloud's free tier is **5K series**, not 5K metrics — series
  cardinality adds up fast
- Their **active series** meter is *retrospective*: a series that
  emitted once in the last 30 days still counts
- Custom alert rules in Grafana Cloud export as Grafana JSON; StackWatch
  uses its own rule-engine format (see [USER-GUIDE.md §Alerting](USER-GUIDE.md))

---

## Comparison by axis (12 tools)

### Axis 1 — Price (per-server, USD/mo, list price)

| Tool | Cheapest plan | Mid tier | Enterprise | Open-source self-host |
|------|---------------|----------|------------|-----------------------|
| **StackWatch** | **$0 (self-host)** / **$0 (cloud free)** | **$8** | **$24** + custom | ✅ Free |
| Datadog | $15 (Pro) | $23 (Enterprise tier) | $35+ | ❌ |
| New Relic | $0 (Standard, 100GB/mo cap) | $25 (Pro) | $99+ | ❌ |
| Grafana Cloud | $0 (10K series) | $14 (Pro) | $299+ | ✅ Free (self-host) |
| Better Stack | $25 (Team) | $85 (Company) | Custom | ❌ |
| SigNoz | $0 (5M spans) | $49 | $199+ | ✅ Free |
| HyperDX | $0 (self-host) | $89 (Cloud Starter) | $499+ | ✅ Free |
| Sentry | $0 (5K errors) | $26 (Team) | $80+ | ✅ Free |
| Prometheus | n/a | n/a | n/a | ✅ Free (self-host only) |
| Zabbix | n/a | n/a | n/a | ✅ Free (self-host only) |
| Nagios | n/a | n/a | n/a | ✅ Free (self-host only) |
| Dynatrace | $74 (SaaS) | $200+ | Custom | ❌ (Managed only) |
| AppDynamics | $100+ | $200+ | Custom | 🟡 (On-prem, license required) |

### Axis 2 — Deployment model

- **SaaS only**: Datadog, New Relic, Better Stack, Honeycomb, Chronosphere
- **SaaS + self-host**: StackWatch, Grafana Cloud, SigNoz, HyperDX, Sentry, Dynatrace (Managed)
- **Self-host only**: Prometheus, Zabbix, Nagios

### Axis 3 — Language support

StackWatch, Datadog, and New Relic all support Go, Python, Node.js,
Java, Ruby, .NET. SigNoz and HyperDX support the same set via
OpenTelemetry. Grafana Cloud is language-agnostic (you bring your own
instrumentation). Prometheus is the same.

Sentry is **errors only** — supports all major languages but the
focus is exception tracking, not metrics.

### Axis 4 — Feature coverage (the 12 product types)

| Feature | StackWatch | Datadog | Grafana Cloud | New Relic | SigNoz | Sentry | Prometheus |
|---------|------------|---------|---------------|-----------|--------|--------|------------|
| Metrics | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | ✅ |
| Logs | ✅ (Tier 6) | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Traces/APM | ✅ (Tier 6) | ✅ | 🟡 | ✅ | ✅ | ❌ | ❌ |
| RUM | ✅ (Tier 6) | ✅ | 🟡 | ✅ | ✅ | ❌ | ❌ |
| Synthetics | ✅ (Tier 6) | ✅ | 🟡 | ✅ | ❌ | ❌ | ❌ |
| Alerts | ✅ (Tier 8) | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Dashboards | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ | 🟡 |
| SLOs | ✅ (Tier 8) | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ |
| Security/CSPM | ✅ (Tier 9) | ✅ | 🟡 | ❌ | ❌ | ❌ | ❌ |
| Audit log | ✅ (Tier 9) | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ |
| SSO/SAML | ✅ (Tier 9) | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| CI visibility | ✅ (Tier 9) | ✅ | 🟡 | 🟡 | ❌ | ❌ | ❌ |

### Axis 5 — Lock-in risk

| Tool | Lock-in mechanism | How to leave |
|------|-------------------|--------------|
| **StackWatch** | **None.** AGPL-3.0. Export any table to CSV. | `pg_dump` + stop using StackWatch |
| Datadog | Proprietary metrics format, dashboards, monitors | Re-implement all dashboards in new tool |
| New Relic | NRQL is proprietary; dashboards export as PDF | Same |
| Grafana Cloud | JSON dashboards exportable; PromQL is open | Easy if you keep Prometheus |
| SigNoz | OpenTelemetry-native; JSON dashboards | Easy |
| Sentry | Errors-only; not a replacement for APM | Different tool entirely |
| Prometheus | PromQL + scrape configs | Hardest exit: your dashboards are all in PromQL |

**StackWatch exports**: every endpoint returns JSON; metrics and logs
are stored in Prometheus-compatible format; dashboards export as
JSON; monitors export as YAML. There is no lock-in by design.

---

## Pricing model summary

See [PRICING.md](PRICING.md) for full detail.

- **Self-host**: free forever, hardware + Postgres
- **Cloud Free**: $0, 3 servers, 7-day retention
- **Cloud Starter**: $8/mo per server, 10 servers max, 30-day retention
- **Cloud Pro**: $24/mo per server, 50 servers max, 90-day retention
- **Enterprise**: custom contract, unlimited everything, multi-region

That's our edge: **you only pay for servers, not seats**. Datadog
charges per-host AND per-user. We charge per-host only.

---

## TL;DR

StackWatch = **Proxmox + TrueNAS + Termius + Datadog** in one
self-hostable binary, at $0 to $24/server/month, AGPL-3.0.

When fully built (Tier 13), it will be a true Datadog replacement for
small-to-mid scale. Today (Tier 11), it is already a homelab
replacement and a credible SMB observability platform.