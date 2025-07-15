package entities

import (
	"time"
)

// UsageEvent represents a raw usage event based on CloudEvents v1.0 specification
type UsageEvent struct {
	OrgId string `json:"org_id"`
	Id    string `json:"id"` // CloudEvent id field

	// Business context (enriched from CloudEvent subject)
	MeterId string `json:"meter_id"`

	// CloudEvents v1.0 fields
	SpecVersion string                 `json:"spec_version"`
	Type        string                 `json:"type"`
	EventId     string                 `json:"event_id"` // CloudEvent id field
	Time        time.Time              `json:"time"`
	Source      string                 `json:"source"`
	Subject     string                 `json:"subject"`
	Data        map[string]interface{} `json:"data"`
	Metadata    map[string]string      `json:"metadata"`
	// Audit
	ReceivedAt time.Time `json:"received_at"`
}

// CalculatedUsageRecord represents a calculated usage record after processing
type CalculatedUsageRecord struct {
	Id                 string `json:"id"`
	OrgId              string `json:"org_id"`
	SubscriptionId     string `json:"subscription_id"`
	SubscriptionItemId string `json:"subscription_item_id"`
	CustomerId         string `json:"customer_id"`
	MeterId            string `json:"meter_id"`

	// Raw event data that was processed
	EventName       string                 `json:"event_name"`
	EventData       map[string]interface{} `json:"event_data"`
	OriginalEventId string                 `json:"original_event_id"`

	// Extracted value based on meter configuration
	ExtractedValue  float64 `json:"extracted_value"`
	AggregationType string  `json:"aggregation_type"`

	// Calculated outputs
	UnitPrice      int64   `json:"unit_price,omitempty"`
	PercentageRate float64 `json:"percentage_rate,omitempty"`
	FixedFee       int64   `json:"fixed_fee,omitempty"`
	TotalAmount    int64   `json:"total_amount"`

	// Reference tracking
	ReferenceId string `json:"reference_id,omitempty"`

	// Timing
	UsageDate     time.Time `json:"usage_date"`
	BillingPeriod string    `json:"billing_period"`

	// Processing
	Processed   bool       `json:"processed"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	InvoiceId   string     `json:"invoice_id,omitempty"`

	// Audit
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UsageProcessingStatus tracks the processing status of usage events for a billing period
type UsageProcessingStatus struct {
	// Identifiers
	OrgID              string
	SubscriptionItemID string
	BillingPeriod      string // Format: "2025-07"

	// Aggregated usage data
	TotalQuantity float64
	TotalAmount   int64
	EventCount    int

	// Processing status
	Processed   bool
	ProcessedAt *time.Time
	InvoiceID   *string

	// Time tracking
	FirstEventTime time.Time
	LastEventTime  time.Time
	LastUpdated    time.Time
}

// UsageEventLog represents an audit log entry for usage-related events
type UsageEventLog struct {
	ID        string
	Timestamp time.Time
	OrgID     string
	EventType string // 'recorded', 'processed', 'corrected', 'refunded'

	// Context
	SubscriptionID     *string
	SubscriptionItemID *string
	CustomerID         *string
	InvoiceID          *string

	// Data
	Amount        *int64
	Quantity      *float64
	EventCount    *int
	BillingPeriod *string

	// Metadata
	TriggeredBy *string
	Reason      *string
	Metadata    map[string]interface{}
}

// MonthlyUsageAggregate represents aggregated usage data for a billing period
type MonthlyUsageAggregate struct {
	SubscriptionID     string
	SubscriptionItemID string
	UsageType          string
	TotalQuantity      float64
	TotalAmount        int64
	ActiveDays         int
	TotalEvents        int
	PeriodStart        time.Time
	PeriodEnd          time.Time
}

// UsageSummary represents a summary of usage for real-time dashboards
type UsageSummary struct {
	Quantity  float64
	Amount    int64
	Events    int
	LastUsage *time.Time
}

// CustomerUsageSummary represents a summary of usage for a customer
type CustomerUsageSummary struct {
	CustomerID      string
	TotalAmount     int64
	TotalEvents     int
	UsageByType     map[string]float64
	AmountByType    map[string]int64
	FirstUsageTime  time.Time
	LastUsageTime   time.Time
	ActiveDays      int
	SubscriptionIDs []string
}

// UsageTypeAnalytics represents analytics data for a specific usage type
type UsageTypeAnalytics struct {
	UsageType       string
	TotalQuantity   float64
	TotalAmount     int64
	TotalEvents     int
	UniqueCustomers int
	DailyAverage    float64
	PeakUsage       float64
	PeakTime        time.Time
}
