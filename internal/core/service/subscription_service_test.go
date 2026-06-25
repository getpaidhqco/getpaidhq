package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

// fakeSettingRepo serves a single configurable setting (used by GetRetryPolicy).
type fakeSettingRepo struct {
	port.SettingRepository
	setting domain.Setting
	findErr error
}

func (r *fakeSettingRepo) FindById(_ context.Context, _, _, _ string) (domain.Setting, error) {
	if r.findErr != nil {
		return domain.Setting{}, r.findErr
	}
	return r.setting, nil
}

// newSubscriptionService wires SubscriptionService with the subset of fakes a
// test needs; unused ports are nil. pubsub defaults to a recorder (the
// constructor subscribes, so it must be non-nil).
func newSubscriptionService(subRepo port.SubscriptionRepository, setting port.SettingRepository, customer port.CustomerRepository, order port.OrderRepository, payment port.PaymentRepository, ps *recordingPubSub) *SubscriptionService {
	if ps == nil {
		ps = &recordingPubSub{}
	}
	// Invoice service resolves the per-cycle charge amount from the linked price.
	// Wire it to yield total = 1000 (unit price 1000 × qty 1), matching the cost
	// the charge-handler tests assert against.
	invOrderRepo := &fakeOrderRepo{items: []domain.OrderItem{{Id: "oi_1", PriceId: "price_1", Quantity: 1}}}
	invPriceRepo := &fakePriceRepo{byId: domain.Price{Id: "price_1", UnitPrice: 1000}}
	invoiceSvc := NewInvoiceService(newFakeInvoiceRepo(), invOrderRepo, invPriceRepo, nil, nil, silentLogger{}, nil, nil, nil)
	// The subscription service's own price repo backs cadence grouping in
	// CreateSubscriptionsForOrder; give it a monthly recurring price.
	subPriceRepo := &fakePriceRepo{byId: domain.Price{Id: "price_1", Category: domain.PriceCategorySubscription, BillingInterval: domain.BillingIntervalMonth, BillingIntervalQty: 1, UnitPrice: 1000}}
	svc, err := NewSubscriptionService(nil, setting, nil, subRepo, customer, order, payment, subPriceRepo, nil, invoiceSvc, ps, lib.ErrorReporter{}, silentLogger{}, nil)
	if err != nil {
		panic(err)
	}
	return svc
}

func TestSubscriptionService_Create(t *testing.T) {
	subRepo := &fakeSubRepo{}
	ps := &recordingPubSub{}
	svc := newSubscriptionService(subRepo, nil, nil, nil, nil, ps)

	got, err := svc.Create(context.Background(), port.CreateSubscriptionInput{
		OrgId: "org_1", PaymentMethodId: "pm_1", Amount: 1000, Currency: "USD",
	})

	require.NoError(t, err)
	assert.Equal(t, "org_1", got.OrgId)
	assert.Len(t, subRepo.created, 1, "subscription persisted")
	assert.True(t, ps.hasTopic(port.TopicSubscriptionCreated))
}

func TestSubscriptionService_PauseSubscription(t *testing.T) {
	t.Run("active subscription is paused", func(t *testing.T) {
		subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive}}
		svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

		got, err := svc.PauseSubscription(context.Background(), port.PauseSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusPaused, got.Status)
		require.Len(t, subRepo.updated, 1)
		assert.Equal(t, domain.SubscriptionStatusPaused, subRepo.updated[0].Status)
	})

	t.Run("already paused is rejected and not re-written", func(t *testing.T) {
		subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPaused}}
		svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

		_, err := svc.PauseSubscription(context.Background(), port.PauseSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

		require.Error(t, err)
		assert.Empty(t, subRepo.updated, "no update on rejection")
	})
}

func TestSubscriptionService_ResumeSubscription(t *testing.T) {
	t.Run("non-paused subscription is rejected", func(t *testing.T) {
		subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive}}
		svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

		_, err := svc.ResumeSubscription(context.Background(), port.ResumeSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

		require.Error(t, err)
		assert.Empty(t, subRepo.updated)
	})

	t.Run("start-new-billing-period resumes to active with a future renewal", func(t *testing.T) {
		subRepo := &fakeSubRepo{sub: domain.Subscription{
			OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPaused,
			BillingInterval: domain.BillingInterval("month"), BillingIntervalQty: 1,
		}}
		svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

		before := time.Now().UTC()
		got, err := svc.ResumeSubscription(context.Background(), port.ResumeSubscriptionInput{
			OrgId: "org_1", Id: "sub_1", ResumeBehavior: domain.StartNewBillingPeriod,
		})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusActive, got.Status)
		assert.True(t, got.RenewsAt.After(before), "renewal moved into the future")
	})

	t.Run("continue-existing-period rejects when next billing is already past", func(t *testing.T) {
		// No billing interval → CalculateNextBillingDate returns the zero time,
		// which is before now → the service refuses to continue the period.
		subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusPaused}}
		svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

		_, err := svc.ResumeSubscription(context.Background(), port.ResumeSubscriptionInput{
			OrgId: "org_1", Id: "sub_1", ResumeBehavior: domain.ContinueExistingBillingPeriod,
		})

		require.Error(t, err)
	})
}

func TestSubscriptionService_CancelSubscription(t *testing.T) {
	t.Run("active subscription is cancelled at period end", func(t *testing.T) {
		renews := time.Now().UTC().Add(72 * time.Hour)
		subRepo := &fakeSubRepo{sub: domain.Subscription{
			OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusActive, RenewsAt: renews,
		}}
		svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

		got, err := svc.CancelSubscription(context.Background(), port.CancelSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusCancelled, got.Status)
		assert.Equal(t, renews, got.CancelAt, "cancellation honours the current period end")
		assert.False(t, got.CancelledAt.IsZero(), "cancellation timestamp recorded")
	})

	t.Run("already cancelled is rejected", func(t *testing.T) {
		subRepo := &fakeSubRepo{sub: domain.Subscription{OrgId: "org_1", Id: "sub_1", Status: domain.SubscriptionStatusCancelled}}
		svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

		_, err := svc.CancelSubscription(context.Background(), port.CancelSubscriptionInput{OrgId: "org_1", Id: "sub_1"})

		require.Error(t, err)
		assert.Empty(t, subRepo.updated)
	})
}

func TestSubscriptionService_HandleSubscriptionChargeSuccess(t *testing.T) {
	charge := func() domain.ChargeResult {
		return domain.ChargeResult{Psp: domain.Paystack, Status: domain.PaymentStatusSucceeded, Amount: 1000, Currency: "USD"}
	}

	t.Run("recurring success advances the cycle and stays active", func(t *testing.T) {
		subRepo := &fakeSubRepo{}
		payRepo := &fakePaymentRepo{}
		ps := &recordingPubSub{}
		svc := newSubscriptionService(subRepo, nil, nil, nil, payRepo, ps)

		sub := domain.Subscription{
			OrgId: "org_1", Id: "sub_1", Cycles: 0, CyclesProcessed: 0, TotalRevenue: 0,
			BillingInterval: domain.BillingInterval("month"), BillingIntervalQty: 1,
		}
		got, err := svc.HandleSubscriptionChargeSuccess(context.Background(), port.SubscriptionChargeInput{Subscription: sub, ChargeResult: charge()})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusActive, got.Status)
		assert.Equal(t, 1, got.CyclesProcessed)
		assert.Equal(t, int64(1000), got.TotalRevenue)
		assert.Equal(t, 0, got.Retries, "retry counter reset on success")
		assert.Len(t, payRepo.created, 1, "payment row recorded")
		assert.True(t, ps.hasTopic(port.TopicSubscriptionPaymentChargeSuccess))
	})

	t.Run("final cycle completes the subscription", func(t *testing.T) {
		subRepo := &fakeSubRepo{}
		payRepo := &fakePaymentRepo{}
		ps := &recordingPubSub{}
		svc := newSubscriptionService(subRepo, nil, nil, nil, payRepo, ps)

		sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", Cycles: 2, CyclesProcessed: 1}
		got, err := svc.HandleSubscriptionChargeSuccess(context.Background(), port.SubscriptionChargeInput{Subscription: sub, ChargeResult: charge()})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusCompleted, got.Status)
		assert.Equal(t, 2, got.CyclesProcessed)
		assert.True(t, ps.hasTopic(port.TopicSubscriptionCompleted))
	})
}

func TestSubscriptionService_HandleSubscriptionChargeFailure(t *testing.T) {
	failCharge := domain.ChargeResult{Psp: domain.Paystack, Status: domain.PaymentStatusFailed, Amount: 1000, Currency: "USD", ErrorReason: "card_declined"}

	t.Run("with retries remaining goes past-due and schedules a retry", func(t *testing.T) {
		subRepo := &fakeSubRepo{}
		payRepo := &fakePaymentRepo{}
		// No setting → default policy (3 attempts). Retries=0 → retry remains.
		settingRepo := &fakeSettingRepo{findErr: errors.New("not set")}
		ps := &recordingPubSub{}
		svc := newSubscriptionService(subRepo, settingRepo, nil, nil, payRepo, ps)

		sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", Retries: 0, RenewsAt: time.Now().UTC()}
		got, err := svc.HandleSubscriptionChargeFailure(context.Background(), port.SubscriptionChargeInput{Subscription: sub, ChargeResult: failCharge})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusPastDue, got.Status)
		assert.Equal(t, 1, got.Retries)
		assert.False(t, got.NextRetryAt.IsZero(), "next retry scheduled")
		assert.Len(t, payRepo.created, 1)
		assert.True(t, ps.hasTopic(port.TopicSubscriptionPastDue), "first past-due publishes the event")
	})

	t.Run("retries exhausted with cancel policy cancels the subscription", func(t *testing.T) {
		subRepo := &fakeSubRepo{}
		payRepo := &fakePaymentRepo{}
		settingRepo := &fakeSettingRepo{findErr: errors.New("not set")} // default policy: 3 attempts, cancel
		ps := &recordingPubSub{}
		svc := newSubscriptionService(subRepo, settingRepo, nil, nil, payRepo, ps)

		sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", Retries: 3, RenewsAt: time.Now().UTC()}
		got, err := svc.HandleSubscriptionChargeFailure(context.Background(), port.SubscriptionChargeInput{Subscription: sub, ChargeResult: failCharge})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusCancelled, got.Status)
		assert.True(t, ps.hasTopic(port.TopicSubscriptionCancelled))
	})

	t.Run("retries exhausted with mark-unpaid policy marks unpaid", func(t *testing.T) {
		subRepo := &fakeSubRepo{}
		payRepo := &fakePaymentRepo{}
		policy, _ := json.Marshal(domain.RetryPolicy{
			RetryAttempts: 2, RetryInterval: domain.RetryIntervalDay, RetryPeriod: 7, FailureAction: domain.FailureActionMarkUnpaid,
		})
		settingRepo := &fakeSettingRepo{setting: domain.Setting{Value: string(policy)}}
		ps := &recordingPubSub{}
		svc := newSubscriptionService(subRepo, settingRepo, nil, nil, payRepo, ps)

		sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", Retries: 2, RenewsAt: time.Now().UTC()}
		got, err := svc.HandleSubscriptionChargeFailure(context.Background(), port.SubscriptionChargeInput{Subscription: sub, ChargeResult: failCharge})

		require.NoError(t, err)
		assert.Equal(t, domain.SubscriptionStatusUnpaid, got.Status)
		assert.True(t, ps.hasTopic(port.TopicSubscriptionUnpaid))
	})
}

func TestSubscriptionService_GetRetryPolicy(t *testing.T) {
	t.Run("missing setting falls back to the default policy", func(t *testing.T) {
		svc := newSubscriptionService(&fakeSubRepo{}, &fakeSettingRepo{findErr: errors.New("not set")}, nil, nil, nil, nil)

		p := svc.GetRetryPolicy(context.Background(), "org_1")

		assert.Equal(t, 3, p.RetryAttempts)
		assert.Equal(t, domain.FailureActionCancel, p.FailureAction)
	})

	t.Run("valid setting is parsed", func(t *testing.T) {
		raw, _ := json.Marshal(domain.RetryPolicy{RetryAttempts: 7, RetryInterval: domain.RetryIntervalHour, RetryPeriod: 2, FailureAction: domain.FailureActionMarkUnpaid})
		svc := newSubscriptionService(&fakeSubRepo{}, &fakeSettingRepo{setting: domain.Setting{Value: string(raw)}}, nil, nil, nil, nil)

		p := svc.GetRetryPolicy(context.Background(), "org_1")

		assert.Equal(t, 7, p.RetryAttempts)
		assert.Equal(t, domain.FailureActionMarkUnpaid, p.FailureAction)
	})

	t.Run("invalid JSON falls back to the default policy", func(t *testing.T) {
		svc := newSubscriptionService(&fakeSubRepo{}, &fakeSettingRepo{setting: domain.Setting{Value: "{not json"}}, nil, nil, nil, nil)

		p := svc.GetRetryPolicy(context.Background(), "org_1")

		assert.Equal(t, 3, p.RetryAttempts)
	})
}

func TestSubscriptionService_MarkAsError(t *testing.T) {
	subRepo := &fakeSubRepo{}
	svc := newSubscriptionService(subRepo, nil, nil, nil, nil, nil)

	err := svc.MarkAsError(context.Background(), domain.Subscription{OrgId: "org_1", Id: "sub_1"}, errors.New("boom"))

	require.NoError(t, err)
	require.Len(t, subRepo.updated, 1)
	assert.Equal(t, domain.SubscriptionStatusError, subRepo.updated[0].Status)
	assert.Equal(t, "boom", subRepo.updated[0].Metadata["error"])
}

func TestSubscriptionService_CreateSubscriptionsForOrder(t *testing.T) {
	subRepo := &fakeSubRepo{}
	orderRepo := &fakeOrderRepo{
		order: domain.Order{OrgId: "org_1", Id: "ord_1", Status: domain.OrderStatusCompleted},
		items: []domain.OrderItem{{OrgId: "org_1", Id: "oi_1", OrderId: "ord_1"}, {OrgId: "org_1", Id: "oi_2", OrderId: "ord_1"}},
	}
	svc := newSubscriptionService(subRepo, nil, nil, orderRepo, nil, nil)

	got, err := svc.CreateSubscriptionsForOrder(context.Background(), "org_1", "ord_1")

	require.NoError(t, err)
	// Both items share one (monthly) cadence → one subscription owning both lines.
	assert.Len(t, got, 1)
	assert.Len(t, subRepo.created, 1)
	assert.Equal(t, domain.SubscriptionStatusActive, subRepo.created[0].Status, "completed order activates its subscription")
}
