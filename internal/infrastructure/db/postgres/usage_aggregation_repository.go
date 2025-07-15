package postgres

import (
	"context"
	"fmt"
	"time"
	
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type UsageAggregationRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewUsageAggregationRepository(usageDb lib.Database, logger logger.Logger) repositories.UsageAggregationRepository {
	pgDatabase, ok := usageDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *PgDatabase")
	}
	return &UsageAggregationRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r *UsageAggregationRepository) GetMonthlyUsage(ctx context.Context, orgID string, billingPeriod time.Time) ([]entities.MonthlyUsageAggregate, error) {
	tx := r.getTransactionFromContext(ctx)
	
	// Format billing period as YYYY-MM
	billingPeriodStr := billingPeriod.Format("2006-01")
	
	query := `
		SELECT 
			subscription_id,
			subscription_item_id,
			usage_type,
			SUM(daily_quantity) as total_quantity,
			SUM(daily_amount) as total_amount,
			COUNT(DISTINCT day) as active_days,
			SUM(daily_events) as total_events,
			MIN(first_event_time) as period_start,
			MAX(last_event_time) as period_end
		FROM usage_daily_billing
		WHERE org_id = $1 
		  AND billing_period = $2
		GROUP BY subscription_id, subscription_item_id, usage_type
		ORDER BY subscription_id, subscription_item_id
	`
	
	rows, err := tx.Query(ctx, query, orgID, billingPeriodStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly usage: %w", err)
	}
	defer rows.Close()
	
	var aggregates []entities.MonthlyUsageAggregate
	for rows.Next() {
		var agg entities.MonthlyUsageAggregate
		
		err := rows.Scan(
			&agg.SubscriptionID,
			&agg.SubscriptionItemID,
			&agg.UsageType,
			&agg.TotalQuantity,
			&agg.TotalAmount,
			&agg.ActiveDays,
			&agg.TotalEvents,
			&agg.PeriodStart,
			&agg.PeriodEnd,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly usage aggregate: %w", err)
		}
		
		aggregates = append(aggregates, agg)
	}
	
	return aggregates, nil
}

func (r *UsageAggregationRepository) GetRealtimeUsage(ctx context.Context, orgID, subscriptionItemID string, since time.Time) (entities.UsageSummary, error) {
	tx := r.getTransactionFromContext(ctx)
	
	query := `
		SELECT 
			COALESCE(SUM(total_quantity), 0) as quantity,
			COALESCE(SUM(total_amount), 0) as amount,
			COALESCE(SUM(event_count), 0) as events,
			MAX(last_event_time) as last_usage
		FROM usage_hourly
		WHERE org_id = $1 
		  AND subscription_item_id = $2
		  AND hour >= $3
	`
	
	var summary entities.UsageSummary
	var lastUsage *time.Time
	
	err := tx.QueryRow(ctx, query, orgID, subscriptionItemID, since).Scan(
		&summary.Quantity,
		&summary.Amount,
		&summary.Events,
		&lastUsage,
	)
	
	if err != nil {
		return entities.UsageSummary{}, fmt.Errorf("failed to get realtime usage: %w", err)
	}
	
	summary.LastUsage = lastUsage
	
	return summary, nil
}

func (r *UsageAggregationRepository) GetCustomerUsage(ctx context.Context, orgID, customerID string, startTime, endTime time.Time) (entities.CustomerUsageSummary, error) {
	tx := r.getTransactionFromContext(ctx)
	
	// First query to get summary data
	summaryQuery := `
		SELECT 
			customer_id,
			SUM(calculated_amount) as total_amount,
			COUNT(*) as total_events,
			MIN(time) as first_usage_time,
			MAX(time) as last_usage_time,
			COUNT(DISTINCT DATE(time)) as active_days,
			ARRAY_AGG(DISTINCT subscription_id) as subscription_ids
		FROM usage_events
		WHERE org_id = $1 
		  AND customer_id = $2
		  AND time >= $3 
		  AND time < $4
		GROUP BY customer_id
	`
	
	var summary entities.CustomerUsageSummary
	var subscriptionIDs []string
	
	err := tx.QueryRow(ctx, summaryQuery, orgID, customerID, startTime, endTime).Scan(
		&summary.CustomerID,
		&summary.TotalAmount,
		&summary.TotalEvents,
		&summary.FirstUsageTime,
		&summary.LastUsageTime,
		&summary.ActiveDays,
		&subscriptionIDs,
	)
	
	if err != nil {
		return entities.CustomerUsageSummary{}, fmt.Errorf("failed to get customer usage summary: %w", err)
	}
	
	summary.SubscriptionIDs = subscriptionIDs
	
	// Second query to get usage by type
	typeQuery := `
		SELECT 
			usage_type,
			SUM(quantity) as total_quantity,
			SUM(calculated_amount) as total_amount
		FROM usage_events
		WHERE org_id = $1 
		  AND customer_id = $2
		  AND time >= $3 
		  AND time < $4
		GROUP BY usage_type
	`
	
	rows, err := tx.Query(ctx, typeQuery, orgID, customerID, startTime, endTime)
	if err != nil {
		return entities.CustomerUsageSummary{}, fmt.Errorf("failed to query usage by type: %w", err)
	}
	defer rows.Close()
	
	summary.UsageByType = make(map[string]float64)
	summary.AmountByType = make(map[string]int64)
	
	for rows.Next() {
		var usageType string
		var quantity float64
		var amount int64
		
		err := rows.Scan(&usageType, &quantity, &amount)
		if err != nil {
			return entities.CustomerUsageSummary{}, fmt.Errorf("failed to scan usage by type: %w", err)
		}
		
		summary.UsageByType[usageType] = quantity
		summary.AmountByType[usageType] = amount
	}
	
	return summary, nil
}

func (r *UsageAggregationRepository) GetUsageTypeAnalytics(ctx context.Context, orgID string, startTime, endTime time.Time) ([]entities.UsageTypeAnalytics, error) {
	tx := r.getTransactionFromContext(ctx)
	
	query := `
		WITH usage_stats AS (
			SELECT 
				usage_type,
				SUM(quantity) as total_quantity,
				SUM(calculated_amount) as total_amount,
				COUNT(*) as total_events,
				COUNT(DISTINCT customer_id) as unique_customers,
				AVG(quantity) as daily_average,
				MAX(quantity) as peak_usage,
				(SELECT time FROM usage_events ue2 
				 WHERE ue2.org_id = ue.org_id AND ue2.usage_type = ue.usage_type 
				 ORDER BY quantity DESC LIMIT 1) as peak_time
			FROM usage_events ue
			WHERE org_id = $1 
			  AND time >= $2 
			  AND time < $3
			GROUP BY org_id, usage_type
		)
		SELECT 
			usage_type,
			total_quantity,
			total_amount,
			total_events,
			unique_customers,
			daily_average,
			peak_usage,
			peak_time
		FROM usage_stats
		ORDER BY total_amount DESC
	`
	
	rows, err := tx.Query(ctx, query, orgID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage type analytics: %w", err)
	}
	defer rows.Close()
	
	var analytics []entities.UsageTypeAnalytics
	for rows.Next() {
		var a entities.UsageTypeAnalytics
		
		err := rows.Scan(
			&a.UsageType,
			&a.TotalQuantity,
			&a.TotalAmount,
			&a.TotalEvents,
			&a.UniqueCustomers,
			&a.DailyAverage,
			&a.PeakUsage,
			&a.PeakTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan usage type analytics: %w", err)
		}
		
		analytics = append(analytics, a)
	}
	
	return analytics, nil
}

func (r *UsageAggregationRepository) RefreshAggregates(ctx context.Context) error {
	tx := r.getTransactionFromContext(ctx)
	
	query := `SELECT refresh_usage_aggregates()`
	
	_, err := tx.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to refresh usage aggregates: %w", err)
	}
	
	return nil
}