package repositories

import (
	"context"
	"payloop/internal/application/dto"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/values"
	"time"
)

type ReportRepository interface {
	GetMRR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	GetARR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	GetRefundTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	GetCustomerChurnTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	GetCustomerChurnRates(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error)
	UpsertSubscription(ctx context.Context, entity entities.Subscription) error
	UpsertPayment(ctx context.Context, entity entities.Payment) error
	UpsertCustomer(ctx context.Context, entity entities.Customer) error
	UpsertRefund(ctx context.Context, entity entities.Refund) error
	UpsertCustomerCohort(ctx context.Context, entity entities.CustomerCohort) error
	StoreDailyMetrics(ctx context.Context, input dto.ProcessDailyMetricsInput) error
	ProcessDailyMetrics(ctx context.Context, d time.Time) error
}
