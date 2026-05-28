package handler

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"getpaidhq/internal/core/domain"
)

func TestPagination_ToDomainPagination(t *testing.T) {
	p := Pagination{Page: 2, Limit: 25, Offset: 25, SortDirection: "asc", SortBy: "created_at"}
	got := p.ToDomainPagination()
	assert.Equal(t, domain.Pagination{Page: 2, Limit: 25, Offset: 25, SortDirection: "asc", SortBy: "created_at"}, got)
}

func TestToSnakeCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"FirstName", "first_name"},
		{"BillingAnchor", "billing_anchor"},
		{"URL", "u_r_l"}, // current impl splits every uppercase
		{"already_snake", "already_snake"},
		{"", ""},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			assert.Equal(t, c.want, toSnakeCase(c.in))
		})
	}
}

// validateOne runs a single struct field through the validator so each tag's
// message branch can be exercised by FormatValidationErrors / validationErrorToText.
func validateOne(t *testing.T, v any) validator.ValidationErrors {
	t.Helper()
	val := validator.New()
	err := val.Struct(v)
	require.Error(t, err, "validator should report a failure for %T", v)
	verr, ok := err.(validator.ValidationErrors)
	require.True(t, ok, "expected validator.ValidationErrors")
	return verr
}

func TestValidationErrorToText_AllTags(t *testing.T) {
	type required struct {
		Field string `validate:"required"`
	}
	type email struct {
		Field string `validate:"email"`
	}
	type oneOf struct {
		Field string `validate:"oneof=a b c"`
	}
	type gt struct {
		Field int `validate:"gt=0"`
	}
	type gte struct {
		Field int `validate:"gte=1"`
	}
	type lt struct {
		Field int `validate:"lt=10"`
	}
	type lte struct {
		Field int `validate:"lte=10"`
	}
	type other struct {
		Field string `validate:"alphanum"`
	}

	cases := []struct {
		name     string
		errs     validator.ValidationErrors
		wantSub  string
	}{
		{"required", validateOne(t, required{}), "required"},
		{"email", validateOne(t, email{Field: "nope"}), "Invalid email"},
		{"oneof", validateOne(t, oneOf{Field: "z"}), "allowed values"},
		{"gt", validateOne(t, gt{Field: 0}), "greater than"},
		{"gte", validateOne(t, gte{Field: 0}), "greater or equal"},
		{"lt", validateOne(t, lt{Field: 100}), "less than"},
		{"lte", validateOne(t, lte{Field: 100}), "less than or equal"},
		{"default", validateOne(t, other{Field: "@@@"}), "Invalid value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := validationErrorToText(c.errs[0])
			assert.Contains(t, got, c.wantSub, "tag %q: got %q", c.errs[0].Tag(), got)
		})
	}
}

func TestFormatValidationErrors_ProducesFieldMessagePairs(t *testing.T) {
	type req struct {
		FirstName string `validate:"required"`
		Quantity  int    `validate:"gt=0"`
	}
	errs := validateOne(t, req{})
	out := FormatValidationErrors(errs)

	require.Len(t, out, 2)
	// Field names are snake_cased, messages come from validationErrorToText.
	fields := map[string]string{out[0]["field"]: out[0]["message"], out[1]["field"]: out[1]["message"]}
	assert.Contains(t, fields, "first_name")
	assert.Contains(t, fields["first_name"], "required")
	assert.Contains(t, fields, "quantity")
	assert.Contains(t, fields["quantity"], "greater than")
}

// ---- response.go entity mappers ----

func TestNewOrderItemFromEntity_MapsFieldsAndPrice(t *testing.T) {
	item := domain.OrderItem{
		Id:        "oi_1",
		OrderId:   "ord_1",
		PriceId:   "price_1",
		ProductId: "prod_1",
		VariantId: "var_1",
		Price: domain.Price{
			Id: "price_1", Currency: domain.Currency("USD"), UnitPrice: 5000,
			BillingInterval: domain.BillingInterval("month"), BillingIntervalQty: 1,
		},
		Description:   "desc",
		Quantity:      2,
		TaxTotal:      100,
		DiscountTotal: 50,
	}
	got := NewOrderItemFromEntity(item)

	assert.Equal(t, "oi_1", got.Id)
	assert.Equal(t, "ord_1", got.OrderId)
	assert.Equal(t, 2, got.Quantity)
	assert.Equal(t, int64(100), got.TaxTotal)
	assert.Equal(t, "price_1", got.Price.Id, "embedded price mapped via NewPriceFromEntity")
	assert.Equal(t, int64(5000), got.Price.UnitPrice)
}

func TestNewVariantFromEntity_MapsPricesArray(t *testing.T) {
	v := domain.Variant{
		Id:   "var_1",
		Name: "Annual",
		Prices: []domain.Price{
			{Id: "price_1", UnitPrice: 10000, Currency: domain.Currency("USD")},
			{Id: "price_2", UnitPrice: 12000, Currency: domain.Currency("EUR")},
		},
	}
	got := NewVariantFromEntity(v)

	assert.Equal(t, "var_1", got.Id)
	assert.Equal(t, "Annual", got.Name)
	require.Len(t, got.Prices, 2)
	assert.Equal(t, "price_1", got.Prices[0].Id)
	assert.Equal(t, int64(12000), got.Prices[1].UnitPrice)
}

func TestNewVariantFromEntity_NoPricesYieldsNilSlice(t *testing.T) {
	got := NewVariantFromEntity(domain.Variant{Id: "var_1", Name: "x"})
	assert.Nil(t, got.Prices)
}

func TestNewPriceFromEntity_AllFields(t *testing.T) {
	p := domain.Price{
		Id:                 "price_1",
		VariantId:          "var_1",
		Label:              "Monthly",
		Category:           domain.PriceCategory("subscription"),
		Scheme:             domain.PriceScheme("fixed"),
		Cycles:             12,
		Currency:           domain.Currency("USD"),
		UnitPrice:          1500,
		MinPrice:           100,
		SuggestedPrice:     2000,
		BillingInterval:    domain.BillingInterval("month"),
		BillingIntervalQty: 1,
		TrialInterval:      domain.BillingInterval("day"),
		TrialIntervalQty:   14,
		TaxCode:            "txcd_1",
	}
	got := NewPriceFromEntity(p)

	assert.Equal(t, p.Id, got.Id)
	assert.Equal(t, p.VariantId, got.VariantId)
	assert.Equal(t, p.Label, got.Label)
	assert.Equal(t, p.Category, got.Category)
	assert.Equal(t, p.Scheme, got.Scheme)
	assert.Equal(t, p.Cycles, got.Cycles)
	assert.Equal(t, p.Currency, got.Currency)
	assert.Equal(t, p.UnitPrice, got.UnitPrice)
	assert.Equal(t, p.MinPrice, got.MinPrice)
	assert.Equal(t, p.SuggestedPrice, got.SuggestedPrice)
	assert.Equal(t, p.BillingInterval, got.BillingInterval)
	assert.Equal(t, p.BillingIntervalQty, got.BillingIntervalQty)
	assert.Equal(t, p.TrialInterval, got.TrialInterval)
	assert.Equal(t, p.TrialIntervalQty, got.TrialIntervalQty)
	assert.Equal(t, p.TaxCode, got.TaxCode)
}

func TestNewProrationDetailsFromEntity_RoundTrip(t *testing.T) {
	d := domain.ProrationDetails{
		CreditAmount:     500,
		DaysCredited:     15,
		OldBillingAnchor: 1,
		NewBillingAnchor: 15,
	}
	got := NewProrationDetailsFromEntity(d)

	assert.Equal(t, d.CreditAmount, got.CreditAmount)
	assert.Equal(t, d.DaysCredited, got.DaysCredited)
	assert.Equal(t, d.OldBillingAnchor, got.OldBillingAnchor)
	assert.Equal(t, d.NewBillingAnchor, got.NewBillingAnchor)
}
