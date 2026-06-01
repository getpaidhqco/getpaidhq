package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// The Temporal activities are thin shims — they should forward each call to the
// service with the same args and propagate the service's result and error
// untouched. These tests pin that wiring contract.

// ---- fakes ----

type fakeDunningService struct {
	port.DunningService
	loadConfigCalls        []struct{ orgId, campaignId string }
	loadConfigReturn       domain.DunningConfig
	loadConfigErr          error
	executeCalls           []struct {
		orgId, campaignId string
		attemptType       domain.DunningAttemptType
	}
	executeReturn          domain.DunningAttempt
	executeErr             error
	updateCampaignCalls    []struct {
		attempt        domain.DunningAttempt
		config         domain.DunningConfig
		attemptContext domain.DunningAttemptContext
	}
	updateCampaignReturn   domain.DunningCampaign
	updateCampaignErr      error
	sendCommunicationCalls []struct {
		orgId, campaignId string
		attemptNumber     int
	}
	sendCommunicationErr   error
	markFailedCalls        []struct{ orgId, campaignId, reason string }
	markFailedReturn       domain.DunningCampaign
	markFailedErr          error
	failAndCancelCalls     []struct{ orgId, campaignId, reason string }
	failAndCancelReturn    domain.DunningCampaign
	failAndCancelErr       error
}

func (f *fakeDunningService) LoadConfigForCampaign(_ context.Context, orgId, campaignId string) (domain.DunningConfig, error) {
	f.loadConfigCalls = append(f.loadConfigCalls, struct{ orgId, campaignId string }{orgId, campaignId})
	return f.loadConfigReturn, f.loadConfigErr
}
func (f *fakeDunningService) ExecuteAttempt(_ context.Context, orgId, campaignId string, t domain.DunningAttemptType) (domain.DunningAttempt, error) {
	f.executeCalls = append(f.executeCalls, struct {
		orgId, campaignId string
		attemptType       domain.DunningAttemptType
	}{orgId, campaignId, t})
	return f.executeReturn, f.executeErr
}
func (f *fakeDunningService) UpdateCampaignWithAttemptResult(_ context.Context, a domain.DunningAttempt, c domain.DunningConfig, ac domain.DunningAttemptContext) (domain.DunningCampaign, error) {
	f.updateCampaignCalls = append(f.updateCampaignCalls, struct {
		attempt        domain.DunningAttempt
		config         domain.DunningConfig
		attemptContext domain.DunningAttemptContext
	}{a, c, ac})
	return f.updateCampaignReturn, f.updateCampaignErr
}
func (f *fakeDunningService) SendCommunication(_ context.Context, orgId, campaignId string, n int) error {
	f.sendCommunicationCalls = append(f.sendCommunicationCalls, struct {
		orgId, campaignId string
		attemptNumber     int
	}{orgId, campaignId, n})
	return f.sendCommunicationErr
}
func (f *fakeDunningService) MarkCampaignFailed(_ context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	f.markFailedCalls = append(f.markFailedCalls, struct{ orgId, campaignId, reason string }{orgId, campaignId, reason})
	return f.markFailedReturn, f.markFailedErr
}
func (f *fakeDunningService) FailCampaignAndCancelSubscription(_ context.Context, orgId, campaignId, reason string) (domain.DunningCampaign, error) {
	f.failAndCancelCalls = append(f.failAndCancelCalls, struct{ orgId, campaignId, reason string }{orgId, campaignId, reason})
	return f.failAndCancelReturn, f.failAndCancelErr
}

type fakeOrderWorkflowService struct {
	port.OrderWorkflowService
	completeCheckoutCalls  []domain.CompleteCheckoutSessionInput
	completeCheckoutReturn domain.Order
	completeCheckoutErr    error
}

func (f *fakeOrderWorkflowService) CompleteCheckoutSession(_ context.Context, in domain.CompleteCheckoutSessionInput) (domain.Order, error) {
	f.completeCheckoutCalls = append(f.completeCheckoutCalls, in)
	return f.completeCheckoutReturn, f.completeCheckoutErr
}

type fakePaymentService struct {
	port.PaymentService
	processRefundCalls  []domain.PaymentWebhookContext
	processRefundReturn domain.Payment
	processRefundErr    error
}

func (f *fakePaymentService) ProcessRefund(_ context.Context, ctx domain.PaymentWebhookContext) (domain.Payment, error) {
	f.processRefundCalls = append(f.processRefundCalls, ctx)
	return f.processRefundReturn, f.processRefundErr
}

type fakeSubscriptionRepository struct {
	port.SubscriptionRepository
	findByOrderIdCalls  []struct{ orgId, orderId string }
	findByOrderIdReturn []domain.Subscription
	findByOrderIdErr    error
	findByIdCalls       []struct{ orgId, id string }
	findByIdReturn      domain.Subscription
	findByIdErr         error
}

func (f *fakeSubscriptionRepository) FindByOrderId(_ context.Context, orgId, orderId string) ([]domain.Subscription, error) {
	f.findByOrderIdCalls = append(f.findByOrderIdCalls, struct{ orgId, orderId string }{orgId, orderId})
	return f.findByOrderIdReturn, f.findByOrderIdErr
}
func (f *fakeSubscriptionRepository) FindById(_ context.Context, orgId, id string) (domain.Subscription, error) {
	f.findByIdCalls = append(f.findByIdCalls, struct{ orgId, id string }{orgId, id})
	return f.findByIdReturn, f.findByIdErr
}

type fakeSubscriptionService struct {
	port.SubscriptionService
	chargeCalls            []domain.Subscription
	chargeReturn           domain.ChargeResult
	chargeErr              error
	handleChargeSuccessCalls []domain.SubscriptionChargeInput
	handleChargeSuccessReturn domain.Subscription
	handleChargeSuccessErr  error
	handleChargeFailureCalls []domain.SubscriptionChargeInput
	handleChargeFailureReturn domain.Subscription
	handleChargeFailureErr  error
	markAsErrorCalls       []struct {
		sub domain.Subscription
		err error
	}
	markAsErrorErr         error
	sendReminderCalls      []struct{ orgId, id string }
	sendReminderErr        error
}

func (f *fakeSubscriptionService) ChargeForBillingPeriod(_ context.Context, s domain.Subscription) (domain.ChargeResult, error) {
	f.chargeCalls = append(f.chargeCalls, s)
	return f.chargeReturn, f.chargeErr
}
func (f *fakeSubscriptionService) HandleSubscriptionChargeSuccess(_ context.Context, in domain.SubscriptionChargeInput) (domain.Subscription, error) {
	f.handleChargeSuccessCalls = append(f.handleChargeSuccessCalls, in)
	return f.handleChargeSuccessReturn, f.handleChargeSuccessErr
}
func (f *fakeSubscriptionService) HandleSubscriptionChargeFailure(_ context.Context, in domain.SubscriptionChargeInput) (domain.Subscription, error) {
	f.handleChargeFailureCalls = append(f.handleChargeFailureCalls, in)
	return f.handleChargeFailureReturn, f.handleChargeFailureErr
}
func (f *fakeSubscriptionService) MarkAsError(_ context.Context, s domain.Subscription, err error) error {
	f.markAsErrorCalls = append(f.markAsErrorCalls, struct {
		sub domain.Subscription
		err error
	}{s, err})
	return f.markAsErrorErr
}
func (f *fakeSubscriptionService) SendRenewalReminder(_ context.Context, orgId, id string) error {
	f.sendReminderCalls = append(f.sendReminderCalls, struct{ orgId, id string }{orgId, id})
	return f.sendReminderErr
}

type fakeWebhookSubService struct {
	port.WebhookSubscriptionService
	calls []port.OutgoingWebhookPayload
	err   error
}

func (f *fakeWebhookSubService) SendWebhook(_ context.Context, p port.OutgoingWebhookPayload) error {
	f.calls = append(f.calls, p)
	return f.err
}

// ---- DunningActivities tests ----

func TestDunningActivities(t *testing.T) {
	svc := &fakeDunningService{}
	a := NewDunningActivities(svc)
	ctx := context.Background()

	t.Run("LoadConfigForCampaign", func(t *testing.T) {
		svc.loadConfigReturn = domain.DunningConfig{ImmediateRetries: domain.ImmediateRetriesConfig{MaxAttempts: 3}}
		got, err := a.LoadConfigForCampaign(ctx, "org_1", "dc_1")
		require.NoError(t, err)
		assert.Equal(t, 3, got.ImmediateRetries.MaxAttempts)
		assert.Equal(t, "org_1", svc.loadConfigCalls[0].orgId)
	})

	t.Run("ExecuteAttempt", func(t *testing.T) {
		svc.executeReturn = domain.DunningAttempt{Id: "att_1"}
		got, err := a.ExecuteAttempt(ctx, "org_1", "dc_1", domain.DunningAttemptTypeImmediate)
		require.NoError(t, err)
		assert.Equal(t, "att_1", got.Id)
		assert.Equal(t, domain.DunningAttemptTypeImmediate, svc.executeCalls[0].attemptType)
	})

	t.Run("UpdateCampaignWithAttemptResult", func(t *testing.T) {
		svc.updateCampaignReturn = domain.DunningCampaign{Id: "dc_1"}
		att := domain.DunningAttempt{Id: "att_1"}
		cfg := domain.DunningConfig{}
		ac := domain.DunningAttemptContext{AttemptNumber: 1}
		got, err := a.UpdateCampaignWithAttemptResult(ctx, att, cfg, ac)
		require.NoError(t, err)
		assert.Equal(t, "dc_1", got.Id)
		assert.Equal(t, att, svc.updateCampaignCalls[0].attempt)
	})

	t.Run("SendCommunication", func(t *testing.T) {
		err := a.SendCommunication(ctx, "org_1", "dc_1", 1)
		require.NoError(t, err)
		assert.Equal(t, 1, svc.sendCommunicationCalls[0].attemptNumber)
	})

	t.Run("MarkCampaignFailed", func(t *testing.T) {
		svc.markFailedReturn = domain.DunningCampaign{Id: "dc_1", Status: domain.DunningStatusFailed}
		got, err := a.MarkCampaignFailed(ctx, "org_1", "dc_1", "reason")
		require.NoError(t, err)
		assert.Equal(t, domain.DunningStatusFailed, got.Status)
	})

	t.Run("FailCampaignAndCancelSubscription", func(t *testing.T) {
		svc.failAndCancelReturn = domain.DunningCampaign{Id: "dc_1", Status: domain.DunningStatusFailed}
		got, err := a.FailCampaignAndCancelSubscription(ctx, "org_1", "dc_1", "reason")
		require.NoError(t, err)
		assert.Equal(t, domain.DunningStatusFailed, got.Status)
	})
}

// ---- OrderActivities tests ----

func TestOrderActivities(t *testing.T) {
	oSvc := &fakeOrderWorkflowService{}
	pSvc := &fakePaymentService{}
	sRepo := &fakeSubscriptionRepository{}
	sSvc := &fakeSubscriptionService{}
	a := NewOrderActivities(oSvc, sSvc, pSvc, sRepo)
	ctx := context.Background()

	t.Run("CompleteOrder", func(t *testing.T) {
		oSvc.completeCheckoutReturn = domain.Order{Id: "ord_1"}
		in := domain.PaymentWebhookContext{OrgId: "org_1", OrderId: "ord_1"}
		got, err := a.CompleteOrder(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, "ord_1", got.Id)
		assert.Equal(t, "ord_1", oSvc.completeCheckoutCalls[0].OrderId)
	})

	t.Run("CompleteOrder_Error", func(t *testing.T) {
		oSvc.completeCheckoutErr = errors.New("boom")
		_, err := a.CompleteOrder(ctx, domain.PaymentWebhookContext{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Can't mark order as completed")
	})

	t.Run("HandlePaymentRefundedEvent", func(t *testing.T) {
		pSvc.processRefundReturn = domain.Payment{Id: "pmt_1"}
		in := domain.PaymentWebhookContext{OrgId: "org_1", OrderId: "ord_1"}
		got, err := a.HandlePaymentRefundedEvent(ctx, in)
		require.NoError(t, err)
		assert.Equal(t, "pmt_1", got.Id)
	})

	t.Run("GetOrderSubscriptions", func(t *testing.T) {
		sRepo.findByOrderIdReturn = []domain.Subscription{{Id: "sub_1"}}
		got, err := a.GetOrderSubscriptions(ctx, "org_1", "ord_1")
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, "sub_1", got[0].Id)
	})

	t.Run("ChargeCustomerForBillingPeriod", func(t *testing.T) {
		sSvc.chargeReturn = domain.ChargeResult{Status: domain.PaymentStatusSucceeded}
		sub := domain.Subscription{Id: "sub_1"}
		got, err := a.ChargeCustomerForBillingPeriod(ctx, sub)
		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusSucceeded, got.Status)
	})

	t.Run("HandleChargeResult_Success", func(t *testing.T) {
		sSvc.handleChargeSuccessReturn = domain.Subscription{Id: "sub_1"}
		sub := domain.Subscription{Id: "sub_1"}
		res := domain.ChargeResult{Status: domain.PaymentStatusSucceeded}
		got, err := a.HandleChargeResult(ctx, sub, res)
		require.NoError(t, err)
		assert.Equal(t, "sub_1", got.Id)
		assert.Len(t, sSvc.handleChargeSuccessCalls, 1)
	})

	t.Run("HandleChargeResult_Failure", func(t *testing.T) {
		sSvc.handleChargeFailureReturn = domain.Subscription{Id: "sub_1"}
		sub := domain.Subscription{Id: "sub_1"}
		res := domain.ChargeResult{Status: domain.PaymentStatusFailed}
		got, err := a.HandleChargeResult(ctx, sub, res)
		require.NoError(t, err)
		assert.Equal(t, "sub_1", got.Id)
		assert.Len(t, sSvc.handleChargeFailureCalls, 1)
	})

	t.Run("ErrorState", func(t *testing.T) {
		sub := domain.Subscription{Id: "sub_1"}
		err := a.ErrorState(ctx, sub, "oops")
		require.NoError(t, err)
		assert.Equal(t, "oops", sSvc.markAsErrorCalls[0].err.Error())
	})

	t.Run("GetSubscription", func(t *testing.T) {
		sRepo.findByIdReturn = domain.Subscription{Id: "sub_1"}
		got, err := a.GetSubscription(ctx, "org_1", "sub_1")
		require.NoError(t, err)
		assert.Equal(t, "sub_1", got.Id)
	})

	t.Run("ProcessReminderEvent", func(t *testing.T) {
		sub := domain.Subscription{OrgId: "org_1", Id: "sub_1"}
		err := a.ProcessReminderEvent(ctx, sub)
		require.NoError(t, err)
		assert.Equal(t, "sub_1", sSvc.sendReminderCalls[0].id)
	})
}

// ---- OutgoingWebhookActivities tests ----

func TestOutgoingWebhookActivities(t *testing.T) {
	svc := &fakeWebhookSubService{}
	a := NewOutgoingWebhookActivities(svc)
	ctx := context.Background()

	t.Run("SendWebhook", func(t *testing.T) {
		payload := port.OutgoingWebhookPayload{}
		err := a.SendWebhook(ctx, payload)
		require.NoError(t, err)
		assert.Len(t, svc.calls, 1)
	})

	t.Run("SendWebhook_Error", func(t *testing.T) {
		svc.err = errors.New("503")
		err := a.SendWebhook(ctx, port.OutgoingWebhookPayload{})
		assert.Error(t, err)
		assert.Equal(t, "503", err.Error())
	})
}
