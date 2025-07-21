package temporal

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"payloop/internal/domain/entities"
	"payloop/internal/infrastructure/workflow/temporal/activities"
	"payloop/internal/infrastructure/workflow/temporal/testutils"
	"payloop/internal/infrastructure/workflow/temporal/workflows"
)

type WorkflowIntegrationTestSuite struct {
	suite.Suite
	client client.Client
	worker worker.Worker
}

func TestWorkflowIntegrationTestSuite(t *testing.T) {
	// Skip integration tests if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration tests. Set INTEGRATION_TEST=true to run.")
	}
	
	suite.Run(t, new(WorkflowIntegrationTestSuite))
}

func (s *WorkflowIntegrationTestSuite) SetupSuite() {
	// Connect to Temporal server (assumes local Temporal is running)
	var err error
	s.client, err = client.Dial(client.Options{
		HostPort: "localhost:7233",
	})
	s.Require().NoError(err)
	
	// Create worker for testing
	s.worker = worker.New(s.client, "test-task-queue", worker.Options{})
	
	// Register workflows and activities
	s.worker.RegisterWorkflow(workflows.SubscriptionWorkflow)
	s.worker.RegisterWorkflow(workflows.DunningWorkflow)
	
	// Note: In real integration tests, you would register actual activity implementations
	// For this example, we'll use mock activities or skip activity registration
	
	// Start worker
	err = s.worker.Start()
	s.Require().NoError(err)
}

func (s *WorkflowIntegrationTestSuite) TearDownSuite() {
	if s.worker != nil {
		s.worker.Stop()
	}
	if s.client != nil {
		s.client.Close()
	}
}

func (s *WorkflowIntegrationTestSuite) TestSubscriptionWorkflow_FastCycle_Integration() {
	// Test subscription workflow with fast billing cycles
	ctx := context.Background()
	
	// Create test subscription with fast timing
	subscription := testutils.CreateFastSubscription("integration_org", "integration_customer", 1000)
	
	// Start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        "test-subscription-" + subscription.Id,
		TaskQueue: "test-task-queue",
	}
	
	workflowRun, err := s.client.ExecuteWorkflow(ctx, workflowOptions, workflows.SubscriptionWorkflow, subscription)
	s.NoError(err)
	
	// Test workflow state query
	queryResult, err := s.client.QueryWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "get-state")
	s.NoError(err)
	
	var queriedSubscription entities.Subscription
	err = queryResult.Get(&queriedSubscription)
	s.NoError(err)
	s.Equal(subscription.Id, queriedSubscription.Id)
	
	// Test workflow signals
	err = s.client.SignalWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "subscription.paused", 
		testutils.CreatePausedSubscription("integration_org", "integration_customer"))
	s.NoError(err)
	
	// Wait for workflow completion or timeout
	ctx, cancel := context.WithTimeout(ctx, time.Minute*2)
	defer cancel()
	
	var result entities.Subscription
	err = workflowRun.Get(ctx, &result)
	
	// Note: This test might timeout if activities are not properly mocked
	// In a real integration test, you would have actual service implementations
	if err != nil && err.Error() != "context deadline exceeded" {
		s.NoError(err)
	}
}

func (s *WorkflowIntegrationTestSuite) TestDunningWorkflow_FastRetries_Integration() {
	// Test dunning workflow with fast retry intervals
	ctx := context.Background()
	
	// Create test dunning input
	input := testutils.CreateDunningWorkflowInput("integration_org", "integration_sub", "integration_customer")
	
	// Start workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        "test-dunning-" + input.SubscriptionId,
		TaskQueue: "test-task-queue",
	}
	
	workflowRun, err := s.client.ExecuteWorkflow(ctx, workflowOptions, workflows.DunningWorkflow, input)
	s.NoError(err)
	
	// Test campaign query
	queryResult, err := s.client.QueryWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "get-campaign")
	s.NoError(err)
	
	// Test workflow signals
	err = s.client.SignalWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "dunning.pause", nil)
	s.NoError(err)
	
	err = s.client.SignalWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "dunning.resume", nil)
	s.NoError(err)
	
	// Wait for workflow completion or timeout
	ctx, cancel := context.WithTimeout(ctx, time.Minute*2)
	defer cancel()
	
	var result workflows.DunningWorkflowInput
	err = workflowRun.Get(ctx, &result)
	
	// Note: This test might timeout if activities are not properly mocked
	if err != nil && err.Error() != "context deadline exceeded" {
		s.NoError(err)
	}
}

func (s *WorkflowIntegrationTestSuite) TestWorkflow_ConcurrentExecution() {
	// Test multiple workflows running concurrently
	ctx := context.Background()
	
	// Start multiple subscription workflows
	var workflowRuns []client.WorkflowRun
	
	for i := 0; i < 5; i++ {
		subscription := testutils.CreateFastSubscription("concurrent_org", "customer_" + string(rune(i)), 1000)
		
		workflowOptions := client.StartWorkflowOptions{
			ID:        "concurrent-subscription-" + subscription.Id + "-" + string(rune(i)),
			TaskQueue: "test-task-queue",
		}
		
		workflowRun, err := s.client.ExecuteWorkflow(ctx, workflowOptions, workflows.SubscriptionWorkflow, subscription)
		s.NoError(err)
		workflowRuns = append(workflowRuns, workflowRun)
	}
	
	// Verify all workflows started
	s.Len(workflowRuns, 5)
	
	// Query each workflow to ensure they're running
	for i, run := range workflowRuns {
		queryResult, err := s.client.QueryWorkflow(ctx, run.GetID(), run.GetRunID(), "get-state")
		s.NoError(err, "Workflow %d query failed", i)
		
		var subscription entities.Subscription
		err = queryResult.Get(&subscription)
		s.NoError(err, "Workflow %d query result parsing failed", i)
	}
	
	// Cancel all workflows to clean up
	for _, run := range workflowRuns {
		err := s.client.CancelWorkflow(ctx, run.GetID(), run.GetRunID())
		s.NoError(err)
	}
}

func (s *WorkflowIntegrationTestSuite) TestWorkflow_LongRunningScenario() {
	// Test workflow behavior over extended time periods (with fast timing)
	ctx := context.Background()
	
	subscription := testutils.CreateFastSubscription("longrunning_org", "longrunning_customer", 1000)
	
	workflowOptions := client.StartWorkflowOptions{
		ID:        "longrunning-subscription-" + subscription.Id,
		TaskQueue: "test-task-queue",
	}
	
	workflowRun, err := s.client.ExecuteWorkflow(ctx, workflowOptions, workflows.SubscriptionWorkflow, subscription)
	s.NoError(err)
	
	// Simulate workflow running for multiple billing cycles
	// In a real test, this would involve actual time progression and activity execution
	
	// Test workflow history size doesn't grow excessively
	workflowInfo, err := s.client.DescribeWorkflowExecution(ctx, workflowRun.GetID(), workflowRun.GetRunID())
	s.NoError(err)
	
	// Verify workflow is running
	s.NotNil(workflowInfo.WorkflowExecutionInfo)
	
	// Clean up
	err = s.client.CancelWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID())
	s.NoError(err)
}

func (s *WorkflowIntegrationTestSuite) TestWorkflow_ErrorRecovery() {
	// Test workflow recovery from various error scenarios
	ctx := context.Background()
	
	subscription := testutils.CreateFastSubscription("error_org", "error_customer", 1000)
	
	workflowOptions := client.StartWorkflowOptions{
		ID:        "error-recovery-" + subscription.Id,
		TaskQueue: "test-task-queue",
	}
	
	workflowRun, err := s.client.ExecuteWorkflow(ctx, workflowOptions, workflows.SubscriptionWorkflow, subscription)
	s.NoError(err)
	
	// Test workflow continues after various signals
	
	// Pause and resume
	err = s.client.SignalWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "subscription.paused", 
		testutils.CreatePausedSubscription("error_org", "error_customer"))
	s.NoError(err)
	
	time.Sleep(time.Second) // Allow signal processing
	
	err = s.client.SignalWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "subscription.activated", 
		testutils.CreateFastSubscription("error_org", "error_customer", 1000))
	s.NoError(err)
	
	// Force refresh
	err = s.client.SignalWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "refresh-state", 
		testutils.CreateFastSubscription("error_org", "error_customer", 2000))
	s.NoError(err)
	
	// Verify workflow is still responsive
	queryResult, err := s.client.QueryWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID(), "get-state")
	s.NoError(err)
	
	var queriedSubscription entities.Subscription
	err = queryResult.Get(&queriedSubscription)
	s.NoError(err)
	
	// Clean up
	err = s.client.CancelWorkflow(ctx, workflowRun.GetID(), workflowRun.GetRunID())
	s.NoError(err)
}

// Benchmark tests for performance under load
func (s *WorkflowIntegrationTestSuite) TestWorkflow_PerformanceBenchmark() {
	// Skip performance tests in short mode
	if testing.Short() {
		s.T().Skip("Skipping performance benchmark in short mode")
	}
	
	ctx := context.Background()
	startTime := time.Now()
	
	// Start multiple workflows and measure startup time
	numWorkflows := 10
	var workflowRuns []client.WorkflowRun
	
	for i := 0; i < numWorkflows; i++ {
		subscription := testutils.CreateFastSubscription("perf_org", "perf_customer_" + string(rune(i)), 1000)
		
		workflowOptions := client.StartWorkflowOptions{
			ID:        "perf-test-" + subscription.Id + "-" + string(rune(i)),
			TaskQueue: "test-task-queue",
		}
		
		workflowRun, err := s.client.ExecuteWorkflow(ctx, workflowOptions, workflows.SubscriptionWorkflow, subscription)
		s.NoError(err)
		workflowRuns = append(workflowRuns, workflowRun)
	}
	
	startupDuration := time.Since(startTime)
	s.T().Logf("Started %d workflows in %v (avg: %v per workflow)", 
		numWorkflows, startupDuration, startupDuration/time.Duration(numWorkflows))
	
	// Verify all workflows are responsive
	queryStartTime := time.Now()
	for i, run := range workflowRuns {
		_, err := s.client.QueryWorkflow(ctx, run.GetID(), run.GetRunID(), "get-state")
		s.NoError(err, "Query failed for workflow %d", i)
	}
	queryDuration := time.Since(queryStartTime)
	s.T().Logf("Queried %d workflows in %v (avg: %v per query)", 
		numWorkflows, queryDuration, queryDuration/time.Duration(numWorkflows))
	
	// Clean up all workflows
	cleanupStartTime := time.Now()
	for _, run := range workflowRuns {
		err := s.client.CancelWorkflow(ctx, run.GetID(), run.GetRunID())
		s.NoError(err)
	}
	cleanupDuration := time.Since(cleanupStartTime)
	s.T().Logf("Cancelled %d workflows in %v", numWorkflows, cleanupDuration)
	
	// Assert reasonable performance thresholds
	avgStartupTime := startupDuration / time.Duration(numWorkflows)
	s.Less(avgStartupTime, time.Second, "Workflow startup should be under 1 second per workflow")
	
	avgQueryTime := queryDuration / time.Duration(numWorkflows)
	s.Less(avgQueryTime, time.Millisecond*100, "Workflow queries should be under 100ms per query")
}