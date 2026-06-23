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
	port.PaymentGateway
	res port.ChargePaymentResponse
}

func (g *chargeRecorderGateway) ChargePayment(ctx context.Context, cmd port.ChargePaymentInput) port.ChargePaymentResponse {
	return g.res
}

type subscriptionChargeGatewayFactory struct {
	port.GatewayFactory
	gw  port.PaymentGateway
	err error
}

func (f *subscriptionChargeGatewayFactory) NewGateway(ctx context.Context, orgId string, pspId string) (port.PaymentGateway, error) {
	return f.gw, f.err
}

type fakeInvoiceRepo struct {
	port.InvoiceRepository
	byCycle map[int]domain.Invoice
	byId    map[string]domain.Invoice
}

func newFakeInvoiceRepo() *fakeInvoiceRepo {
	return &fakeInvoiceRepo{byCycle: map[int]domain.Invoice{}, byId: map[string]domain.Invoice{}}
}

func (r *fakeInvoiceRepo) Create(_ context.Context, inv domain.Invoice) (domain.Invoice, error) {
	r.byCycle[inv.Cycle] = inv
	r.byId[inv.Id] = inv
	return inv, nil
}

func (r *fakeInvoiceRepo) FindBySubscriptionCycle(_ context.Context, _, _ string, cycle int) (domain.Invoice, error) {
	if inv, ok := r.byCycle[cycle]; ok {
		return inv, nil
	}
	return domain.Invoice{}, port.ErrNotFound
}

func (r *fakeInvoiceRepo) FindById(_ context.Context, _, id string) (domain.Invoice, error) {
	if inv, ok := r.byId[id]; ok {
		return inv, nil
	}
	return domain.Invoice{}, port.ErrNotFound
}

func (r *fakeInvoiceRepo) Update(_ context.Context, inv domain.Invoice) (domain.Invoice, error) {
	r.byId[inv.Id] = inv
	r.byCycle[inv.Cycle] = inv
	return inv, nil
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
		Currency: "USD", PspId: domain.Paystack,
	}

	cus := domain.Customer{Id: cusId, OrgId: orgId}
	pm := domain.PaymentMethod{Id: pmId, OrgId: orgId, Token: "tok_1"}

	setup := func() (*fakeSubRepo, *fakeCustomerRepo, *subscriptionChargeGatewayFactory, *SubscriptionService) {
		sr := &fakeSubRepo{sub: sub}
		cr := &fakeCustomerRepo{customer: cus, paymentMethod: pm}
		gf := &subscriptionChargeGatewayFactory{}
		er := lib.NewErrorReporter(silentLogger{})
		or := &fakeOrderRepo{items: []domain.OrderItem{{Id: "oi_1", PriceId: "price_1", Quantity: 1}}}
		pr := &fakePriceRepo{byId: domain.Price{Id: "price_1", UnitPrice: 1000}}
		is := NewInvoiceService(newFakeInvoiceRepo(), or, pr, nil, nil, silentLogger{}, nil, nil)
		svc, _ := NewSubscriptionService(nil, nil, nil, sr, cr, or, nil, pr, gf, is, &recordingPubSub{}, er, silentLogger{}, nil)
		return sr, cr, gf, svc
	}

	t.Run("successful charge returns mapped result", func(t *testing.T) {
		_, _, gf, svc := setup()
		gf.gw = &chargeRecorderGateway{res: port.ChargePaymentResponse{
			Status:        port.ChargePaymentStatusSuccess,
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
		gf.gw = &chargeRecorderGateway{res: port.ChargePaymentResponse{
			Status:      port.ChargePaymentStatusGatewayError,
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
