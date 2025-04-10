package dto

import (
	"payloop/internal/domain/common"
	"time"
)

type DataChangeEvent struct {
	Operation common.Operation
	Entity    common.Entity
	NewObject interface{}
	OldObject interface{}
}

type ProcessDailyMetricsInput struct {
	OrgId    string    `json:"org_id"`
	Date     time.Time `json:"date"`
	Timezone string    `json:"timezone"`
}
