package service

import (
	"testing"

	"getpaidhq/internal/core/domain"
)

func line(id string, cat domain.PriceCategory) orderLine {
	return orderLine{
		item:  domain.OrderItem{Id: id},
		price: domain.Price{Id: "p_" + id, Category: cat},
	}
}

func ids(lines []orderLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.item.Id
	}
	return out
}

func TestSubscriptionAnchors(t *testing.T) {
	const sub = domain.PriceCategorySubscription
	const met = domain.PriceCategoryMetered
	const once = domain.OneTime

	tests := []struct {
		name  string
		lines []orderLine
		want  []string
	}{
		{"fixed only", []orderLine{line("a", sub)}, []string{"a"}},
		{"metered only is its own subscription", []orderLine{line("a", met)}, []string{"a"}},
		{"fixed + metered: only the fixed anchors (metered billed on it)", []orderLine{line("plan", sub), line("usage", met)}, []string{"plan"}},
		{"two fixed + metered: both fixed anchor, metered does not", []orderLine{line("p1", sub), line("p2", sub), line("u", met)}, []string{"p1", "p2"}},
		{"two metered, no plan: first metered anchors once", []orderLine{line("in", met), line("out", met)}, []string{"in"}},
		{"one-time only: no subscription", []orderLine{line("a", once)}, nil},
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
