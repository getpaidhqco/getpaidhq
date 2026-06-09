package service

import (
	"testing"

	"getpaidhq/internal/core/domain"
)

// recurring builds a recurring line at a given cadence (fixed unless a meter id
// is given). metered lines also carry a billing interval — usage is billed at a
// cadence too.
func recurring(id string, interval domain.BillingInterval, qty int, meterId string) orderLine {
	return orderLine{
		item: domain.OrderItem{Id: id},
		price: domain.Price{
			Id:                 "p_" + id,
			Category:           domain.PriceCategorySubscription,
			BillingInterval:    interval,
			BillingIntervalQty: qty,
			BillableMetricId:   meterId,
		},
	}
}

// oneTime builds a non-recurring line (no billing interval → starts no subscription).
func oneTime(id string) orderLine {
	return orderLine{item: domain.OrderItem{Id: id}, price: domain.Price{Id: "p_" + id, Category: domain.OneTime, BillingInterval: domain.BillingIntervalNone}}
}

func groupItemIds(groups [][]orderLine) [][]string {
	out := make([][]string, len(groups))
	for i, g := range groups {
		ids := make([]string, len(g))
		for j, l := range g {
			ids[j] = l.item.Id
		}
		out[i] = ids
	}
	return out
}

func TestGroupIntoSubscriptions(t *testing.T) {
	month := domain.BillingIntervalMonth
	year := domain.BillingIntervalYear

	t.Run("flat + metered on one cadence form one group of two", func(t *testing.T) {
		groups := groupIntoSubscriptions([]orderLine{
			recurring("plan", month, 1, ""),
			recurring("usage", month, 1, "tokens"),
		})
		got := groupItemIds(groups)
		if len(got) != 1 || len(got[0]) != 2 {
			t.Fatalf("want one group of two lines, got %v", got)
		}
	})

	t.Run("two cadences form two groups", func(t *testing.T) {
		groups := groupIntoSubscriptions([]orderLine{
			recurring("monthly", month, 1, ""),
			recurring("yearly", year, 1, ""),
		})
		if got := groupItemIds(groups); len(got) != 2 {
			t.Fatalf("want two groups, got %v", got)
		}
	})

	t.Run("one-time line is not grouped", func(t *testing.T) {
		if groups := groupIntoSubscriptions([]orderLine{oneTime("setup")}); len(groups) != 0 {
			t.Fatalf("want no groups, got %v", groupItemIds(groups))
		}
	})

	t.Run("metered-only on one cadence is its own subscription", func(t *testing.T) {
		groups := groupIntoSubscriptions([]orderLine{
			recurring("in", month, 1, "ingress"),
			recurring("out", month, 1, "egress"),
		})
		got := groupItemIds(groups)
		if len(got) != 1 || len(got[0]) != 2 {
			t.Fatalf("want one group of two metered lines, got %v", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		if groups := groupIntoSubscriptions(nil); len(groups) != 0 {
			t.Fatalf("want no groups, got %v", groups)
		}
	})
}

// The credit-risk rule: a metered line is capped at monthly, so an annual base
// + usage (even if the usage price is configured annually) splits into an annual
// base subscription and a separate monthly usage subscription.
func TestGroupIntoSubscriptions_MeteredCappedToMonthly(t *testing.T) {
	year := domain.BillingIntervalYear
	groups := groupIntoSubscriptions([]orderLine{
		recurring("base", year, 1, ""),        // annual base
		recurring("usage", year, 1, "tokens"), // usage, configured annual → forced monthly
	})
	if len(groups) != 2 {
		t.Fatalf("annual base + usage must split into two subscriptions, got %v", groupItemIds(groups))
	}
}
