package domain

import "time"

type RecurringRevenue struct {
	Period    time.Time `json:"period"`
	Total     float64   `json:"total"`
	Count     float64   `json:"count"`
	GrowthMoM float64   `json:"growth_mom"`
	Type      string    `json:"type"`
}

type Refunds struct {
	Period    time.Time `json:"period"`
	Total     float64   `json:"total"`
	Count     float64   `json:"count"`
	GrowthMoM float64   `json:"growth_mom"`
	Type      string    `json:"type"`
}
