package domain

// AggregationType is how a BillableMetric turns raw usage events into a quantity.
type AggregationType string

const (
	AggregationCount       AggregationType = "count"        // number of events (no field)
	AggregationSum         AggregationType = "sum"          // sum of a numeric field
	AggregationMax         AggregationType = "max"          // largest value of a numeric field
	AggregationLatest      AggregationType = "latest"       // last reported numeric value
	AggregationWeightedSum AggregationType = "weighted_sum" // time-weighted standing level; requires CarryOver (a flow meter resets each period, so a time average would underbill)
	AggregationUniqueCount AggregationType = "unique_count" // distinct values of a field (usually a string id)
)
