package handler

import (
	"time"

	"getpaidhq/internal/core/port"
	"getpaidhq/internal/core/service"
)

// IngestEventResult is the per-event outcome in an ingest batch, aligned by index
// to the request's events array.
type IngestEventResult struct {
	Index  int    `json:"index"`
	Id     string `json:"id,omitempty"`
	Status string `json:"status"`          // recorded | duplicate | accepted | rejected
	Error  string `json:"error,omitempty"` // set only when status is "rejected"
}

// IngestEventsResponse is the result of POST /usage/ingest.
type IngestEventsResponse struct {
	Results []IngestEventResult `json:"results"`
}

func NewIngestEventsResponse(results []port.IngestResult) IngestEventsResponse {
	out := make([]IngestEventResult, len(results))
	for i, r := range results {
		status := r.Status
		if status == "" {
			status = port.IngestRecorded
		}
		out[i] = IngestEventResult{Index: i, Id: r.Id, Status: string(status), Error: r.Error}
	}
	return IngestEventsResponse{Results: out}
}

// MeterUsageResponse is one meter's usage quantity for the period.
type MeterUsageResponse struct {
	MetricCode  string `json:"metric_code"`
	Aggregation string `json:"aggregation"`
	Quantity    string `json:"quantity"` // decimal string — preserves precision
}

// SubscriptionUsageResponse is a subscription's usage for its current billing period.
type SubscriptionUsageResponse struct {
	SubscriptionId     string               `json:"subscription_id"`
	CurrentPeriodStart time.Time            `json:"current_period_start"`
	CurrentPeriodEnd   time.Time            `json:"current_period_end"`
	Meters             []MeterUsageResponse `json:"meters"`
}

func NewSubscriptionUsageResponse(u service.SubscriptionUsage) SubscriptionUsageResponse {
	meters := make([]MeterUsageResponse, len(u.Meters))
	for i, m := range u.Meters {
		meters[i] = MeterUsageResponse{
			MetricCode:  m.MetricCode,
			Aggregation: string(m.Aggregation),
			Quantity:    m.Quantity.String(),
		}
	}
	return SubscriptionUsageResponse{
		SubscriptionId:     u.SubscriptionId,
		CurrentPeriodStart: u.CurrentPeriodStart,
		CurrentPeriodEnd:   u.CurrentPeriodEnd,
		Meters:             meters,
	}
}
