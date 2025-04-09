package values

import "time"

type RecurringRevenue struct {
	Period    time.Time `json:"period"`
	Total     float64   `json:"total"`
	GrowthMoM float64   `json:"growth_mom"`
	Type      string    `json:"type"`
}
