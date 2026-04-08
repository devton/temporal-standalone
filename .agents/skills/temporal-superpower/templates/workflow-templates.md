# Temporal Workflow Templates

## Template 1: Basic Workflow with Activity

```go
package workflow

import (
    "time"
    "go.temporal.io/sdk/temporal"
    "go.temporal.io/sdk/workflow"
)

// Input/Output types
type ProcessInput struct {
    ID     string
    Data   string
}

type ProcessOutput struct {
    Result string
}

// Workflow Definition
func ProcessWorkflow(ctx workflow.Context, input ProcessInput) (ProcessOutput, error) {
    logger := workflow.GetLogger(ctx)
    logger.Info("Workflow started", "ID", input.ID)

    // Activity options
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:    time.Minute,
            MaximumAttempts:    3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    // Execute activity
    var result string
    err := workflow.ExecuteActivity(ctx, ProcessActivity, input.Data).Get(ctx, &result)
    if err != nil {
        return ProcessOutput{}, err
    }

    logger.Info("Workflow completed", "result", result)
    return ProcessOutput{Result: result}, nil
}
```

---

## Template 2: Signal-Driven Workflow

```go
// Workflow that responds to signals
func OrderWorkflow(ctx workflow.Context, orderID string) error {
    order := Order{
        ID:     orderID,
        Status: "created",
    }

    // Register signal handlers
    workflow.SetSignalHandler(ctx, "cancel", func(ctx workflow.Context, reason string) error {
        order.Status = "cancelled"
        order.CancelReason = reason
        return nil
    })

    workflow.SetSignalHandler(ctx, "addItem", func(ctx workflow.Context, item Item) error {
        order.Items = append(order.Items, item)
        return nil
    })

    // Set up query handler
    workflow.SetQueryHandler(ctx, "getOrder", func() (Order, error) {
        return order, nil
    })

    // Wait for order completion
    workflow.Await(ctx, func() bool {
        return order.Status == "cancelled" || len(order.Items) > 0
    })

    if order.Status == "cancelled" {
        return workflow.NewContinueAsNewError(ctx, CancelOrderWorkflow, order)
    }

    return processOrder(ctx, order)
}
```

---

## Template 3: Saga Pattern Workflow

```go
func TransferWorkflow(ctx workflow.Context, transfer TransferRequest) error {
    // Setup activity options
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    var compensations []func()

    // Step 1: Validate
    err := workflow.ExecuteActivity(ctx, ValidateTransfer, transfer).Get(ctx, nil)
    if err != nil {
        return err
    }

    // Step 2: Debit
    err = workflow.ExecuteActivity(ctx, DebitAccount, transfer.From, transfer.Amount).Get(ctx, nil)
    if err != nil {
        return err
    }
    compensations = append(compensations, func() {
        workflow.ExecuteActivity(ctx, CreditAccount, transfer.From, transfer.Amount).Get(ctx, nil)
    })

    // Step 3: Credit (may fail)
    err = workflow.ExecuteActivity(ctx, CreditAccount, transfer.To, transfer.Amount).Get(ctx, nil)
    if err != nil {
        // Run compensations in reverse
        for i := len(compensations) - 1; i >= 0; i-- {
            compensations[i]()
        }
        return err
    }

    // Step 4: Confirm
    return workflow.ExecuteActivity(ctx, ConfirmTransfer, transfer).Get(ctx, nil)
}
```

---

## Template 4: Child Workflow Orchestration

```go
func BatchWorkflow(ctx workflow.Context, batchID string, items []string) ([]Result, error) {
    var results []Result
    var futures []workflow.Future

    // Process items in parallel via child workflows
    for i, item := range items {
        childID := fmt.Sprintf("%s-item-%d", batchID, i)
        childOpts := workflow.ChildWorkflowOptions{
            WorkflowID: childID,
        }
        childCtx := workflow.WithChildOptions(ctx, childOpts)
        
        future := workflow.ExecuteChildWorkflow(childCtx, ItemWorkflow, item)
        futures = append(futures, future)
    }

    // Collect results
    for _, future := range futures {
        var result Result
        if err := future.Get(ctx, &result); err != nil {
            return nil, err
        }
        results = append(results, result)
    }

    return results, nil
}
```

---

## Template 5: Long-Running with Heartbeat

```go
func LongRunningActivity(ctx context.Context, input ProcessInput) (*ProcessOutput, error) {
    logger := activity.GetLogger(ctx)
    
    // Get heartbeat details for resume
    var lastProcessedIndex int
    if heartbeatDetails, ok := activity.GetHeartbeatDetails(ctx); ok {
        lastProcessedIndex = heartbeatDetails.(int)
    }

    totalItems := len(input.Items)
    
    for i := lastProcessedIndex; i < totalItems; i++ {
        // Check cancellation
        if ctx.Err() == context.Canceled {
            // Return partially processed state
            return nil, temporal.NewHeartbeatDetailsError(i)
        }

        // Process item
        if err := processItem(ctx, input.Items[i]); err != nil {
            return nil, err
        }

        // Heartbeat with progress
        activity.Heartbeat(ctx, i+1)
        logger.Info("Progress", "completed", i+1, "total", totalItems)
    }

    return &ProcessOutput{Status: "completed"}, nil
}
```

---

## Template 6: Wait for Signal with Timeout

```go
func ApprovalWorkflow(ctx workflow.Context, request ApprovalRequest) error {
    logger := workflow.GetLogger(ctx)
    
    // Send notification
    err := workflow.ExecuteActivity(ctx, SendNotification, request).Get(ctx, nil)
    if err != nil {
        return err
    }

    // Wait for approval or timeout
    approved := false
    approvalCh := workflow.GetSignalChannel(ctx, "approve")
    
    selector := workflow.NewSelector(ctx)
    selector.AddReceive(approvalCh, func(c workflow.ReceiveChannel, more bool) {
        c.Receive(ctx, &approved)
    })
    selector.AddFuture(workflow.NewTimer(ctx, request.Timeout), func(f workflow.Future) {
        logger.Info("Approval timed out")
    })
    
    selector.Select(ctx)

    if !approved {
        return workflow.ExecuteActivity(ctx, HandleRejection, request).Get(ctx, nil)
    }

    return workflow.ExecuteActivity(ctx, HandleApproval, request).Get(ctx, nil)
}
```

---

## Template 7: Cron Workflow

```go
func ScheduledWorkflow(ctx workflow.Context) error {
    info := workflow.GetInfo(ctx)
    
    // Get schedule metadata from search attributes
    scheduledBy := extractSearchAttribute(ctx, "TemporalScheduledById")
    scheduledStart := extractSearchAttribute(ctx, "TemporalScheduledStartTime")
    
    logger := workflow.GetLogger(ctx)
    logger.Info("Scheduled execution", "by", scheduledBy, "start", scheduledStart)
    
    // Do scheduled work
    return workflow.ExecuteActivity(ctx, DoScheduledWork, scheduledBy, scheduledStart).Get(ctx, nil)
}

// Client code to create schedule
func createSchedule(c client.Client) {
    scheduleClient := c.ScheduleClient()
    scheduleClient.Create(ctx, client.ScheduleOptions{
        ID: "daily-cleanup",
        Spec: client.ScheduleSpec{
            CronExpressions: []string{"0 0 * * *"}, // Daily at midnight
        },
        Action: &client.ScheduleWorkflowAction{
            ID:        "cleanup-workflow",
            Workflow:  ScheduledWorkflow,
            TaskQueue: "cleanup-queue",
        },
    })
}
```

---

## Template 8: Activity with Retry and Timeout

```go
func ResilientWorkflow(ctx workflow.Context, input string) (string, error) {
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute,
        ScheduleToCloseTimeout: 5 * time.Minute, // Total including retries
        ScheduleToStartTimeout: time.Minute,      // Queue wait time
        HeartbeatTimeout: 30 * time.Second,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second * 2,
            BackoffCoefficient: 2.0,
            MaximumInterval:    time.Minute,
            MaximumAttempts:    5,
            NonRetryableErrorTypes: []string{
                "InvalidInputError",
                "PermissionDeniedError",
            },
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)

    var result string
    err := workflow.ExecuteActivity(ctx, MyActivity, input).Get(ctx, &result)
    return result, err
}
```
