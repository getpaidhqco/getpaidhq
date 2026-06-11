package handler

import (
	"time"

	"github.com/shopspring/decimal"

	"getpaidhq/internal/core/domain"
)

// Pagination documents the shape of the paging response payload. The
// request-side helper that reads the four query params lives in
// context.go (GetPagination).
type Pagination struct {
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
	SortDirection string `json:"sort_order"`
	SortBy        string `json:"sort_by"`
}

func (p Pagination) ToDomainPagination() domain.Pagination {
	return domain.Pagination{
		Page:          p.Page,
		Limit:         p.Limit,
		Offset:        p.Offset,
		SortDirection: p.SortDirection,
		SortBy:        p.SortBy,
	}
}

// ---------------------------------------------------------------------------
// Address
// ---------------------------------------------------------------------------

// Address represents a physical address in request DTOs.
type Address struct {
	FirstName  string         `json:"first_name"`
	LastName   string         `json:"last_name"`
	Email      string         `json:"email"`
	Phone      string         `json:"phone"`
	Line1      string         `json:"line1"`
	Line2      string         `json:"line2"`
	City       string         `json:"city"`
	State      string         `json:"state"`
	PostalCode string         `json:"postal_code"`
	Country    domain.Country `json:"country"`
}

// ---------------------------------------------------------------------------
// Cart requests
// ---------------------------------------------------------------------------

type AddItemRequest struct {
	ProductId string `json:"product_id" validate:"required"`
	PriceId   string `json:"price_id" validate:"required"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemRequest struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"`
}

// ---------------------------------------------------------------------------
// Order requests
// ---------------------------------------------------------------------------

type CartInput struct {
	Currency string     `json:"currency"`
	Items    []CartItem `json:"items"`
}

type CartItem struct {
	ProductId string `json:"product_id" validate:"required"`
	PriceId   string `json:"price_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"required"`
}

type CreateOrderRequest struct {
	Customer        CreateOrderRequestCustomer `json:"customer" validate:"required"`
	PaymentMethodId string                     `json:"payment_method_id"`
	SessionId       string                     `json:"session_id"`
	PspId           string                     `json:"psp_id" validate:"required"`

	// Cart is required if SessionId is not provided
	Cart     CartInput         `json:"cart"`
	Metadata map[string]string `json:"metadata"`
	Options  map[string]string `json:"options"`
}

type CreateOrderRequestCustomer struct {
	ID        string            `json:"id"`
	Email     string            `json:"email"`
	FirstName string            `json:"first_name"`
	LastName  string            `json:"last_name"`
	Phone     string            `json:"phone"`
	Metadata  map[string]string `json:"metadata"`
}

type CompleteOrderRequest struct {
	PaymentMethodId string                          `json:"payment_method_id"`
	PaymentMethod   CompleteOrderInputPaymentMethod `json:"payment_method"`
	Payment         CompleteOrderRequestPayment     `json:"payment"`
	Metadata        map[string]string               `json:"metadata"`
}

type CompleteOrderInputPaymentMethod struct {
	Psp            string            `json:"psp"`
	Name           string            `json:"name"`
	IsDefault      bool              `json:"is_default"`
	BillingAddress Address           `json:"billing_address"`
	Type           string            `json:"type"`
	Details        any               `json:"details"`
	Token          string            `json:"token"`
	Metadata       map[string]string `json:"metadata"`
}

type CompleteOrderRequestPayment struct {
	PspId       string            `json:"psp_id"`
	Reference   string            `json:"reference"`
	Amount      int64             `json:"amount"`
	CompletedAt string            `json:"completed_at"`
	Metadata    map[string]string `json:"metadata"`
	Currency    string            `json:"currency"`
}

// ---------------------------------------------------------------------------
// Subscription requests
// ---------------------------------------------------------------------------

type CreateSubscriptionRequest struct {
	PaymentMethodId string `json:"payment_method_id" validate:"required"`

	Activate bool `json:"activate"`

	Amount   int    `json:"amount"  validate:"required"`
	Currency string `json:"currency"  validate:"required"`

	BillingInterval    domain.BillingInterval `json:"billing_interval"  validate:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  validate:"required"`
	Cycles             int                    `json:"cycles"`

	TrialInterval    domain.BillingInterval `json:"trial_interval"`
	TrialIntervalQty int                    `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}

type ActivateSubscriptionRequest struct {
	PaymentMethodId string `json:"payment_method_id" validate:"required"`

	Amount   int    `json:"amount"  validate:"required"`
	Currency string `json:"currency"  validate:"required"`

	BillingInterval    domain.BillingInterval `json:"billing_interval"  validate:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  validate:"required"`
	Cycles             int                    `json:"cycles"`

	TrialInterval    domain.BillingInterval `json:"trial_interval"`
	TrialIntervalQty int                    `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}

type PauseSubscriptionRequest struct {
	Reason string `json:"reason"`
}

type UpdateBillingAnchorRequest struct {
	// BillingAnchor is the new billing anchor as a day between 1 and 31.
	BillingAnchor int                  `json:"billing_anchor" validate:"required,gte=1,lte=31"`
	ProrationMode domain.ProrationMode `json:"proration_mode" validate:"required,oneof=none prorate"`
}

type ResumeSubscriptionRequest struct {
	ResumeBehavior domain.SubscriptionResumeBehavior `json:"resume_behavior"`
}

// ---------------------------------------------------------------------------
// Customer requests
// ---------------------------------------------------------------------------

type CreateCustomerRequest struct {
	Email          string            `json:"email" validate:"required"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	BillingAddress domain.Address    `json:"billing_address"`
	Phone          string            `json:"phone"`
	Metadata       map[string]string `json:"metadata"`
}

type CreatePaymentMethodRequest struct {
	Psp  string `json:"psp" validate:"required"`
	Name string `json:"name" validate:"required"`

	// Type of payment method, e.g. card, bank account, etc.
	Type           domain.PaymentMethodType `json:"type" validate:"required"`
	Details        any                      `json:"details"`
	Token          string                   `json:"token" validate:"required"`
	IsDefault      bool                     `json:"is_default"`
	BillingAddress domain.Address           `json:"billing_address"`
	Metadata       map[string]string        `json:"metadata"`
}

type UpdatePaymentMethodRequest struct {
	Name           string                   `json:"name"`
	Type           domain.PaymentMethodType `json:"type"`
	Details        any                      `json:"details"`
	Token          string                   `json:"token"`
	IsDefault      bool                     `json:"is_default"`
	BillingAddress domain.Address           `json:"billing_address"`
	Metadata       map[string]string        `json:"metadata"`
}

// CreatePaymentMethodInput wraps a CreatePaymentMethodRequest with org/customer context.
type CreatePaymentMethodInput struct {
	CreatePaymentMethodRequest
	OrgId      string
	CustomerId string
}

// UpdatePaymentMethodInput wraps an UpdatePaymentMethodRequest with org/customer/pm context.
type UpdatePaymentMethodInput struct {
	UpdatePaymentMethodRequest
	OrgId           string
	PaymentMethodId string
	CustomerId      string
}

// ---------------------------------------------------------------------------
// Product requests
// ---------------------------------------------------------------------------

type CreateProductRequest struct {
	Name        string                        `json:"name" validate:"required"`
	Description string                        `json:"description"`
	Metadata    map[string]string             `json:"metadata"`
	Variants    []CreateProductVariantRequest `json:"variants" validate:"required,dive"`
}

type UpdateProductRequest struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateVariantRequest struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type UpdateVariantRequest struct {
	Name        string            `json:"name" validate:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateProductVariantRequest struct {
	Name        string                      `json:"name" validate:"required"`
	Description string                      `json:"description"`
	Metadata    map[string]string           `json:"metadata"`
	Prices      []CreateProductPriceRequest `json:"prices" validate:"required,dive"`
}

type CreateProductPriceRequest struct {
	Label              string                 `json:"label" validate:"omitempty,min=1,max=255"`
	Category           domain.PriceCategory   `json:"category" validate:"required,oneof=one_time subscription free variable"`
	Scheme             domain.PriceScheme     `json:"scheme" validate:"required,oneof=fixed tiered volume graduated"`
	Cycles             int                    `json:"cycles" validate:"omitempty,gt=0"`
	Currency           string                 `json:"currency" validate:"required,iso4217"`
	UnitPrice          int64                  `json:"unit_price" validate:"gte=0"`
	// UnitCount is how many units unit_price buys (fixed scheme only): rate =
	// unit_price/unit_count cents per unit ("$1 per 1000 calls" = 100/1000).
	// Omitted or 1 = per single unit.
	UnitCount          int                    `json:"unit_count" validate:"omitempty,gte=1"`
	MinPrice           int64                  `json:"min_price" validate:"omitempty,gte=0"`
	SuggestedPrice     int64                  `json:"suggested_price" validate:"omitempty,gte=0"`
	BillingInterval    domain.BillingInterval `json:"billing_interval" validate:"omitempty,oneof=none minute hour day week month year"`
	BillingIntervalQty int                    `json:"billing_interval_qty" validate:"omitempty,gt=0,lte=999"`
	TrialInterval      domain.BillingInterval `json:"trial_interval" validate:"omitempty,oneof=none minute hour day week month year"`
	TrialIntervalQty   int                    `json:"trial_interval_qty" validate:"omitempty,gt=0,lte=999"`
	TaxCode            string                 `json:"tax_code" validate:"omitempty,alphanum"`
	BillableMetricId   string                 `json:"billable_metric_id" validate:"omitempty"`
	Tiers              []PriceTierRequest     `json:"tiers" validate:"omitempty,dive"`
	// FilterField/FilterValue scope a metered price to one slice of its meter (a value
	// of one of the meter's filters). filter_field empty = whole meter; filter_field
	// set with empty filter_value = the default/catch-all charge.
	FilterField string `json:"filter_field" validate:"omitempty,max=255"`
	FilterValue string `json:"filter_value" validate:"omitempty,max=255"`
	// Proration switches for prices on weighted_sum carry-over meters; inert otherwise.
	ProrateOnIncrease bool              `json:"prorate_on_increase"`
	CreditOnDecrease  bool              `json:"credit_on_decrease"`
	Metadata          map[string]string `json:"metadata"`
}

type CreatePriceRequest struct {
	VariantId          string                 `json:"variant_id" validate:"required"`
	Category           domain.PriceCategory   `json:"category" validate:"required,oneof=one_time subscription free variable"`
	Scheme             domain.PriceScheme     `json:"scheme" validate:"required,oneof=fixed tiered volume graduated"`
	Cycles             int                    `json:"cycles" validate:"omitempty,gt=0"`
	Label              string                 `json:"label"`
	Currency           string                 `json:"currency" validate:"required,iso4217"`
	UnitPrice          int64                  `json:"unit_price" validate:"gte=0"`
	// UnitCount is how many units unit_price buys (fixed scheme only); see
	// CreateProductPriceRequest.UnitCount.
	UnitCount          int                    `json:"unit_count" validate:"omitempty,gte=1"`
	MinPrice           int64                  `json:"min_price" validate:"omitempty,gte=0"`
	SuggestedPrice     int64                  `json:"suggested_price" validate:"omitempty,gte=0"`
	BillingInterval    domain.BillingInterval `json:"billing_interval" validate:"omitempty,oneof=none minute hour day week month year"`
	BillingIntervalQty int                    `json:"billing_interval_qty" validate:"omitempty,gt=0,lte=999"`
	TrialInterval      domain.BillingInterval `json:"trial_interval" validate:"omitempty,oneof=none minute hour day week month year"`
	TrialIntervalQty   int                    `json:"trial_interval_qty" validate:"omitempty,gt=0,lte=999"`
	TaxCode            string                 `json:"tax_code" validate:"omitempty,alphanum"`
	BillableMetricId   string                 `json:"billable_metric_id" validate:"omitempty"`
	Tiers              []PriceTierRequest     `json:"tiers" validate:"omitempty,dive"`
	FilterField        string                 `json:"filter_field" validate:"omitempty,max=255"`
	FilterValue        string                 `json:"filter_value" validate:"omitempty,max=255"`
	ProrateOnIncrease  bool                   `json:"prorate_on_increase"`
	CreditOnDecrease   bool                   `json:"credit_on_decrease"`
	Metadata           map[string]string      `json:"metadata"`
}

// PriceTierRequest is one rate band for graduated/volume/tiered schemes. Decimal
// amounts are sent as strings to preserve precision (e.g. sub-cent per-unit rates).
// to_value empty/"0" means the last, unbounded tier.
type PriceTierRequest struct {
	FromValue     string `json:"from_value" validate:"omitempty,numeric"`
	ToValue       string `json:"to_value" validate:"omitempty,numeric"`
	PerUnitAmount string `json:"per_unit_amount" validate:"omitempty,numeric"`
	FlatAmount    int64  `json:"flat_amount" validate:"omitempty,gte=0"`
}

// toDomainTiers parses the string decimal fields into domain.PriceTier.
func toDomainTiers(in []PriceTierRequest) ([]domain.PriceTier, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]domain.PriceTier, len(in))
	for i, t := range in {
		from, err := decimalOrZero(t.FromValue)
		if err != nil {
			return nil, err
		}
		to, err := decimalOrZero(t.ToValue)
		if err != nil {
			return nil, err
		}
		per, err := decimalOrZero(t.PerUnitAmount)
		if err != nil {
			return nil, err
		}
		out[i] = domain.PriceTier{FromValue: from, ToValue: to, PerUnitAmount: per, FlatAmount: t.FlatAmount}
	}
	return out, nil
}

func decimalOrZero(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}

// ---------------------------------------------------------------------------
// Org requests: see org_request.go (CreateOrgRequest)
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// PSP requests
// ---------------------------------------------------------------------------

type CreateGatewayRequest struct {
	Name     string            `json:"name" validate:"required"`
	PspId    string            `json:"psp" validate:"required"`
	Settings map[string]string `json:"settings" validate:"required"`
}

// ---------------------------------------------------------------------------
// Report requests
// ---------------------------------------------------------------------------

type GetMRRRequest struct {
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
}

type GetARRRequest struct {
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date" validate:"required"`
}

// ---------------------------------------------------------------------------
// Webhook subscription requests
// ---------------------------------------------------------------------------

type CreateWebhookSubscriptionRequest struct {
	Url    string   `json:"url" validate:"required"`
	Events []string `json:"events" validate:"required"`
	Secret string   `json:"secret"`
}
