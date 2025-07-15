package entities

import "time"

// UsageLineItem represents aggregated usage for an invoice line item
type UsageLineItem struct {
	SubscriptionItemId string    `json:"subscription_item_id"`
	MeterId           string    `json:"meter_id"`
	MeterName         string    `json:"meter_name"`
	AggregatedValue   float64   `json:"aggregated_value"`
	TotalAmount       int64     `json:"total_amount"`
	EventCount        int       `json:"event_count"`
	PeriodStart       time.Time `json:"period_start"`
	PeriodEnd         time.Time `json:"period_end"`
}