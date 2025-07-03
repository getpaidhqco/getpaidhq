package entities

import (
	"time"
)

// UsageEvent represents a single usage event in the system
type UsageEvent struct {
	// Time when the usage event occurred
	Time time.Time

	// Organization and subscription context
	OrgID              string
	SubscriptionID     string
	SubscriptionItemID string
	CustomerID         string

	// Usage data
	UsageType        string
	Quantity         *float64 // Pointer to allow nil for percentage-based pricing
	TransactionValue *int64   // Pointer to allow nil for unit-based pricing
	CalculatedAmount int64    // Final calculated amount in cents

	// References for idempotency and tracking
	ReferenceID   *string
	ReferenceType *string
	Metadata      map[string]interface{}
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
	Amount       *int64
	Quantity     *float64
	EventCount   *int
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
