# Tier 14 Phase 14.9 — Specification

## 1. URL pattern
- `/proxmox/users` — users
- `/proxmox/users/:userid/tokens` — user's tokens
- `/proxmox/permissions` — permissions
- `/proxmox/pools` — pools

## 2. Users

Table: User ID | First Name | Last Name | Email | Role | Enabled | 2FA | Expires | Actions
- "+ Create user" button → form dialog
- Edit per row → form pre-filled
- Delete per row → confirm

## 3. API Tokens

Table: Token ID | Comment | Expire | Privilege Separation | Actions
- "+ Create token" button → form dialog (comment, expire, privsep)
- On create: show the **plain-text token once** (copy-to-clipboard)
- Revoke per row

## 4. Permissions

Table: Path | Role | Actions
- "+ Grant permission" button → form (path + role)
- Revoke per row

## 5. Pools

Table: Pool ID | Comment | Members | Actions
- "+ Create pool" button → form (id + comment)
- Delete per row
- "+ Add member" inline → choose VM/CT (later phase)

## 6. Acceptance criteria
As listed in proposal + standard modular/quality checks.
