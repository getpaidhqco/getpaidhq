package response

import (
	"payloop/internal/domain/entities"
	"time"
)

// Invoice represents the response for an invoice
type Invoice struct {
	Id             string                 `json:"id"`
	CustomerId     string                 `json:"customer_id,omitempty"`
	OrderId        string                 `json:"order_id,omitempty"`
	SubscriptionId string                 `json:"subscription_id,omitempty"`
	SequenceId     string                 `json:"sequence_id"`
	DocNumber      string                 `json:"doc_number"`
	Type           entities.DocumentType  `json:"type"`
	InvoiceType    entities.InvoiceType   `json:"invoice_type"`
	Status         entities.InvoiceStatus `json:"status"`
	IsImmutable    bool                   `json:"is_immutable"`
	Currency       string                 `json:"currency"`
	SubTotal       int                    `json:"sub_total"`
	TaxTotal       int                    `json:"tax_total"`
	DiscountTotal  int                    `json:"discount_total"`
	Total          int                    `json:"total"`
	AmountPaid     int                    `json:"amount_paid"`
	AmountDue      int                    `json:"amount_due"`
	TaxProvider    string                 `json:"tax_provider,omitempty"`
	IssuedAt       time.Time              `json:"issued_at,omitempty"`
	DueAt          time.Time              `json:"due_at,omitempty"`
	PaidAt         time.Time              `json:"paid_at,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
	CustomerNotes  string                 `json:"customer_notes,omitempty"`
	Metadata       map[string]string      `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	LineItems      []InvoiceLineItem      `json:"line_items,omitempty"`
	Payments       []Payment              `json:"payments,omitempty"`
}

// InvoiceLineItem represents the response for an invoice line item
type InvoiceLineItem struct {
	Id            string            `json:"id"`
	ProductId     string            `json:"product_id,omitempty"`
	VariantId     string            `json:"variant_id,omitempty"`
	PriceId       string            `json:"price_id,omitempty"`
	Description   string            `json:"description"`
	Category      string            `json:"category,omitempty"`
	Quantity      float64           `json:"quantity"`
	UnitPrice     int               `json:"unit_price"`
	LineTotal     int               `json:"line_total"`
	DiscountType  string            `json:"discount_type,omitempty"`
	DiscountValue int               `json:"discount_value,omitempty"`
	DiscountTotal int               `json:"discount_total"`
	TaxCode       string            `json:"tax_code,omitempty"`
	TaxRate       int               `json:"tax_rate,omitempty"`
	TaxAmount     int               `json:"tax_amount,omitempty"`
	TaxExempt     bool              `json:"tax_exempt"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// InvoiceHistory represents the response for an invoice history entry
type InvoiceHistory struct {
	Id        string                        `json:"id"`
	Action    entities.InvoiceHistoryAction `json:"action"`
	Field     string                        `json:"field,omitempty"`
	OldValue  interface{}                   `json:"old_value,omitempty"`
	NewValue  interface{}                   `json:"new_value,omitempty"`
	UserEmail string                        `json:"user_email,omitempty"`
	Reason    string                        `json:"reason,omitempty"`
	Timestamp time.Time                     `json:"timestamp"`
}

// NewInvoiceFromEntity creates a new Invoice response from an entity
func NewInvoiceFromEntity(entity entities.Invoice) Invoice {

	lineItems := make([]InvoiceLineItem, len(entity.LineItems))
	for i, item := range entity.LineItems {
		lineItems[i] = NewInvoiceLineItemFromEntity(item)
	}

	return Invoice{
		Id:             entity.Id,
		CustomerId:     entity.CustomerId,
		OrderId:        entity.OrderId,
		SubscriptionId: entity.SubscriptionId,
		SequenceId:     entity.SequenceId,
		DocNumber:      entity.DocNumber,
		Type:           entity.Type,
		InvoiceType:    entity.InvoiceType,
		Status:         entity.Status,
		IsImmutable:    entity.IsImmutable,
		Currency:       entity.Currency,
		SubTotal:       entity.SubTotal,
		TaxTotal:       entity.TaxTotal,
		DiscountTotal:  entity.DiscountTotal,
		Total:          entity.Total,
		AmountPaid:     entity.AmountPaid,
		AmountDue:      entity.AmountDue,
		TaxProvider:    entity.TaxProvider,
		IssuedAt:       entity.IssuedAt,
		DueAt:          entity.DueAt,
		PaidAt:         entity.PaidAt,
		Notes:          entity.Notes,
		CustomerNotes:  entity.CustomerNotes,
		Metadata:       entity.Metadata,
		CreatedAt:      entity.CreatedAt,
		UpdatedAt:      entity.UpdatedAt,
		LineItems:      lineItems,
	}
}

// NewInvoiceLineItemFromEntity creates a new InvoiceLineItem response from an entity
func NewInvoiceLineItemFromEntity(entity entities.InvoiceLineItem) InvoiceLineItem {
	return InvoiceLineItem{
		Id:            entity.Id,
		ProductId:     entity.ProductId,
		VariantId:     entity.VariantId,
		PriceId:       entity.PriceId,
		Description:   entity.Description,
		Category:      entity.Category,
		Quantity:      entity.Quantity,
		UnitPrice:     entity.UnitPrice,
		LineTotal:     entity.LineTotal,
		DiscountType:  entity.DiscountType,
		DiscountValue: entity.DiscountValue,
		DiscountTotal: entity.DiscountTotal,
		TaxCode:       entity.TaxCode,
		TaxRate:       entity.TaxRate,
		TaxAmount:     entity.TaxAmount,
		TaxExempt:     entity.TaxExempt,
		Metadata:      entity.Metadata,
	}
}

// NewInvoiceHistoryFromEntity creates a new InvoiceHistory response from an entity
func NewInvoiceHistoryFromEntity(entity entities.InvoiceHistory) InvoiceHistory {
	return InvoiceHistory{
		Id:        entity.Id,
		Action:    entity.Action,
		Field:     entity.Field,
		OldValue:  entity.OldValue,
		NewValue:  entity.NewValue,
		UserEmail: entity.UserEmail,
		Reason:    entity.Reason,
		Timestamp: entity.Timestamp,
	}
}
