package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"getpaidhq/internal/core/domain"
)

// ---------------------------------------------------------------------------
// Pagination
// ---------------------------------------------------------------------------

const (
	PageDefault  = "0"
	LimitDefault = "10"
	PageTag      = "page"
	LimitTag     = "limit"
)

// Pagination holds paging parameters extracted from the query string.
// swagger:parameters listSubscriptions
type Pagination struct {
	Page          int    `json:"page"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
	SortDirection string `json:"sort_order"`
	SortBy        string `json:"sort_by"`
}

// GetPagination reads pagination parameters from the gin context and returns a domain.Pagination.
func GetPagination(c *gin.Context) domain.Pagination {
	page, err := strconv.Atoi(c.DefaultQuery(PageTag, PageDefault))
	if err != nil || page < 1 {
		page = 0
	}

	limit, err := strconv.Atoi(c.DefaultQuery(LimitTag, LimitDefault))
	if err != nil {
		limit = 10
	}
	sortOrder := c.DefaultQuery("sort_order", "desc")
	sortBy := c.DefaultQuery("sort_by", "created_at")

	return domain.Pagination{
		Page:          page,
		Limit:         limit,
		Offset:        page * limit,
		SortBy:        sortBy,
		SortDirection: sortOrder,
	}
}

// ToDomainPagination converts handler Pagination to domain.Pagination.
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
	ProductId string `json:"product_id" binding:"required"`
	PriceId   string `json:"price_id" binding:"required"`
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
	ProductId string `json:"product_id" binding:"required"`
	PriceId   string `json:"price_id" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
}

type CreateOrderRequest struct {
	Customer        CreateOrderRequestCustomer `json:"customer" binding:"required"`
	PaymentMethodId string                     `json:"payment_method_id"`
	SessionId       string                     `json:"session_id"`
	PspId           string                     `json:"psp_id" binding:"required"`

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
	PaymentMethodId string `json:"payment_method_id" binding:"required"`

	Activate bool `json:"activate"`

	Amount   int    `json:"amount"  binding:"required"`
	Currency string `json:"currency"  binding:"required"`

	BillingInterval    domain.BillingInterval `json:"billing_interval"  binding:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  binding:"required"`
	Cycles             int                    `json:"cycles"`

	TrialInterval    domain.BillingInterval `json:"trial_interval"`
	TrialIntervalQty int                    `json:"trial_interval_qty"`

	Metadata map[string]string `json:"metadata"`
}

type ActivateSubscriptionRequest struct {
	PaymentMethodId string `json:"payment_method_id" binding:"required"`

	Amount   int    `json:"amount"  binding:"required"`
	Currency string `json:"currency"  binding:"required"`

	BillingInterval    domain.BillingInterval `json:"billing_interval"  binding:"required"`
	BillingIntervalQty int                    `json:"billing_interval_qty"  binding:"required"`
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
	BillingAnchor int                  `json:"billing_anchor" binding:"required,gte=1,lte=31"`
	ProrationMode domain.ProrationMode `json:"proration_mode" binding:"required,oneof=none prorate"`
}

type ResumeSubscriptionRequest struct {
	ResumeBehavior domain.SubscriptionResumeBehavior `json:"resume_behavior"`
}

// ---------------------------------------------------------------------------
// Customer requests
// ---------------------------------------------------------------------------

type CreateCustomerRequest struct {
	Email          string            `json:"email" binding:"required"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	BillingAddress domain.Address    `json:"billing_address"`
	Phone          string            `json:"phone"`
	Metadata       map[string]string `json:"metadata"`
}

type CreatePaymentMethodRequest struct {
	Psp  string `json:"psp" binding:"required"`
	Name string `json:"name" binding:"required"`

	// Type of payment method, e.g. card, bank account, etc.
	Type           domain.PaymentMethodType `json:"type" binding:"required"`
	Details        any                      `json:"details"`
	Token          string                   `json:"token" binding:"required"`
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
	Name        string                        `json:"name" binding:"required"`
	Description string                        `json:"description"`
	Metadata    map[string]string             `json:"metadata"`
	Variants    []CreateProductVariantRequest `json:"variants" binding:"required,dive"`
}

type UpdateProductRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateVariantRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type UpdateVariantRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type CreateProductVariantRequest struct {
	Name        string                      `json:"name" binding:"required"`
	Description string                      `json:"description"`
	Metadata    map[string]string           `json:"metadata"`
	Prices      []CreateProductPriceRequest `json:"prices" binding:"required,dive"`
}

type CreateProductPriceRequest struct {
	Label              string                 `json:"label" binding:"omitempty,min=1,max=255"`
	Category           domain.PriceCategory   `json:"category" binding:"required,oneof=one_time subscription free variable"`
	Scheme             domain.PriceScheme     `json:"scheme" binding:"required,oneof=fixed tiered volume graduated"`
	Cycles             int                    `json:"cycles" binding:"omitempty,gt=0"`
	Currency           string                 `json:"currency" binding:"required,iso4217"`
	UnitPrice          int64                  `json:"unit_price" binding:"required,gte=0"`
	MinPrice           int64                  `json:"min_price" binding:"omitempty,gte=0"`
	SuggestedPrice     int64                  `json:"suggested_price" binding:"omitempty,gte=0"`
	BillingInterval    domain.BillingInterval `json:"billing_interval" binding:"omitempty,oneof=none minute hour day week month year"`
	BillingIntervalQty int                    `json:"billing_interval_qty" binding:"omitempty,gt=0,lte=999"`
	TrialInterval      domain.BillingInterval `json:"trial_interval" binding:"omitempty,oneof=none minute hour day week month year"`
	TrialIntervalQty   int                    `json:"trial_interval_qty" binding:"omitempty,gt=0,lte=999"`
	TaxCode            string                 `json:"tax_code" binding:"omitempty,alphanum"`
	Metadata           map[string]string      `json:"metadata"`
}

type CreatePriceRequest struct {
	VariantId          string                 `json:"variant_id" binding:"required"`
	Category           domain.PriceCategory   `json:"category" binding:"required,oneof=one_time subscription free variable"`
	Scheme             domain.PriceScheme     `json:"scheme" binding:"required,oneof=fixed tiered volume graduated"`
	Cycles             int                    `json:"cycles" binding:"omitempty,gt=0"`
	Label              string                 `json:"label"`
	Currency           string                 `json:"currency" binding:"required,iso4217"`
	UnitPrice          int64                  `json:"unit_price" binding:"required,gte=0"`
	MinPrice           int64                  `json:"min_price" binding:"omitempty,gte=0"`
	SuggestedPrice     int64                  `json:"suggested_price" binding:"omitempty,gte=0"`
	BillingInterval    domain.BillingInterval `json:"billing_interval" binding:"omitempty,oneof=none minute hour day week month year"`
	BillingIntervalQty int                    `json:"billing_interval_qty" binding:"omitempty,gt=0,lte=999"`
	TrialInterval      domain.BillingInterval `json:"trial_interval" binding:"omitempty,oneof=none minute hour day week month year"`
	TrialIntervalQty   int                    `json:"trial_interval_qty" binding:"omitempty,gt=0,lte=999"`
	TaxCode            string                 `json:"tax_code" binding:"omitempty,alphanum"`
	Metadata           map[string]string      `json:"metadata"`
}

// ---------------------------------------------------------------------------
// Org requests
// ---------------------------------------------------------------------------

type CreateOrgInput struct {
	Name     string            `json:"name" binding:"required"`
	Country  string            `json:"country" binding:"required"`
	Timezone string            `json:"timezone" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}

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
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

type GetARRRequest struct {
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required"`
}

// ---------------------------------------------------------------------------
// Webhook subscription requests
// ---------------------------------------------------------------------------

type CreateWebhookSubscriptionRequest struct {
	Url    string   `json:"url" binding:"required"`
	Events []string `json:"events" binding:"required"`
	Secret string   `json:"secret"`
}
