# AGENTS.md - AI Agent Instructions

Instructions for AI coding agents working on the temporal-standalone project.

## Project Overview

Temporal Standalone is a complete Temporal development environment with:
- **Temporal Server** (v1.24.3) with PostgreSQL
- **Temporal UI** (v2.48.1) with custom API Keys feature
- **Casdoor** (OIDC Provider) for authentication
- **Custom UI** built via Git Submodule + Overlays approach

## Quick Start

```bash
# 1. Configure environment
cp .env.example .env
# Edit .env if accessing via network IP

# 2. Start services
docker compose up -d

# 3. Check health
docker ps --format "table {{.Names}}\t{{.Status}}"
```

## Architecture

```
temporal-standalone/
├── docker-compose.yml          # Main services
├── docker-compose.override.yml # Custom UI build
├── .env                        # Configuration (not in git)
├── .env.example               # Configuration template
├── config/
│   ├── temporal/              # Temporal server config
│   └── ui/                    # UI config (template)
├── ui-custom/                 # Custom UI source
│   ├── upstream/              # Git submodule (temporalio/ui)
│   ├── overlays/              # Custom modifications
│   └── Dockerfile.custom      # Multi-stage build
├── scripts/                   # Setup and utility scripts
└── .agents/                   # AI agent resources
    └── skills/                # Reusable agent skills
```

## Key URLs (configurable via .env)

| Service | Default URL |
|---------|-------------|
| Temporal UI | http://localhost:8080 |
| Temporal Server | localhost:7233 |
| Casdoor | http://localhost:8000 |
| PostgreSQL | localhost:5432 |

## Configuration

All configuration is via `.env` file:

```env
# Network
TEMPORAL_HOST=localhost          # Use IP for network access

# Ports
TEMPORAL_UI_PORT=8080
TEMPORAL_SERVER_PORT=7233
CASDOOR_PORT=8000

# OIDC (from Casdoor)
CASDOOR_CLIENT_ID=xxx
CASDOOR_CLIENT_SECRET=xxx

# Security
JWT_SECRET=change-in-production
```

## Custom UI Development

The custom UI uses a **Git Submodule + Overlays** approach:

### Structure

```
ui-custom/
├── upstream/              # git submodule (temporalio/ui)
├── overlays/              # Your customizations
│   ├── server/server/route/api_keys.go
│   ├── src/routes/(app)/settings/api-keys/+page.svelte
│   └── src/lib/holocene/user-menu.svelte
├── Dockerfile.custom
└── scripts/apply-overlays.sh
```

### Build Process

1. Copy `upstream/` to build context
2. Apply `overlays/*` on top
3. Build frontend (pnpm) + backend (Go)
4. Generate config from env vars at runtime

### Updating Upstream

```bash
cd ui-custom/upstream
git pull origin main
cd ../..
git add ui-custom/upstream
```

### Adding Custom Features

Create files in `ui-custom/overlays/` with same directory structure as upstream:

```bash
# Example: Add new route
mkdir -p ui-custom/overlays/src/routes/(app)/my-feature
echo "<script>...</script>" > ui-custom/overlays/src/routes/(app)/my-feature/+page.svelte
```

## API Keys Feature

The custom UI includes an API Keys management feature:

- **Backend:** `ui-custom/overlays/server/server/route/api_keys.go`
- **Frontend:** `ui-custom/overlays/src/routes/(app)/settings/api-keys/+page.svelte`
- **Menu Entry:** `ui-custom/overlays/src/lib/holocene/user-menu.svelte`

### Endpoints

- `GET /api/v1/api-keys` - List keys
- `POST /api/v1/api-keys` - Create key
- `DELETE /api/v1/api-keys/:id` - Delete key

### JWT Tokens

API keys are signed JWTs using `JWT_SECRET` from environment.

## Git Repository

```bash
# Remote
git@github.com:devton/temporal-standalone.git

# Push with SSH key
GIT_SSH_COMMAND="ssh -i ~/.ssh/temporal-standalone-git -o IdentitiesOnly=yes" git push origin master
```

## Troubleshooting

### CSRF Token Issues

If accessing via IP, set `TEMPORAL_HOST` in `.env`:

```env
TEMPORAL_HOST=192.168.2.68
```

Then restart: `docker compose down && docker compose up -d`

### UI Build Fails

```bash
# Rebuild from scratch
docker compose build --no-cache temporal-ui
```

### OIDC Login Fails

1. Verify Casdoor is running: `curl http://localhost:8000/api/get-health`
2. Check client ID/secret in `.env` matches Casdoor application
3. Verify redirect URLs in Casdoor app settings

## Skills

See `.agents/skills/temporal-superpower/` for reusable workflows:
- CLI reference
- Local development
- Production checklist
- Workflow templates
