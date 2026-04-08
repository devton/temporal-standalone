# Temporal CLI Commands Reference

## Workflow Management

### Start Workflow
```bash
# Basic start
temporal workflow start \
  --type MyWorkflow \
  --task-queue my-queue \
  --input '{"name": "test"}'

# With workflow ID
temporal workflow start \
  --type MyWorkflow \
  --task-queue my-queue \
  --workflow-id my-workflow-123 \
  --input '{"name": "test"}'

# With timeout
temporal workflow start \
  --type MyWorkflow \
  --task-queue my-queue \
  --execution-timeout 3600 \
  --run-timeout 600
```

### Execute Workflow (Wait for completion)
```bash
temporal workflow execute \
  --type MyWorkflow \
  --task-queue my-queue \
  --workflow-id my-workflow-123 \
  --input '{"name": "test"}'
```

### List Workflows
```bash
# List all open workflows
temporal workflow list --query "ExecutionStatus = 'Running'"

# List with search attributes
temporal workflow list --query "CustomerId = '12345' AND ExecutionStatus = 'Running'"

# List with pagination
temporal workflow list --query "ExecutionStatus = 'Completed'" --limit 100
```

### Describe Workflow
```bash
temporal workflow describe \
  --workflow-id my-workflow-123 \
  --namespace my-namespace
```

### Show Workflow History
```bash
# Full history
temporal workflow show --workflow-id my-workflow-123

# Follow updates
temporal workflow show --workflow-id my-workflow-123 --follow
```

### Signal Workflow
```bash
# Basic signal
temporal workflow signal \
  --workflow-id my-workflow-123 \
  --name cancel \
  --input 'true'

# Signal with start (create if not exists)
temporal workflow signal \
  --workflow-id my-workflow-123 \
  --name update-status \
  --input 'processing' \
  --signal-with-start
```

### Query Workflow
```bash
temporal workflow query \
  --workflow-id my-workflow-123 \
  --type get-status
```

### Cancel Workflow
```bash
temporal workflow cancel \
  --workflow-id my-workflow-123 \
  --namespace my-namespace
```

### Terminate Workflow
```bash
temporal workflow terminate \
  --workflow-id my-workflow-123 \
  --reason "Manual termination"
```

### Reset Workflow
```bash
# Reset to first workflow task
temporal workflow reset \
  --workflow-id my-workflow-123 \
  --type FirstWorkflowTask \
  --reason "Retry after fix"

# Reset to specific event ID
temporal workflow reset \
  --workflow-id my-workflow-123 \
  --event-id 50 \
  --reason "Retry from specific point"
```

---

## Namespace Management

### Create Namespace
```bash
temporal operator namespace create my-namespace \
  --retention 7d \
  --description "Production namespace"
```

### Describe Namespace
```bash
temporal operator namespace describe my-namespace
```

### List Namespaces
```bash
temporal operator namespace list
```

### Update Namespace
```bash
temporal operator namespace update my-namespace \
  --retention 14d
```

---

## Search Attributes

### Create Search Attribute
```bash
temporal operator search-attribute create \
  --name CustomerId \
  --type Text

# Types: Text, Keyword, Int, Double, Bool, Datetime, KeywordList
```

### List Search Attributes
```bash
temporal operator search-attribute list
```

---

## Task Queue Management

### Describe Task Queue
```bash
temporal task-queue describe \
  --task-queue my-queue \
  --namespace my-namespace
```

---

## Schedule Management

### Create Schedule
```bash
temporal schedule create \
  --schedule-id my-schedule \
  --cron "0 0 * * *" \
  --workflow-type MyWorkflow \
  --task-queue my-queue \
  --input '{"param": "value"}'
```

### Describe Schedule
```bash
temporal schedule describe --schedule-id my-schedule
```

### Update Schedule
```bash
temporal schedule update \
  --schedule-id my-schedule \
  --cron "0 */6 * * *"
```

### Delete Schedule
```bash
temporal schedule delete --schedule-id my-schedule
```

---

## Server Management

### Start Development Server
```bash
# Basic
temporal server start-dev

# With UI
temporal server start-dev --ui-port 8080

# With persistence
temporal server start-dev --db-filename /data/temporal.db

# With custom port
temporal server start-dev --port 7234
```

---

## Batch Operations

### Start Batch Operation
```bash
temporal batch start \
  --query "ExecutionStatus = 'Running' AND StartTime < '2024-01-01'" \
  --reason-type Terminate \
  --reason "Cleanup old workflows"
```

### Describe Batch Job
```bash
temporal batch describe --job-id batch-job-123
```

---

## Environment Configuration

```bash
# Set environment
temporal env set default.address localhost:7233
temporal env set default.namespace my-namespace

# Set TLS
temporal env set default.tls.cert-path /path/to/cert
temporal env set default.tls.key-path /path/to/key
```

---

## Useful Query Examples

```bash
# Find stuck workflows
temporal workflow list --query "ExecutionStatus = 'Running' AND StartTime < 'now-1d'"

# Find by customer
temporal workflow list --query "CustomerId = 'cust-123'"

# Find by type and status
temporal workflow list --query "WorkflowType = 'OrderWorkflow' AND ExecutionStatus = 'Running'"

# Find failed workflows
temporal workflow list --query "ExecutionStatus = 'Failed' AND CloseTime > 'now-1h'"
```
