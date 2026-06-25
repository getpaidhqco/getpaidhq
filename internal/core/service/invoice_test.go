package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

// mapPriceRepo resolves prices by id (the single-value fakePriceRepo can't back a
// multi-item order).
type mapPriceRepo struct {
	port.PriceRepository
	byId map[string]domain.Price
}

func (r *mapPriceRepo) FindById(_ context.Context, _, id string) (domain.Price, error) {
	if p, ok := r.byId[id]; ok {
		return p, nil
	}
	return domain.Price{}, port.ErrNotFound
}

func TestInvoiceService_BuildForBillingPeriod_Multiline(t *testing.T) {
	// One order = flat plan ($50) + input tokens (3c/unit) + output tokens (15c/unit).
	plan := domain.Price{OrgId: "org_1", Id: "price_plan", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 5000}
	inPrice := domain.Price{OrgId: "org_1", Id: "price_in", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 3, BillableMetricId: "met_in"}
	outPrice := domain.Price{OrgId: "org_1", Id: "price_out", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 15, BillableMetricId: "met_out"}

	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{
		{OrgId: "org_1", Id: "oi_plan", OrderId: "ord_1", PriceId: "price_plan", Quantity: 1},
		{OrgId: "org_1", Id: "oi_in", OrderId: "ord_1", PriceId: "price_in"},
		{OrgId: "org_1", Id: "oi_out", OrderId: "ord_1", PriceId: "price_out"},
	}}
	priceRepo := &mapPriceRepo{byId: map[string]domain.Price{"price_plan": plan, "price_in": inPrice, "price_out": outPrice}}

	meters := &usageMeterRepo{byId: map[string]domain.BillableMetric{
		"met_in":  {OrgId: "org_1", Id: "met_in", Code: "input_tokens", Aggregation: domain.AggregationCount},
		"met_out": {OrgId: "org_1", Id: "met_out", Code: "output_tokens", Aggregation: domain.AggregationCount},
	}}
	customers := &usageCustomerRepo{byId: map[string]domain.Customer{"cus_1": {OrgId: "org_1", Id: "cus_1"}}}
	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", OrderId: "ord_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive}
	es := &usageEventStore{count: 100} // 100 units measured for each meter
	usage := newUsageSvc(meters, customers, &usageSubRepo{metered: []domain.Subscription{sub}}, es)

	svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, nil, usage, nil, silentLogger{}, nil, nil, nil)
	inv, err := svc.BuildForBillingPeriod(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.LineItems) != 3 {
		t.Fatalf("want 3 lines (flat + 2 usage), got %d", len(inv.LineItems))
	}
	// flat 5000 + input 100*3=300 + output 100*15=1500 = 6800
	if inv.Total != 6800 {
		t.Errorf("multiline total = %d, want 6800 (5000 + 300 + 1500)", inv.Total)
	}
	var base, usageLines int
	for _, l := range inv.LineItems {
		switch l.Kind {
		case domain.InvoiceLineKindBase:
			base++
		case domain.InvoiceLineKindUsage:
			usageLines++
		}
	}
	if base != 1 || usageLines != 2 {
		t.Errorf("want 1 base + 2 usage lines, got base=%d usage=%d", base, usageLines)
	}
}

func newInvoiceServiceForTest(price domain.Price, usage *UsageService) *InvoiceService {
	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{{Id: "oi_1", PriceId: "price_1", Quantity: 1}}}
	priceRepo := &fakePriceRepo{byId: price}
	return NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, nil, usage, nil, silentLogger{}, nil, nil, nil)
}

// fakeInvoiceSettingsResolver returns a fixed InvoiceSettings for reference tests.
type fakeInvoiceSettingsResolver struct{ cfg domain.InvoiceSettings }

func (r fakeInvoiceSettingsResolver) ResolveInvoiceSettings(_ context.Context, _ string) (domain.InvoiceSettings, error) {
	return r.cfg, nil
}

func TestInvoiceService_BuildForBillingPeriod_FormatsReference(t *testing.T) {
	price := domain.Price{OrgId: "org_1", Id: "price_1", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 1000}
	orderRepo := &fakeOrderRepo{items: []domain.OrderItem{{Id: "oi_1", PriceId: "price_1", Quantity: 1}}}
	priceRepo := &fakePriceRepo{byId: price}
	resolver := fakeInvoiceSettingsResolver{cfg: domain.InvoiceSettings{Prefix: "INV-", Padding: 6}}
	svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, nil, nil, nil, silentLogger{}, nil, nil, resolver)

	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive}
	inv, err := svc.BuildForBillingPeriod(context.Background(), sub)
	require.NoError(t, err)
	require.EqualValues(t, 1, inv.Number)
	require.Equal(t, "INV-000001", inv.Reference)
}

func TestInvoiceService_BuildForBillingPeriod_Trial(t *testing.T) {
	// ADR 0003: a trial waives the base fee on a non-metered price → no base line.
	price := domain.Price{OrgId: "org_1", Id: "price_1", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 1000}
	svc := newInvoiceServiceForTest(price, nil)

	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusTrial}
	inv, err := svc.BuildForBillingPeriod(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.LineItems) != 0 {
		t.Errorf("trial should waive the base line, got %d lines", len(inv.LineItems))
	}
	if inv.Total != 0 {
		t.Errorf("trial base-waived invoice total = %d, want 0", inv.Total)
	}
}

func TestInvoiceService_BuildForBillingPeriod_NonTrialBaseLine(t *testing.T) {
	price := domain.Price{OrgId: "org_1", Id: "price_1", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 1000}
	svc := newInvoiceServiceForTest(price, nil)

	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive}
	inv, err := svc.BuildForBillingPeriod(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.LineItems) != 1 || inv.Total != 1000 {
		t.Fatalf("active sub should bill base line 1000, got lines=%d total=%d", len(inv.LineItems), inv.Total)
	}
	if inv.Number != 1 {
		t.Errorf("first created invoice number = %d, want 1", inv.Number)
	}
	if inv.LineItems[0].Kind != domain.InvoiceLineKindBase {
		t.Errorf("expected base line kind, got %s", inv.LineItems[0].Kind)
	}
}

func TestInvoiceService_CounterMethods(t *testing.T) {
	repo := newFakeInvoiceRepo()
	svc := NewInvoiceService(repo, nil, nil, nil, nil, nil, silentLogger{}, nil, nil, nil)
	ctx := context.Background()

	first, err := svc.NextInvoiceNumber(ctx, "org_1")
	require.NoError(t, err)
	require.EqualValues(t, 1, first)

	require.NoError(t, svc.SetInvoiceCounter(ctx, "org_1", 41))
	next, err := svc.NextInvoiceNumber(ctx, "org_1")
	require.NoError(t, err)
	require.EqualValues(t, 42, next)

	otherOrg, err := svc.NextInvoiceNumber(ctx, "org_2")
	require.NoError(t, err)
	require.EqualValues(t, 1, otherOrg)
}

func TestInvoiceService_BuildForBillingPeriod_Metered(t *testing.T) {
	// Metered Fixed price @ 10 cents/unit, count meter measuring 3 events → 30 cents.
	price := domain.Price{OrgId: "org_1", Id: "price_1", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 10, BillableMetricId: "met_1"}

	meters := &usageMeterRepo{byId: map[string]domain.BillableMetric{"met_1": countMeter()}}
	customers := &usageCustomerRepo{byId: map[string]domain.Customer{"cus_1": {OrgId: "org_1", Id: "cus_1"}}}
	es := &usageEventStore{count: 3}
	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive}
	usage := newUsageSvc(meters, customers, &usageSubRepo{metered: []domain.Subscription{sub}}, es)

	svc := newInvoiceServiceForTest(price, usage)
	inv, err := svc.BuildForBillingPeriod(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.LineItems) != 1 {
		t.Fatalf("metered invoice should have one usage line, got %d", len(inv.LineItems))
	}
	if inv.LineItems[0].Kind != domain.InvoiceLineKindUsage {
		t.Errorf("expected usage line kind, got %s", inv.LineItems[0].Kind)
	}
	if inv.Total != 30 {
		t.Errorf("metered total = %d, want 30 (3 units × 10c)", inv.Total)
	}
}

// minimalInvoiceRepo is a minimal InvoiceRepository fake for transition tests.
// It holds a single invoice that FindById always returns, and Update stores.
type minimalInvoiceRepo struct {
	inv domain.Invoice
}

func (r *minimalInvoiceRepo) Create(_ context.Context, in domain.Invoice) (domain.Invoice, error) {
	r.inv = in
	return in, nil
}
func (r *minimalInvoiceRepo) NextInvoiceNumber(_ context.Context, _ string) (int64, error) {
	return 1, nil
}
func (r *minimalInvoiceRepo) SetInvoiceCounter(_ context.Context, _ string, _ int64) error {
	return nil
}
func (r *minimalInvoiceRepo) Update(_ context.Context, in domain.Invoice) (domain.Invoice, error) {
	r.inv = in
	return in, nil
}
func (r *minimalInvoiceRepo) FindById(_ context.Context, _, _ string) (domain.Invoice, error) {
	return r.inv, nil
}
func (r *minimalInvoiceRepo) FindBySubscriptionCycle(_ context.Context, _, _ string, _ int) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}
func (r *minimalInvoiceRepo) FindOrderInvoice(_ context.Context, _, _ string) (domain.Invoice, error) {
	return domain.Invoice{}, port.ErrNotFound
}
func (r *minimalInvoiceRepo) FindBySubscriptionId(_ context.Context, _, _ string, _ domain.Pagination) ([]domain.Invoice, int, error) {
	return nil, 0, nil
}
func (r *minimalInvoiceRepo) List(_ context.Context, _ string, _ domain.Pagination) ([]domain.Invoice, int, error) {
	return nil, 0, nil
}

func TestInvoiceServiceTransitions(t *testing.T) {
	repo := &minimalInvoiceRepo{inv: domain.Invoice{Id: "inv_1", Status: domain.InvoiceStatusOpen}}
	s := &InvoiceService{invoiceRepository: repo}
	ctx := context.Background()

	got, err := s.MarkUncollectible(ctx, "org", "inv_1")
	require.NoError(t, err)
	require.Equal(t, domain.InvoiceStatusUncollectible, got.Status)

	repo.inv = domain.Invoice{Id: "inv_1", Status: domain.InvoiceStatusPaid}
	_, err = s.MarkUncollectible(ctx, "org", "inv_1")
	require.ErrorIs(t, err, domain.ErrInvalidInvoiceTransition)
}
