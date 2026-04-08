# Local Development Setup for Temporal

## Docker Compose Quick Start

```yaml
# docker-compose.yml
version: "3.8"

services:
  postgresql:
    image: postgres:13
    environment:
      POSTGRES_USER: temporal
      POSTGRES_PASSWORD: temporal
      POSTGRES_DB: temporal
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  temporal:
    image: temporalio/auto-setup:1.24.3
    depends_on:
      - postgresql
    environment:
      - DB=postgresql
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgresql
    ports:
      - "7233:7233"
    volumes:
      - ./config/dynamicconfig.yaml:/etc/temporal/config/dynamicconfig.yaml

  temporal-ui:
    image: temporalio/ui:2.27.0
    depends_on:
      - temporal
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
    ports:
      - "8080:8080"

volumes:
  postgres_data:
```

## Temporal CLI (Recommended)

```bash
# Install CLI
go install github.com/temporalio/cli/temporal@latest

# Start with UI
temporal server start-dev --ui-port 8080

# With PostgreSQL persistence
temporal server start-dev \
  --db-port 5432 \
  --db-user temporal \
  --db-password temporal \
  --db-name temporal

# With custom config
temporal server start-dev --config ./config
```

## Namespace Setup

```bash
# Create namespace
temporal operator namespace create my-namespace \
  --retention 7d \
  --description "Production namespace"

# Register search attributes
temporal operator search-attribute create \
  --namespace my-namespace \
  --name CustomerId \
  --type Text
```

## Worker Setup Template

```go
package main

import (
    "context"
    "log"
    
    "go.temporal.io/sdk/client"
    "go.temporal.io/sdk/worker"
)

func main() {
    // Connect to Temporal
    c, err := client.Dial(client.Options{
        HostPort:  "localhost:7233",
        Namespace: "default",
    })
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer c.Close()
    
    // Create worker
    w := worker.New(c, "my-task-queue", worker.Options{
        MaxConcurrentActivityExecutionSize: 100,
        MaxConcurrentWorkflowTaskExecutionSize: 100,
    })
    
    // Register workflows and activities
    w.RegisterWorkflow(MyWorkflow)
    w.RegisterActivity(MyActivity)
    
    // Start worker
    if err := w.Run(worker.InterruptCh()); err != nil {
        log.Fatalf("Worker failed: %v", err)
    }
}
```

## Environment Variables

```bash
# .env
TEMPORAL_ADDRESS=localhost:7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=my-task-queue
```
