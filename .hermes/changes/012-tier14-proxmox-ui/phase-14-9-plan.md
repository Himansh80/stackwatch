# Tier 14 Phase 14.9 Plan

## Frontend (4 new pages)

### ProxmoxUsers.tsx (~280 LOC)
- Table of users
- "+ Create user" form
- Edit dialog
- Delete confirm
- Show 2FA / enabled / role chips

### ProxmoxUserTokens.tsx (~200 LOC)
- Sub-page for one user
- Table of tokens
- Create form (comment, expire, privsep)
- Show token once after creation
- Revoke

### ProxmoxPermissions.tsx (~250 LOC)
- Table of permissions
- Grant form (path, role, propagate)
- Revoke

### ProxmoxPools.tsx (~220 LOC)
- Table of pools
- Create form (id, comment)
- Delete
- Member list (read-only for 14.9)

## Order
1. Users
2. Tokens (sub-page)
3. Permissions
4. Pools
5. Routes
6. Sidebar
7. tsc + build + deploy
8. Commit
