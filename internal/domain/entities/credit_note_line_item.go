package entities

import (
	"time"
)

type CreditNoteLineItem struct {
	OrgId        string    `json:"org_id"`
	CreditNoteId string    `json:"credit_note_id"`
	Id           string    `json:"id"`
	Description  string    `json:"description"`
	Quantity     float64   `json:"quantity"`
	UnitPrice    int       `json:"unit_price"`
	Amount       int       `json:"amount"`
	TaxAmount    int       `json:"tax_amount,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}