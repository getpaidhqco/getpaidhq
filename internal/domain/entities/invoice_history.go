package entities

import (
	"time"
)

type InvoiceHistoryAction string

const (
	InvoiceHistoryActionCreated      InvoiceHistoryAction = "created"
	InvoiceHistoryActionUpdated      InvoiceHistoryAction = "updated"
	InvoiceHistoryActionSent         InvoiceHistoryAction = "sent"
	InvoiceHistoryActionViewed       InvoiceHistoryAction = "viewed"
	InvoiceHistoryActionPaid         InvoiceHistoryAction = "paid"
	InvoiceHistoryActionPartialPaid  InvoiceHistoryAction = "partial_paid"
	InvoiceHistoryActionOverdue      InvoiceHistoryAction = "overdue"
	InvoiceHistoryActionReminded     InvoiceHistoryAction = "reminded"
	InvoiceHistoryActionVoided       InvoiceHistoryAction = "voided"
	InvoiceHistoryActionCredited     InvoiceHistoryAction = "credited"
	InvoiceHistoryActionRefunded     InvoiceHistoryAction = "refunded"
	InvoiceHistoryActionDisputed     InvoiceHistoryAction = "disputed"
	InvoiceHistoryActionAdjusted     InvoiceHistoryAction = "adjusted"
)

type InvoiceHistory struct {
	OrgId      string               `json:"org_id"`
	Id         string               `json:"id"`
	InvoiceId  string               `json:"invoice_id"`
	Action     InvoiceHistoryAction `json:"action"`
	Field      string               `json:"field,omitempty"`
	OldValue   interface{}          `json:"old_value,omitempty"`
	NewValue   interface{}          `json:"new_value,omitempty"`
	UserId     string               `json:"user_id,omitempty"`
	UserEmail  string               `json:"user_email,omitempty"`
	IpAddress  string               `json:"ip_address,omitempty"`
	UserAgent  string               `json:"user_agent,omitempty"`
	Reason     string               `json:"reason,omitempty"`
	Metadata   map[string]string    `json:"metadata,omitempty"`
	Timestamp  time.Time            `json:"timestamp"`
}