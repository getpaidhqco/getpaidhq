package prices

// BillingInterval represents the billing interval for a price.
type BillingInterval string

const (
	BillingIntervalNone   BillingInterval = "none"
	BillingIntervalSecond BillingInterval = "second"
	BillingIntervalMinute BillingInterval = "minute"
	BillingIntervalHour   BillingInterval = "hour"
	BillingIntervalDay    BillingInterval = "day"
	BillingIntervalWeek   BillingInterval = "week"
	BillingIntervalMonth  BillingInterval = "month"
	BillingIntervalYear   BillingInterval = "year"
)

type PriceCategory string

const (
	OneTime                   PriceCategory = "one_time"
	PriceCategorySubscription PriceCategory = "subscription"
	PriceCategoryUsage        PriceCategory = "usage"
	PriceCategoryHybrid       PriceCategory = "hybrid"
	Free                      PriceCategory = "free"
	Variable                  PriceCategory = "variable"
)

type PriceScheme string

const (
	Fixed     PriceScheme = "fixed"
	Tiered    PriceScheme = "tiered"
	Volume    PriceScheme = "volume"
	Graduated PriceScheme = "graduated"
)

// UsageType represents the type of usage for a price.
type UsageType string

const (
	UsageTypeMetered  UsageType = "metered"
	UsageTypeLicensed UsageType = "licensed"
)

// AggregationType represents how usage is aggregated for billing.
type AggregationType string

const (
	AggregationTypeSum              AggregationType = "sum"
	AggregationTypeMax              AggregationType = "max"
	AggregationTypeAverage          AggregationType = "average"
	AggregationTypeLastDuringPeriod AggregationType = "last_during_period"
)

// UnitType represents the unit being measured for usage-based billing.
type UnitType string

const (
	UnitTypeCount        UnitType = "count"
	UnitTypeTransactions UnitType = "transactions"
	UnitTypeGbHours      UnitType = "gb_hours"
	UnitTypeApiCalls     UnitType = "api_calls"
	UnitTypeStorage      UnitType = "storage"
	UnitTypeBandwidth    UnitType = "bandwidth"
	UnitTypeUsers        UnitType = "users"
	UnitTypeSeats        UnitType = "seats"
	UnitTypeCustom       UnitType = "custom"
)
