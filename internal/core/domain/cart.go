package domain

import (
	"errors"
	"time"
)

var ErrItemNotFound = errors.New("item not found")

// CartData holds the computed cart totals and line items.
type CartData struct {
	Currency      string         `json:"currency"`
	Total         int64          `json:"total"`
	SubTotal      int64          `json:"sub_total"`
	DiscountTotal int64          `json:"discount"`
	ShippingTotal int64          `json:"shipping"`
	TaxTotal      int64          `json:"tax"`
	Items         []CartLineItem `json:"items"`
}

// CartLineItem represents a line item inside a cart.
type CartLineItem struct {
	Id            string        `json:"id"`
	ProductId     string        `json:"product_id"`
	Price         CartItemPrice `json:"price"`
	Description   string        `json:"description"`
	Quantity      int64         `json:"quantity"`
	UnitPrice     int64         `json:"unit_price"`
	TaxTotal      int64         `json:"tax_total"`
	SubTotal      int64         `json:"sub_total"`
	DiscountTotal int64         `json:"discount_total"`
	ShippingTotal int64         `json:"shipping_total"`
	Total         int64         `json:"total"`
}

// Calculate recalculates the line item totals.
func (i *CartLineItem) Calculate() {
	i.SubTotal = i.UnitPrice * i.Quantity
	i.Total = i.SubTotal
}

// CartSummary provides a summary view of the cart.
type CartSummary struct {
	Type      CartType `json:"type"`
	DueNow    int64    `json:"due_now"`
	DueFuture int64    `json:"due_future"`
}

// CartItemPrice represents the price snapshot attached to a cart line item.
type CartItemPrice struct {
	Id                 string          `json:"id"`
	Category           PriceCategory   `json:"category"`
	Scheme             PriceScheme     `json:"scheme"`
	Currency           string          `json:"currency"`
	Cycles             int64           `json:"cycles"`
	UnitPrice          int64           `json:"unit_price"`
	MinPrice           int64           `json:"min_price"`
	SuggestedPrice     int64           `json:"suggested_price"`
	BillingInterval    BillingInterval `json:"billing_interval"`
	BillingIntervalQty int64           `json:"billing_interval_qty"`
	TrialInterval      BillingInterval `json:"trial_interval"`
	TrialIntervalQty   int64           `json:"trial_interval_qty"`
	TaxCode            string          `json:"tax_code"`
}

// CartType distinguishes one-time vs subscription carts.
type CartType string

const (
	CartTypeOnceOff      CartType = "once_off"
	CartTypeSubscription CartType = "subscription"
)

// AddItemInput is the command object for adding a product to a cart.
type AddItemInput struct {
	ProductId string
	PriceId   string
	Quantity  int
}

// PriceToCartItemPrice converts a domain Price entity into a CartItemPrice snapshot.
func PriceToCartItemPrice(p Price) CartItemPrice {
	return CartItemPrice{
		Id:                 p.Id,
		Category:           p.Category,
		Scheme:             p.Scheme,
		Currency:           string(p.Currency),
		Cycles:             int64(p.Cycles),
		UnitPrice:          p.UnitPrice,
		BillingInterval:    p.BillingInterval,
		BillingIntervalQty: int64(p.BillingIntervalQty),
		TrialInterval:      p.TrialInterval,
		TrialIntervalQty:   int64(p.TrialIntervalQty),
		TaxCode:            p.TaxCode,
	}
}

// Cart is the persisted cart entity.
type Cart struct {
	OrgId     string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id        string            `gorm:"column:id;primaryKey" json:"id"`
	Data      CartData          `gorm:"column:data;serializer:json" json:"data"`
	Status    string            `gorm:"-" json:"status"`
	Total     int64             `gorm:"-" json:"total"`
	Metadata  map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Cart) TableName() string { return "carts" }

// Calculate recalculates the cart totals from its items.
func (c *Cart) Calculate() {
	var total int64
	var subTotal int64
	var discountTotal int64
	var taxTotal int64

	for idx := range c.Data.Items {
		c.Data.Items[idx].Calculate()
		discountTotal += c.Data.Items[idx].DiscountTotal
		taxTotal += c.Data.Items[idx].TaxTotal
		subTotal += c.Data.Items[idx].SubTotal
		total = subTotal
	}

	c.Data.Total = total
	c.Data.SubTotal = subTotal
	c.Data.TaxTotal = taxTotal
	c.Data.DiscountTotal = discountTotal
	c.Total = total
}

// RemoveItem removes an item by its Id and recalculates the cart.
func (c *Cart) RemoveItem(id string) {
	for i, item := range c.Data.Items {
		if item.Id == id {
			c.Data.Items = append(c.Data.Items[:i], c.Data.Items[i+1:]...)
			break
		}
	}
	c.Calculate()
}

// AdjustQuantity changes the quantity of an item by its Id and recalculates.
func (c *Cart) AdjustQuantity(id string, quantity int64) error {
	for i, item := range c.Data.Items {
		if item.Id == id {
			c.Data.Items[i].Quantity = quantity
			c.Calculate()
			return nil
		}
	}
	return ErrItemNotFound
}

type CartStatus string

const (
	CartStatusPending   CartStatus = "pending"
	CartStatusCompleted CartStatus = "completed"
	CartStatusExpired   CartStatus = "expired"
)

type AddProductCommand struct {
	OrgId     string `json:"org_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}

type RemoveItemCommand struct {
	OrgId  string `json:"org_id"`
	CartId string `json:"cart_id"`
	Id     string `json:"id"`
}

type AdjustCommand struct {
	OrgId     string `json:"org_id"`
	CartId    string `json:"cart_id"`
	ProductId string `json:"product_id"`
	PriceId   string `json:"price_id"`
	Quantity  int    `json:"quantity"`
}

type CreateCartInput struct {
	OrgId    string            `json:"org_id"`
	Cart     Cart              `json:"carts"`
	Metadata map[string]string `json:"metadata"`
}
