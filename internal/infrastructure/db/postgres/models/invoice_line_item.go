package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type InvoiceLineItem struct {
	OrgId         string             `json:"org_id"`
	InvoiceId     string             `json:"invoice_id"`
	Id            string             `json:"id"`
	ProductId     pgtype.Text        `json:"product_id"`
	VariantId     pgtype.Text        `json:"variant_id"`
	PriceId       pgtype.Text        `json:"price_id"`
	Description   string             `json:"description"`
	Category      pgtype.Text        `json:"category"`
	Quantity      float64            `json:"quantity"`
	UnitPrice     int                `json:"unit_price"`
	LineTotal     int                `json:"line_total"`
	DiscountType  pgtype.Text        `json:"discount_type"`
	DiscountValue int                `json:"discount_value"`
	DiscountTotal int                `json:"discount_total"`
	TaxCode       pgtype.Text        `json:"tax_code"`
	TaxRate       int                `json:"tax_rate"`
	TaxAmount     int                `json:"tax_amount"`
	TaxExempt     bool               `json:"tax_exempt"`
	Seq           int                `json:"seq"`
	Metadata      []byte             `json:"metadata"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
	UpdatedAt     pgtype.Timestamptz `json:"updated_at"`
}

func (i *InvoiceLineItem) ToEntity() entities.InvoiceLineItem {
	var metadata map[string]string

	// Handle JSON fields
	if i.Metadata != nil {
		_ = json.Unmarshal(i.Metadata, &metadata)
	}

	return entities.InvoiceLineItem{
		OrgId:         i.OrgId,
		InvoiceId:     i.InvoiceId,
		Id:            i.Id,
		ProductId:     i.ProductId.String,
		VariantId:     i.VariantId.String,
		PriceId:       i.PriceId.String,
		Description:   i.Description,
		Category:      i.Category.String,
		Quantity:      i.Quantity,
		UnitPrice:     i.UnitPrice,
		LineTotal:     i.LineTotal,
		DiscountType:  i.DiscountType.String,
		DiscountValue: i.DiscountValue,
		DiscountTotal: i.DiscountTotal,
		TaxCode:       i.TaxCode.String,
		TaxRate:       i.TaxRate,
		TaxAmount:     i.TaxAmount,
		TaxExempt:     i.TaxExempt,
		Seq:           i.Seq,
		Metadata:      metadata,
		CreatedAt:     i.CreatedAt.Time,
		UpdatedAt:     i.UpdatedAt.Time,
	}
}