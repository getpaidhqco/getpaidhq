package domain

import "time"

// RecurringRevenue is the row shape the reporting layer used to return
// from /reports/* endpoints. The reporting layer is currently torn down
// (see internal/adapter/postgres/report_repo.go for context); this type
// is kept so the stub repo's method signatures remain meaningful when
// the layer is revived.
type RecurringRevenue struct {
	Period    time.Time `json:"period"`
	Total     float64   `json:"total"`
	Count     float64   `json:"count"`
	GrowthMoM float64   `json:"growth_mom"`
	Type      string    `json:"type"`
}
