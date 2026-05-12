# Standalone Activities — Feature Guide

## What Is It?

Standalone Activities (Public Preview, Temporal 1.31+) lets you execute Activities directly via the Temporal Client — **without a Workflow wrapper**.

```bash
# Before: needed a Workflow to run an Activity
temporal workflow start --type MyWorkflow ...

# After: run Activity directly
temporal activity execute \
  --type MyActivity \
  --input '"hello"' \
  --task-queue my-queue
```

## Use Cases

- **Ad-hoc operations**: Run scripts, maintenance tasks without deploying a Workflow
- **Testing**: Quickly test Activity logic without Workflow scaffolding
- **CLI operations**: One-off commands via terminal
- **Human-triggered actions**: Manual interventions via UI or CLI

## Prerequisites

| Requirement | Our Status |
|---|---|
| Temporal Server ≥ 1.31.0 | ✅ Running 1.31.0 |
| Temporal CLI ≥ 1.7.0 | ⚠️ Need to verify |
| Namespace capability enabled | ❌ NOT enabled |
| Worker running with Activity registered | ✅ (per-namespace) |

## How to Enable

Standalone Activities is a **namespace capability**, not a dynamic config flag. It must be set via gRPC `UpdateNamespace`.

### Method 1: grpcurl (quick test)

```bash
# Install grpcurl if needed
# Then call UpdateNamespace with capability

grpcurl -plaintext \
  -d '{
    "namespace": "default",
    "updateMask": {"paths": ["postUpdateSpec"]},
    "postUpdateSpec": {
      "capabilities": {
        "standaloneActivities": true
      }
    }
  }' \
  192.168.2.68:7233 \
  temporal.api.workflowservice.v1.WorkflowService/UpdateNamespace
```

**Note**: The exact proto field path may differ. Check `temporal.api.namespace.v1.NamespaceInfo.capabilities` in the proto definitions.

### Method 2: Go script (recommended)

```go
package main

import (
    "context"
    "fmt"
    "log"

    "go.temporal.io/api/workflowservice/v1"
    "go.temporal.io/sdk/client"
)

func main() {
    c, err := client.Dial(client.Options{
        HostPort: "192.168.2.68:7233",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    // UpdateNamespace with standalone_activities capability
    ctx := context.Background()
    _, err = c.WorkflowService().UpdateNamespace(ctx, &workflowservice.UpdateNamespaceRequest{
        Namespace: "default",
        // ... capabilities config
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Standalone Activities enabled!")
}
```

### Method 3: Wait for CLI support

Temporal CLI v1.7.0+ may add a flag like:
```bash
temporal operator namespace update default --enable-standalone-activities
```

## Verification

```bash
# Check namespace capabilities
docker exec temporal-setup temporal operator namespace describe default \
  --address temporal:7233

# Should see "standaloneActivities: true" in capabilities

# Test activity execution
temporal activity execute \
  --type my-test-activity \
  --task-queue default \
  --address 192.168.2.68:7233
```

## Current Status (2026-05-12)

- Server upgraded to 1.31.0 ✅
- Schema migrated to v1.19 ✅
- Capability NOT yet enabled — blocked on finding correct gRPC field path
- Need to verify CLI version and test grpcurl approach
