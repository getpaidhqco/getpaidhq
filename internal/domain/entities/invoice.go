package entities

import (
	"fmt"
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
	LineItems      []InvoiceLineItem `json:"line_items,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Business methods for Invoice aggregate

// AddLineItem adds a line item to the invoice and recalculates totals
func (i *Invoice) AddLineItem(item InvoiceLineItem) error {
	if i.IsImmutable {
		return fmt.Errorf("cannot add line item to immutable invoice")
	}
	
	// Set invoice reference
	item.InvoiceId = i.Id
	item.OrgId = i.OrgId
	
	// Add to line items
	i.LineItems = append(i.LineItems, item)
	
	// Recalculate totals
	i.RecalculateTotals()
	
	return nil
}

// UpdateLineItem updates an existing line item and recalculates totals
func (i *Invoice) UpdateLineItem(lineItemId string, updates InvoiceLineItem) error {
	if i.IsImmutable {
		return fmt.Errorf("cannot update line item on immutable invoice")
	}
	
	for idx, item := range i.LineItems {
		if item.Id == lineItemId {
			// Preserve core identifiers
			updates.Id = item.Id
			updates.InvoiceId = i.Id
			updates.OrgId = i.OrgId
			updates.CreatedAt = item.CreatedAt
			
			// Update the line item
			i.LineItems[idx] = updates
			
			// Recalculate totals
			i.RecalculateTotals()
			
			return nil
		}
	}
	
	return fmt.Errorf("line item with id %s not found", lineItemId)
}

// RemoveLineItem removes a line item from the invoice and recalculates totals
func (i *Invoice) RemoveLineItem(lineItemId string) error {
	if i.IsImmutable {
		return fmt.Errorf("cannot remove line item from immutable invoice")
	}
	
	for idx, item := range i.LineItems {
		if item.Id == lineItemId {
			// Remove the item
			i.LineItems = append(i.LineItems[:idx], i.LineItems[idx+1:]...)
			
			// Recalculate totals
			i.RecalculateTotals()
			
			return nil
		}
	}
	
	return fmt.Errorf("line item with id %s not found", lineItemId)
}

// RecalculateTotals recalculates all invoice totals based on line items
func (i *Invoice) RecalculateTotals() {
	var subTotal, taxTotal, discountTotal int
	
	for _, item := range i.LineItems {
		itemTotal := int(item.Quantity * float64(item.UnitPrice))
		subTotal += itemTotal
		discountTotal += item.DiscountTotal
		taxTotal += item.TaxAmount
	}
	
	i.SubTotal = subTotal
	i.TaxTotal = taxTotal
	i.DiscountTotal = discountTotal
	i.Total = subTotal + taxTotal - discountTotal
	i.AmountDue = i.Total - i.AmountPaid
}

// GetLineItemById returns a line item by its ID
func (i *Invoice) GetLineItemById(lineItemId string) (InvoiceLineItem, bool) {
	for _, item := range i.LineItems {
		if item.Id == lineItemId {
			return item, true
		}
	}
	return InvoiceLineItem{}, false
}

// IsComplete returns true if the invoice has line items
func (i *Invoice) IsComplete() bool {
	return len(i.LineItems) > 0
}

// GetLineItemCount returns the number of line items
func (i *Invoice) GetLineItemCount() int {
	return len(i.LineItems)
}

// ValidateTotals checks if the calculated totals match the stored totals
func (i *Invoice) ValidateTotals() error {
	originalSubTotal := i.SubTotal
	originalTaxTotal := i.TaxTotal
	originalDiscountTotal := i.DiscountTotal
	originalTotal := i.Total
	
	// Temporarily recalculate
	i.RecalculateTotals()
	
	// Check for discrepancies
	if i.SubTotal != originalSubTotal {
		return fmt.Errorf("subtotal mismatch: calculated %d, stored %d", i.SubTotal, originalSubTotal)
	}
	if i.TaxTotal != originalTaxTotal {
		return fmt.Errorf("tax total mismatch: calculated %d, stored %d", i.TaxTotal, originalTaxTotal)
	}
	if i.DiscountTotal != originalDiscountTotal {
		return fmt.Errorf("discount total mismatch: calculated %d, stored %d", i.DiscountTotal, originalDiscountTotal)
	}
	if i.Total != originalTotal {
		return fmt.Errorf("total mismatch: calculated %d, stored %d", i.Total, originalTotal)
	}
	
	return nil
}
