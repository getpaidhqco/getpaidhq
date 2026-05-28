package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// The Hatchet steps are thin shims — they should forward each call to the
// service with the same args and propagate the service's result and error
// untouched. These tests pin that wiring contract so a step can't silently
// diverge from its service method.

// ---- fakes ----

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

// fakeDunningService embeds port.DunningService and records the methods the
// step bundle forwards into. Returned values and errors are configurable per
// method so we can assert pass-through.
type fakeDunningService struct {
	port.DunningService

	loadConfigCalls       []loadConfigCall
	loadConfigReturn      domain.DunningConfig
	loadConfigErr         error
	executeCalls          []executeCall
	executeReturn         domain.DunningAttempt
	executeErr            error
	updateCampaignCalls   []updateCampaignCall
	updateCampaignReturn  domain.DunningCampaign
	updateCampaignErr     error
	sendCommunicationCalls []sendCommunicationCall
	sendCommunicationErr  error
	markFailedCalls       []markFailedCall
	markFailedReturn      domain.DunningCampaign
	markFailedErr         error
	failAndCancelCalls    []markFailedCall
	failAndCancelReturn   domain.DunningCampaign
	failAndCancelErr      error
}

type loadConfigCall struct{ orgId, campaignId string }
type executeCall struct {
	orgId, campaignId string
	attemptType       domain.DunningAttemptType
}
type updateCampaignCall struct {
	attempt domain.DunningAttempt
	config  domain.DunningConfig
	ctx     domain.DunningAttemptContext
}
type sendCommunicationCall struct {
	orgId, campaignId string
	attemptNumber     int
}
type markFailedCall struct{ orgId, campaignId, reason string }

func (f *fakeDunningService) LoadConfigForCampaign(_ context.Context, orgId, campaignId string) (domain.DunningConfig, error) {
	f.loadConfigCalls = append(f.loadConfigCalls, loadConfigCall{orgId, campaignId})
	return f.loadConfigReturn, f.loadConfigErr
}
func (f *fakeDunningService) ExecuteAttempt(_ context.Context, orgId, campaignId string, t domain.DunningAttemptType) (domain.DunningAttempt, error) {
	f.executeCalls = append(f.executeCalls, executeCall{orgId, campaignId, t})
	return f.executeReturn, f.executeErr
}
func (f *fakeDunningService) UpdateCampaignWithAttemptResult(_ context.Context, a domain.DunningAttempt, c domain.DunningConfig, ac domain.DunningAttemptContext) (domain.DunningCampaign, error) {
	f.updateCampaignCalls = append(f.updateCampaignCalls, updateCampaignCall{a, c, ac})
	return f.updateCampaignReturn, f.updateCampaignErr
}
func (f *fakeDunningService) SendCommunication(_ context.Context, orgId, campaignId string, n int) error {
	f.sendCommunicationCalls = append(f.sendCommunicationCalls, sendCommunicationCall{orgId, campaignId, n})
	return f.sendCommunicationErr
}
func (f *fakeDunningService) MarkCampaignFailed(_ context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	f.markFailedCalls = append(f.markFailedCalls, markFailedCall{orgId, campaignId, reason})
	return f.markFailedReturn, f.markFailedErr
}
func (f *fakeDunningService) FailCampaignAndCancelSubscription(_ context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	f.failAndCancelCalls = append(f.failAndCancelCalls, markFailedCall{orgId, campaignId, reason})
	return f.failAndCancelReturn, f.failAndCancelErr
}

type fakeWebhookSubService struct {
	calls []port.OutgoingWebhookPayload
	err   error
}

func (f *fakeWebhookSubService) SendWebhook(_ context.Context, p port.OutgoingWebhookPayload) error {
	f.calls = append(f.calls, p)
	return f.err
}

// ---- DunningSteps tests ----

func TestDunningSteps_LoadConfigForCampaign(t *testing.T) {
	svc := &fakeDunningService{loadConfigReturn: domain.DunningConfig{EscalationRules: domain.EscalationRulesConfig{CancelAfterAttempt: 9}}}
	s := NewDunningSteps(noopLogger{}, svc)

	got, err := s.LoadConfigForCampaign(context.Background(), "org_1", "dc_1")

	require.NoError(t, err)
	assert.Equal(t, 9, got.EscalationRules.CancelAfterAttempt)
	require.Len(t, svc.loadConfigCalls, 1)
	assert.Equal(t, loadConfigCall{"org_1", "dc_1"}, svc.loadConfigCalls[0])
}

func TestDunningSteps_LoadConfigForCampaign_PropagatesError(t *testing.T) {
	svc := &fakeDunningService{loadConfigErr: errors.New("boom")}
	s := NewDunningSteps(noopLogger{}, svc)

	_, err := s.LoadConfigForCampaign(context.Background(), "org_1", "dc_1")

	assert.ErrorContains(t, err, "boom")
}

func TestDunningSteps_ExecuteAttempt(t *testing.T) {
	svc := &fakeDunningService{executeReturn: domain.DunningAttempt{Id: "att_1", Status: domain.PaymentStatusSucceeded}}
	s := NewDunningSteps(noopLogger{}, svc)

	got, err := s.ExecuteAttempt(context.Background(), "org_1", "dc_1", domain.DunningAttemptTypeProgressive)

	require.NoError(t, err)
	assert.Equal(t, "att_1", got.Id)
	require.Len(t, svc.executeCalls, 1)
	assert.Equal(t, executeCall{"org_1", "dc_1", domain.DunningAttemptTypeProgressive}, svc.executeCalls[0])
}

func TestDunningSteps_UpdateCampaignWithAttemptResult(t *testing.T) {
	svc := &fakeDunningService{updateCampaignReturn: domain.DunningCampaign{Id: "dc_1", Status: domain.DunningStatusRecovered}}
	s := NewDunningSteps(noopLogger{}, svc)

	att := domain.DunningAttempt{DunningCampaignId: "dc_1", AttemptNumber: 3, Status: domain.PaymentStatusSucceeded}
	cfg := domain.DunningConfig{EscalationRules: domain.EscalationRulesConfig{CancelAfterAttempt: 5}}
	ac := domain.DunningAttemptContext{AttemptNumber: 3, WasSubscriptionSuspended: true}

	got, err := s.UpdateCampaignWithAttemptResult(context.Background(), att, cfg, ac)

	require.NoError(t, err)
	assert.Equal(t, domain.DunningStatusRecovered, got.Status)
	require.Len(t, svc.updateCampaignCalls, 1)
	c := svc.updateCampaignCalls[0]
	assert.Equal(t, att, c.attempt)
	assert.Equal(t, cfg, c.config)
	assert.Equal(t, ac, c.ctx)
}

func TestDunningSteps_SendCommunication(t *testing.T) {
	svc := &fakeDunningService{}
	s := NewDunningSteps(noopLogger{}, svc)

	require.NoError(t, s.SendCommunication(context.Background(), "org_1", "dc_1", 3))
	require.Len(t, svc.sendCommunicationCalls, 1)
	assert.Equal(t, sendCommunicationCall{"org_1", "dc_1", 3}, svc.sendCommunicationCalls[0])

	svc.sendCommunicationErr = errors.New("queue down")
	assert.Error(t, s.SendCommunication(context.Background(), "org_1", "dc_1", 4), "service error propagates")
}

func TestDunningSteps_MarkCampaignFailed(t *testing.T) {
	svc := &fakeDunningService{markFailedReturn: domain.DunningCampaign{Status: domain.DunningStatusFailed}}
	s := NewDunningSteps(noopLogger{}, svc)

	got, err := s.MarkCampaignFailed(context.Background(), "org_1", "dc_1", "max_attempts_reached")

	require.NoError(t, err)
	assert.Equal(t, domain.DunningStatusFailed, got.Status)
	require.Len(t, svc.markFailedCalls, 1)
	assert.Equal(t, markFailedCall{"org_1", "dc_1", "max_attempts_reached"}, svc.markFailedCalls[0])
}

func TestDunningSteps_FailCampaignAndCancelSubscription(t *testing.T) {
	svc := &fakeDunningService{failAndCancelReturn: domain.DunningCampaign{Status: domain.DunningStatusFailed}}
	s := NewDunningSteps(noopLogger{}, svc)

	got, err := s.FailCampaignAndCancelSubscription(context.Background(), "org_1", "dc_1", "exhausted")

	require.NoError(t, err)
	assert.Equal(t, domain.DunningStatusFailed, got.Status)
	require.Len(t, svc.failAndCancelCalls, 1)
	assert.Equal(t, markFailedCall{"org_1", "dc_1", "exhausted"}, svc.failAndCancelCalls[0])
}

// ---- OutgoingWebhookSteps tests ----

func TestOutgoingWebhookSteps_SendWebhook(t *testing.T) {
	t.Run("forwards payload to the service", func(t *testing.T) {
		whSvc := &fakeWebhookSubService{}
		s := NewOutgoingWebhookSteps(noopLogger{}, nil, nil, whSvc, nil)
		payload := port.OutgoingWebhookPayload{}

		require.NoError(t, s.SendWebhook(context.Background(), payload))
		require.Len(t, whSvc.calls, 1)
	})

	t.Run("propagates the service error", func(t *testing.T) {
		whSvc := &fakeWebhookSubService{err: errors.New("503")}
		s := NewOutgoingWebhookSteps(noopLogger{}, nil, nil, whSvc, nil)

		assert.ErrorContains(t, s.SendWebhook(context.Background(), port.OutgoingWebhookPayload{}), "503")
	})
}
