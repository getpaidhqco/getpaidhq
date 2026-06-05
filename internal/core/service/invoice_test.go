package service

import (
	"context"
	"testing"

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
	inPrice := domain.Price{OrgId: "org_1", Id: "price_in", Category: domain.PriceCategoryMetered, Scheme: domain.Fixed, UnitPrice: 3, BillableMetricId: "met_in"}
	outPrice := domain.Price{OrgId: "org_1", Id: "price_out", Category: domain.PriceCategoryMetered, Scheme: domain.Fixed, UnitPrice: 15, BillableMetricId: "met_out"}

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
	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", OrderId: "ord_1", OrderItemId: "oi_plan", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive}
	es := &usageEventStore{count: 100} // 100 units measured for each meter
	usage := newUsageSvc(meters, customers, &usageSubRepo{metered: []domain.Subscription{sub}}, es)

	svc := NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, usage, nil, silentLogger{})
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
	return NewInvoiceService(newFakeInvoiceRepo(), orderRepo, priceRepo, usage, nil, silentLogger{})
}

func TestInvoiceService_BuildForBillingPeriod_Trial(t *testing.T) {
	// ADR 0003: a trial waives the base fee on a non-metered price → no base line.
	price := domain.Price{OrgId: "org_1", Id: "price_1", Category: domain.PriceCategorySubscription, Scheme: domain.Fixed, UnitPrice: 1000}
	svc := newInvoiceServiceForTest(price, nil)

	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", OrderItemId: "oi_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusTrial}
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

	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", OrderItemId: "oi_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive}
	inv, err := svc.BuildForBillingPeriod(context.Background(), sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(inv.LineItems) != 1 || inv.Total != 1000 {
		t.Fatalf("active sub should bill base line 1000, got lines=%d total=%d", len(inv.LineItems), inv.Total)
	}
	if inv.LineItems[0].Kind != domain.InvoiceLineKindBase {
		t.Errorf("expected base line kind, got %s", inv.LineItems[0].Kind)
	}
}

func TestInvoiceService_BuildForBillingPeriod_Metered(t *testing.T) {
	// Metered Fixed price @ 10 cents/unit, count meter measuring 3 events → 30 cents.
	price := domain.Price{OrgId: "org_1", Id: "price_1", Category: domain.PriceCategoryMetered, Scheme: domain.Fixed, UnitPrice: 10, BillableMetricId: "met_1"}

	meters := &usageMeterRepo{byId: map[string]domain.BillableMetric{"met_1": countMeter()}}
	customers := &usageCustomerRepo{byId: map[string]domain.Customer{"cus_1": {OrgId: "org_1", Id: "cus_1"}}}
	es := &usageEventStore{count: 3}
	sub := domain.Subscription{OrgId: "org_1", Id: "sub_1", OrderItemId: "oi_1", CustomerId: "cus_1", Status: domain.SubscriptionStatusActive}
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
