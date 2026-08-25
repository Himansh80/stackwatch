# StackWatch — Frequently Asked Questions

**Last updated:** 2026-08-25

Cross-references: [INSTALL.md](INSTALL.md) · [USER-GUIDE.md](USER-GUIDE.md) ·
[ARCHITECTURE.md](ARCHITECTURE.md) · [PRICING.md](PRICING.md) ·
[TROUBLESHOOTING.md](TROUBLESHOOTING.md) · [SECURITY.md](SECURITY.md).

---

## General

### What is StackWatch?

A self-hostable monitoring + remote-admin platform that replaces a
stack of single-purpose tools (Proxmox UI + TrueNAS UI + Termius +
Cockpit + Datadog + Grafana) with one binary. License is AGPL-3.0
for the OSS edition; commercial licenses available for Enterprise.

### Why not just use Datadog / New Relic?

- **Cost.** $24 per host per month (StackWatch Pro) vs $15–23
  per host **plus** per-feature gates in Datadog. Self-hosted is
  **free** if you have a VM. See [PRICING.md §ROI](PRICING.md#roi-calculator-vs-datadog).
- **Data residency.** Compliance regimes (HIPAA, GDPR, ITAR)
  forbid sending data to third-party SaaS. StackWatch ships your
  data home.
- **Self-hostable.** Most SaaS observability isn't.
- **No lock-in.** Every endpoint is JSON; every metric + log exports
  as PromQL/JSONL; every dashboard exports as JSON.

### Why AGPL?

We want to make money on **SaaS** (Starter / Pro / Enterprise) and
**hardware integrations** (OEM) without preventing anyone from
self-hosting. AGPL prevents SaaS competitors from forking and
running a parallel service without disclosing their changes.

If you're an enterprise and need a permissive license, contact
`sales@stackwatch.io` for a commercial license.

### Is StackWatch really "Datadog parity"?

Honest answer: **mostly yes, with gaps at the high end.** See the
[COMPARISON.md §What we LOSE on](COMPARISON.md#what-stackwatch-loses-on-today-be-honest)
section for the full list. The big remaining gaps:

- Mobile app (Tier 13, target Q1 2027)
- Full PagerDuty-class on-call rotations (basic rotation in Tier 8;
  full PagerDuty parity later)
- 700+ pre-built integrations (we ship ~30)

Everything else — metrics, logs, traces, RUM, synthetics, alerts,
dashboards, security, CSPM, CI visibility, SSO, audit log — is
shipped.

### Can StackWatch replace both Datadog AND Termius?

Yes. The browser SSH terminal (Tier 3) is Termius-class: real SSH
over WebSocket, SFTP file browser, known-hosts trust list, encrypted
credentials vault. It's bundled in every plan including Free.

### Is there a hosted version, or only self-host?

Both. Our managed cloud is at `https://stackwatch.smarthomelab.fun/`
and runs on the same binaries as self-host. Pick whichever fits your
ops capacity.

---

## Installation

### Do I need Postgres?

Yes. StackWatch uses Postgres 14+ for metadata + audit log + metrics
metadata. (Note: high-volume metrics may move to a TSDB in Tier 6.)

### Can I run everything in Docker?

**Yes** — see [INSTALL.md §Docker](INSTALL.md#docker-install). The
shipped `docker-compose.yml` runs api-gateway, web-terminal,
truenas-connector, and Postgres on one host. For production, run
Postgres on a separate host or use RDS/Cloud SQL.

### Can I run it without Postgres?

No — every tier assumes Postgres. If you need SQLite or embedded,
self-host a single Tiny tier setup and add your own Postgres.

### Can I run it on Kubernetes?

**Yes** — see [INSTALL.md §Kubernetes](INSTALL.md#kubernetes-install).
A Helm chart is shipped at `deploy/helm/stackwatch/`. Tested on
Kubernetes 1.27+.

### Can I run it on macOS for development?

**Yes** — see [INSTALL.md §macOS](INSTALL.md#macos-install-development).
Three `darwin` binaries build natively; postgres runs via Homebrew.

### What's the minimum hardware?

- Homelab (≤ 10 servers): **2 vCPU, 4 GB RAM, 20 GB disk**
- SMB (≤ 50 servers): **4 vCPU, 8 GB RAM, 100 GB disk**
- Enterprise (200+ servers): **16 vCPU, 32 GB RAM, 1 TB disk**

See [INSTALL.md §Hardware](INSTALL.md#hardware) for the full matrix.

### Can I install the agent on Windows?

**Yes.** Windows binaries (`api-gateway-windows.exe`,
`web-terminal-windows.exe`, `truenas-connector-windows.exe`) are
shipped in `bin/`. The agent runs as a Windows Service. Install
instructions are in [INSTALL.md](INSTALL.md).

---

## Operations

### How do I add a new server?

```bash
# Get your install command from the dashboard:
# Dashboard → Servers → Add → copy "Linux/macOS/Windows" one-liner
curl -fsSL https://stackwatch.example.com/api/v1/installers/linux/amd64/installer \
  | sudo BACKEND=https://stackwatch.example.com \
      INGEST_KEY=ios_a_xxxxxxxx bash
```

The agent pulls its own binary, configures systemd, and registers with
the api-gateway in under 60 seconds.

### What is the agent?

A single Go binary (~10 MB) on each monitored host. It listens on
`127.0.0.1:9101` (or `0.0.0.0:9101` for remote access), reports
metrics every 30s, and exposes `/health` + `/fs` + `/exec` for the
gateway to query.

### Why is my agent "down" but SSH works?

Two reasons typically:

- **Bind address**: agent defaults to loopback. Set `-listen 0.0.0.0:9101`
  for remote API access.
- **Ingest key revoked**: the api-key was rotated but the installer copy
  still uses the old key. Re-run `install` from the dashboard.

### How do I upgrade?

```bash
git pull origin main
make build
sudo systemctl restart api-gateway web-terminal truenas-connector
```

Migrations are applied automatically on boot. See
[INSTALL.md §Upgrading](INSTALL.md#upgrading).

### How do I back up?

Tier 11 PL5 ships a backup CLI:

```bash
/opt/stackwatch/bin/backup-cli snapshot --label="pre-upgrade"
# → /opt/stackwatch/backups/2026-08-25-pre-upgrade.tar.gz.enc
```

Encrypted with AES-256-GCM. Restore with `--from-snapshot`.

---

## Multi-tenant

### Can I run StackWatch for multiple customers?

**Yes** — one instance of the binary can hold many tenants. Each tenant
sees only their data; you (the operator) can `su` to view any tenant
via `super_admin` bypass.

### Can I white-label?

**Yes** (OEM tier). Subject to contract. Branding is set at deploy
time via env (`BRAND_NAME`, `BRAND_PRIMARY_COLOR`, `BRAND_LOGO_URL`).
See [PRICING.md §Self-host](PRICING.md#self-host-is-free-forever) for the
commercial terms.

### How does multi-tenant isolation work?

Three layers:

1. **Application layer**: every handler checks JWT `tenant_id` claim
   against the row's `tenant_id` (RLS pattern).
2. **Postgres RLS** (Tier 9): the `ios_rls` role can only see rows
   where `tenant_id = current_setting('app.tenant_id')`.
3. **Network layer**: each tenant's integrations (Proxmox / TrueNAS)
   are scoped to the tenant — tenant A's Proxmox token cannot reach
   tenant B's Proxmox host.

### Can I have one tenant in EU and one in US?

Yes (Enterprise). Each tenant's data plane is pinned to a region.
Cross-region replication is optional and configurable per-tenant.

---

## Security + compliance

### Is StackWatch SOC 2 compliant?

The Tier 9 release ships SOC 2 Type 1 controls. We're pursuing Type 2
in Q4 2026. Customers on Pro and Enterprise get the SOC 2 report
under NDA.

### Do you support SAML SSO?

Yes (Tier 9). All major IdPs supported: Okta, Azure AD, Google
Workspace, OneLogin, JumpCloud, Auth0. SAML 2.0 + OIDC 2.0.

### Do you support SCIM?

Yes (Tier 9). Auto-provisioning + de-provisioning from your IdP.
Tested with Okta, Azure AD, Google Workspace.

### Can I require MFA?

Yes. TOTP (Google Authenticator, Authy, 1Password) is built in. SMS
MFA on the roadmap.

### Is my data encrypted at rest?

Yes. Postgres TDE (or LUKS at the disk layer) + AES-256-GCM for
backups. TLS 1.3 for all in-transit data.

### Can I do a security audit myself?

Yes. The audit log is tamper-evident (hash-chained, Tier 9) and
exportable to CSV. You can replay any action sequence.

---

## Pricing

### How much does it cost?

[PRICING.md](PRICING.md). Summary: **Free forever** (self-host) or
**$0–$24/server/mo** (cloud), Enterprise custom.

### Can I self-host for free, forever?

**Yes.** AGPL-3.0. No metering, no call-home, no license key. The
source is open.

### What if I exceed my plan limits?

The api-gateway returns **HTTP 402 Payment Required** with
`error.code="limit_exceeded"`. We email you at 80% utilization so
you can upgrade before hitting it. Your data is **not deleted**.

### Do you charge per-seat?

**No.** Adding your engineer to a Pro account is free. Forever. We
charge per-server only. See [PRICING.md §Why we're 10× cheaper](PRICING.md#why-were-10x-cheaper-than-datadog).

### Can I get a refund?

Within 30 days of any charge, yes — full refund. After 30 days, we
pro-rate the unused portion.

### Do you have a non-profit / education discount?

Yes — **50% off** Starter and Pro. Contact `support@stackwatch.io`
with proof of status.

---

## Migration

### How do I migrate from Datadog?

See [COMPARISON.md §Migration from Datadog](COMPARISON.md#migrating-from-datadog).
Short version: run side-by-side for 1 week, add hosts in groups of 10,
cutover the last group, cancel Datadog. The Tier 11.5 migration CLI
exports monitors as YAML.

### How do I migrate from Grafana Cloud?

See [COMPARISON.md §Migration from Grafana Cloud](COMPARISON.md#migrating-from-grafana-cloud).
Short version: export dashboards as JSON, point your Prometheus
scrape config at the StackWatch agent endpoints (which expose
Prometheus-format `/metrics`), disable `remote_write`.

### How do I migrate from Prometheus + Grafana?

Easy. Install the StackWatch agent on every Prometheus-scrape target;
the agent exposes Prometheus-format metrics on port 9101. Point
Prometheus at the agent. Then import your Grafana dashboards as JSON
into StackWatch. Retire Grafana when ready.

### How do I migrate from Zabbix / Nagios?

Longer. Zabbix templates and Nagios checks are not auto-importable.
You'll rebuild alert rules + dashboards in StackWatch. We recommend
running both side-by-side for 30 days.

---

## Troubleshooting

### "Permission denied" trying to run binary

`chmod +x /opt/stackwatch/bin/api-gateway-linux`

### Login returns 401

Check JWT secret is set (`echo $IOS_SECRET_KEY`). If unset, every
restart invalidates all existing tokens.

### "Connection refused" on remote agent

Agent binds loopback by default. Restart with `-listen 0.0.0.0:9101`.

### Health check returns `db: "down"`

Postgres is unreachable. Check:

- `DATABASE_URL` is correct
- Postgres is running (`systemctl status postgresql`)
- Network ACL allows port 5432 from api-gateway host

### Metrics not showing up

Check the agent is running (`systemctl status stackwatch-agent`).
Check the agent's logs (`journalctl -u stackwatch-agent -n 50`).
The most common issue is a firewall blocking port 9101.

### Alert rule never fires

Check:

- The query is valid (test in the metrics explorer first)
- The `for:` window has elapsed (5m means 5 minutes of sustained)
- The channel is configured and verified (test the channel)

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for the full catalog.

---

## License / commercial

### Can I resell?

OEM tier contract grants resale rights. Self-hosting for your own
company doesn't require a contract (AGPL).

### Can I get a commercial license?

Yes — contact `sales@stackwatch.io`.

### Do you offer SLA support?

Yes. Pro tier: priority email + chat (8h response). Enterprise tier:
SLA-backed (1h response on P1 incidents).

### What's the AGPL actually require?

If you modify StackWatch and **operate it as a network service**,
you must publish your modifications under AGPL-3.0. Internal use
(employees using StackWatch on your own infrastructure) is exempt.
Selling StackWatch-as-a-Service requires the OEM contract.

If you're an enterprise that needs to fork StackWatch without
publishing your fork, the commercial license removes that
requirement.

---

## Where to get more help

- [INSTALL.md](INSTALL.md) — install + upgrade
- [USER-GUIDE.md](USER-GUIDE.md) — per-page tour
- [ARCHITECTURE.md](ARCHITECTURE.md) — design + data flow diagrams
- [TESTING.md](TESTING.md) — verify your install
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) — common failures
- [SECURITY.md](SECURITY.md) — security model
- [PRICING.md](PRICING.md) — plans + ROI calculator
- GitHub: https://github.com/Himanshu7613/stackwatch/issues
- Email: `support@stackwatch.io` (help) · `sales@stackwatch.io` (sales)