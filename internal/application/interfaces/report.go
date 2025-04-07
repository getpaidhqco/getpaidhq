package interfaces

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/values"
	"time"
)

type ReportService interface {
	GetMonthlyRecurringRevenue(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	GetAnnualRecurringRevenue(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	HandlePublishedEvent(topic string, data []byte)
	ProcessDataChange(event dto.DataChangeEvent)
}
