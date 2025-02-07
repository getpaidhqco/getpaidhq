package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type SubscriptionRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewSubscriptionRepository(database lib.Database, logger lib.Logger) repositories.SubscriptionRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SubscriptionRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r SubscriptionRepository) FindById(ctx context.Context, orgId string, id string) (entities.Subscription, error) {
	var subscription entities.Subscription
	query := `SELECT org_id, id, order_id, customer_id, status, payment_method_id, start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor, trial_ends_at, cancel_at, ends_at, last_charge, renews_at, retries, next_retry, currency, amount, metadata, cycles_processed, total_revenue, cancelled_at, created_at, updated_at
			  FROM subscriptions
			  WHERE org_id = @org_id AND id = @id;`
	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&subscription.OrgId,
		&subscription.Id,
		&subscription.OrderId,
		&subscription.CustomerId,
		&subscription.Status,
		&subscription.PaymentMethodId,
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
		r.logger.Error(`failed to find Subscription by id`, err.Error())
		return entities.Subscription{}, err
	}
	return subscription, nil
}

func (r SubscriptionRepository) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]entities.Subscription, error) {
	var subscriptions []entities.Subscription
	query := `SELECT  org_id,
				  id,
				  order_id,
				  customer_id,
				  status,
				  payment_method_id,
				  start_date,
				  end_date,
				  billing_interval,
				  billing_interval_qty,
				  cycles,
				  billing_anchor,
				  trial_ends_at,
				  cancel_at,
				  ends_at,
				  last_charge,
				  renews_at,
				  retries,
				  next_retry,
				  currency,
				  amount,
				  metadata,
				  cycles_processed,
				  total_revenue,
				  cancelled_at,
				  created_at,
				  updated_at
				FROM subscriptions
				WHERE org_id = @org_id AND order_id = @order_id;`
	rows, err := r.Pool.Query(ctx,
		query,
		pgx.NamedArgs{
			"org_id":   orgId,
			"order_id": orderId,
		})
	if err != nil {
		r.logger.Error(`failed to find Subscriptions`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var subscription entities.Subscription

		err := rows.Scan(
			&subscription.OrgId,
			&subscription.Id,
			&subscription.OrderId,
			&subscription.CustomerId,
			&subscription.Status,
			&subscription.PaymentMethodId,
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
			r.logger.Error(`failed to scan Subscription`, err.Error())
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return subscriptions, nil
}

func (r SubscriptionRepository) Create(ctx context.Context, entity entities.Subscription) (entities.Subscription, error) {

	var subscription entities.Subscription
	query := `INSERT INTO subscriptions (org_id, id, order_id, customer_id, status, start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor, trial_ends_at, cancel_at, ends_at, last_charge, renews_at, retries, next_retry, currency, amount, metadata, cycles_processed, total_revenue, cancelled_at, created_at, updated_at) 
			  VALUES (@org_id, @id, @order_id,@customer_id, @status, @start_date, @end_date, @billing_interval, @billing_interval_qty, @cycles, @billing_anchor, @trial_ends_at, @cancel_at, @ends_at, @last_charge, @renews_at, @retries, @next_retry, @currency, @amount, @metadata, @cycles_processed, @total_revenue, @cancelled_at, NOW(), NOW())
			  RETURNING org_id, id, customer_id, status, start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor, trial_ends_at, cancel_at, ends_at, last_charge, renews_at, retries, next_retry, currency, amount, metadata, cycles_processed, total_revenue, cancelled_at, created_at, updated_at`

	metaJson, _ := json.Marshal(entity.Metadata)

	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"order_id":             entity.OrderId,
		"customer_id":          entity.CustomerId,
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
		&subscription.OrgId,
		&subscription.Id,
		&subscription.CustomerId,
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

	subscription.OrderId = entity.OrderId

	return subscription, nil
}

func (r SubscriptionRepository) Update(ctx context.Context, entity entities.Subscription) (entities.Subscription, error) {

	query := `UPDATE subscriptions
			  SET status=@status, payment_method_id=@payment_method_id, start_date=@start_date, end_date=@end_date, billing_interval=@billing_interval, billing_interval_qty=@billing_interval_qty, cycles=@cycles, billing_anchor=@billing_anchor, trial_ends_at=@trial_ends_at, cancel_at=@cancel_at, ends_at=@ends_at, last_charge=@last_charge, renews_at=@renews_at, retries=@retries, next_retry=@next_retry, currency=@currency, amount=@amount, metadata=@metadata, cycles_processed=@cycles_processed, total_revenue=@total_revenue, cancelled_at=@cancelled_at, updated_at=NOW()
			  WHERE org_id=@org_id AND id=@id
			  RETURNING org_id, id, order_id, customer_id, payment_method_id, status, start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor, trial_ends_at, cancel_at, ends_at, last_charge, renews_at, retries, next_retry, currency, amount, metadata, cycles_processed, total_revenue, cancelled_at, created_at, updated_at`

	metaJson, _ := json.Marshal(entity.Metadata)

	var subscription entities.Subscription
	err := r.Pool.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"status":               entity.Status,
		"start_date":           entity.StartDate,
		"payment_method_id":    entity.PaymentMethodId,
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
		&subscription.OrgId,
		&subscription.Id,
		&subscription.OrderId,
		&subscription.CustomerId,
		&subscription.PaymentMethodId,
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
		r.logger.Error(`failed to update Subscription`, err.Error())
		return entities.Subscription{}, err
	}

	return subscription, nil
}
