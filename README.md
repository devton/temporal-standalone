# Temporal Standalone

A production-ready Temporal server setup using Docker Compose with PostgreSQL persistence.

## Services

| Service | Image | Port | Description |
|---------|-------|------|-------------|
| **PostgreSQL** | postgres:13 | 5432 | Primary database for Temporal |
| **Temporal Server** | temporalio/auto-setup:1.24.3 | 7233 | Temporal server with auto-setup |
| **Temporal UI** | temporalio/ui:2.31.0 | 8080 | Web UI for workflow management |

## Quick Start

```bash
# Start all services
make up

# Check status
make status

# View logs
make logs

# Stop all services
make down
```

## Access

- **Temporal UI**: http://localhost:8080
- **Temporal Server**: localhost:7233 (gRPC)
- **PostgreSQL**: localhost:5432

## Default Credentials

| Service | Username | Password |
|---------|----------|----------|
| PostgreSQL | temporal | temporal |
| Temporal UI | - | - (no auth) |

## Prerequisites

- Docker Engine 20.10+
- Docker Compose v2.0+

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Temporal UI   │────▶│ Temporal Server │────▶│   PostgreSQL    │
│   (port 8080)   │     │   (port 7233)   │     │   (port 5432)   │
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

## Usage with Temporal CLI

```bash
# Install temporal CLI
go install go.temporal.io/sdk/tools/tctl@latest

# Configure to use local server
export TEMPORAL_CLI_ADDRESS=localhost:7233

# List namespaces
temporal operator namespace list

# Start a workflow (example)
temporal workflow start --task-queue my-task-queue --type MyWorkflow
```

## Development

### Connecting from your application

```go
package main

import (
    "go.temporal.io/sdk/client"
)

func main() {
    c, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        panic(err)
    }
    defer c.Close()
    // Use client...
}
```

### Python SDK

```python
from temporalio.client import Client

async def main():
    client = await Client.connect("localhost:7233")
    # Use client...
```

## Data Persistence

Data is persisted in a Docker volume `postgres_data`. To reset:

```bash
make down-v  # Removes volumes
make up      # Fresh start
```

## Makefile Commands

| Command | Description |
|---------|-------------|
| `make up` | Start all services |
| `make down` | Stop all services |
| `make down-v` | Stop and remove volumes |
| `make logs` | View all logs |
| `make status` | Show container status |
| `make ps` | Alias for status |
| `make restart` | Restart all services |

## Troubleshooting

### Container health checks failing

```bash
# Check logs
docker compose logs temporal

# Restart services
make restart
```

### Connection refused

Wait for all services to be healthy (usually 30-60 seconds on first start).

### Database errors

The auto-setup image handles schema migrations automatically. If issues persist:

```bash
make down-v
make up
```

## Version Compatibility

| Component | Version |
|-----------|---------|
| Temporal Server | 1.24.3 |
| Temporal UI | 2.31.0 |
| PostgreSQL | 13 |

## License

MIT
