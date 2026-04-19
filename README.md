# Temporal Standalone

Complete Temporal environment with PostgreSQL, Casdoor (OIDC), and UI.

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/new?repo=github.com/devton/temporal-standalone)

## Services

| Service | Port | URL |
|---------|------|-----|
| Temporal Server | 7233 | `localhost:7233` |
| Temporal UI | 8080 | http://localhost:8080 |
| Casdoor (OIDC) | 8000 | http://localhost:8000 |
| PostgreSQL | 5432 | `localhost:5432` |

## Quick Start

1. Copy the example file and adjust if needed:

```bash
cp .env.example .env
# Edit .env to change TEMPORAL_HOST if you need to access via IP
```

2. Start the services:

```bash
docker compose up -d
```

Wait for all services to become healthy:

```bash
docker ps --format "table {{.Names}}\t{{.Status}}"
```

## Configure OIDC (Casdoor)

### Option A: Automatic Script

Run the setup script:

```bash
./scripts/setup-casdoor.sh
```

The script automatically creates:
- Organization `temporal`
- Application `temporal-ui` with Client ID/Secret
- Test user `testuser` with password `Temporal123!`

### Option B: Manual Configuration

Access Casdoor Admin UI and configure manually.

#### 1. Access Casdoor Admin

Open http://localhost:8000

- **User:** admin
- **Password:** 123
- **Organization:** built-in

#### 2. Create Organization

1. Go to **Organizations** → **Add**
2. Fill in:
   - **Name:** `temporal`
   - **DisplayName:** Temporal
   - **Website:** `http://localhost:8080`
3. Click **Save**

#### 3. Create Application

1. Go to **Applications** → **Add**
2. Fill in:
   - **Organization:** temporal
   - **Name:** `temporal-ui`
   - **Client ID:** `temporal-ui`
   - **Client secret:** `temporal-ui-secret`
   - **Redirect URLs:**
     - `http://localhost:8080/auth/callback`
     - `http://localhost:8080`
   - **Token format:** JWT
   - **Expire in hours:** 168 (7 days)
   - **Grant types:** authorization_code, refresh_token
3. Click **Save**

#### 4. Create Test User

1. Go to **Users** → **Add**
2. Fill in:
   - **Organization:** temporal
   - **Name:** `testuser`
   - **Password:** `Temporal123!`
   - **Email:** `testuser@temporal.local`
   - **Email verified:** true
3. Click **Save**

#### 5. Test Login

1. Access http://localhost:8080
2. You will be redirected to Casdoor
3. Login with:
   - **Organization:** temporal
   - **Username:** testuser
   - **Password:** `Temporal123!`
4. After login, you will be redirected back to the UI

## Namespaces

The `default` namespace is configured with:

- **Retention:** 720h (30 days)
- **History Archival:** Enabled (`file:///tmp/temporal_archival/development`)
- **Visibility Archival:** Enabled (`file:///tmp/temporal_vis_archival/development`)

### Verify Namespace

```bash
docker exec temporal-server temporal operator namespace describe default
```

### Update Namespace

```bash
# Enable archival (already done by setup)
docker exec temporal-server temporal operator namespace update default \
  --history-archival-state enabled \
  --visibility-archival-state enabled \
  --retention 720h
```

## Using with CLI

```bash
# Without auth (server without auth enabled)
temporal workflow list --address localhost:7233

# List namespaces
temporal operator namespace list --address localhost:7233

# Execute test workflow
temporal workflow execute \
  --address localhost:7233 \
  --namespace default \
  --task-queue test \
  --type test \
  --input '"hello"'
```

## Using with SDK

### Go

```go
import "go.temporal.io/sdk/client"

func main() {
    c, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()
    
    // Use client...
}
```

### Python

```python
from temporalio.client import Client

async def main():
    client = await Client.connect(
        "localhost:7233",
        namespace="default"
    )
    
    # Use client...
```

### TypeScript

```typescript
import { Client } from '@temporalio/client';

const client = new Client({
  address: 'localhost:7233',
  namespace: 'default',
});

// Use client...
```

## Archival

Archived workflows are stored in Docker volumes:

- **History:** `archive_data` → `/tmp/temporal_archival`
- **Visibility:** `archive_vis_data` → `/tmp/temporal_vis_archival`

### Verify Archival

```bash
# List files in container
docker exec temporal-server ls -la /tmp/temporal_archival/development/
docker exec temporal-server ls -la /tmp/temporal_vis_archival/development/
```

### MinIO (S3 Backend)

To use MinIO as S3 backend:

1. Uncomment `minio` and `minio-init` sections in `docker-compose.yml`
2. Update `dynamicconfig/docker.yaml`:

```yaml
history.archival:
  - value:
      enableReadFromArchival: true
      enableWriteToArchival: true
      uri: "s3://temporal-archive/"
      s3:
        endpoint: "minio:9000"
        accessKey: "temporal"
        secretKey: "temporal_archive_secret"
        region: "us-east-1"
        usePathStyleAccess: true
```

## Manual JWT Token

To generate a JWT token manually (for CLI testing):

```bash
./scripts/generate-jwt.sh
```

The token will be saved to `config/jwt/token.txt`.

### Use Token

```bash
export TEMPORAL_CLI_AUTH_TOKEN=$(cat config/jwt/token.txt)
temporal workflow list --address localhost:7233
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Docker Network                           │
│                    temporal-network                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐│
│  │  PostgreSQL  │  │   Casdoor    │  │   Temporal Server    ││
│  │    :5432     │  │    :8000     │  │       :7233          ││
│  │  (database)  │  │    (OIDC)    │  │   (auto-setup)       ││
│  └──────────────┘  └──────────────┘  └──────────────────────┘│
│                                            ┌──────────────────┤│
│                                            │  Temporal UI     ││
│                                            │     :8080       ││
│                                            │  (OIDC client)  ││
│                                            └──────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Troubleshooting

### UI doesn't redirect to login

1. Check if Auth is enabled:
   ```bash
   curl http://localhost:8080/api/v1/settings | jq .Auth
   ```
2. Check logs: `docker logs temporal-ui`
3. Restart UI: `docker compose restart temporal-ui`

### "discovery failed" error on login

1. Check if Casdoor is accessible:
   ```bash
   curl http://localhost:8000/.well-known/openid-configuration
   ```
2. Check if the application exists in Casdoor
3. Verify Client ID/Secret are correct

### Casdoor doesn't start

1. Check if `casdoor` database was created:
   ```bash
   docker exec temporal-postgres psql -U postgres -l | grep casdoor
   ```
2. If not, create it manually:
   ```bash
   docker exec temporal-postgres psql -U postgres -c "CREATE DATABASE casdoor;"
   ```

### Server doesn't start

1. Check logs:
   ```bash
   docker logs temporal-server 2>&1 | tail -50
   ```
2. Common issues:
   - Database not ready: wait longer
   - Invalid JWT key source: check Casdoor URL
   - Auth enabled without complete configuration: disable auth on server

### Archived workflow doesn't appear

1. Check if archival is enabled on the namespace
2. Check if the workflow was closed longer than retention time
3. Check archival files in the container

## Advanced Configuration

### Enable Auth on Server

To enable authentication on the Temporal server (not just UI):

1. Update `dynamicconfig/docker.yaml`:

```yaml
frontend.auth:
  - value:
      jwtKeySource:
        - keySourceKind: "jwks"
          url: "http://casdoor:8000/.well-known/jwks"
      claimMapper:
        name: "jwt"
      authorizer:
        name: "default"
```

2. Add to `docker-compose.yml`:

```yaml
environment:
  - TEMPORAL_JWT_KEY_SOURCE1=http://casdoor:8000/.well-known/jwks
```

**Warning:** This may block internal system workflows. Use with caution.

### Multiple Namespaces

```bash
# Create production namespace
temporal operator namespace create production \
  --retention 720h \
  --history-archival-state enabled \
  --visibility-archival-state enabled
```

### PostgreSQL Backup

```bash
# Backup
docker exec temporal-postgres pg_dump -U temporal temporal > backup_temporal.sql
docker exec temporal-postgres pg_dump -U temporal temporal_visibility > backup_visibility.sql

# Restore
cat backup_temporal.sql | docker exec -i temporal-postgres psql -U temporal
cat backup_visibility.sql | docker exec -i temporal-postgres psql -U temporal
```

## License

MIT

## Network Note

By default, all services use `localhost`. To access from another machine on the network:

1. Copy the environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` and change `TEMPORAL_HOST`:
   ```env
   TEMPORAL_HOST=192.168.1.100
   ```

3. Restart services:
   ```bash
   docker compose down && docker compose up -d
   ```

All URLs (OIDC, callbacks, CORS) will be automatically adjusted.
