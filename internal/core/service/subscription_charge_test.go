package service

import (
	"context"
	"errors"
	"sync"
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
	byOrder map[string]domain.Invoice
	mu      sync.Mutex
	counter map[string]int64
}

func newFakeInvoiceRepo() *fakeInvoiceRepo {
	return &fakeInvoiceRepo{byCycle: map[int]domain.Invoice{}, byId: map[string]domain.Invoice{}, byOrder: map[string]domain.Invoice{}, counter: map[string]int64{}}
}

func (r *fakeInvoiceRepo) Create(_ context.Context, inv domain.Invoice) (domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byCycle[inv.Cycle] = inv
	r.byId[inv.Id] = inv
	if inv.OrderId != "" {
		r.byOrder[inv.OrderId] = inv
	}
	return inv, nil
}

func (r *fakeInvoiceRepo) FindOrderInvoice(_ context.Context, _, orderId string) (domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inv, ok := r.byOrder[orderId]; ok {
		return inv, nil
	}
	return domain.Invoice{}, port.ErrNotFound
}

func (r *fakeInvoiceRepo) FindBySubscriptionCycle(_ context.Context, _, _ string, cycle int) (domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inv, ok := r.byCycle[cycle]; ok {
		return inv, nil
	}
	return domain.Invoice{}, port.ErrNotFound
}

func (r *fakeInvoiceRepo) FindById(_ context.Context, _, id string) (domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inv, ok := r.byId[id]; ok {
		return inv, nil
	}
	return domain.Invoice{}, port.ErrNotFound
}

func (r *fakeInvoiceRepo) Update(_ context.Context, inv domain.Invoice) (domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byId[inv.Id] = inv
	r.byCycle[inv.Cycle] = inv
	return inv, nil
}

func (r *fakeInvoiceRepo) NextInvoiceNumber(_ context.Context, orgId string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter[orgId]++
	return r.counter[orgId], nil
}

func (r *fakeInvoiceRepo) SetInvoiceCounter(_ context.Context, orgId string, value int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counter[orgId] = value
	return nil
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
		is := NewInvoiceService(newFakeInvoiceRepo(), or, pr, &fakeSubRepo{}, nil, nil, silentLogger{}, noopDiscountRepo{}, noopCouponRepo{}, noopReservationRepo{}, defaultSettingsResolver{})
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
