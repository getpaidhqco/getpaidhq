package handler

import (
	"time"

	"getpaidhq/internal/core/port"
)

// IngestEventsRequest is the body for POST /usage/ingest — a batch of 1..N usage
// events. Each element is validated and ingested independently (per-event results).
type IngestEventsRequest struct {
	Events []RecordEventRequest `json:"events" validate:"required,min=1,max=500,dive"`
}

// ToInputs maps the batch to service inputs, scoped to the org.
func (r IngestEventsRequest) ToInputs(orgId string) []port.RecordEventInput {
	inputs := make([]port.RecordEventInput, len(r.Events))
	for i, e := range r.Events {
		inputs[i] = e.ToInput(orgId)
	}
	return inputs
}

// RecordEventRequest is one usage event in an ingest batch. Exactly one of customer_id /
// external_customer_id should be set (enforced in the service). metric_code is required.
type RecordEventRequest struct {
	CustomerId         string            `json:"customer_id"`
	ExternalCustomerId string            `json:"external_customer_id"`
	MetricCode         string            `json:"metric_code" validate:"required"`
	SubscriptionId     string            `json:"subscription_id"`
	ExternalId         string            `json:"external_id"`
	Timestamp          time.Time         `json:"timestamp"`
	Metadata           map[string]string `json:"metadata"`
}

func (r RecordEventRequest) ToInput(orgId string) port.RecordEventInput {
	return port.RecordEventInput{
		OrgId:              orgId,
		CustomerId:         r.CustomerId,
		ExternalCustomerId: r.ExternalCustomerId,
		MetricCode:         r.MetricCode,
		SubscriptionId:     r.SubscriptionId,
		ExternalId:         r.ExternalId,
		Timestamp:          r.Timestamp,
		Metadata:           r.Metadata,
	}
}
