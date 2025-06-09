package entities

import (
	"time"
)

type InvoiceLineItem struct {
	OrgId         string            `json:"org_id"`
	InvoiceId     string            `json:"invoice_id"`
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
	Seq           int               `json:"seq,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}