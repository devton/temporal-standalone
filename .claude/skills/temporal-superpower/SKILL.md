---
name: temporal-superpower
description: Complete guide to Temporal workflows - from basics to advanced patterns. Covers workflows, activities, signals, queries, updates, child workflows, saga pattern, Nexus, interceptors, encryption, retry policies, scheduling, and production best practices.
version: 2.1.0
tags: [temporal, workflow, distributed-systems, durability, saga, orchestration]
languages: [go, python, typescript]
repository: git@github.com:devton/temporal-standalone.git
---

# Temporal Superpower

Master guide for building durable, scalable workflows with Temporal - from fundamentals to advanced production patterns.

## Quick Reference

| Concept | Description |
|---------|-------------|
| **Workflow** | Durable function that orchestrates activities |
| **Activity** | Unit of work that interacts with external systems |
| **Signal** | Async message sent TO workflow |
| **Query** | Sync request TO workflow (returns state) |
| **Update** | Sync request TO workflow (modifies state) |
| **Task Queue** | FIFO queue workers poll for tasks |
| **Event History** | Append-only log of workflow state |

---

## Part 1: Fundamentals

### 1.1 Core Concepts

**Temporal Platform** consists of:
- **Temporal Service** (Server + Persistence + Visibility)
- **Worker Processes** (execute your code)

**Key Properties of Workflows:**
1. **Durable** - State survives failures
2. **Reactive** - Respond to external events
3. **Scalable** - Millions of concurrent executions
4. **Deterministic** - Same input = same result (on replay)

### 1.2 Hello World - Go

```go
package main

import (
    "context"
    "go.temporal.io/sdk/client"
    "go.temporal.io/sdk/workflow"
    "go.temporal.io/sdk/activity"
    "go.temporal.io/sdk/worker"
)

// Workflow Definition
func GreetingWorkflow(ctx workflow.Context, name string) (string, error) {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute,
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    var result string
    err := workflow.ExecuteActivity(ctx, GreetActivity, name).Get(ctx, &result)
    return result, err
}

// Activity Definition
func GreetActivity(ctx context.Context, name string) (string, error) {
    return "Hello " + name + "!", nil
}

// Worker Setup
func main() {
    c, _ := client.Dial(client.Options{HostPort: "localhost:7233"})
    defer c.Close()

    w := worker.New(c, "greeting-task-queue", worker.Options{})
    w.RegisterWorkflow(GreetingWorkflow)
    w.RegisterActivity(GreetActivity)
    
    w.Run(worker.InterruptCh())
}
```

### 1.3 Hello World - Python

```python
from datetime import timedelta
from temporalio import activity, workflow
from temporalio.client import Client
from temporalio.worker import Worker

@activity.defn
def greet_activity(name: str) -> str:
    return f"Hello {name}!"

@workflow.defn
class GreetingWorkflow:
    @workflow.run
    async def run(self, name: str) -> str:
        return await workflow.execute_activity(
            greet_activity, name,
            start_to_close_timeout=timedelta(seconds=10)
        )

async def main():
    client = await Client.connect("localhost:7233")
    async with Worker(
        client, task_queue="greeting-task-queue",
        workflows=[GreetingWorkflow],
        activities=[greet_activity]
    ):
        result = await client.execute_workflow(
            GreetingWorkflow.run, "World",
            id="greeting-workflow-id",
            task_queue="greeting-task-queue"
        )
        print(result)
```

### 1.4 Hello World - TypeScript

```typescript
import { proxyActivities } from '@temporalio/workflow';
import type * as activities from './activities';

const { greetActivity } = proxyActivities<typeof activities>({
  startToCloseTimeout: '1 minute',
});

export async function greetingWorkflow(name: string): Promise<string> {
  return await greetActivity(name);
}
```

---

## Part 2: Activities

### 2.1 Activity Options

```go
ao := workflow.ActivityOptions{
    // Timeouts
    StartToCloseTimeout: time.Minute,      // Max time for single activity task
    ScheduleToCloseTimeout: 5 * time.Minute, // Total time including retries
    ScheduleToStartTimeout: time.Minute,    // Time waiting in queue
    
    // Heartbeat for long-running activities
    HeartbeatTimeout: 10 * time.Second,
    
    // Retry configuration
    RetryPolicy: &temporal.RetryPolicy{
        InitialInterval:    time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval:    time.Minute,
        MaximumAttempts:    5,
    },
    
    // Cancellation behavior
    WaitForCancellation: true,
}
ctx = workflow.WithActivityOptions(ctx, ao)
```

### 2.2 Activity Heartbeating

Essential for long-running activities to detect failures:

```go
func LongRunningActivity(ctx context.Context, input string) (string, error) {
    logger := activity.GetLogger(ctx)
    
    for i := 0; i < 100; i++ {
        // Check if activity was canceled
        if ctx.Err() == context.Canceled {
            // Perform cleanup
            return "", ctx.Err()
        }
        
        // Heartbeat to signal progress
        activity.Heartbeat(ctx, i) // Pass progress as details
        
        // Do work
        processChunk(ctx, input, i)
    }
    
    return "completed", nil
}
```

**Python heartbeat:**
```python
@activity.defn
async def long_running_activity(input_data: str) -> str:
    for i in range(100):
        activity.heartbeat(i)  # Pass progress
        await process_chunk(input_data, i)
    return "completed"
```

### 2.3 Retry Policies

```go
retryPolicy := &temporal.RetryPolicy{
    InitialInterval:    time.Second,     // First retry after 1s
    BackoffCoefficient: 2.0,             // Exponential backoff
    MaximumInterval:    time.Minute,     // Cap at 1 minute
    MaximumAttempts:    5,               // Max 5 retries
    NonRetryableErrorTypes: []string{   // Don't retry these
        "InvalidArgumentError",
        "PermissionDeniedError",
    },
}
```

---

## Part 3: Signals, Queries, Updates

### 3.1 Signals (Async Input)

Signals are async messages sent TO a workflow:

```go
// Workflow with signal handler
func OrderWorkflow(ctx workflow.Context, orderID string) error {
    var status string = "created"
    var cancelRequested bool
    
    // Set up signal channel
    cancelChannel := workflow.GetSignalChannel(ctx, "cancel-order")
    
    // Register signal handler (recommended)
    err := workflow.SetSignalHandler(ctx, "update-status", func(ctx workflow.Context, newStatus string) error {
        status = newStatus
        return nil
    })
    if err != nil {
        return err
    }
    
    // Poll for signals in main loop
    for {
        selector := workflow.NewSelector(ctx)
        selector.AddReceive(cancelChannel, func(c workflow.ReceiveChannel, more bool) {
            c.Receive(ctx, &cancelRequested)
        })
        selector.AddFuture(workflow.NewTimer(ctx, time.Second), func(f workflow.Future) {})
        selector.Select(ctx)
        
        if cancelRequested {
            status = "cancelled"
            break
        }
        // ... process order
        break
    }
    return nil
}

// Client sends signal
func cancelOrder(c client.Client, workflowID string) {
    c.SignalWorkflow(ctx, workflowID, "", "cancel-order", true)
}
```

**Python signals:**
```python
@workflow.defn
class OrderWorkflow:
    def __init__(self) -> None:
        self.status = "created"
        self.cancelled = False
    
    @workflow.run
    async def run(self, order_id: str) -> None:
        await workflow.wait_condition(lambda: self.cancelled)
        self.status = "cancelled"
    
    @workflow.signal
    def cancel_order(self) -> None:
        self.cancelled = True
    
    @workflow.signal
    def update_status(self, status: str) -> None:
        self.status = status
```

### 3.2 Queries (Sync State Lookup)

Queries return state WITHOUT modifying it:

```go
func CounterWorkflow(ctx workflow.Context) error {
    counter := 0
    
    // Register query handler
    err := workflow.SetQueryHandler(ctx, "get-counter", func() (int, error) {
        return counter, nil
    })
    if err != nil {
        return err
    }
    
    // ... workflow logic
    return nil
}

// Client query
resp, err := c.QueryWorkflow(ctx, workflowID, "", "get-counter")
var count int
resp.Get(&count)
```

**Python queries:**
```python
@workflow.defn
class CounterWorkflow:
    def __init__(self) -> None:
        self.counter = 0
    
    @workflow.run
    async def run(self) -> None:
        await workflow.wait_condition(lambda: self.counter >= 100)
    
    @workflow.query
    def get_counter(self) -> int:
        return self.counter
```

### 3.3 Updates (Sync State Modification)

Updates are synchronous - modify state AND return result:

```go
func CounterWorkflow(ctx workflow.Context) error {
    counter := 0
    
    // Register update handler with validator
    err := workflow.SetUpdateHandlerWithOptions(ctx, "increment",
        func(ctx workflow.Context, amount int) (int, error) {
            old := counter
            counter += amount
            return old, nil
        },
        workflow.UpdateHandlerOptions{
            Validator: func(ctx workflow.Context, amount int) error {
                if amount < 0 {
                    return fmt.Errorf("amount must be non-negative")
                }
                return nil
            },
        },
    )
    if err != nil {
        return err
    }
    
    // ... workflow logic
    return nil
}

// Client update
handle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
    WorkflowID:   workflowID,
    UpdateName:   "increment",
    Args:         []interface{}{5},
})
var previous int
handle.Get(ctx, &previous)
```

**Python updates:**
```python
@workflow.defn
class CounterWorkflow:
    def __init__(self) -> None:
        self.counter = 0
    
    @workflow.run
    async def run(self) -> int:
        await workflow.wait_condition(lambda: self.counter >= 100)
        return self.counter
    
    @workflow.update
    def increment(self, amount: int) -> int:
        old = self.counter
        self.counter += amount
        return old
    
    @workflow.update.validator(increment)
    def increment_validate(self, amount: int) -> None:
        if amount < 0:
            raise ValueError("amount must be non-negative")
```

---

## Part 4: Advanced Workflow Patterns

### 4.1 Child Workflows

Spawn workflows from within workflows:

```go
func ParentWorkflow(ctx workflow.Context) error {
    childOpts := workflow.ChildWorkflowOptions{
        WorkflowID:         "child-" + workflow.GetInfo(ctx).WorkflowExecution.ID,
        ParentClosePolicy:  enums.PARENT_CLOSE_POLICY_TERMINATE,
    }
    ctx = workflow.WithChildOptions(ctx, childOpts)
    
    var result string
    err := workflow.ExecuteChildWorkflow(ctx, ChildWorkflow, "input").Get(ctx, &result)
    return err
}
```

**Parent Close Policies:**
- `PARENT_CLOSE_POLICY_TERMINATE` - Kill child when parent closes
- `PARENT_CLOSE_POLICY_ABANDON` - Let child run independently
- `PARENT_CLOSE_POLICY_REQUEST_CANCEL` - Request child cancellation

**Python child workflows:**
```python
@workflow.defn
class ParentWorkflow:
    @workflow.run
    async def run(self) -> str:
        return await workflow.execute_child_workflow(
            ChildWorkflow.run, "input",
            id="child-workflow-id"
        )
```

### 4.2 Continue-As-New

Restart workflow with fresh history (for long-running workflows):

```go
func LongRunningWorkflow(ctx workflow.Context, processed int) error {
    // Process in batches to avoid history growth
    batchSize := 1000
    
    for i := 0; i < batchSize; i++ {
        // Process item
        processItem(ctx, processed + i)
    }
    
    newProcessed := processed + batchSize
    
    // Check if we should continue
    info := workflow.GetInfo(ctx)
    if info.GetCurrentHistoryLength() > 40000 || info.GetCurrentHistorySize() > 40*1024*1024 {
        return workflow.NewContinueAsNewError(ctx, LongRunningWorkflow, newProcessed)
    }
    
    // Continue processing...
    return nil
}
```

**Python:**
```python
@workflow.defn
class LongRunningWorkflow:
    @workflow.run
    async def run(self, processed: int) -> None:
        batch_size = 1000
        for i in range(batch_size):
            await self.process_item(processed + i)
        
        # Continue as new if history is large
        if workflow.info().get_current_history_length() > 40000:
            workflow.continue_as_new(processed + batch_size)
```

### 4.3 Parallel Execution (Branch)

```go
func ParallelWorkflow(ctx workflow.Context, items []string) ([]string, error) {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute,
    }
    ctx = workflow.WithActivityOptions(ctx, ao)
    
    // Start all activities in parallel
    var futures []workflow.Future
    for _, item := range items {
        future := workflow.ExecuteActivity(ctx, ProcessActivity, item)
        futures = append(futures, future)
    }
    
    // Collect results
    var results []string
    for _, future := range futures {
        var result string
        if err := future.Get(ctx, &result); err != nil {
            return nil, err
        }
        results = append(results, result)
    }
    return results, nil
}
```

### 4.4 Selector Pattern

Wait on multiple channels simultaneously:

```go
func SelectorWorkflow(ctx workflow.Context) error {
    signalCh := workflow.GetSignalChannel(ctx, "my-signal")
    timer := workflow.NewTimer(ctx, time.Minute)
    
    selector := workflow.NewSelector(ctx)
    
    selector.AddReceive(signalCh, func(c workflow.ReceiveChannel, more bool) {
        var signal string
        c.Receive(ctx, &signal)
        // Handle signal
    })
    
    selector.AddFuture(timer, func(f workflow.Future) {
        // Handle timeout
    })
    
    selector.Select(ctx) // Wait for first match
    return nil
}
```

---

## Part 5: Saga Pattern

Implement distributed transactions with compensation:

```go
func TransferMoneySaga(ctx workflow.Context, details TransferDetails) error {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)
    
    var compensations []func()
    
    // Step 1: Withdraw
    err := workflow.ExecuteActivity(ctx, Withdraw, details).Get(ctx, nil)
    if err != nil {
        return err
    }
    compensations = append(compensations, func() {
        workflow.ExecuteActivity(ctx, WithdrawCompensation, details).Get(ctx, nil)
    })
    
    // Step 2: Deposit
    err = workflow.ExecuteActivity(ctx, Deposit, details).Get(ctx, nil)
    if err != nil {
        // Run compensations in reverse order
        for i := len(compensations) - 1; i >= 0; i-- {
            compensations[i]()
        }
        return err
    }
    
    return nil
}
```

**TypeScript saga:**
```typescript
interface Compensation {
  message: string;
  fn: () => Promise<void>;
}

export async function openAccount(params: OpenAccount): Promise<void> {
  const compensations: Compensation[] = [];

  try {
    await createAccount({ accountId: params.accountId });
    
    compensations.unshift({
      message: 'reversing add address',
      fn: () => clearPostalAddresses({ accountId: params.accountId }),
    });
    
    await addClient({ accountId: params.accountId, clientEmail: params.clientEmail });
    
    compensations.unshift({
      message: 'reversing add client',
      fn: () => removeClient({ accountId: params.accountId }),
    });

  } catch (err) {
    // Run compensations in reverse order
    for (const comp of compensations) {
      try { await comp.fn(); } catch { /* swallow */ }
    }
    throw err;
  }
}
```

---

## Part 6: Dynamic Workflows (DSL)

Execute workflows defined by external configuration:

```go
type Workflow struct {
    Variables map[string]string
    Root      Statement
}

type Statement struct {
    Activity *ActivityInvocation
    Sequence *Sequence
    Parallel *Parallel
}

func DSLWorkflow(ctx workflow.Context, dslWorkflow Workflow) error {
    bindings := make(map[string]string)
    for k, v := range dslWorkflow.Variables {
        bindings[k] = v
    }
    
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 10 * time.Second,
    }
    ctx = workflow.WithActivityOptions(ctx, ao)
    
    return dslWorkflow.Root.execute(ctx, bindings)
}

func (s *Statement) execute(ctx workflow.Context, bindings map[string]string) error {
    if s.Parallel != nil {
        return s.Parallel.execute(ctx, bindings)
    }
    if s.Sequence != nil {
        return s.Sequence.execute(ctx, bindings)
    }
    if s.Activity != nil {
        return s.Activity.execute(ctx, bindings)
    }
    return nil
}

func (p *Parallel) execute(ctx workflow.Context, bindings map[string]string) error {
    childCtx, cancel := workflow.WithCancel(ctx)
    selector := workflow.NewSelector(ctx)
    var activityErr error
    
    for _, branch := range p.Branches {
        f := executeAsync(branch, childCtx, bindings)
        selector.AddFuture(f, func(f workflow.Future) {
            if err := f.Get(ctx, nil); err != nil {
                cancel() // Cancel all on first failure
                activityErr = err
            }
        })
    }
    
    for i := 0; i < len(p.Branches); i++ {
        selector.Select(ctx)
        if activityErr != nil {
            return activityErr
        }
    }
    return nil
}
```

---

## Part 7: Nexus (Cross-Namespace Communication)

Connect workflows across namespace/cluster/cloud boundaries:

### Service Definition

```go
// service/api.go
package service

const HelloServiceName = "my-hello-service"

const HelloOperationName = "say-hello"

type HelloInput struct {
    Name     string
    Language Language
}

type HelloOutput struct {
    Message string
}
```

### Handler Implementation

```go
// handler/app.go
var HelloOperation = temporalnexus.NewWorkflowRunOperation(
    service.HelloOperationName,
    HelloHandlerWorkflow,
    func(ctx context.Context, input service.HelloInput, opts nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
        return client.StartWorkflowOptions{
            ID: opts.RequestID, // Dedupe with request ID
        }, nil
    },
)

func HelloHandlerWorkflow(ctx workflow.Context, input service.HelloInput) (service.HelloOutput, error) {
    switch input.Language {
    case EN:
        return service.HelloOutput{Message: "Hello " + input.Name}, nil
    case FR:
        return service.HelloOutput{Message: "Bonjour " + input.Name}, nil
    }
    return service.HelloOutput{}, fmt.Errorf("unsupported language")
}
```

### Caller Workflow

```go
// caller/workflows.go
func HelloCallerWorkflow(ctx workflow.Context, name string) (string, error) {
    c := workflow.NewNexusClient("my-nexus-endpoint", service.HelloServiceName)
    
    fut := c.ExecuteOperation(ctx, service.HelloOperationName, 
        service.HelloInput{Name: name, Language: service.EN},
        workflow.NexusOperationOptions{})
    
    var res service.HelloOutput
    if err := fut.Get(ctx, &res); err != nil {
        return "", err
    }
    return res.Message, nil
}
```

---

## Part 8: Timers and Delays

### 8.1 Basic Timer

```go
func TimerWorkflow(ctx workflow.Context) error {
    // Sleep for 30 seconds
    workflow.Sleep(ctx, 30*time.Second)
    
    // Or use NewTimer for more control
    timer := workflow.NewTimer(ctx, time.Hour)
    if err := timer.Get(ctx, nil); err != nil {
        return err
    }
    return nil
}
```

### 8.2 Updatable Timer

Timer that can be modified at runtime:

```go
func UpdatableTimerWorkflow(ctx workflow.Context, initialWakeTime time.Time) error {
    timer := &UpdatableTimer{wakeUpTime: initialWakeTime}
    
    // Set up query to check current wake time
    workflow.SetQueryHandler(ctx, "GetWakeUpTime", func() (time.Time, error) {
        return timer.GetWakeUpTime(), nil
    })
    
    // Set up signal to update wake time
    updateCh := workflow.GetSignalChannel(ctx, "UpdateWakeUpTime")
    
    return timer.SleepUntil(ctx, initialWakeTime, updateCh)
}

type UpdatableTimer struct {
    wakeUpTime time.Time
}

func (u *UpdatableTimer) SleepUntil(ctx workflow.Context, wakeTime time.Time, updateCh workflow.ReceiveChannel) error {
    u.wakeUpTime = wakeTime
    
    for ctx.Err() == nil {
        timerCtx, cancel := workflow.WithCancel(ctx)
        duration := u.wakeUpTime.Sub(workflow.Now(timerCtx))
        timer := workflow.NewTimer(timerCtx, duration)
        
        selector := workflow.NewSelector(ctx)
        selector.AddFuture(timer, func(f workflow.Future) {
            // Timer fired
        })
        selector.AddReceive(updateCh, func(c workflow.ReceiveChannel, more bool) {
            cancel() // Cancel current timer
            c.Receive(ctx, &u.wakeUpTime) // Update wake time
        })
        selector.Select(timerCtx)
    }
    return ctx.Err()
}
```

---

## Part 9: Cancellation and Cleanup

```go
func CancellableWorkflow(ctx workflow.Context) error {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 30 * time.Minute,
        HeartbeatTimeout:    5 * time.Second,
        WaitForCancellation: true, // Wait for activity to handle cancellation
    }
    ctx = workflow.WithActivityOptions(ctx, ao)
    
    var a *Activities
    
    // Defer cleanup on cancellation
    defer func() {
        if errors.Is(ctx.Err(), workflow.ErrCanceled) {
            // Use disconnected context for cleanup
            newCtx, _ := workflow.NewDisconnectedContext(ctx)
            workflow.ExecuteActivity(newCtx, a.CleanupActivity).Get(ctx, nil)
        }
    }()
    
    err := workflow.ExecuteActivity(ctx, a.LongRunningActivity).Get(ctx, nil)
    return err
}
```

---

## Part 10: Distributed Mutex

Coordinate exclusive access across workflows:

```go
func SampleWorkflowWithMutex(ctx workflow.Context, resourceID string) error {
    currentWorkflowID := workflow.GetInfo(ctx).WorkflowExecution.ID
    mutex := NewMutex(currentWorkflowID, "my-use-case")
    
    // Acquire lock (blocks until obtained)
    unlock, err := mutex.Lock(ctx, resourceID, 10*time.Minute)
    if err != nil {
        return err
    }
    defer unlock() // Release lock when done
    
    // Safe to access exclusive resource
    return processExclusiveResource(ctx, resourceID)
}

type Mutex struct {
    currentWorkflowID string
    lockNamespace     string
}

func (m *Mutex) Lock(ctx workflow.Context, resourceID string, timeout time.Duration) (UnlockFunc, error) {
    // Signal mutex workflow to request lock
    activityCtx := workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{
        ScheduleToCloseTimeout: time.Minute,
    })
    
    var execution workflow.Execution
    err := workflow.ExecuteLocalActivity(activityCtx,
        SignalWithStartMutexWorkflowActivity, m.lockNamespace,
        resourceID, m.currentWorkflowID, timeout).Get(ctx, &execution)
    if err != nil {
        return nil, err
    }
    
    // Wait for lock acquisition signal
    workflow.GetSignalChannel(ctx, AcquireLockSignalName).Receive(ctx, nil)
    
    // Return unlock function
    return func() error {
        return workflow.SignalExternalWorkflow(ctx, execution.ID, execution.RunID,
            "release-lock", true).Get(ctx, nil)
    }, nil
}
```

---

## Part 11: Schedules

```go
// Create a schedule that runs every hour
func createSchedule(c client.Client) {
    scheduleID := "my-schedule"
    workflowID := "scheduled-workflow"
    
    // Create schedule client
    scheduleClient := c.ScheduleClient()
    
    // Create the schedule
    _, err := scheduleClient.Create(ctx, client.ScheduleOptions{
        ID: scheduleID,
        Spec: client.ScheduleSpec{
            CronExpressions: []string{"0 * * * *"}, // Every hour
        },
        Action: &client.ScheduleWorkflowAction{
            ID:        workflowID,
            Workflow:  MyScheduledWorkflow,
            TaskQueue: "my-task-queue",
            Args:      []interface{}{"scheduled-input"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }
}

// Workflow that runs on a schedule
func MyScheduledWorkflow(ctx workflow.Context, input string) error {
    // Access schedule metadata
    info := workflow.GetInfo(ctx)
    scheduledByID := info.SearchAttributes.IndexedFields["TemporalScheduledById"]
    startTime := info.SearchAttributes.IndexedFields["TemporalScheduledStartTime"]
    
    return processScheduledJob(ctx, input)
}
```

---

## Part 12: Encryption and Data Converters

### 12.1 Custom Data Converter

```go
type EncryptionCodec struct {
    key []byte
}

func (c *EncryptionCodec) Encode(payloads []*common.Payload) ([]*common.Payload, error) {
    result := make([]*common.Payload, len(payloads))
    for i, p := range payloads {
        encrypted, err := encrypt(p.Data, c.key)
        if err != nil {
            return nil, err
        }
        result[i] = &common.Payload{
            Metadata: map[string][]byte{
                "encoding": []byte("encrypted"),
            },
            Data: encrypted,
        }
    }
    return result, nil
}

func (c *EncryptionCodec) Decode(payloads []*common.Payload) ([]*common.Payload, error) {
    result := make([]*common.Payload, len(payloads))
    for i, p := range payloads {
        if string(p.Metadata["encoding"]) != "encrypted" {
            result[i] = p
            continue
        }
        decrypted, err := decrypt(p.Data, c.key)
        if err != nil {
            return nil, err
        }
        result[i] = &common.Payload{Data: decrypted}
    }
    return result, nil
}
```

### 12.2 Codec Server

```go
// HTTP server for UI/tctl to decode encrypted payloads
func main() {
    http.HandleFunc("/decode", func(w http.ResponseWriter, r *http.Request) {
        var payloads []*common.Payload
        json.NewDecoder(r.Body).Decode(&payloads)
        
        codec := &EncryptionCodec{key: getKey()}
        decoded, _ := codec.Decode(payloads)
        
        json.NewEncoder(w).Encode(decoded)
    })
    
    http.HandleFunc("/encode", func(w http.ResponseWriter, r *http.Request) {
        var payloads []*common.Payload
        json.NewDecoder(r.Body).Decode(&payloads)
        
        codec := &EncryptionCodec{key: getKey()}
        encoded, _ := codec.Encode(payloads)
        
        json.NewEncoder(w).Encode(encoded)
    })
    
    http.ListenAndServe(":8081", nil)
}
```

---

## Part 13: Best Practices

### 13.1 Determinism Rules

**NEVER use in workflows:**
- `time.Now()` → Use `workflow.Now(ctx)`
- `time.Sleep()` → Use `workflow.Sleep(ctx, ...)`
- `rand.*` → Use `workflow.NewUUID(ctx)` or SideEffect
- Global state → Use workflow state
- Blocking calls → Use workflow APIs
- Network I/O → Use Activities

```go
// WRONG - Non-deterministic
func BadWorkflow(ctx workflow.Context) error {
    timestamp := time.Now() // Non-deterministic!
    rand.Seed(time.Now().UnixNano())
    id := rand.Intn(1000) // Non-deterministic!
    return nil
}

// CORRECT - Deterministic
func GoodWorkflow(ctx workflow.Context) error {
    timestamp := workflow.Now(ctx) // Deterministic
    id := workflow.NewUUID(ctx)    // Deterministic
    return nil
}
```

### 13.2 Activity Design

```go
// Activity should be idempotent
func IdempotentActivity(ctx context.Context, input ProcessInput) error {
    // Use unique ID to dedupe
    existing, _ := checkExisting(ctx, input.ID)
    if existing != nil {
        return nil // Already processed
    }
    
    // Process with transaction
    return processWithTransaction(ctx, input)
}
```

### 13.3 Workflow Testing

```go
func TestMyWorkflow(t *testing.T) {
    testSuite := &testsuite.WorkflowTestSuite{}
    env := testSuite.NewTestWorkflowEnvironment()
    
    // Mock activities
    env.OnActivity(MyActivity, mock.Anything, "input").Return("output", nil)
    
    // Run workflow
    env.ExecuteWorkflow(MyWorkflow, "test-input")
    
    require.True(t, env.IsWorkflowCompleted())
    require.NoError(t, env.GetWorkflowError())
    
    var result string
    require.NoError(t, env.GetWorkflowResult(&result))
}
```

---

## Part 14: Production Checklist

### Development
- [ ] All workflow code is deterministic
- [ ] Activities are idempotent
- [ ] Proper timeout values set
- [ ] Retry policies configured
- [ ] Heartbeats for long activities

### Security
- [ ] Encryption codec for sensitive data
- [ ] Codec server for UI decryption
- [ ] mTLS enabled
- [ ] Namespace isolation
- [ ] JWT/OIDC authentication (Cloud)

### Operations
- [ ] Worker versioning strategy
- [ ] Monitoring and alerting configured
- [ ] Log aggregation set up
- [ ] Search attributes defined
- [ ] Retention period configured

### Scaling
- [ ] History size limits monitored
- [ ] Continue-as-new for long workflows
- [ ] Task queue partitioning
- [ ] Worker autoscaling configured

---

## Quick Commands

```bash
# Start local server
temporal server start-dev

# Run workflow
temporal workflow start --type MyWorkflow --task-queue my-queue

# Query workflow
temporal workflow query --type get-status -w my-workflow-id

# Signal workflow
temporal workflow signal --type cancel -w my-workflow-id

# Describe workflow
temporal workflow describe -w my-workflow-id

# View workflow history
temporal workflow show -w my-workflow-id
```

---

## References

- [Temporal Docs](https://docs.temporal.io)
- [Go SDK Reference](https://pkg.go.dev/go.temporal.io/sdk)
- [Python SDK Reference](https://python.temporal.io)
- [TypeScript SDK Reference](https://typescript.temporal.io)
- [Samples Repository](https://github.com/temporalio/samples-go)
