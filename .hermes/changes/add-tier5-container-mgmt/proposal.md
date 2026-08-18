# Proposal: Tier 5 — Container Management (Portainer + Watchtower)

## Why
StackWatch today lets you see a server's CPU/mem/disk (Tier 4) but
not what's running ON it. Operators need to:
- List containers, images, volumes, networks
- Start/stop/restart/pause/unpause containers
- Read container logs and stream stats (CPU/mem/net)
- Get an interactive shell into a container (exec)
- Create containers from images or compose stacks
- Pull / remove images
- Manage volumes and networks
- Auto-update watched containers (Watchtower-style)

## What changes
- New `ContainerHandler` with 13 endpoint groups (C1-C11 + Watchtower)
- New routes: `/api/v1/containers/*`, `/api/v1/images/*`, `/api/v1/volumes/*`,
  `/api/v1/networks/*`, `/api/v1/stacks/*`, `/api/v1/templates/*`,
  `/api/v1/watchtower/*`
- Frontend: 8 pages (Containers list, Detail, Create, Images, Volumes,
  Networks, Stacks, Templates)
- No new binary — reuse Tier 3 SSH bridge via Tier 4 web-terminal

## Impact
- Areas affected: api-gateway, web-terminal, frontend
- Breaking changes: **none** — purely additive
- Migration needed: **none** — no new tables
- Server impact: requires Docker installed on managed hosts
- Test data: docker on .113 (root, /usr/bin/docker 29.7.2)
