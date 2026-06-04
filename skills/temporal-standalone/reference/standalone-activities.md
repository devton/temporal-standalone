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

Standalone Activities is controlled by a **dynamic config key**, NOT by a gRPC `UpdateNamespace` call.
The `standaloneActivities` field in `DescribeNamespace` response is dynamically computed from the server's config at query time — it's not stored in the database.

**Config key:** `activity.enableStandalone` (default: `false`)

Source: `chasm/lib/activity/config.go` in Temporal Server v1.31.0

### Method: Dynamic Config YAML

Add to `config/temporal/dynamicconfig/docker.yaml`:

```yaml
activity.enableStandalone:
  - value: true
    constraints:
      namespace: "default"
  - value: true
    constraints:
      namespace: "<namespace-name-or-id>"
```

The key supports `NamespaceBoolSetting` — you can enable per-namespace or use empty constraints to enable globally.

After editing, restart the server:
```bash
docker restart temporal-server
```

### Why gRPC UpdateNamespace doesn't work

`UpdateNamespaceRequest` only exposes `update_info` (description, owner, data, state) and `config` — neither contains a `capabilities` field. The `capabilities` struct on `NamespaceInfo` (proto field 7) is read-only/computed server-side from dynamic config.

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
