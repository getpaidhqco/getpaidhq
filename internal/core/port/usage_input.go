package port

import "time"

// RecordEventInput is the parameter of UsageService.RecordEvent. The orgId comes from
// the authenticated context; the HTTP request DTO maps to this via ToInput.
type RecordEventInput struct {
	OrgId              string
	CustomerId         string
	ExternalCustomerId string
	MetricCode         string
	SubscriptionId     string
	ExternalId         string
	Timestamp          time.Time
	Metadata           map[string]string
}
