package postgres

import (
	"context"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/values"
	"payloop/internal/lib"
	"time"
)

type ReportRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewReportRepository(reportingDb lib.Database, logger logger.Logger) repositories.ReportRepository {
	pgDatabase, ok := reportingDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return ReportRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r ReportRepository) GetMRR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	tx := r.getTransactionFromContext(ctx)

	mrr := make([]values.RecurringRevenue, 0)
	query := `
		SELECT DATE_TRUNC('month', completed_at) date, 
		       SUM(amount),
		       'mrr'
		FROM payments 
		WHERE org_id = $1 AND status = 'succeeded' AND recurring = true 
		and completed_at between $2 and $3
		GROUP BY DATE_TRUNC('month', completed_at)
	`

	rows, err := tx.Query(ctx, query, orgId, startDate, endDate)
	if err != nil {
		r.logger.Error("failed to execute query", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var revenue values.RecurringRevenue
		if err := rows.Scan(
			&revenue.Period,
			&revenue.Total,
			&revenue.Type,
		); err != nil {
			r.logger.Error("failed to scan row", err)
			return nil, err
		}
		mrr = append(mrr, revenue)
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return mrr, nil
}

func (r ReportRepository) GetARR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	tx := r.getTransactionFromContext(ctx)

	mrr := make([]values.RecurringRevenue, 0)
	query := `
		SELECT DATE_TRUNC('month', completed_at) month, 
		       DATE_TRUNC('month', completed_at) month, 
		       SUM(amount), 
		'mrr'
		FROM payments 
		WHERE org_id = $1 AND status = 'completed' AND recurring = true 
		GROUP BY DATE_TRUNC('month', completed_at)
	`

	rows, err := tx.Query(ctx, query, orgId)
	if err != nil {
		r.logger.Error("failed to execute query", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var revenue values.RecurringRevenue
		if err := rows.Scan(
			&revenue.Period,
			&revenue.Total,
		); err != nil {
			r.logger.Error("failed to scan row", err)
			return nil, err
		}
		mrr = append(mrr, revenue)
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return mrr, nil
}

func (r ReportRepository) GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {
	tx := r.getTransactionFromContext(ctx)

	mrr := make([]values.RecurringRevenue, 0)
	query := `
		SELECT DATE_TRUNC('month', completed_at) month, 
		       SUM(amount), 
		'mrr'
		FROM payments 
		WHERE org_id = $1 AND status = 'completed' AND recurring = true 
		GROUP BY DATE_TRUNC('month', completed_at)
	`

	rows, err := tx.Query(ctx, query, orgId)
	if err != nil {
		r.logger.Error("failed to execute query", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var revenue values.RecurringRevenue
		if err := rows.Scan(
			&revenue.Period,
			&revenue.Total,
		); err != nil {
			r.logger.Error("failed to scan row", err)
			return nil, err
		}
		mrr = append(mrr, revenue)
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return mrr, nil
}

func (r ReportRepository) StoreDailyMetrics(ctx context.Context, orgId string, date time.Time) ([]values.RecurringRevenue, error) {
	tx := r.getTransactionFromContext(ctx)

	mrr := make([]values.RecurringRevenue, 0)
	query := `
		SELECT DATE_TRUNC('month', completed_at) month, 
		       SUM(amount), 
		'mrr'
		FROM payments 
		WHERE org_id = $1 AND status = 'completed' AND recurring = true 
		GROUP BY DATE_TRUNC('month', completed_at)
	`

	rows, err := tx.Query(ctx, query, orgId)
	if err != nil {
		r.logger.Error("failed to execute query", err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var revenue values.RecurringRevenue
		if err := rows.Scan(
			&revenue.Period,
			&revenue.Total,
		); err != nil {
			r.logger.Error("failed to scan row", err)
			return nil, err
		}
		mrr = append(mrr, revenue)
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return mrr, nil
}
