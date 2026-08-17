# StackWatch — Frequently Asked Questions

**Last updated:** 2026-08-17

---

## General

### What is StackWatch?

A self-hostable monitoring + remote-admin platform that replaces a stack
of single-purpose tools (Proxmox UI + TrueNAS UI + Termius + Cockpit +
Datadog) with one binary. License is AGPL-3.0 for the OSS edition.

### Why not just use Datadog / New Relic?

- **Cost.** $8 per host per month is reasonable; Datadog Pro is $15-23
  per host. New Relic is similar. For an SMB running 50 hosts, that's
  $900-1150/month vs $400/month. Self-hosted is **free** if you have a VM.
- **Data residency.** Some compliance regimes (HIPAA, GDPR, ITAR)
  forbid sending data to a third-party SaaS. StackWatch ships your
  data home.
- **Self-hostable.** Most SaaS doesn't.

### Why AGPL?

We want to make money on **SaaS** (Tier 2 Pro) and **hardware integrations**
(Tier 4 OEM) without preventing anyone from self-hosting. AGPL prevents
SaaS competitors from forking and running a parallel service without
disclosing their changes.

If you're an enterprise and need a permissive license, contact
`sales@stackwatch.io` for a commercial license.

---

## Installation

### Do I need Postgres?

Yes. StackWatch uses Postgres 14+ for metadata + audit log + metrics
metadata. (Note: high-volume metrics may move to a TSDB in Tier 6.)

### Can I run everything in Docker?

Not yet. The Tier 4 (build plan §4.0) Docker + docker-compose stack is on
the roadmap. Currently you run two Linux binaries + Postgres.

### Can I run it without Postgres?

No — every tier assumes Postgres. If you need SQLite or embedded,
self-host a single Tiny tier setup and add your own Postgres.

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

### What is the agent?

A single Go binary (~10 MB) on each monitored host. It listens on
`127.0.0.1:9101` (or `0.0.0.0:9101` for remote access), reports metrics
every 30s, and exposes `/health` + `/fs` + `/exec` for the gateway to query.

### Why is my agent "down" but SSH works?

Two reasons typically:
- **Bind address**: agent defaults to loopback. Set `-listen 0.0.0.0:9101`
  for remote API access.
- **Ingest key revoked**: the api-key was rotated but the installer copy
  still uses the old key. Re-run `install` from the dashboard.

---

## Multi-tenant

### Can I run StackWatch for multiple customers?

Yes — one instance of the binary can hold many tenants. Each tenant
sees only their data; you (the operator) can `su` to view any tenant
via `super_admin` bypass.

### Can I white-label?

Yes (Tier 4 OEM). Subject to contract. Branding is set at deploy
time via env (`BRAND_NAME`, `BRAND_PRIMARY_COLOR`, `BRAND_LOGO_URL`).

---

## Roadmap

### When is mobile?

Tier 13. Rough target: Q1 2027.

### When is alerting?

Tier 8 (Intelligence & Alerting). Already partially built for the
agent-installed servers; full rule engine ships in Tier 8.

### When is push-to-mobile?

Tier 13.MO2. Firebase Cloud Messaging + Apple Push Notification Service.
Rough target: Q1 2027.

---

## Troubleshooting

### "Permission denied" trying to run binary

`chmod +x /opt/stackwatch/bin/api-gateway-linux`

### Login returns 401

Check JWT secret is set (`echo $IOS_SECRET_KEY`). If unset, every restart
invalidates all existing tokens.

### "Connection refused" on remote agent

Agent binds loopback by default. Restart with `-listen 0.0.0.0:9101`.

---

## License / commercial

### Can I resell?

Tier 4 OEM contract grants resale rights. Self-hosting for your own
company doesn't require a contract (AGPL).

### Can I get a commercial license?

Yes — contact `sales@stackwatch.io`.

### Do you offer SLA support?

Self-Hosted Business tier ($500/year) includes 24h email + chat SLA.
Cloud Pro ($8/host/month) includes the same.
