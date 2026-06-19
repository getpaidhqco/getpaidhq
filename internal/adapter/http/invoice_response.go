package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// InvoiceLineItemResponse is the API shape of one invoice line. Quantity/UnitAmount
// are decimals serialized as strings to preserve precision (fractional usage/rates).
type InvoiceLineItemResponse struct {
	Id          string            `json:"id"`
	PriceId     string            `json:"price_id"`
	Kind        string            `json:"kind"` // base | usage
	Description string            `json:"description"`
	Quantity    string            `json:"quantity"`
	UnitAmount  string            `json:"unit_amount"` // cents (may be fractional)
	Total       int64             `json:"total"`       // cents
	Metadata    map[string]string `json:"metadata"`
}

type InvoiceResponse struct {
	Id             string                    `json:"id"`
	SubscriptionId string                    `json:"subscription_id"`
	CustomerId     string                    `json:"customer_id"`
	OrderId        string                    `json:"order_id"`
	Status         string                    `json:"status" validate:"oneof=draft open paid uncollectible void"`
	Currency       string                    `json:"currency"`
	Subtotal       int64                     `json:"subtotal"`
	Total          int64                     `json:"total"`
	Cycle          int                       `json:"cycle"`
	PeriodStart    time.Time                 `json:"period_start"`
	PeriodEnd      time.Time                 `json:"period_end"`
	LineItems      []InvoiceLineItemResponse `json:"line_items"`
	Metadata       map[string]string         `json:"metadata"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
}

func NewInvoiceResponse(inv domain.Invoice) InvoiceResponse {
	lines := make([]InvoiceLineItemResponse, len(inv.LineItems))
	for i, l := range inv.LineItems {
		lines[i] = InvoiceLineItemResponse{
			Id:          l.Id,
			PriceId:     l.PriceId,
			Kind:        string(l.Kind),
			Description: l.Description,
			Quantity:    l.Quantity.String(),
			UnitAmount:  l.UnitAmount.String(),
			Total:       l.Total,
			Metadata:    l.Metadata,
		}
	}
	return InvoiceResponse{
		Id:             inv.Id,
		SubscriptionId: inv.SubscriptionId,
		CustomerId:     inv.CustomerId,
		OrderId:        inv.OrderId,
		Status:         string(inv.Status),
		Currency:       inv.Currency,
		Subtotal:       inv.Subtotal,
		Total:          inv.Total,
		Cycle:          inv.Cycle,
		PeriodStart:    inv.PeriodStart,
		PeriodEnd:      inv.PeriodEnd,
		LineItems:      lines,
		Metadata:       inv.Metadata,
		CreatedAt:      inv.CreatedAt,
		UpdatedAt:      inv.UpdatedAt,
	}
}
