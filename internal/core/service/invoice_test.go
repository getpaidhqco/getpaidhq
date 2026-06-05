package service

import (
	"context"
	"testing"

	"getpaidhq/internal/core/domain"
)

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
