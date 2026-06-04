package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
	"getpaidhq/internal/lib"
)

type chargeRecorderGateway struct {
	domain.GatewayProvider
	res domain.ChargePaymentResponse
}

func (g *chargeRecorderGateway) ChargePayment(ctx context.Context, cmd domain.ChargePaymentCommand) domain.ChargePaymentResponse {
	return g.res
}

type subscriptionChargeGatewayFactory struct {
	port.GatewayFactory
	gw  domain.GatewayProvider
	err error
}

func (f *subscriptionChargeGatewayFactory) NewGateway(ctx context.Context, orgId string, pspId string) (domain.GatewayProvider, error) {
	return f.gw, f.err
}

func TestSubscriptionService_ChargeForBillingPeriod(t *testing.T) {
	const (
		orgId = "org_1"
		subId = "sub_1"
		cusId = "cus_1"
		pmId  = "pm_1"
	)

	sub := domain.Subscription{
		OrgId: orgId, Id: subId, CustomerId: cusId, PaymentMethodId: pmId,
		Amount: 1000, Currency: "USD", PspId: domain.Paystack,
	}

	cus := domain.Customer{Id: cusId, OrgId: orgId}
	pm := domain.PaymentMethod{Id: pmId, OrgId: orgId, Token: "tok_1"}

	setup := func() (*fakeSubRepo, *fakeCustomerRepo, *subscriptionChargeGatewayFactory, *SubscriptionService) {
		sr := &fakeSubRepo{sub: sub}
		cr := &fakeCustomerRepo{customer: cus, paymentMethod: pm}
		gf := &subscriptionChargeGatewayFactory{}
		er := lib.NewErrorReporter(silentLogger{})
		svc, _ := NewSubscriptionService(nil, nil, nil, sr, cr, nil, nil, nil, gf, &recordingPubSub{}, er, silentLogger{}, nil)
		return sr, cr, gf, svc
	}

	t.Run("successful charge returns mapped result", func(t *testing.T) {
		_, _, gf, svc := setup()
		gf.gw = &chargeRecorderGateway{res: domain.ChargePaymentResponse{
			Status:        domain.ChargePaymentStatusSuccess,
			Psp:           domain.Paystack,
			AmountCharged: 1000,
			Reference:     "ref_1",
		}}

		got, err := svc.ChargeForBillingPeriod(context.Background(), sub)

		require.NoError(t, err)
		assert.Equal(t, domain.PaymentStatusSucceeded, got.Status)
		assert.Equal(t, int64(1000), got.Amount)
		assert.Equal(t, "ref_1", got.Reference)
	})

	t.Run("gateway error returns wrapped error", func(t *testing.T) {
		_, _, gf, svc := setup()
		gf.gw = &chargeRecorderGateway{res: domain.ChargePaymentResponse{
			Status:      domain.GatewayError,
			ErrorReason: "insufficient_funds",
		}}

		_, err := svc.ChargeForBillingPeriod(context.Background(), sub)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "gateway error: insufficient_funds")
	})

	t.Run("subscription not found propagation", func(t *testing.T) {
		sr, _, _, svc := setup()
		sr.findErr = errors.New("not found")

		_, err := svc.ChargeForBillingPeriod(context.Background(), sub)
		require.Error(t, err)
	})
}
