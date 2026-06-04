//go:build integration

package hatchet

import (
	"context"
	"os"
	"testing"
	"time"

	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/adapter/hatchet/steps"
	"getpaidhq/internal/adapter/hatchet/workflows"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// TestHatchetIntegration connects to the Docker Compose hatchet-lite and runs
// a real DAG workflow end-to-end.
//
// Prerequisites:
//
//	docker compose -f docker/docker-compose.yml up -d postgresql hatchet-lite
//
// Run:
//
//	HATCHET_CLIENT_TOKEN=*** \
//	go test -v -tags=integration -run TestHatchetIntegration ./internal/adapter/hatchet/
func TestHatchetIntegration(t *testing.T) {
	token := os.Getenv("HATCHET_CLIENT_TOKEN")
	if token == "" {
		t.Skip("HATCHET_CLIENT_TOKEN not set — skipping")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to hatchet-lite (reads env vars for host/namespace/TLS)
	client, err := hatchet.NewClient()
	require.NoError(t, err, "Hatchet client should connect to local compose service")

	// Register a minimal test worker that runs the DunningAttemptWorkflow
	// with fake steps so we can verify the engine wiring end-to-end.
	fakeLogger := &noopLogger{}
	fakeDunningSvc := &fakeDunningServiceForTest{}
	dunningSteps := steps.NewDunningSteps(fakeLogger, fakeDunningSvc)

	dunningAttemptWF := workflows.NewDunningAttemptWorkflow(client, dunningSteps)

	worker, err := client.NewWorker("test-worker",
		hatchet.WithWorkflows(dunningAttemptWF),
		hatchet.WithSlots(5),
	)
	require.NoError(t, err, "Should create worker")

	// Run the worker in the background
	workerCtx, workerCancel := context.WithCancel(ctx)
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		_ = worker.StartBlocking(workerCtx)
	}()
	defer workerCancel()

	// Give the worker a moment to register
	time.Sleep(1 * time.Second)

	// Run a dunning-attempt workflow
	runCtx, runCancel := context.WithTimeout(ctx, 15*time.Second)
	defer runCancel()

	res, err := client.Run(runCtx, "dunning-attempt", workflows.DunningAttemptInput{
		OrgId:         "test-org",
		CampaignId:    "test-campaign",
		AttemptNumber: 1,
		AttemptType:   domain.DunningAttemptTypeProgressive,
	}, hatchet.WithRunKey("test-attempt-1"))
	require.NoError(t, err, "Should start workflow")

	var attempt domain.DunningAttempt
	require.NoError(t, res.TaskOutput("execute-attempt").Into(&attempt))
	assert.Equal(t, "test-attempt-id", attempt.Id)
	assert.Equal(t, domain.PaymentStatusSucceeded, attempt.Status)

	// Shutdown
	workerCancel()
	<-doneCh
	t.Log("Hatchet-lite integration test passed — workflow ran end-to-end")
}

// ---- test doubles ----

type noopLogger struct{}

func (noopLogger) Debug(string, ...any)  {}
func (noopLogger) Info(string, ...any)   {}
func (noopLogger) Warn(string, ...any)   {}
func (noopLogger) Error(string, ...any)  {}
func (noopLogger) Fatal(string, ...any)  {}
func (noopLogger) Debugf(string, ...any) {}
func (noopLogger) Infof(string, ...any)  {}
func (noopLogger) Warnf(string, ...any)  {}
func (noopLogger) Errorf(string, ...any) {}
func (noopLogger) Panicf(string, ...any) {}
func (noopLogger) Fatalf(string, ...any) {}
func (noopLogger) Sync() error           { return nil }

type fakeDunningServiceForTest struct {
	port.DunningService // embed to satisfy the interface — only override ExecuteAttempt
}

func (f *fakeDunningServiceForTest) ExecuteAttempt(_ context.Context, orgId, campaignId string, _ domain.DunningAttemptType) (domain.DunningAttempt, error) {
	return domain.DunningAttempt{
		OrgId:             orgId,
		Id:                "test-attempt-id",
		DunningCampaignId: campaignId,
		AttemptNumber:     1,
		Status:            domain.PaymentStatusSucceeded,
	}, nil
}

// Compile-time check: fakeDunningServiceForTest satisfies the interface
var _ port.DunningService = (*fakeDunningServiceForTest)(nil)
