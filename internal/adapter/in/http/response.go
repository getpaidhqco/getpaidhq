package handler

import (
	"encoding/json"
	"time"

	"payloop/internal/core/domain"
	"payloop/internal/infrastructure/cart"
)

// ---------------------------------------------------------------------------
// List / Meta
// ---------------------------------------------------------------------------

// ListResponse is a generic response for list endpoints.
// swagger:response listResponse
type ListResponse struct {
	Data interface{} `json:"data"`
	Meta Meta        `json:"meta"`
}

type Meta struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// ---------------------------------------------------------------------------
// Order
// ---------------------------------------------------------------------------

type OrderResponse struct {
	Id         string            `json:"id"`
	CustomerId string            `json:"customer_id"`
	Customer   CustomerResponse  `json:"customer"`
	Reference  string            `json:"reference"`
	Status     string            `json:"status"`
	SessionId  string            `json:"session_id"`
	CartId     string            `json:"cart_id"`
	Items      []OrderItemResponse `json:"items"`
	Currency   string            `json:"currency"`
	Total      int64             `json:"total"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  time.Time         `json:"created_at"`
}

func NewOrderFromEntity(entity domain.Order) OrderResponse {
	var items []OrderItemResponse
	for _, item := range entity.Items {
		items = append(items, NewOrderItemFromEntity(item))
	}

	return OrderResponse{
		Id:         entity.Id,
		CustomerId: entity.CustomerId,
		Customer:   NewCustomerFromEntity(entity.Customer),
		Reference:  entity.Reference,
		Items:      items,
		Status:     string(entity.Status),
		SessionId:  entity.SessionId,
		CartId:     entity.CartId,
		Currency:   entity.Currency,
		Total:      entity.Total,
		Metadata:   entity.Metadata,
		CreatedAt:  entity.CreatedAt,
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

func NewOrderItemFromEntity(entity domain.OrderItem) OrderItemResponse {
	return OrderItemResponse{
		Id:            entity.Id,
		OrderId:       entity.OrderId,
		PriceId:       entity.PriceId,
		ProductId:     entity.ProductId,
		VariantId:     entity.VariantId,
		Price:         NewPriceFromEntity(entity.Price),
		Description:   entity.Description,
		Quantity:      entity.Quantity,
		TaxTotal:      entity.TaxTotal,
		DiscountTotal: entity.DiscountTotal,
		Subtotal:      entity.Subtotal,
		Total:         entity.Total,
		Metadata:      entity.Metadata,
		CreatedAt:     entity.CreatedAt,
		UpdatedAt:     entity.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// Subscription
// ---------------------------------------------------------------------------

type SubscriptionResponse struct {
	Id          string                    `json:"id"`
	Status      domain.SubscriptionStatus `json:"status"`
	Currency    string                    `json:"currency"`
	Amount      int64                     `json:"amount"`
	OrderId     string                    `json:"order_id"`
	OrderItemId string                    `json:"order_item_id"`

	PaymentMethodId string `json:"payment_method_id,omitempty"`

	StartDate          time.Time              `json:"start_date,omitempty"`
	EndDate            time.Time              `json:"end_date,omitempty,omitzero"`
	BillingInterval    domain.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	Cycles             int                    `json:"cycles"`
	BillingAnchor      int                    `json:"billing_anchor"`
	TrialEndsAt        time.Time              `json:"trial_ends_at,omitempty,omitzero"`
	CancelAt           time.Time              `json:"cancel_at,omitempty,omitzero"`
	EndsAt             time.Time              `json:"ends_at,omitempty,omitzero"`
	LastCharge         time.Time              `json:"last_charge,omitempty,omitzero"`
	RenewsAt           time.Time              `json:"renews_at,omitempty,omitzero"`

	CurrentPeriodStart time.Time `json:"current_period_start,omitempty,omitzero"`
	CurrentPeriodEnd   time.Time `json:"current_period_end,omitempty,omitzero"`

	Retries     int       `json:"retries"`
	NextRetryAt time.Time `json:"next_retry,omitempty,omitzero"`

	Customer CustomerResponse `json:"customer"`

	Metadata        map[string]string `json:"metadata"`
	CyclesProcessed int               `json:"cycles_processed"`
	TotalRevenue    int64             `json:"total_revenue"`
	CancelledAt     time.Time         `json:"cancelled_at,omitempty,omitzero"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

func NewSubscriptionFromEntity(entity domain.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		Id:                 entity.Id,
		OrderId:            entity.OrderId,
		OrderItemId:        entity.OrderItemId,
		Customer:           NewCustomerFromEntity(entity.Customer),
		Status:             entity.Status,
		PaymentMethodId:    entity.PaymentMethodId,
		StartDate:          entity.StartDate,
		EndDate:            entity.EndDate,
		BillingInterval:    entity.BillingInterval,
		BillingIntervalQty: entity.BillingIntervalQty,
		Cycles:             entity.Cycles,
		BillingAnchor:      entity.BillingAnchor,
		TrialEndsAt:        entity.TrialEndsAt,
		CancelAt:           entity.CancelAt,
		EndsAt:             entity.EndsAt,
		LastCharge:         entity.LastCharge,
		RenewsAt:           entity.RenewsAt,
		CurrentPeriodStart: entity.CurrentPeriodStart,
		CurrentPeriodEnd:   entity.CurrentPeriodEnd,
		Retries:            entity.Retries,
		NextRetryAt:        entity.NextRetryAt,
		Currency:           entity.Currency,
		Amount:             entity.Amount,
		Metadata:           entity.Metadata,
		CyclesProcessed:    entity.CyclesProcessed,
		TotalRevenue:       entity.TotalRevenue,
		CancelledAt:        entity.CancelledAt,
		CreatedAt:          entity.CreatedAt,
		UpdatedAt:          entity.UpdatedAt,
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
	BillingAddress domain.Address    `json:"billing_address,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
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
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Variants    []VariantResponse `json:"variants,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func NewProductFromEntity(entity domain.Product) ProductResponse {
	var variants []VariantResponse
	for _, variant := range entity.Variants {
		variants = append(variants, NewVariantFromEntity(variant))
	}
	return ProductResponse{
		Id:          entity.Id,
		Name:        entity.Name,
		Description: entity.Description,
		Variants:    variants,
		Metadata:    entity.Metadata,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
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

func NewVariantFromEntity(entity domain.Variant) VariantResponse {
	var prices []PriceResponse
	for _, price := range entity.Prices {
		prices = append(prices, NewPriceFromEntity(price))
	}
	return VariantResponse{
		Id:        entity.Id,
		Name:      entity.Name,
		Prices:    prices,
		CreatedAt: entity.CreatedAt,
		UpdatedAt: entity.UpdatedAt,
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
	MinPrice           int64                  `json:"min_price"`
	SuggestedPrice     int64                  `json:"suggested_price"`
	BillingInterval    domain.BillingInterval `json:"billing_interval"`
	BillingIntervalQty int                    `json:"billing_interval_qty"`
	TrialInterval      domain.BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int                    `json:"trial_interval_qty"`
	TaxCode            string                 `json:"tax_code"`
	Metadata           map[string]string      `json:"metadata"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

func NewPriceFromEntity(entity domain.Price) PriceResponse {
	return PriceResponse{
		Id:                 entity.Id,
		VariantId:          entity.VariantId,
		Category:           entity.Category,
		Scheme:             entity.Scheme,
		Label:              entity.Label,
		Cycles:             entity.Cycles,
		Currency:           entity.Currency,
		UnitPrice:          entity.UnitPrice,
		MinPrice:           entity.MinPrice,
		SuggestedPrice:     entity.SuggestedPrice,
		BillingInterval:    entity.BillingInterval,
		BillingIntervalQty: entity.BillingIntervalQty,
		TrialInterval:      entity.TrialInterval,
		TrialIntervalQty:   entity.TrialIntervalQty,
		TaxCode:            entity.TaxCode,
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
	NewPeriodStart     time.Time `json:"new_period_start,omitempty"`
	NewPeriodEnd       time.Time `json:"new_period_end,omitempty"`
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
	cart.CartData
}

func ToCartResponse(entity domain.Cart) CartResponse {
	data, err := json.Marshal(entity.Data)
	if err != nil {
		// handle error
	}
	var cartData cart.CartData
	if err := json.Unmarshal(data, &cartData); err != nil {
		// handle error
	}

	return CartResponse{
		CartData: entity.Data.(cart.CartData),
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
