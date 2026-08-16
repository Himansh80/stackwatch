# StackWatch — User Guide

**For:** End users (after Tier 1 ships)
**Tier 0 scope:** Authentication only — see TESTING.md

---

## What StackWatch does

StackWatch is one interface that replaces:

- **Datadog** — monitoring + APM + logs
- **Proxmox UI** — VMs / containers / storage
- **TrueNAS UI** — ZFS / NFS / SMB / iSCSI
- **Portainer + Watchtower** — Docker management
- **Termix + Chrome Remote Desktop** — SSH/RDP/VNC
- **Cockpit** — server admin
- **Netdata + Prometheus + Grafana + Loki** — metrics/logs
- **Homarr** — homelab dashboard

Open-source. Self-hostable. Strict modular architecture.

## Tier 0 features (this is what's live)

### Authentication

- Sign up at `/signup` (Tier 0: via curl only; UI ships Tier 1)
- Log in at `/login`
- JWT token (24h TTL)
- Change password
- Log out

### What ships next (Tier 1 — Proxmox replacement)

- Add a Proxmox host (single click)
- See all VMs / LXC across all hosts in one list
- Start/stop/reboot/delete VMs
- Manage storage (ZFS/LVM/Ceph)
- Configure networking (bridges/VLANs/bonds)
- Backup + restore

## How to use (Tier 0 — auth only)

### Login

1. Open the web UI (after Tier 1 ships: https://your-domain/)
2. Enter email + password
3. Click "Sign in"

### Sign up (admin only — to invite new users)

```bash
# Server admin creates new accounts
curl -X POST https://your-domain/api/v1/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"new@user.com","password":"StrongPwd2026","full_name":"New User","tenant_name":"Org Name"}'
```

### Forgot password

Tier 0: returns `ok` but doesn't send email yet.
Tier 1+: integrates with Resend SMTP — real email sent within 30s.

### Change your password

Once logged in:
1. Click your avatar (top right) → "Change password"
2. Enter old + new (8+ chars)
3. Save

You stay logged in, but next login uses the new password.

### Sign out

Click avatar → "Sign out". JWT is discarded locally.

## Roles (planned)

| Role | What they can do |
|------|------------------|
| `super_admin` | Everything. Platform owner. |
| `admin` | Full access to their tenant. |
| `operator` | Manage servers + alerts, can't change billing. |
| `viewer` | Read-only access. |

Tier 0 ships `admin` role only. Other roles ship with their tiers.

## Common questions

**Q: I forgot my password. What do I do?**
A: Tier 0 has no self-service reset (returns ok but no email). Ask your admin to create a new account, or use psql to reset: `UPDATE users SET password_hash = '<bcrypt>', must_change_password = true WHERE email = '...'`.

**Q: How long does my login last?**
A: 24 hours. After that, log in again.

**Q: Can I use StackWatch from my phone?**
A: Tier 13 ships the mobile app (iOS + Android). Until then, the web UI works on mobile browsers but isn't optimized.

**Q: Is my data secure?**
A: Yes — JWT tokens, bcrypt password hashing, tenant isolation. Tier 9 ships SOC 2-grade features (RBAC, audit logs, encryption at rest).

## Keyboard shortcuts (Tier 0 — minimal)

| Shortcut | Action |
|----------|--------|
| `Ctrl+Enter` | Submit form |
| `Esc` | Cancel modal |

Tier 1+ will add global shortcut bar.

## Where to get help

- **Issues:** GitHub Issues (when repo is public)
- **Docs:** `docs/` folder in the repo
- **Live system:** http://192.168.0.115:8080/health

---

## Roadmap

| Tier | What | Status |
|------|------|--------|
| 0 | Auth + base structure | ✅ Shipped |
| 1 | Proxmox UI replacement | 🔄 Next |
| 2 | TrueNAS UI replacement | ⏳ |
| 3 | RDP/VNC/SSH (Chrome RD + Termix) | ⏳ |
| 4 | Server admin (Cockpit) | ⏳ |
| 5 | Container mgmt (Portainer + Watchtower) | ⏳ |
| 6 | Metrics depth (Netdata + Prom + Grafana + Loki) | ⏳ |
| 7 | Datadog full platform | ⏳ |
| 8 | Intelligence & alerting | ⏳ |
| 9 | Security & enterprise | ⏳ |
| 10 | Homelab dashboard (Homarr) | ⏳ |
| 11 | Platform & commerce | ⏳ |
| 12 | Docs & GTM | ⏳ |
| 13 | Mobile apps | ⏳ |
