package domain

// BillingInterval represents the billing interval for a price.
type BillingInterval string

const (
	BillingIntervalNone   BillingInterval = "none"
	BillingIntervalSecond BillingInterval = "second"
	BillingIntervalMinute BillingInterval = "minute"
	BillingIntervalDay    BillingInterval = "day"
	BillingIntervalWeek   BillingInterval = "week"
	BillingIntervalMonth  BillingInterval = "month"
	BillingIntervalYear   BillingInterval = "year"
)

type PriceCategory string

const (
	OneTime                   PriceCategory = "one_time"
	PriceCategorySubscription PriceCategory = "subscription"
	Free                      PriceCategory = "free"
	Variable                  PriceCategory = "variable"
	// Note: there is no "metered" category. Metering is a pricing method, orthogonal
	// to cadence — a metered price is a subscription with a meter (see Price.IsMetered).
)

type PriceScheme string

const (
	Fixed     PriceScheme = "fixed"
	Tiered    PriceScheme = "tiered"
	Volume    PriceScheme = "volume"
	Graduated PriceScheme = "graduated"
	// Package bills every STARTED block of UnitCount units at UnitPrice cents —
	// ceil(units/UnitCount) × UnitPrice ("$5 per started 1,000 SMS"). The round-up
	// sibling of Fixed: same (UnitPrice, UnitCount) pair, but a partial block owes
	// the full block instead of prorating. Metered only, flat (no tiers).
	Package PriceScheme = "package"
)
