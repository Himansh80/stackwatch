# GitHub Setup Instructions

**Status:** Repo `stackwatch` not yet created on GitHub. SSH key verified.

## What I need from you

### Option A: Add SSH key to existing GitHub account (RECOMMENDED)

Your SSH key is already authorized on `Himanshu7613/ios-platform-v2` (verified).

The public key is at `C:/Users/himan/.ssh/hermes_ed25519.pub`:

```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHRt2m+ESZmhfGfQ3PZUZKTUW147zDLw7wBJztVmbKMT
```

**Steps:**
1. Go to https://github.com/settings/keys
2. Click "New SSH key"
3. Title: `Hermes StackWatch`
4. Key: paste the public key above
5. Click "Add SSH key"

If `stackwatch` repo already exists under your account, I can push immediately.

### Option B: Create the repo

1. Go to https://github.com/new
2. Repository name: `stackwatch`
3. Description: `Self-hosted Datadog + Proxmox + TrueNAS + Portainer + Cockpit + Netdata + Grafana + Loki + Chrome RD + Homarr replacement. Strict modular architecture. AGPL-3.`
4. Visibility: `Private` (recommended) or `Public`
5. **DO NOT** initialize with README, .gitignore, or license (we already have them)
6. Click "Create repository"

Once the repo exists, I'll push. Verify with:
```
curl -sI https://github.com/Himanshu7613/stackwatch | head -1
```

---

## Once repo is ready

I will push with:
```
git push -u origin main
```

The first commit (11da342) has:
- PROJECT.md
- MASTER_BUILD_PLAN.md
- MODULARITY_RULES.md
- VERIFICATION_CONTRACT.md
- RESEARCH.md
- README.md
- .gitignore
- journal.md
- decisions.md
- errors.md
- docs/INDEX.md
