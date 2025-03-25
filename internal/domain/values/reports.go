package values

import "time"

type RecurringRevenue struct {
	Period time.Time `json:"period"`
	Total  int64     `json:"total"`
	Type   string    `json:"type"`
}
