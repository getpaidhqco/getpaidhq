package repository

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/lib"
)

type SubscriptionRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewSubscriptionRepository(database lib.Database, logger lib.Logger) SubscriptionRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SubscriptionRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r *SubscriptionRepository) WithTrx(trxHandle interface{}) *SubscriptionRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r *SubscriptionRepository) FindById(ctx context.Context, acctId string, id string) (entities.Subscription, error) {
	var subscription entities.Subscription
	err := r.Pool.QueryRow(ctx,
		`SELECT * FROM subscriptions WHERE acct_id=@acct_id AND id=@id`,
		pgx.NamedArgs{
			"acct_id": acctId,
			"id":      id,
		}).Scan(
		&subscription.AccountId,
		&subscription.Id,
		&subscription.OrderId,
		&subscription.Status,
		&subscription.StartDate,
		&subscription.EndDate,
		&subscription.BillingInterval,
		&subscription.BillingIntervalQty,
		&subscription.Cycles,
		&subscription.BillingAnchor,
		&subscription.TrialEndsAt,
		&subscription.CancelAt,
		&subscription.EndsAt,
		&subscription.LastCharge,
		&subscription.RenewsAt,
		&subscription.Retries,
		&subscription.NextRetry,
		&subscription.Currency,
		&subscription.Amount,
		&subscription.Metadata,
		&subscription.CyclesProcessed,
		&subscription.TotalRevenue,
		&subscription.CancelledAt,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find Subscription`, err.Error())
		return entities.Subscription{}, errors.New("not found")
	}
	return subscription, nil
}

func (r *SubscriptionRepository) Create(ctx context.Context, entity entities.Subscription) (entities.Subscription, error) {

	var subscription entities.Subscription

	query := `INSERT INTO subscriptions (acct_id, id, order_id, status, start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor, trial_ends_at, cancel_at, ends_at, last_charge, renews_at, retries, next_retry, currency, amount, metadata, cycles_processed, total_revenue, cancelled_at, created_at, updated_at) 
			  VALUES (@acct_id, @id, @order_id, @status, @start_date, @end_date, @billing_interval, @billing_interval_qty, @cycles, @billing_anchor, @trial_ends_at, @cancel_at, @ends_at, @last_charge, @renews_at, @retries, @next_retry, @currency, @amount, @metadata, @cycles_processed, @total_revenue, @cancelled_at, NOW(), NOW())
			  RETURNING acct_id, id, order_id, status, start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor, trial_ends_at, cancel_at, ends_at, last_charge, renews_at, retries, next_retry, currency, amount, metadata, cycles_processed, total_revenue, cancelled_at, created_at, updated_at`

	metaJson, _ := json.Marshal(entity.Metadata)

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"acct_id":              entity.AccountId,
		"id":                   entity.Id,
		"order_id":             entity.OrderId,
		"status":               entity.Status,
		"start_date":           entity.StartDate,
		"end_date":             entity.EndDate,
		"billing_interval":     entity.BillingInterval,
		"billing_interval_qty": entity.BillingIntervalQty,
		"cycles":               entity.Cycles,
		"billing_anchor":       entity.BillingAnchor,
		"trial_ends_at":        entity.TrialEndsAt,
		"cancel_at":            entity.CancelAt,
		"ends_at":              entity.EndsAt,
		"last_charge":          entity.LastCharge,
		"renews_at":            entity.RenewsAt,
		"retries":              entity.Retries,
		"next_retry":           entity.NextRetry,
		"currency":             entity.Currency,
		"amount":               entity.Amount,
		"metadata":             metaJson,
		"cycles_processed":     entity.CyclesProcessed,
		"total_revenue":        entity.TotalRevenue,
		"cancelled_at":         entity.CancelledAt,
	}).Scan(
		&subscription.AccountId,
		&subscription.Id,
		&subscription.OrderId,
		&subscription.Status,
		&subscription.StartDate,
		&subscription.EndDate,
		&subscription.BillingInterval,
		&subscription.BillingIntervalQty,
		&subscription.Cycles,
		&subscription.BillingAnchor,
		&subscription.TrialEndsAt,
		&subscription.CancelAt,
		&subscription.EndsAt,
		&subscription.LastCharge,
		&subscription.RenewsAt,
		&subscription.Retries,
		&subscription.NextRetry,
		&subscription.Currency,
		&subscription.Amount,
		&subscription.Metadata,
		&subscription.CyclesProcessed,
		&subscription.TotalRevenue,
		&subscription.CancelledAt,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to insert Subscription`, err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}
