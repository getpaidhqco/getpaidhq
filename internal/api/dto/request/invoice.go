package request

import (
	"payloop/internal/domain/entities"
	"time"
)

// CreateInvoiceRequest represents the request to create a new invoice
type CreateInvoiceRequest struct {
	CustomerId     string                         `json:"customer_id" binding:"required"`
	OrderId        string                         `json:"order_id,omitempty"`
	SubscriptionId string                         `json:"subscription_id,omitempty"`
	Type           entities.DocumentType          `json:"type" binding:"required"`
	InvoiceType    entities.InvoiceType           `json:"invoice_type" binding:"required"`
	Currency       string                         `json:"currency" binding:"required"`
	DueAt          time.Time                      `json:"due_at,omitempty"`
	Notes          string                         `json:"notes,omitempty"`
	CustomerNotes  string                         `json:"customer_notes,omitempty"`
	Metadata       map[string]string              `json:"metadata,omitempty"`
	LineItems      []CreateInvoiceLineItemRequest `json:"line_items,omitempty"`
}

// UpdateInvoiceRequest represents the request to update an existing invoice
type UpdateInvoiceRequest struct {
	Notes         string            `json:"notes,omitempty"`
	CustomerNotes string            `json:"customer_notes,omitempty"`
	DueAt         time.Time         `json:"due_at,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// CreateInvoiceLineItemRequest represents the request to create a new invoice line item
type CreateInvoiceLineItemRequest struct {
	ProductId     string            `json:"product_id,omitempty"`
	VariantId     string            `json:"variant_id,omitempty"`
	PriceId       string            `json:"price_id,omitempty"`
	Description   string            `json:"description" binding:"required"`
	Category      string            `json:"category,omitempty"`
	Quantity      float64           `json:"quantity" binding:"required"`
	UnitPrice     int               `json:"unit_price" binding:"required"`
	DiscountType  string            `json:"discount_type,omitempty"`
	DiscountValue int               `json:"discount_value,omitempty"`
	TaxCode       string            `json:"tax_code,omitempty"`
	TaxRate       int               `json:"tax_rate,omitempty"`
	TaxExempt     bool              `json:"tax_exempt,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// UpdateInvoiceLineItemRequest represents the request to update an existing invoice line item
type UpdateInvoiceLineItemRequest struct {
	Description   string            `json:"description,omitempty"`
	Category      string            `json:"category,omitempty"`
	Quantity      float64           `json:"quantity,omitempty"`
	UnitPrice     int               `json:"unit_price,omitempty"`
	DiscountType  string            `json:"discount_type,omitempty"`
	DiscountValue int               `json:"discount_value,omitempty"`
	TaxCode       string            `json:"tax_code,omitempty"`
	TaxRate       int               `json:"tax_rate,omitempty"`
	TaxExempt     bool              `json:"tax_exempt,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// InvoiceActionRequest represents the request to perform an action on an invoice
type InvoiceActionRequest struct {
	Action string `json:"action" binding:"required"`
	Reason string `json:"reason,omitempty"`
}

// GenerateInvoicePDFRequest represents the request to generate a PDF for an invoice
type GenerateInvoicePDFRequest struct {
	TemplateName string `json:"template_name" binding:"required"`
	OutputPath   string `json:"output_path,omitempty"`
}

// CreateInvoicePaymentLinkRequest represents the request to create a payment link for an invoice
type CreateInvoicePaymentLinkRequest struct {
	ExpiresAt  string                 `json:"expires_at,omitempty"`
	SuccessUrl string                 `json:"success_url,omitempty"`
	CancelUrl  string                 `json:"cancel_url,omitempty"`
	Config     map[string]interface{} `json:"config,omitempty"`
}
