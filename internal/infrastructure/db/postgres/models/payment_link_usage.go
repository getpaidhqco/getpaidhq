package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type PaymentLinkUsage struct {
	Id            string      `json:"id"`
	OrgId         string      `json:"org_id"`
	PaymentLinkId string      `json:"payment_link_id"`
	SessionId     pgtype.Text `json:"session_id,omitempty"`
	CustomerId    pgtype.Text `json:"customer_id,omitempty"`
	EventType     string      `json:"event_type"`
	IpAddress     pgtype.Text `json:"ip_address,omitempty"`
	UserAgent     pgtype.Text `json:"user_agent,omitempty"`
	Referer       pgtype.Text `json:"referer,omitempty"`
	Country       pgtype.Text `json:"country,omitempty"`
	Metadata      []byte      `json:"metadata,omitempty"`
	Timestamp     pgtype.Date `json:"timestamp"`
}

func (p *PaymentLinkUsage) ToEntity() entities.PaymentLinkUsage {
	return entities.PaymentLinkUsage{
		Id:            p.Id,
		OrgId:         p.OrgId,
		PaymentLinkId: p.PaymentLinkId,
		SessionId:     p.SessionId.String,
		CustomerId:    p.CustomerId.String,
		EventType:     p.EventType,
		IpAddress:     p.IpAddress.String,
		UserAgent:     p.UserAgent.String,
		Referer:       p.Referer.String,
		Country:       p.Country.String,
		Metadata:      p.Metadata,
		Timestamp:     p.Timestamp.Time,
	}
}