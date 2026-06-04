package port_test

import (
	"testing"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"github.com/stretchr/testify/assert"
)

// TestCreateSubscriptionInput_ToSubscription mirrors the old domain.TestNewFromCreateInput.
// It moved here when NewFromCreateInput was replaced by CreateSubscriptionInput.ToSubscription.
func TestCreateSubscriptionInput_ToSubscription(t *testing.T) {
	now := time.Now().UTC()

	t.Run("no trial -> StartDate is now, TrialEndsAt zero", func(t *testing.T) {
		s := port.CreateSubscriptionInput{
			OrgId:              "org_1",
			Amount:             1000,
			Currency:           "USD",
			BillingInterval:    domain.BillingIntervalMonth,
			BillingIntervalQty: 1,
			TrialInterval:      domain.BillingIntervalNone,
		}.ToSubscription()
		assert.Equal(t, "org_1", s.OrgId)
		assert.Equal(t, domain.SubscriptionStatusPending, s.Status)
		assert.WithinDuration(t, now, s.StartDate, 5*time.Second)
		assert.True(t, s.TrialEndsAt.IsZero())
		assert.Equal(t, s.StartDate.Day(), s.BillingAnchor)
		assert.NotEmpty(t, s.Id)
	})

	trialCases := []struct {
		name     string
		interval domain.BillingInterval
		qty      int
		offset   func(time.Time) time.Time
	}{
		{"minute trial", domain.BillingIntervalMinute, 30, func(t time.Time) time.Time { return t.Add(30 * time.Minute) }},
		{"hour trial", domain.BillingInterval("hour"), 4, func(t time.Time) time.Time { return t.Add(4 * time.Hour) }},
		{"day trial", domain.BillingIntervalDay, 7, func(t time.Time) time.Time { return t.AddDate(0, 0, 7) }},
		{"week trial", domain.BillingIntervalWeek, 2, func(t time.Time) time.Time { return t.AddDate(0, 0, 14) }},
		{"month trial", domain.BillingIntervalMonth, 1, func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }},
		{"year trial", domain.BillingIntervalYear, 1, func(t time.Time) time.Time { return t.AddDate(1, 0, 0) }},
	}

	for _, tc := range trialCases {
		t.Run(tc.name+" pushes StartDate out and sets TrialEndsAt", func(t *testing.T) {
			before := time.Now().UTC()
			s := port.CreateSubscriptionInput{
				OrgId:              "org_1",
				Amount:             1000,
				Currency:           "USD",
				BillingInterval:    domain.BillingIntervalMonth,
				BillingIntervalQty: 1,
				TrialInterval:      tc.interval,
				TrialIntervalQty:   tc.qty,
			}.ToSubscription()
			assert.WithinDuration(t, tc.offset(before), s.StartDate, 5*time.Second)
			assert.Equal(t, s.StartDate, s.TrialEndsAt, "TrialEndsAt mirrors the pushed StartDate")
		})
	}
}

// TestCreatePriceInput_ToPrice mirrors the old domain.TestNewPrice_IntervalDefaulting.
// It moved here when NewPrice was replaced by CreatePriceInput.ToPrice.
func TestCreatePriceInput_ToPrice(t *testing.T) {
	t.Run("empty billing and trial intervals default to none", func(t *testing.T) {
		p := port.CreatePriceInput{
			Currency:  "USD",
			UnitPrice: 1000,
		}.ToPrice("org_1", "var_1")
		assert.Equal(t, domain.BillingIntervalNone, p.BillingInterval)
		assert.Equal(t, domain.BillingIntervalNone, p.TrialInterval)
		assert.Equal(t, "org_1", p.OrgId)
		assert.Equal(t, "var_1", p.VariantId)
		assert.Equal(t, domain.Currency("USD"), p.Currency)
		assert.NotEmpty(t, p.Id, "id generated")
	})

	t.Run("explicit intervals are preserved", func(t *testing.T) {
		p := port.CreatePriceInput{
			Currency:           "EUR",
			UnitPrice:          500,
			BillingInterval:    domain.BillingIntervalMonth,
			BillingIntervalQty: 1,
			TrialInterval:      domain.BillingIntervalDay,
			TrialIntervalQty:   14,
		}.ToPrice("org_1", "var_1")
		assert.Equal(t, domain.BillingIntervalMonth, p.BillingInterval)
		assert.Equal(t, domain.BillingIntervalDay, p.TrialInterval)
		assert.Equal(t, 14, p.TrialIntervalQty)
	})
}
