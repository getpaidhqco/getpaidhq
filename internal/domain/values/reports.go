package values

import "time"

type RecurringRevenue struct {
	Period time.Time `json:"period"`
	Total  float64   `json:"total"`
	Type   string    `json:"type"`
}
