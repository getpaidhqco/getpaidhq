package entities

import (
	"time"
)

// DocumentType represents the type of document
type DocumentType string

const (
	DocumentTypeInvoice    DocumentType = "invoice"    // Standard invoice for payment
	DocumentTypeProforma   DocumentType = "proforma"   // Proforma invoice (quote/estimate)
	DocumentTypeQuote      DocumentType = "quote"      // Formal quote/proposal
	DocumentTypeReceipt    DocumentType = "receipt"    // Payment receipt
	DocumentTypeStatement  DocumentType = "statement"  // Account statement
)

type Document struct {
	OrgId          string            `json:"org_id"`
	Id             string            `json:"id"`
	InvoiceId      string            `json:"invoice_id,omitempty"`
	CreditNoteId   string            `json:"credit_note_id,omitempty"`
	Filename       string            `json:"filename"`
	OriginalName   string            `json:"original_name"`
	ContentType    string            `json:"content_type"`
	Size           int               `json:"size"`
	StorageProvider string            `json:"storage_provider"`
	StorageKey     string            `json:"storage_key"`
	Url            string            `json:"url,omitempty"`
	Type           DocumentType      `json:"type"`
	Purpose        string            `json:"purpose,omitempty"`
	IsPublic       bool              `json:"is_public"`
	AccessToken    string            `json:"access_token,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}
