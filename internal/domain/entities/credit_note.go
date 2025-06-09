package entities

import (
	"time"
)

type CreditNoteReason string

const (
	CreditNoteReasonCorrection   CreditNoteReason = "correction"
	CreditNoteReasonCancellation CreditNoteReason = "cancellation"
	CreditNoteReasonRefund       CreditNoteReason = "refund"
	CreditNoteReasonAdjustment   CreditNoteReason = "adjustment"
	CreditNoteReasonTaxAdjustment CreditNoteReason = "tax_adjustment"
)

type CreditNote struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	SequenceId  string            `json:"sequence_id"`
	DocNumber   string            `json:"doc_number"`
	InvoiceId   string            `json:"invoice_id,omitempty"`
	Reason      CreditNoteReason  `json:"reason"`
	ReasonNote  string            `json:"reason_note,omitempty"`
	Currency    string            `json:"currency"`
	Amount      int               `json:"amount"`
	TaxAmount   int               `json:"tax_amount,omitempty"`
	Status      string            `json:"status"`
	AppliedAt   time.Time         `json:"applied_at,omitempty"`
	Notes       string            `json:"notes,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}