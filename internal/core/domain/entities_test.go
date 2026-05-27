package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrder_SetMetadata(t *testing.T) {
	t.Run("initialises nil map", func(t *testing.T) {
		o := &Order{}
		got := o.SetMetadata(map[string]string{"k": "v"})
		assert.Same(t, o, got)
		assert.Equal(t, map[string]string{"k": "v"}, o.Metadata)
	})

	t.Run("merges and overwrites", func(t *testing.T) {
		o := &Order{Metadata: map[string]string{"a": "1", "b": "2"}}
		o.SetMetadata(map[string]string{"b": "x", "c": "3"})
		assert.Equal(t, map[string]string{"a": "1", "b": "x", "c": "3"}, o.Metadata)
	})
}

func TestPayment_SetMetadata(t *testing.T) {
	t.Run("initialises nil map", func(t *testing.T) {
		p := &Payment{}
		got := p.SetMetadata(map[string]string{"k": "v"})
		assert.Same(t, p, got)
		assert.Equal(t, map[string]string{"k": "v"}, p.Metadata)
	})

	t.Run("merges and overwrites", func(t *testing.T) {
		p := &Payment{Metadata: map[string]string{"a": "1"}}
		p.SetMetadata(map[string]string{"a": "9", "b": "2"})
		assert.Equal(t, map[string]string{"a": "9", "b": "2"}, p.Metadata)
	})
}

func TestAddress_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		addr Address
		want bool
	}{
		{"zero value is empty", Address{}, true},
		{"only first name set is not empty", Address{FirstName: "Jane"}, false},
		{"only email set is not empty", Address{Email: "a@b.com"}, false},
		{"only country set is not empty", Address{Country: Country("US")}, false},
		{"fully populated is not empty", Address{
			FirstName: "Jane", LastName: "Doe", Email: "a@b.com", Phone: "123",
			Line1: "1 St", Line2: "Apt 2", City: "Town", State: "ST",
			PostalCode: "0000", Country: Country("US"),
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.addr.IsEmpty())
		})
	}
}

func TestParseAddress(t *testing.T) {
	t.Run("maps known fields", func(t *testing.T) {
		got := ParseAddress(map[string]any{
			"first_name": "Jane",
			"last_name":  "Doe",
			"email":      "jane@example.com",
			"line1":      "1 Main St",
			"city":       "Townsville",
			"country":    "US",
		})
		assert.Equal(t, "Jane", got.FirstName)
		assert.Equal(t, "Doe", got.LastName)
		assert.Equal(t, "jane@example.com", got.Email)
		assert.Equal(t, "1 Main St", got.Line1)
		assert.Equal(t, "Townsville", got.City)
		assert.Equal(t, Country("US"), got.Country)
	})

	t.Run("empty map yields empty address", func(t *testing.T) {
		got := ParseAddress(map[string]any{})
		assert.True(t, got.IsEmpty())
	})

	t.Run("unknown keys are ignored", func(t *testing.T) {
		got := ParseAddress(map[string]any{"unknown_field": "x", "first_name": "A"})
		assert.Equal(t, "A", got.FirstName)
	})
}

func TestCardDetail_GetExpiryDate(t *testing.T) {
	tests := []struct {
		name  string
		card  CardDetail
		want  time.Time
	}{
		{
			name: "valid month/year -> first of month UTC",
			card: CardDetail{ExpiryMonth: "12", ExpiryYear: "2030"},
			want: time.Date(2030, time.December, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "single digit month parses",
			card: CardDetail{ExpiryMonth: "3", ExpiryYear: "2026"},
			want: time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "non-numeric values default to zero -> month 0 normalises to Dec of prior year",
			card: CardDetail{ExpiryMonth: "ab", ExpiryYear: "cd"},
			want: time.Date(0, time.Month(0), 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.card.GetExpiryDate())
		})
	}
}

func TestParsePaymentMethodDetails(t *testing.T) {
	t.Run("card type parses into CardDetail", func(t *testing.T) {
		got, err := ParsePaymentMethodDetails(PaymentMethodTypeCard, map[string]any{
			"brand":        "visa",
			"last4":        "4242",
			"expiry_month": "11",
			"expiry_year":  "2029",
		})
		require.NoError(t, err)
		card, ok := got.(CardDetail)
		require.True(t, ok, "result should be a CardDetail")
		assert.Equal(t, "visa", card.Brand)
		assert.Equal(t, "4242", card.Last4)
		assert.Equal(t, time.Date(2029, time.November, 1, 0, 0, 0, 0, time.UTC), card.GetExpiryDate())
	})

	t.Run("unknown type returns error", func(t *testing.T) {
		got, err := ParsePaymentMethodDetails(PaymentMethodType("paypal"), map[string]any{})
		assert.Error(t, err)
		assert.Nil(t, got)
	})
}

func TestParsePaymentWebhookContext(t *testing.T) {
	t.Run("round-trips a populated struct", func(t *testing.T) {
		src := PaymentWebhookContext{
			Type:    PaymentSuccess,
			OrgId:   "org_1",
			OrderId: "order_1",
			Psp:     Paystack,
			Status:  "ok",
			Payment: GatewayPayment{
				Currency:  "USD",
				Reference: "ref_1",
				PspId:     "psp_1",
				Amount:    1500,
				Status:    PaymentStatusSucceeded,
			},
			Customer:      GatewayCustomer{Id: "cus_1", Email: "a@b.com"},
			PaymentMethod: GatewayPaymentMethod{PspId: "pm_1", Token: "tok_1", IsRecurring: true},
		}

		got, err := ParsePaymentWebhookContext(src)
		require.NoError(t, err)
		assert.Equal(t, src.Type, got.Type)
		assert.Equal(t, src.OrgId, got.OrgId)
		assert.Equal(t, src.OrderId, got.OrderId)
		assert.Equal(t, src.Psp, got.Psp)
		assert.Equal(t, src.Payment.Amount, got.Payment.Amount)
		assert.Equal(t, src.Payment.Status, got.Payment.Status)
		assert.Equal(t, src.Customer.Email, got.Customer.Email)
		assert.True(t, got.PaymentMethod.IsRecurring)
	})

	t.Run("parses from a generic map", func(t *testing.T) {
		got, err := ParsePaymentWebhookContext(map[string]any{
			"type":     "payment.success",
			"org_id":   "org_9",
			"order_id": "order_9",
		})
		require.NoError(t, err)
		assert.Equal(t, PaymentSuccess, got.Type)
		assert.Equal(t, "org_9", got.OrgId)
		assert.Equal(t, "order_9", got.OrderId)
	})

	t.Run("unmarshalable input returns error", func(t *testing.T) {
		// A channel cannot be marshalled to JSON.
		_, err := ParsePaymentWebhookContext(make(chan int))
		assert.Error(t, err)
	})
}

func TestNewPrice_IntervalDefaulting(t *testing.T) {
	t.Run("empty billing and trial intervals default to none", func(t *testing.T) {
		p := NewPrice("org_1", "var_1", CreatePriceInput{
			Currency:  "USD",
			UnitPrice: 1000,
		})
		assert.Equal(t, BillingIntervalNone, p.BillingInterval)
		assert.Equal(t, BillingIntervalNone, p.TrialInterval)
		assert.Equal(t, "org_1", p.OrgId)
		assert.Equal(t, "var_1", p.VariantId)
		assert.Equal(t, Currency("USD"), p.Currency)
		assert.NotEmpty(t, p.Id, "id generated")
	})

	t.Run("explicit intervals are preserved", func(t *testing.T) {
		p := NewPrice("org_1", "var_1", CreatePriceInput{
			Currency:           "EUR",
			UnitPrice:          500,
			BillingInterval:    BillingIntervalMonth,
			BillingIntervalQty: 1,
			TrialInterval:      BillingIntervalDay,
			TrialIntervalQty:   14,
		})
		assert.Equal(t, BillingIntervalMonth, p.BillingInterval)
		assert.Equal(t, BillingIntervalDay, p.TrialInterval)
		assert.Equal(t, 14, p.TrialIntervalQty)
	})
}

func TestNewFromCreateInput(t *testing.T) {
	now := time.Now().UTC()

	t.Run("no trial -> StartDate is now, TrialEndsAt zero", func(t *testing.T) {
		s := NewFromCreateInput(CreateSubscriptionInput{
			OrgId:              "org_1",
			Amount:             1000,
			Currency:           "USD",
			BillingInterval:    BillingIntervalMonth,
			BillingIntervalQty: 1,
			TrialInterval:      BillingIntervalNone,
		})
		assert.Equal(t, "org_1", s.OrgId)
		assert.Equal(t, SubscriptionStatusPending, s.Status)
		assert.WithinDuration(t, now, s.StartDate, 5*time.Second)
		assert.True(t, s.TrialEndsAt.IsZero())
		assert.Equal(t, s.StartDate.Day(), s.BillingAnchor)
		assert.NotEmpty(t, s.Id)
	})

	trialCases := []struct {
		name     string
		interval BillingInterval
		qty      int
		offset   func(time.Time) time.Time
	}{
		{"minute trial", BillingIntervalMinute, 30, func(t time.Time) time.Time { return t.Add(30 * time.Minute) }},
		{"hour trial", BillingIntervalHour(), 4, func(t time.Time) time.Time { return t.Add(4 * time.Hour) }},
		{"day trial", BillingIntervalDay, 7, func(t time.Time) time.Time { return t.AddDate(0, 0, 7) }},
		{"week trial", BillingIntervalWeek, 2, func(t time.Time) time.Time { return t.AddDate(0, 0, 14) }},
		{"month trial", BillingIntervalMonth, 1, func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }},
		{"year trial", BillingIntervalYear, 1, func(t time.Time) time.Time { return t.AddDate(1, 0, 0) }},
	}

	for _, tc := range trialCases {
		t.Run(tc.name+" pushes StartDate out and sets TrialEndsAt", func(t *testing.T) {
			before := time.Now().UTC()
			s := NewFromCreateInput(CreateSubscriptionInput{
				OrgId:              "org_1",
				Amount:             1000,
				Currency:           "USD",
				BillingInterval:    BillingIntervalMonth,
				BillingIntervalQty: 1,
				TrialInterval:      tc.interval,
				TrialIntervalQty:   tc.qty,
			})
			assert.WithinDuration(t, tc.offset(before), s.StartDate, 5*time.Second)
			assert.Equal(t, s.StartDate, s.TrialEndsAt, "TrialEndsAt mirrors the pushed StartDate")
		})
	}
}

func TestTableNames(t *testing.T) {
	tests := []struct {
		name  string
		got   string
		want  string
	}{
		{"subscription", Subscription{}.TableName(), "subscriptions"},
		{"order", Order{}.TableName(), "orders"},
		{"order_item", OrderItem{}.TableName(), "order_items"},
		{"payment", Payment{}.TableName(), "payments"},
		{"payment_method", PaymentMethod{}.TableName(), "payment_methods"},
		{"refund", Refund{}.TableName(), "refunds"},
		{"price", Price{}.TableName(), "prices"},
		{"cart", Cart{}.TableName(), "carts"},
		{"customer", Customer{}.TableName(), "customers"},
		{"customer_cohort", CustomerCohort{}.TableName(), "customer_cohorts"},
		{"cohort", Cohort{}.TableName(), "cohorts"},
		{"metadata_store", MetadataStore{}.TableName(), "metadata_store"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
