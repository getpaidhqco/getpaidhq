package models

import (
	"encoding/json"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
	"time"
)

type InvoiceHistory struct {
	OrgId      string             `json:"org_id"`
	Id         string             `json:"id"`
	InvoiceId  string             `json:"invoice_id"`
	Action     string             `json:"action"`
	Field      pgtype.Text        `json:"field"`
	OldValue   []byte             `json:"old_value"`
	NewValue   []byte             `json:"new_value"`
	UserId     pgtype.Text        `json:"user_id"`
	UserEmail  pgtype.Text        `json:"user_email"`
	IpAddress  pgtype.Text        `json:"ip_address"`
	UserAgent  pgtype.Text        `json:"user_agent"`
	Reason     pgtype.Text        `json:"reason"`
	Metadata   []byte             `json:"metadata"`
	Timestamp  time.Time          `json:"timestamp"`
}

func (h *InvoiceHistory) ToEntity() entities.InvoiceHistory {
	var oldValue interface{}
	var newValue interface{}
	var metadata map[string]string

	// Handle JSON fields
	if h.OldValue != nil {
		_ = json.Unmarshal(h.OldValue, &oldValue)
	}

	if h.NewValue != nil {
		_ = json.Unmarshal(h.NewValue, &newValue)
	}

	if h.Metadata != nil {
		_ = json.Unmarshal(h.Metadata, &metadata)
	}

	return entities.InvoiceHistory{
		OrgId:      h.OrgId,
		Id:         h.Id,
		InvoiceId:  h.InvoiceId,
		Action:     entities.InvoiceHistoryAction(h.Action),
		Field:      h.Field.String,
		OldValue:   oldValue,
		NewValue:   newValue,
		UserId:     h.UserId.String,
		UserEmail:  h.UserEmail.String,
		IpAddress:  h.IpAddress.String,
		UserAgent:  h.UserAgent.String,
		Reason:     h.Reason.String,
		Metadata:   metadata,
		Timestamp:  h.Timestamp,
	}
}