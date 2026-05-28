package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// fakeDunningEngine records port.DunningEngine calls.
type fakeDunningEngine struct {
	mu          sync.Mutex
	started     []domain.StartDunningWorkflowInput
	signals     []string
	cancels     int
	wfId, runId string
	startErr    error
}

func (e *fakeDunningEngine) StartDunningWorkflow(_ context.Context, in domain.StartDunningWorkflowInput) (string, string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.started = append(e.started, in)
	return e.wfId, e.runId, e.startErr
}
func (e *fakeDunningEngine) SignalDunningWorkflow(_ context.Context, signal string, _ domain.DunningCampaign, _ any) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.signals = append(e.signals, signal)
	return nil
}
func (e *fakeDunningEngine) CancelDunningWorkflow(_ context.Context, _ domain.DunningCampaign) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cancels++
	return nil
}

// ---- SubscriptionOrchestrationService: narrow op + engine signal ----

func TestSubscriptionOrchestration_PauseSignalsEngineAndPublishes(t *testing.T) {
	subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive}}
	ps := &recordingPubSub{}
	engine := &recordingEngine{}
	narrow := newSubscriptionService(subRepo, nil, nil, nil, nil, ps)
	svc := NewSubscriptionOrchestrationService(narrow, engine, silentLogger{})

	got, err := svc.PauseSubscription(context.Background(), domain.PauseSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusPaused, got.Status)
	assert.Contains(t, engine.updates, "subscription.paused", "engine signalled with the pause update")
	assert.True(t, ps.hasTopic(port.TopicSubscriptionPaused))
}

func TestSubscriptionOrchestration_DoesNotSignalWhenNarrowRejects(t *testing.T) {
	// Already paused → narrow rejects → engine must not be signalled.
	subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPaused}}
	engine := &recordingEngine{}
	narrow := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)
	svc := NewSubscriptionOrchestrationService(narrow, engine, silentLogger{})

	_, err := svc.PauseSubscription(context.Background(), domain.PauseSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

	require.Error(t, err)
	assert.Empty(t, engine.updates, "no engine signal when the DB transition is rejected")
}

func TestSubscriptionOrchestration_CancelSignalsEngine(t *testing.T) {
	subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive}}
	ps := &recordingPubSub{}
	engine := &recordingEngine{}
	narrow := newSubscriptionService(subRepo, nil, nil, nil, nil, ps)
	svc := NewSubscriptionOrchestrationService(narrow, engine, silentLogger{})

	got, err := svc.CancelSubscription(context.Background(), domain.CancelSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusCancelled, got.Status)
	assert.Contains(t, engine.updates, port.TopicSubscriptionCancelled)
	assert.True(t, ps.hasTopic(port.TopicSubscriptionCancelled))
}

func TestSubscriptionOrchestration_ActivateStartsWorkflow(t *testing.T) {
	subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPending}}
	engine := &recordingEngine{}
	narrow := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)
	svc := NewSubscriptionOrchestrationService(narrow, engine, silentLogger{})

	got, err := svc.Activate(context.Background(), "org_1", "sub_1")

	require.NoError(t, err)
	assert.Equal(t, domain.SubscriptionStatusActive, got.Status)
	assert.Len(t, engine.started, 1, "subscription workflow started")
}

func TestSubscriptionOrchestration_EngineErrorPropagates(t *testing.T) {
	subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive}}
	engine := &recordingEngine{updateErr: errors.New("engine down")}
	narrow := newSubscriptionService(subRepo, nil, nil, nil, nil, &recordingPubSub{})
	svc := NewSubscriptionOrchestrationService(narrow, engine, silentLogger{})

	_, err := svc.PauseSubscription(context.Background(), domain.PauseSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

	require.Error(t, err, "engine signalling failure surfaces to the caller")
}

// ---- DunningOrchestrationService: auto-open campaign + engine handles ----

func newDunningOrchestration(dr *fakeDunningRepo, sr *fakeSubRepo, cr *fakeCustomerRepo, eng port.DunningEngine, ps *recordingPubSub) *DunningOrchestrationService {
	if ps == nil {
		ps = &recordingPubSub{}
	}
	// CreateCampaign validates the subscription + customer exist, so the narrow
	// service needs both repos wired.
	narrow := NewDunningService(dr, sr, cr, nil, nil, nil, ps, lib.ErrorReporter{}, silentLogger{})
	svc, err := NewDunningOrchestrationService(narrow, eng, ps, lib.ErrorReporter{}, silentLogger{})
	if err != nil {
		panic(err)
	}
	return svc
}

func TestDunningOrchestration_StartCreatesCampaignAndStoresHandles(t *testing.T) {
	dr := &fakeDunningRepo{}
	sr := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1"}}
	cr := &fakeCustomerRepo{customer: domain.Customer{Id: "cust_1"}}
	engine := &fakeDunningEngine{wfId: "dunning-runner", runId: "run_123"}
	svc := newDunningOrchestration(dr, sr, cr, engine, nil)

	got, err := svc.StartDunningWorkflow(context.Background(), domain.StartDunningWorkflowInput{
		OrgId: "org_1", SubscriptionId: "sub_1", CustomerId: "cust_1", FailedAmount: 5000, Currency: "USD",
		InitialFailureReason: "card_declined",
	})

	require.NoError(t, err)
	assert.Len(t, dr.createdCampaigns, 1, "campaign created")
	assert.NotEmpty(t, dr.createdCampaigns[0].ConfigSnapshot, "config snapshot stored at start")
	assert.Len(t, engine.started, 1, "engine asked to start the dunning run")
	assert.Equal(t, "dunning-runner", got.WorkflowId)
	assert.Equal(t, "run_123", got.WorkflowRunId)
}

func TestDunningOrchestration_ChargeFailureEventOpensCampaign(t *testing.T) {
	// This is the auto-dunning flow: a subscription.payment.charge.failed event
	// must open a campaign and start the runner.
	dr := &fakeDunningRepo{}
	sr := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1"}}
	cr := &fakeCustomerRepo{customer: domain.Customer{Id: "cust_1"}}
	engine := &fakeDunningEngine{wfId: "dunning-runner", runId: "run_1"}
	svc := newDunningOrchestration(dr, sr, cr, engine, nil)

	// Build the envelope exactly as SubscriptionService.HandleSubscriptionChargeFailure publishes it.
	envelope, err := json.Marshal(port.PubSubPayload{
		OrgId: "org_1",
		Topic: port.TopicSubscriptionPaymentChargeFailed,
		Data: map[string]any{
			"subscription":  domain.Subscription{OrgId: "org_1", Id: "sub_1", CustomerId: "cust_1"},
			"charge_result": domain.ChargeResult{Amount: 5000, Currency: "USD", ErrorReason: "insufficient_funds", ErrorCode: "51"},
		},
	})
	require.NoError(t, err)

	svc.HandleSubscriptionChargeFailure(port.TopicSubscriptionPaymentChargeFailed, envelope)

	require.Len(t, dr.createdCampaigns, 1, "charge-failure event opened a dunning campaign")
	assert.Equal(t, "sub_1", dr.createdCampaigns[0].SubscriptionId)
	assert.Len(t, engine.started, 1, "dunning runner started for the campaign")
}

func TestDunningOrchestration_BadEnvelopeOpensNothing(t *testing.T) {
	dr := &fakeDunningRepo{}
	engine := &fakeDunningEngine{}
	svc := newDunningOrchestration(dr, &fakeSubRepo{}, &fakeCustomerRepo{}, engine, nil)

	svc.HandleSubscriptionChargeFailure(port.TopicSubscriptionPaymentChargeFailed, []byte("not json"))

	assert.Empty(t, dr.createdCampaigns, "no campaign on malformed event")
	assert.Empty(t, engine.started)
}

func TestDunningOrchestration_PauseResumeCancelSignalEngine(t *testing.T) {
	mk := func(status domain.DunningStatus) (*fakeDunningRepo, *fakeDunningEngine, *DunningOrchestrationService) {
		dr := &fakeDunningRepo{campaign: domain.DunningCampaign{OrgId: "org_1", Id: "dc_1", Status: status}}
		engine := &fakeDunningEngine{}
		svc := newDunningOrchestration(dr, &fakeSubRepo{}, &fakeCustomerRepo{}, engine, nil)
		return dr, engine, svc
	}

	t.Run("pause signals dunning.pause", func(t *testing.T) {
		_, engine, svc := mk(domain.DunningStatusActive)
		_, err := svc.PauseCampaign(context.Background(), domain.PauseDunningCampaignInput{OrgId: "org_1", CampaignId: "dc_1"})
		require.NoError(t, err)
		assert.Contains(t, engine.signals, "dunning.pause")
	})

	t.Run("resume signals dunning.resume", func(t *testing.T) {
		_, engine, svc := mk(domain.DunningStatusPaused)
		_, err := svc.ResumeCampaign(context.Background(), domain.ResumeDunningCampaignInput{OrgId: "org_1", CampaignId: "dc_1"})
		require.NoError(t, err)
		assert.Contains(t, engine.signals, "dunning.resume")
	})

	t.Run("cancel cancels the workflow", func(t *testing.T) {
		_, engine, svc := mk(domain.DunningStatusActive)
		_, err := svc.CancelCampaign(context.Background(), domain.CancelDunningCampaignInput{OrgId: "org_1", CampaignId: "dc_1"})
		require.NoError(t, err)
		assert.Equal(t, 1, engine.cancels)
	})
}
