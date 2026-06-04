package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// MeterEvent is one recorded use, attached to a Customer (never a Subscription at
// record time). Identify the customer with CustomerId (ours) or ExternalCustomerId
// (the merchant's) — exactly one. ExternalId is the caller's own id for the event and
// doubles as the dedup key. SubscriptionId is optional attribution (blank =
// unattributed; billed by the customer's earliest metered subscription for the meter).
type MeterEvent struct {
	OrgId              string
	Id                 string
	CustomerId         string
	ExternalCustomerId string
	MetricCode         string
	SubscriptionId     string
	ExternalId         string
	Metadata           map[string]string
	Value              decimal.Decimal // numeric field pulled from Metadata at ingest (0 for count/unique_count)
	Timestamp          time.Time
	CreatedAt          time.Time
}
