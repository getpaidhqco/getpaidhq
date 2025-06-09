package entities

import (
	"time"
)

type InvoiceStatus string
type InvoiceType string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusSent      InvoiceStatus = "sent"
	InvoiceStatusPaid      InvoiceStatus = "paid"
	InvoiceStatusOverdue   InvoiceStatus = "overdue"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
	InvoiceStatusRefunded  InvoiceStatus = "refunded"
)

const (
	InvoiceTypeInitial      InvoiceType = "initial"
	InvoiceTypeRecurring    InvoiceType = "recurring"
	InvoiceTypeUsage        InvoiceType = "usage"
	InvoiceTypeAdjustment   InvoiceType = "adjustment"
	InvoiceTypeSetup        InvoiceType = "setup"
	InvoiceTypeCancellation InvoiceType = "cancellation"
	InvoiceTypeRefund       InvoiceType = "refund"
)

type Invoice struct {
	OrgId          string            `json:"org_id"`
	Id             string            `json:"id"`
	CustomerId     string            `json:"customer_id,omitempty"`
	OrderId        string            `json:"order_id,omitempty"`
	SubscriptionId string            `json:"subscription_id,omitempty"`
	SequenceId     string            `json:"sequence_id"`
	DocNumber      string            `json:"doc_number"`
	Type           DocumentType      `json:"type"`
	InvoiceType    InvoiceType       `json:"invoice_type"`
	Status         InvoiceStatus     `json:"status"`
	IsImmutable    bool              `json:"is_immutable"`
	Currency       string            `json:"currency"`
	SubTotal       int               `json:"sub_total"`
	TaxTotal       int               `json:"tax_total"`
	DiscountTotal  int               `json:"discount_total"`
	Total          int               `json:"total"`
	AmountPaid     int               `json:"amount_paid"`
	AmountDue      int               `json:"amount_due"`
	TaxProvider    string            `json:"tax_provider,omitempty"`
	TaxTransactionId string          `json:"tax_transaction_id,omitempty"`
	TaxBreakdown   map[string]interface{} `json:"tax_breakdown,omitempty"`
	IssuedAt       time.Time         `json:"issued_at,omitempty"`
	DueAt          time.Time         `json:"due_at,omitempty"`
	PaidAt         time.Time         `json:"paid_at,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	CustomerNotes  string            `json:"customer_notes,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	ExchangeRate   int               `json:"exchange_rate,omitempty"`
	BaseCurrency   string            `json:"base_currency,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
