package models

import "time"

type DailyMetric struct {
	OrgId              string
	Date               time.Time
	MRR                int64
	ARR                int64
	CustomerCount      int
	ChurnRate          int64
	ARPU               int64
	CLTV               int64
	SuccessfulPayments int
	FailedPayments     int
	Refunds            int64
	CreatedAt          time.Time
}
