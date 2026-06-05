package service

import (
	"testing"

	"getpaidhq/internal/core/domain"
)

// fixed builds a fixed (non-metered) recurring subscription item.
func fixed(id string) orderLine {
	return orderLine{item: domain.OrderItem{Id: id}, price: domain.Price{Id: "p_" + id, Category: domain.PriceCategorySubscription}}
}

// metered builds a usage-priced subscription item (metered = has a meter attached).
func metered(id string) orderLine {
	return orderLine{item: domain.OrderItem{Id: id}, price: domain.Price{Id: "p_" + id, Category: domain.PriceCategorySubscription, BillableMetricId: "met_" + id}}
}

// oneTime builds a non-recurring item (never anchors a subscription).
func oneTime(id string) orderLine {
	return orderLine{item: domain.OrderItem{Id: id}, price: domain.Price{Id: "p_" + id, Category: domain.OneTime}}
}

func ids(lines []orderLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.item.Id
	}
	return out
}

func TestSubscriptionAnchors(t *testing.T) {
	tests := []struct {
		name  string
		lines []orderLine
		want  []string
	}{
		{"fixed only", []orderLine{fixed("a")}, []string{"a"}},
		{"metered only is its own subscription", []orderLine{metered("a")}, []string{"a"}},
		{"fixed + metered: only the fixed anchors (metered billed on it)", []orderLine{fixed("plan"), metered("usage")}, []string{"plan"}},
		{"two fixed + metered: both fixed anchor, metered does not", []orderLine{fixed("p1"), fixed("p2"), metered("u")}, []string{"p1", "p2"}},
		{"two metered, no plan: first metered anchors once", []orderLine{metered("in"), metered("out")}, []string{"in"}},
		{"one-time only: no subscription", []orderLine{oneTime("a")}, nil},
		{"empty", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ids(subscriptionAnchors(tt.lines))
			if len(got) != len(tt.want) {
				t.Fatalf("anchors = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("anchors = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
