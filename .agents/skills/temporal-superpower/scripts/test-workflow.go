package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.temporal.io/sdk/testsuite"
	"github.com/stretchr/testify/require"
)

// WorkflowTestSuite provides deterministic testing for Temporal workflows
type WorkflowTestSuite struct {
	suite *testsuite.WorkflowTestSuite
	env   *testsuite.TestWorkflowEnvironment
}

func NewWorkflowTestSuite() *WorkflowTestSuite {
	suite := &testsuite.WorkflowTestSuite{}
	return &WorkflowTestSuite{
		suite: suite,
		env:   suite.NewTestWorkflowEnvironment(),
	}
}

// Test basic workflow execution
func TestBasicWorkflow(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Mock activity
	env.OnActivity(MyActivity, mock.Anything, "test-input").
		Return("test-output", nil)

	env.ExecuteWorkflow(MyWorkflow, "test-input")

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result string
	require.NoError(t, env.GetWorkflowResult(&result))
	require.Equal(t, "test-output", result)
}

// Test workflow with signals
func TestSignalWorkflow(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Send signal after workflow starts
	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow("cancel-signal", true)
	}, time.Second*5)

	env.ExecuteWorkflow(SignalWorkflow, "test-input")

	require.True(t, env.IsWorkflowCompleted())
}

// Test workflow with queries
func TestQueryWorkflow(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	var queryResult string
	env.RegisterDelayedCallback(func() {
		resp, err := env.QueryWorkflow("get-status", nil)
		require.NoError(t, err)
		require.NoError(t, resp.Get(&queryResult))
		require.Equal(t, "processing", queryResult)
	}, time.Second*2)

	env.ExecuteWorkflow(QueryableWorkflow, "test-input")

	require.True(t, env.IsWorkflowCompleted())
}

// Test workflow cancellation
func TestCancellationWorkflow(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Cancel after 5 seconds
	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second*5)

	env.ExecuteWorkflow(CancellableWorkflow, "test-input")

	require.True(t, env.IsWorkflowCompleted())
	// Cancellation should be handled gracefully
}

// Test workflow failure and retry
func TestWorkflowFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Activity fails twice, then succeeds
	callCount := 0
	env.OnActivity(MyActivity, mock.Anything, mock.Anything).
		Return(func(ctx context.Context, input string) (string, error) {
			callCount++
			if callCount < 3 {
				return "", fmt.Errorf("transient error")
			}
			return "success", nil
		})

	env.ExecuteWorkflow(MyWorkflow, "test-input")

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// Test workflow with child workflow
func TestChildWorkflow(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Mock child workflow
	env.OnWorkflow(ChildWorkflow, mock.Anything, "child-input").
		Return("child-output", nil)

	env.ExecuteWorkflow(ParentWorkflow, "parent-input")

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

// Test saga workflow with compensation
func TestSagaWorkflowFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Step 1 succeeds
	env.OnActivity(WithdrawActivity, mock.Anything, mock.Anything).
		Return(nil).Once()
	
	// Step 2 fails
	env.OnActivity(DepositActivity, mock.Anything, mock.Anything).
		Return(fmt.Errorf("deposit failed")).Once()
	
	// Compensation should be called
	env.OnActivity(WithdrawCompensation, mock.Anything, mock.Anything).
		Return(nil).Once()

	env.ExecuteWorkflow(SagaWorkflow, TransferDetails{Amount: 100})

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

// Test workflow timeout
func TestWorkflowTimeout(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	// Set workflow timeout
	env.SetWorkflowTimeout(time.Minute)

	// Activity takes too long (simulated)
	env.OnActivity(SlowActivity, mock.Anything, mock.Anything).
		Return("done", nil).After(time.Hour)

	env.ExecuteWorkflow(TimedWorkflow, "test-input")

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}

// Run all tests
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
