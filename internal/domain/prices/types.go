package prices

// PriceCategory represents the category of a price.
type PriceCategory string

const (
	OneTime      PriceCategory = "one_time"
	Subscription PriceCategory = "subscription"
	Free         PriceCategory = "free"
	Variable     PriceCategory = "variable"
)

type PriceScheme string

const (
	Fixed     PriceScheme = "fixed"
	Tiered    PriceScheme = "tiered"
	Volume    PriceScheme = "volume"
	Graduated PriceScheme = "graduated"
)

// BillingInterval represents the billing interval for a price.
type BillingInterval string

const (
	BillingIntervalNone  BillingInterval = "none"
	BillingIntervalDay   BillingInterval = "day"
	BillingIntervalWeek  BillingInterval = "week"
	BillingIntervalMonth BillingInterval = "month"
	BillingIntervalYear  BillingInterval = "year"
)
