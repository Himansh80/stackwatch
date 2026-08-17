# StackWatch — Reseller / OEM Guide

**Last updated:** 2026-08-17

This guide is for partners integrating StackWatch under their own brand
or as part of a managed services offering. Internal use is fully covered
under AGPL-3.0 — you do **not** need this guide if you just self-host
for your own company.

---

## What "reselling StackWatch" means

Three flavors:

1. **Managed service** — you host StackWatch on your infra, customers
   log into your instance, you bill them monthly
2. **White-label** — your branding, your customer support, your invoices
3. **OEM integration** — StackWatch is a sub-system of your product

Flavors 1 and 2 require **Tier 4 OEM** license. Flavor 3 always requires
a custom contract. Email `sales@stackwatch.io`.

---

## What you CAN'T do without a license

- ❌ Run StackWatch **as a hosted service** that you charge users for
- ❌ Strip the AGPL notice and rebrand as your own
- ❌ Re-skin the dashboard and ship under another name
- ❌ Embed StackWatch in a commercial product and sell that product

## What you CAN do without a license (AGPL-3.0)

- ✅ Self-host for your company's internal IT
- ✅ Run StackWatch as a free, no-charge tool for non-profit / community
- ✅ Copy portions of the code into other AGPL projects
- ✅ Audit, learn from, and extend for your own non-commercial use

---

## What you get with Tier 4 OEM ($500+/year or custom)

- Branded login page (logo + colors)
- Re-sellable commercial license to your customers
- Right to ship a derivative work under your own name
- Co-marketing in the StackWatch reseller directory
- Private Slack channel for priority issues
- Quarterly roadmap review calls

---

## Branding

Env vars set at deploy time:

```bash
BRAND_NAME="Your Company StackWatch"
BRAND_PRIMARY_COLOR="#1a73e8"
BRAND_LOGO_URL="https://cdn.example.com/logo.svg"
BRAND_SUPPORT_EMAIL="support@example.com"
BRAND_TERMS_URL="https://example.com/terms"
```

The dashboard reads these on each render. The login page honors
`BRAND_NAME` + `BRAND_LOGO_URL`. Invoices use `BRAND_NAME` + terms URL.

---

## Pricing your offering

You can price your offering however you want. Reasonable options:

| Bundled with | Price |
|--------------|-------|
| MSP per-host | Match StackWatch ($8/svr) |
| Full-stack service | $50-100/svr/month |
| Hardware + monitoring | Add $5-10 to hardware MSRP |

Note: we are NOT going to undercut you. Once you have a license, we will
not license the same Tier 4 to your direct competitor in your region.

---

## Sales playbook

1. **Lead with consolidation**: "Stop paying for Datadog + Termius +
   Cockpit + Proxmox UI."
2. **Lead with self-host**: "Your data never leaves your VPC."
3. **Lead with cost**: "8 dollars per server per month, all-in."

For SMB IT (5-50 servers), the math is undeniable:
- Datadog Pro: 50 × $15 = $750/mo
- New Relic Pro: 50 × $25 = $1,250/mo
- StackWatch self-hosted: $0/mo (hardware ~$20/mo amortized)
- StackWatch Pro: 50 × $8 = $400/mo

---

## Co-marketing

Resellers get:
- Listing on `stackwatch.io/partners`
- Badges: "StackWatch Authorized Reseller"
- Co-branded press releases for big deals
- Joint webinars with StackWatch engineering

---

## Support escalation

Tier 1 (your customer) → Tier 2 (your team) → **Tier 3 (StackWatch
engineering)** via your private Slack channel.

---

## What StackWatch engineering does not do

- ❌ Sell directly to your customers
- ❌ Set your pricing
- ❌ Handle your customers' billing
- ❌ Provide end-user support

---

## Renewal

Annual contract. 30 days notice to terminate.

---

## FAQ for resellers

### Q: Can I sell exclusively in my country?

Yes — exclusive regional rights are negotiable above 5,000 servers
committed.

### Q: How long is the contract?

Annual, auto-renewing.

### Q: Can I get source-only (no support)?

Yes, ~$200/year discount.

### Q: Is there a revenue share if I bring the customer directly?

Yes — up to 25% in year 1, on deals we close jointly.

### Q: Can I make modifications and not contribute them back?

Yes, with the commercial license. The AGPL forbids this.

---

For more information, email `sales@stackwatch.io`.
