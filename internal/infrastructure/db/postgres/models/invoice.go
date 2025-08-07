package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Invoice struct {
	OrgId           string                 `json:"org_id"`
	Id              string                 `json:"id"`
	CustomerId      pgtype.Text            `json:"customer_id"`
	OrderId         pgtype.Text            `json:"order_id"`
	SubscriptionId  pgtype.Text            `json:"subscription_id"`
	SequenceId      string                 `json:"sequence_id"`
	DocNumber       string                 `json:"doc_number"`
	Type            string                 `json:"type"`
	InvoiceType     string                 `json:"invoice_type"`
	Status          string                 `json:"status"`
	IsImmutable     bool                   `json:"is_immutable"`
	Currency        string                 `json:"currency"`
	SubTotal        int                    `json:"sub_total"`
	TaxTotal        int                    `json:"tax_total"`
	DiscountTotal   int                    `json:"discount_total"`
	Total           int                    `json:"total"`
	AmountPaid      int                    `json:"amount_paid"`
	AmountDue       int                    `json:"amount_due"`
	TaxProvider     pgtype.Text            `json:"tax_provider"`
	TaxTransactionId pgtype.Text           `json:"tax_transaction_id"`
	TaxBreakdown    []byte                 `json:"tax_breakdown"`
	IssuedAt        pgtype.Timestamptz     `json:"issued_at"`
	DueAt           pgtype.Timestamptz     `json:"due_at"`
	PaidAt          pgtype.Timestamptz     `json:"paid_at"`
	DeliveredAt     pgtype.Timestamptz     `json:"delivered_at"`             // When invoice was emailed to customer
	VoidedAt        pgtype.Timestamptz     `json:"voided_at"`                // When invoice was voided
	MarkedUncollectibleAt pgtype.Timestamptz `json:"marked_uncollectible_at"` // When marked as bad debt
	Notes           pgtype.Text            `json:"notes"`
	CustomerNotes   pgtype.Text            `json:"customer_notes"`
	Metadata        []byte                 `json:"metadata"`
	ExchangeRate    int                    `json:"exchange_rate"`
	BaseCurrency    pgtype.Text            `json:"base_currency"`
	CreatedAt       pgtype.Timestamptz     `json:"created_at"`
	UpdatedAt       pgtype.Timestamptz     `json:"updated_at"`
}

func (i *Invoice) ToEntity() entities.Invoice {
	var taxBreakdown map[string]interface{}
	var metadata map[string]string

	// Handle JSON fields
	if i.TaxBreakdown != nil {
		_ = json.Unmarshal(i.TaxBreakdown, &taxBreakdown)
	}

	if i.Metadata != nil {
		_ = json.Unmarshal(i.Metadata, &metadata)
	}

	return entities.Invoice{
		OrgId:           i.OrgId,
		Id:              i.Id,
		CustomerId:      i.CustomerId.String,
		OrderId:         i.OrderId.String,
		SubscriptionId:  i.SubscriptionId.String,
		SequenceId:      i.SequenceId,
		DocNumber:       i.DocNumber,
		Type:            entities.DocumentType(i.Type),
		InvoiceType:     entities.InvoiceType(i.InvoiceType),
		Status:          entities.InvoiceStatus(i.Status),
		IsImmutable:     i.IsImmutable,
		Currency:        i.Currency,
		SubTotal:        i.SubTotal,
		TaxTotal:        i.TaxTotal,
		DiscountTotal:   i.DiscountTotal,
		Total:           i.Total,
		AmountPaid:      i.AmountPaid,
		AmountDue:       i.AmountDue,
		TaxProvider:     i.TaxProvider.String,
		TaxTransactionId: i.TaxTransactionId.String,
		TaxBreakdown:    taxBreakdown,
		IssuedAt:        i.IssuedAt.Time,
		DueAt:           i.DueAt.Time,
		PaidAt:          i.PaidAt.Time,
		DeliveredAt:     i.DeliveredAt.Time,
		VoidedAt:        i.VoidedAt.Time,
		MarkedUncollectibleAt: i.MarkedUncollectibleAt.Time,
		Notes:           i.Notes.String,
		CustomerNotes:   i.CustomerNotes.String,
		Metadata:        metadata,
		ExchangeRate:    i.ExchangeRate,
		BaseCurrency:    i.BaseCurrency.String,
		CreatedAt:       i.CreatedAt.Time,
		UpdatedAt:       i.UpdatedAt.Time,
	}
}
