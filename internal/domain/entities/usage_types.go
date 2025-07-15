package entities

// UsageType represents the type of usage for a subscription item
type UsageType string

const (
	// UsageTypeMetered is the only usage type in the system
	// All usage is treated as "metered" with different aggregation methods
	UsageTypeMetered UsageType = "metered"
)

// AggregationType defines how usage is calculated for billing
type AggregationType string

const (
	// AggregationTypeSum adds all usage during the period (most common)
	AggregationTypeSum AggregationType = "sum"
	
	// AggregationTypeMax bills for the highest usage point during the period
	AggregationTypeMax AggregationType = "max"
	
	// AggregationTypeAverage bills based on average usage during the period
	AggregationTypeAverage AggregationType = "average"
	
	// AggregationTypeLastDuringPeriod bills based on the final value in the period
	AggregationTypeLastDuringPeriod AggregationType = "last_during_period"
)

// UnitType defines what is being measured
type UnitType string

const (
	// UnitTypeCount is a simple quantity (API calls, SMS, emails)
	UnitTypeCount UnitType = "count"
	
	// UnitTypeGBHours is storage over time
	UnitTypeGBHours UnitType = "gb_hours"
	
	// UnitTypeMinutes is time-based usage
	UnitTypeMinutes UnitType = "minutes"
	
	// UnitTypeMB is data transfer
	UnitTypeMB UnitType = "mb"
	
	// UnitTypeGB is data transfer in gigabytes
	UnitTypeGB UnitType = "gb"
	
	// UnitTypeTransactions is payment processing (count + value)
	UnitTypeTransactions UnitType = "transactions"
	
	// UnitTypeCents is monetary amounts for percentage calculations
	UnitTypeCents UnitType = "cents"
	
	// UnitTypeDollars is monetary amounts for percentage calculations
	UnitTypeDollars UnitType = "dollars"
	
	// UnitTypeSeats is active users or licenses
	UnitTypeSeats UnitType = "seats"
)