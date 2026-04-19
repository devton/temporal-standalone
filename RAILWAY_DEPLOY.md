# Temporal Standalone - Railway Deployment

One-click deploy of Temporal + Custom UI + Casdoor OIDC on Railway.

## Architecture

```
Railway PostgreSQL Plugin
        │
        ├── casdoor (OIDC Identity Provider + auto-seed)
        ├── temporal (Server with auto-setup/schema migration)
        ├── temporal-setup (One-shot namespace config)
        └── temporal-ui (Custom UI with API Keys feature)
```

## One-Click Deploy

[Click here to deploy on Railway](https://railway.app/new)

Or manually: Create a new project from this repo, select `railway-compose` branch.

## Step-by-Step Setup

### 1. Create Railway Project

1. Go to [railway.app/new](https://railway.app/new)
2. Click **"Deploy from GitHub repo"**
3. Select `devton/temporal-standalone`
4. Choose the `railway-compose` branch
5. Railway will detect `docker-compose.railway.yml`

### 2. Add PostgreSQL Plugin

1. In your Railway project, click **"New"** → **"Database"** → **"PostgreSQL"**
2. Railway creates a PostgreSQL instance and injects `DATABASE_URL`, `PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`

### 3. Set Environment Variables

In the Railway project settings, add these variables:

**Required:**

| Variable | Description | Example |
|----------|-------------|---------|
| `TEMPORAL_HOST` | Your public domain or Railway URL | `temporal.up.railway.app` |
| `JWT_SECRET` | Secret for API key signing | Random 32+ char string |

**Optional (defaults provided):**

| Variable | Default | Description |
|----------|---------|-------------|
| `CASDOOR_CLIENT_ID` | `temporal-ui` | Casdoor app client ID (auto-seeded) |
| `CASDOOR_CLIENT_SECRET` | `temporal-ui-secret` | Casdoor app client secret (auto-seeded) |
| `CASDOOR_ADMIN_PASSWORD` | `123` | Casdoor admin password |
| `POSTGRES_PASSWORD` | (from plugin) | Auto-provided by Railway PostgreSQL |

### 4. Configure Public URLs

After deploy, Railway assigns a public URL to each service. Update these:

1. **Casdoor origin**: Set `CASDOOR_ORIGIN` to your Casdoor service URL
2. **UI callback**: Set `TEMPORAL_UI_CALLBACK` to `https://your-ui-url.railway.app/auth/sso/callback`
3. **TEMPORAL_HOST**: Set to your UI service URL (without protocol)
4. **JWT source**: The Temporal server needs Casdoor's JWKS URL — update if using custom domain

### 5. Casdoor Auto-Seed

On first boot, Casdoor automatically:
- Creates the `temporal` organization
- Creates the `temporal-ui` application with OIDC config
- Creates a test user (`testuser` / `Temporal123!`)

Check Casdoor logs for the generated Client ID and Secret, then set them as `CASDOOR_CLIENT_ID` and `CASDOOR_CLIENT_SECRET` in Railway.

### 6. Access Your Temporal UI

Open your temporal-ui Railway URL. Login with:
- Organization: `temporal`
- Username: `testuser`
- Password: `Temporal123!`

## Service Details

### Casdoor (OIDC Provider)
- Image: `casbin/casdoor:v1.693.0` with auto-seed overlay
- Auto-creates org, app, and test user on first boot
- Admin dashboard at port 8000 (login: admin / 123)

### Temporal Server
- Image: `temporalio/auto-setup:1.24.3` with wait-for-postgres
- Auto-runs schema migration on first boot
- Then switches to normal server operation
- Dynamic config included for archival + retention

### Temporal Setup (One-shot)
- Waits for server healthy, configures namespaces
- Runs once then exits successfully

### Temporal UI (Custom)
- Builds from `temporalio/ui` v2.48.3 + custom overlays
- Features: API Keys management, namespace isolation, OIDC auth
- Build time: ~5 minutes (Node + Go multi-stage)

## Custom Domains

To use custom domains:

1. Add a custom domain in Railway for each service
2. Update `TEMPORAL_HOST` to your UI domain
3. Update `CASDOOR_ORIGIN` to your Casdoor domain
4. Update `TEMPORAL_UI_CALLBACK` to `https://ui.yourdomain.com/auth/sso/callback`
5. In Casdoor admin, update the application's redirect URLs

## Environment Reference

All env vars are defined in `docker-compose.railway.yml`. Railway auto-injects PostgreSQL plugin vars (`PGHOST`, `PGPORT`, `PGUSER`, `PGPASSWORD`). 

For local development, use the standard `docker-compose.yml` + `docker-compose.override.yml`.
