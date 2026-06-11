package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/service"
)

// ---------------------------------------------------------------------------
// List / Meta
// ---------------------------------------------------------------------------

// ListResponse is a generic response for list endpoints.
// swagger:response listResponse
type ListResponse struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

type Meta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// EmptyResponse is the response body for endpoints that return 204 No Content
// (e.g. DELETEs). It exists because Fuego reflects on the typed return to
// register an OpenAPI schema component; an anonymous `struct{}` has no
// reflect.Type.Name() and gets registered with the empty string, which the
// spec validator then rejects with `identifier "" is not supported`. A named
// (even if zero-field) type gives the component a stable identifier.
type EmptyResponse struct{}

// ---------------------------------------------------------------------------
// Order
// ---------------------------------------------------------------------------

type OrderResponse struct {
	Id         string              `json:"id"`
	CustomerId string              `json:"customer_id"`
	Customer   CustomerResponse    `json:"customer"`
	Reference  string              `json:"reference"`
	Status     string              `json:"status"`
	SessionId  string              `json:"session_id"`
	CartId     string              `json:"cart_id"`
	Items      []OrderItemResponse `json:"items"`
	Currency   string              `json:"currency"`
	Total      int64               `json:"total"`
	Metadata   map[string]string   `json:"metadata"`
	CreatedAt  time.Time           `json:"created_at"`
}

// NewOrderResponseFromDetails composes the order response DTO from the
// service-layer read model. Items render their nested Price via
// NewOrderItemResponseFromDetails.
func NewOrderResponseFromDetails(d service.OrderDetails) OrderResponse {
	items := make([]OrderItemResponse, len(d.Items))
	for i, it := range d.Items {
		items[i] = NewOrderItemResponseFromDetails(it)
	}
	return OrderResponse{
		Id:         d.Order.Id,
		CustomerId: d.Order.CustomerId,
		Customer:   NewCustomerFromEntity(d.Customer),
		Reference:  d.Order.Reference,
		Items:      items,
		Status:     string(d.Order.Status),
		SessionId:  d.Order.SessionId,
		CartId:     d.Order.CartId,
		Currency:   d.Order.Currency,
		Total:      d.Order.Total,
		Metadata:   d.Order.Metadata,
		CreatedAt:  d.Order.CreatedAt,
	}
}

// ---------------------------------------------------------------------------
// OrderItem
// ---------------------------------------------------------------------------

type OrderItemResponse struct {
	Id            string            `json:"id"`
	OrderId       string            `json:"order_id"`
	ProductId     string            `json:"product_id"`
	VariantId     string            `json:"variant_id"`
	PriceId       string            `json:"price_id"`
	Price         PriceResponse     `json:"price"`
	Description   string            `json:"description"`
	Quantity      int               `json:"quantity"`
	TaxTotal      int64             `json:"tax_total"`
	DiscountTotal int64             `json:"discount_total"`
	Subtotal      int64             `json:"sub_total"`
	Total         int64             `json:"total"`
	Metadata      map[string]string `json:"metadata"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// NewOrderItemResponseFromDetails composes the order-item response DTO from
// the service-layer read model (Item + Price pair).
func NewOrderItemResponseFromDetails(d service.OrderItemDetails) OrderItemResponse {
	return OrderItemResponse{
		Id:            d.Item.Id,
		OrderId:       d.Item.OrderId,
		PriceId:       d.Item.PriceId,
		ProductId:     d.Item.ProductId,
		VariantId:     d.Item.VariantId,
		Price:         NewPriceFromEntity(d.Price),
		Description:   d.Item.Description,
		Quantity:      d.Item.Quantity,
		TaxTotal:      d.Item.TaxTotal,
		DiscountTotal: d.Item.DiscountTotal,
		Subtotal:      d.Item.Subtotal,
		Total:         d.Item.Total,
		Metadata:      d.Item.Metadata,
		CreatedAt:     d.Item.CreatedAt,
		UpdatedAt:     d.Item.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Subscription
// ---------------------------------------------------------------------------

type SubscriptionResponse struct {
	Id       string                    `json:"id"`
	Status   domain.SubscriptionStatus `json:"status"`
	Currency string                    `json:"currency"`
	OrderId  string                    `json:"order_id"`

	PaymentMethodId string `json:"payment_method_id,omitempty"`

	StartDate          time.Time              `json:"start_date"`
	EndDate            time.Time              `json:"end_date,omitzero"`
	BillingInterval    domain.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	Cycles             int                    `json:"cycles"`
	BillingAnchor      int                    `json:"billing_anchor"`
	TrialEndsAt        time.Time              `json:"trial_ends_at,omitzero"`
	CancelAt           time.Time              `json:"cancel_at,omitzero"`
	EndsAt             time.Time              `json:"ends_at,omitzero"`
	LastCharge         time.Time              `json:"last_charge,omitzero"`
	RenewsAt           time.Time              `json:"renews_at,omitzero"`

	CurrentPeriodStart time.Time `json:"current_period_start,omitzero"`
	CurrentPeriodEnd   time.Time `json:"current_period_end,omitzero"`

	Retries     int       `json:"retries"`
	NextRetryAt time.Time `json:"next_retry,omitzero"`

	Customer CustomerResponse `json:"customer"`

	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int64             `json:"total_revenue"`
	CancelledAt     time.Time         `json:"cancelled_at,omitzero"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// NewSubscriptionResponseFromDetails composes the subscription response DTO
// from the service-layer read model (Subscription + Customer pair).
func NewSubscriptionResponseFromDetails(d service.SubscriptionDetails) SubscriptionResponse {
	s := d.Subscription
	return SubscriptionResponse{
		Id:                 s.Id,
		OrderId:            s.OrderId,
		Customer:           NewCustomerFromEntity(d.Customer),
		Status:             s.Status,
		PaymentMethodId:    s.PaymentMethodId,
		StartDate:          s.StartDate,
		EndDate:            s.EndDate,
		BillingInterval:    s.BillingInterval,
		BillingIntervalQty: s.BillingIntervalQty,
		Cycles:             s.Cycles,
		BillingAnchor:      s.BillingAnchor,
		TrialEndsAt:        s.TrialEndsAt,
		CancelAt:           s.CancelAt,
		EndsAt:             s.EndsAt,
		LastCharge:         s.LastCharge,
		RenewsAt:           s.RenewsAt,
		CurrentPeriodStart: s.CurrentPeriodStart,
		CurrentPeriodEnd:   s.CurrentPeriodEnd,
		Retries:            s.Retries,
		NextRetryAt:        s.NextRetryAt,
		Currency:           s.Currency,
		Metadata:           s.Metadata,
		CyclesProcessed:    s.CyclesProcessed,
		TotalRevenue:       s.TotalRevenue,
		CancelledAt:        s.CancelledAt,
		CreatedAt:          s.CreatedAt,
		UpdatedAt:          s.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Customer
// ---------------------------------------------------------------------------

type CustomerResponse struct {
	Id             string            `json:"id"`
	Name           string            `json:"name,omitempty"`
	Email          string            `json:"email"`
	FirstName      string            `json:"first_name,omitempty"`
	LastName       string            `json:"last_name,omitempty"`
	Phone          string            `json:"phone,omitempty"`
	BillingAddress domain.Address    `json:"billing_address"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

func NewCustomerFromEntity(entity domain.Customer) CustomerResponse {
	return CustomerResponse{
		Id:             entity.Id,
		Email:          entity.Email,
		FirstName:      entity.FirstName,
		LastName:       entity.LastName,
		BillingAddress: entity.BillingAddress,
		Metadata:       entity.Metadata,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Product
// ---------------------------------------------------------------------------

type ProductResponse struct {
	Id          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Status      domain.ProductStatus `json:"status"`
	ArchivedAt  *time.Time           `json:"archived_at,omitempty"`
	Variants    []VariantResponse    `json:"variants,omitempty"`
	Metadata    map[string]string    `json:"metadata,omitempty"`
	CreatedAt   time.Time            `json:"created_at"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

// NewProductResponseFromDetails composes the product response DTO from the
// service-layer read model (product + variants + nested prices).
func NewProductResponseFromDetails(d service.ProductDetails) ProductResponse {
	variants := make([]VariantResponse, len(d.Variants))
	for i, v := range d.Variants {
		variants[i] = NewVariantResponseFromDetails(v)
	}
	return ProductResponse{
		Id:          d.Product.Id,
		Name:        d.Product.Name,
		Description: d.Product.Description,
		Status:      d.Product.Status,
		ArchivedAt:  d.Product.ArchivedAt,
		Variants:    variants,
		Metadata:    d.Product.Metadata,
		CreatedAt:   d.Product.CreatedAt,
		UpdatedAt:   d.Product.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Variant
// ---------------------------------------------------------------------------

type VariantResponse struct {
	Id        string          `json:"id"`
	Name      string          `json:"name"`
	Prices    []PriceResponse `json:"prices,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// NewVariantResponseFromDetails composes the variant response DTO from the
// service-layer read model (Variant + Prices).
func NewVariantResponseFromDetails(d service.VariantDetails) VariantResponse {
	prices := make([]PriceResponse, len(d.Prices))
	for i, p := range d.Prices {
		prices[i] = NewPriceFromEntity(p)
	}
	return VariantResponse{
		Id:        d.Variant.Id,
		Name:      d.Variant.Name,
		Prices:    prices,
		CreatedAt: d.Variant.CreatedAt,
		UpdatedAt: d.Variant.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Price
// ---------------------------------------------------------------------------

type PriceResponse struct {
	Id                 string                 `json:"id"`
	VariantId          string                 `json:"variant_id"`
	Label              string                 `json:"label"`
	Category           domain.PriceCategory   `json:"category"`
	Scheme             domain.PriceScheme     `json:"scheme"`
	Cycles             int                    `json:"cycles"`
	Currency           domain.Currency        `json:"currency"`
	UnitPrice          int64                  `json:"unit_price"`
	UnitCount          int                    `json:"unit_count"` // units unit_price buys; 1 = per single unit
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    domain.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      domain.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`
	BillableMetricId   string                 `json:"billable_metric_id"`
	Tiers              []PriceTierResponse    `json:"tiers"`
	FilterField        string                 `json:"filter_field"`
	FilterValue        string                 `json:"filter_value"`
	ProrateOnIncrease  bool                   `json:"prorate_on_increase"`
	CreditOnDecrease   bool                   `json:"credit_on_decrease"`
	Metadata           map[string]string      `json:"metadata"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// PriceTierResponse mirrors a domain.PriceTier; decimal fields are strings.
type PriceTierResponse struct {
	FromValue     string `json:"from_value"`
	ToValue       string `json:"to_value"`
	PerUnitAmount string `json:"per_unit_amount"`
	FlatAmount    int64  `json:"flat_amount"`
}

func NewPriceFromEntity(entity domain.Price) PriceResponse {
	tiers := make([]PriceTierResponse, len(entity.Tiers))
	for i, t := range entity.Tiers {
		tiers[i] = PriceTierResponse{
			FromValue:     t.FromValue.String(),
			ToValue:       t.ToValue.String(),
			PerUnitAmount: t.PerUnitAmount.String(),
			FlatAmount:    t.FlatAmount,
		}
	}
	return PriceResponse{
		Id:                 entity.Id,
		VariantId:          entity.VariantId,
		Category:           entity.Category,
		Scheme:             entity.Scheme,
		Label:              entity.Label,
		Cycles:             entity.Cycles,
		Currency:           entity.Currency,
		UnitPrice:          entity.UnitPrice,
		UnitCount:          entity.UnitCount,
		MinPrice:           entity.MinPrice,
		SuggestedPrice:     entity.SuggestedPrice,
		BillingInterval:    entity.BillingInterval,
		BillingIntervalQty: entity.BillingIntervalQty,
		TrialInterval:      entity.TrialInterval,
		TrialIntervalQty:   entity.TrialIntervalQty,
		TaxCode:            entity.TaxCode,
		BillableMetricId:   entity.BillableMetricId,
		Tiers:              tiers,
		FilterField:        entity.FilterField,
		FilterValue:        entity.FilterValue,
		ProrateOnIncrease:  entity.ProrateOnIncrease,
		CreditOnDecrease:   entity.CreditOnDecrease,
		Metadata:           entity.Metadata,
		CreatedAt:          entity.CreatedAt,
		UpdatedAt:          entity.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Payment
// ---------------------------------------------------------------------------

type PaymentResponse struct {
	Id             string               `json:"id"`
	PspId          string               `json:"psp_id"`
	Reference      string               `json:"reference"`
	OrderId        string               `json:"order_id"`
	SubscriptionId string               `json:"subscription_id"`
	InvoiceId      string               `json:"invoice_id"`
	Status         domain.PaymentStatus `json:"status"`
	Currency       string               `json:"currency"`
	Amount         int64                `json:"amount"`
	PspFee         int64                `json:"psp_fee"`
	PlatformFee    int64                `json:"platform_fee"`
	NetAmount      int64                `json:"net_amount"`
	Metadata       map[string]string    `json:"metadata"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

func NewPaymentFromEntity(entity domain.Payment) PaymentResponse {
	return PaymentResponse{
		Id:             entity.Id,
		PspId:          entity.PspId,
		Reference:      entity.Reference,
		OrderId:        entity.OrderId,
		SubscriptionId: entity.SubscriptionId,
		InvoiceId:      entity.InvoiceId,
		Status:         entity.Status,
		Currency:       entity.Currency,
		Amount:         entity.Amount,
		PspFee:         entity.PspFee,
		PlatformFee:    entity.PlatformFee,
		NetAmount:      entity.NetAmount,
		Metadata:       entity.Metadata,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// ProrationDetails
// ---------------------------------------------------------------------------

type ProrationDetailsResponse struct {
	CreditAmount       int       `json:"credit_amount"`
	DaysCredited       int       `json:"days_credited"`
	CurrentPeriodStart time.Time `json:"current_period_start"`
	CurrentPeriodEnd   time.Time `json:"current_period_end"`
	OldBillingAnchor   int       `json:"old_billing_anchor,omitempty"`
	NewBillingAnchor   int       `json:"new_billing_anchor,omitempty"`
	NewPeriodStart     time.Time `json:"new_period_start"`
	NewPeriodEnd       time.Time `json:"new_period_end"`
}

func NewProrationDetailsFromEntity(details domain.ProrationDetails) ProrationDetailsResponse {
	return ProrationDetailsResponse{
		CreditAmount:       details.CreditAmount,
		DaysCredited:       details.DaysCredited,
		CurrentPeriodStart: details.CurrentPeriodStart,
		CurrentPeriodEnd:   details.CurrentPeriodEnd,
		OldBillingAnchor:   details.OldBillingAnchor,
		NewBillingAnchor:   details.NewBillingAnchor,
		NewPeriodStart:     details.NewPeriodStart,
		NewPeriodEnd:       details.NewPeriodEnd,
	}
}

// ---------------------------------------------------------------------------
// Gateway / PSP
// ---------------------------------------------------------------------------

type GatewayResponse struct {
	Id        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewGatewayFromEntity(entity domain.PspConfig) GatewayResponse {
	return GatewayResponse{
		Id:        entity.Id,
		Name:      entity.Name,
		UpdatedAt: entity.UpdatedAt,
		CreatedAt: entity.CreatedAt,
	}
}

// ---------------------------------------------------------------------------
// Cart
// ---------------------------------------------------------------------------

type CartResponse struct {
	Data any `json:"data"`
}

func ToCartResponse(entity domain.Cart) CartResponse {
	return CartResponse{
		Data: entity.Data,
	}
}

// ---------------------------------------------------------------------------
// Cart item mapping (orders)
// ---------------------------------------------------------------------------

func ToCartItems(cartItems []CartItem) []domain.CartItem {
	var items []domain.CartItem
	for _, item := range cartItems {
		items = append(items, domain.CartItem{
			ProductId: item.ProductId,
			PriceId:   item.PriceId,
			Quantity:  item.Quantity,
		})
	}
	return items
}
