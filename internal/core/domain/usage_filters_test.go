package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestBillableMetric_FilterValues(t *testing.T) {
	m := BillableMetric{Filters: []MetricFilter{
		{Field: "type", Values: []string{"SMS", "MMS"}},
		{Field: "region", Values: []string{"us"}},
	}}
	if got := m.FilterValues("type"); len(got) != 2 || got[0] != "SMS" || got[1] != "MMS" {
		t.Fatalf("FilterValues(type) = %v", got)
	}
	if got := m.FilterValues("missing"); got != nil {
		t.Fatalf("FilterValues(missing) = %v, want nil", got)
	}
}

func TestPrice_IsDefaultFilter(t *testing.T) {
	cases := []struct {
		field, value string
		want         bool
	}{
		{"", "", false},        // not filtered at all
		{"type", "SMS", false}, // a specific slice
		{"type", "", true},     // the catch-all charge
	}
	for _, c := range cases {
		p := Price{FilterField: c.field, FilterValue: c.value}
		if p.IsDefaultFilter() != c.want {
			t.Errorf("IsDefaultFilter(field=%q,value=%q) = %v, want %v", c.field, c.value, p.IsDefaultFilter(), c.want)
		}
	}
}

func TestUsageLineFromPriceGrouped(t *testing.T) {
	// $0.05/msg, 200 messages for project=acme.
	p := Price{OrgId: "org_1", Id: "price_1", Label: "Messages", Scheme: Fixed, UnitPrice: 5}
	line := UsageLineFromPriceGrouped("org_1", "inv_1", p, "project", "acme", decimal.NewFromInt(200))

	if line.Kind != InvoiceLineKindUsage {
		t.Errorf("kind = %v, want usage", line.Kind)
	}
	if line.Total != 1000 { // 200 × 5 cents
		t.Errorf("total = %d, want 1000", line.Total)
	}
	if got := line.Metadata["project"]; got != "acme" {
		t.Errorf("metadata[project] = %q, want acme", got)
	}
	if line.Description != "Messages (project=acme)" {
		t.Errorf("description = %q", line.Description)
	}
	// Same rate as the ungrouped line — only the quantity differs.
	plain := UsageLineFromPrice("org_1", "inv_1", p, decimal.NewFromInt(200))
	if !line.UnitAmount.Equal(plain.UnitAmount) || line.Total != plain.Total {
		t.Errorf("grouped rate/total diverged from ungrouped: %s/%d vs %s/%d",
			line.UnitAmount, line.Total, plain.UnitAmount, plain.Total)
	}
}
