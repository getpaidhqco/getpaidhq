package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type SubscriptionRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewSubscriptionRepository(primaryDb lib.Database, logger port.Logger) port.SubscriptionRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return SubscriptionRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r SubscriptionRepository) FindById(ctx context.Context, orgId string, id string) (domain.Subscription, error) {
	tx := r.getTransactionFromContext(ctx)

	var subscription models.Subscription
	var customer models.Customer
	query := `SELECT s.org_id, s.id, s.psp_id, s.order_id, s.order_item_id, s.customer_id, s.status, s.payment_method_id, s.start_date, s.end_date,
       s.billing_interval, s.billing_interval_qty, s.cycles, s.billing_anchor, s.trial_ends_at, s.cancel_at, s.ends_at,
       s.last_charge,
       s.renews_at,
       s.current_period_start,
       s.current_period_end,
       s.retries, s.next_retry, s.currency, s.amount, s.metadata, s.cycles_processed,
       s.total_revenue, s.cancelled_at, s.created_at, s.updated_at,
       c.org_id, c.id, c.first_name, c.last_name, c.email, c.created_at, c.updated_at
   FROM subscriptions s
   JOIN customers c ON s.org_id=c.org_id AND s.customer_id = c.id
   WHERE s.org_id = @org_id AND s.id = @id;`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&subscription.OrgId,
		&subscription.Id,
		&subscription.PspId,
		&subscription.OrderId,
		&subscription.OrderItemId,
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
		&subscription.CurrentPeriodStart,
		&subscription.CurrentPeriodEnd,
		&subscription.Retries,
		&subscription.NextRetryAt,
		&subscription.Currency,
		&subscription.Amount,
		&subscription.Metadata,
		&subscription.CyclesProcessed,
		&subscription.TotalRevenue,
		&subscription.CancelledAt,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,

		&customer.OrgId,
		&customer.Id,
		&customer.FirstName,
		&customer.LastName,
		&customer.Email,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		r.logger.Error(`failed to find Subscription by id`, err.Error())
		return domain.Subscription{}, err
	}
	subscription.Customer = customer
	return subscription.ToEntity(), nil
}

func (r SubscriptionRepository) FindByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.Subscription, error) {
	tx := r.getTransactionFromContext(ctx)
	var subscriptions = make([]domain.Subscription, 0)
	query := `SELECT s.org_id, s.id, s.psp_id, s.order_id, s.order_item_id, s.customer_id,
       s.status, s.payment_method_id, s.start_date, s.end_date,
       s.billing_interval, s.billing_interval_qty, s.cycles, s.billing_anchor, s.trial_ends_at, s.cancel_at, s.ends_at,
       s.last_charge, s.renews_at,
       s.current_period_start,
       s.current_period_end, s.retries, s.next_retry, s.currency, s.amount, s.metadata, s.cycles_processed,
       s.total_revenue, s.cancelled_at, s.created_at, s.updated_at,

       oi.org_id, oi.id, oi.price_id, oi.quantity, oi.description,oi.created_at, oi.updated_at
			FROM subscriptions s
			JOIN order_items oi ON s.org_id = oi.org_id AND s.order_id = oi.order_id
			WHERE s.org_id = @org_id AND s.order_id = @order_id;`
	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"order_id": orderId,
	})
	if err != nil {
		r.logger.Error(`failed to find Subscriptions by order id`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var subscription models.Subscription
		var orderItem models.OrderItem
		err := rows.Scan(
			&subscription.OrgId,
			&subscription.Id,
			&subscription.PspId,
			&subscription.OrderId,
			&subscription.OrderItemId,
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
			&subscription.CurrentPeriodStart,
			&subscription.CurrentPeriodEnd,
			&subscription.Retries,
			&subscription.NextRetryAt,
			&subscription.Currency,
			&subscription.Amount,
			&subscription.Metadata,
			&subscription.CyclesProcessed,
			&subscription.TotalRevenue,
			&subscription.CancelledAt,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,

			&orderItem.OrgId,
			&orderItem.Id,
			&orderItem.PriceId,
			&orderItem.Quantity,
			&orderItem.Description,
			&orderItem.CreatedAt,
			&orderItem.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan Subscription`, "err", err.Error())
			return nil, err
		}
		subscription.OrderItem = orderItem
		subscriptions = append(subscriptions, subscription.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return subscriptions, nil
}

func (r SubscriptionRepository) Create(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO subscriptions (org_id, id, psp_id, payment_method_id, order_id, order_item_id, customer_id, status,
                           start_date, end_date, billing_interval, billing_interval_qty, cycles, billing_anchor,
                           trial_ends_at, cancel_at, ends_at, last_charge, renews_at,
                           current_period_start, current_period_end, retries, next_retry,
                           currency, amount, metadata, cycles_processed, total_revenue, cancelled_at,
                           created_at, updated_at)
			  VALUES (@org_id, @id, @psp_id, @payment_method_id, @order_id, @order_item_id, @customer_id, @status,
			          @start_date, @end_date, @billing_interval, @billing_interval_qty, @cycles, @billing_anchor,
			          @trial_ends_at, @cancel_at, @ends_at, @last_charge, @renews_at,
			          @current_period_start, @current_period_end, @retries, @next_retry,
			          @currency, @amount, @metadata, @cycles_processed, @total_revenue, @cancelled_at,
			          NOW(), NOW())
`
	args := entityToNamedArgs(entity)
	_, err := tx.Exec(ctx, query, args)

	if err != nil {
		r.logger.Error(`failed to insert Subscription`, err.Error())
		return domain.Subscription{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r SubscriptionRepository) Update(ctx context.Context, entity domain.Subscription) (domain.Subscription, error) {
	tx := r.getTransactionFromContext(ctx)
	query := `UPDATE subscriptions
			  SET status=@status,
			      payment_method_id=@payment_method_id,
			      start_date=@start_date, end_date=@end_date,
			      billing_interval=@billing_interval,
			      billing_interval_qty=@billing_interval_qty,
			      cycles=@cycles,
			      billing_anchor=@billing_anchor,
			      trial_ends_at=@trial_ends_at,
			      cancel_at=@cancel_at,
			      ends_at=@ends_at,
			      last_charge=@last_charge,
			      renews_at=@renews_at,
			      current_period_start=@current_period_start,
			      current_period_end=@current_period_end,
			      retries=@retries,
			      next_retry=@next_retry,
			      currency=@currency,
			      amount=@amount,
			      metadata=@metadata,
			      cycles_processed=@cycles_processed,
			      total_revenue=@total_revenue,
			      cancelled_at=@cancelled_at,
			      updated_at=NOW()
			  WHERE org_id=@org_id AND id=@id
`
	args := entityToNamedArgs(entity)
	_, err := tx.Exec(ctx, query, args)

	if err != nil {
		r.logger.Error(`failed to update Subscription`, "err", err.Error())
		return domain.Subscription{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r SubscriptionRepository) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Subscription, int, error) {
	tx := r.getTransactionFromContext(ctx)
	r.logger.Debugf("sort_dir[%s] sort_col[%s]", p.SortDirection, p.SortBy)

	var subscriptions = make([]domain.Subscription, 0)
	var count int
	query := `SELECT s.org_id, s.id, s.order_id, s.order_item_id, s.customer_id, s.status, s.payment_method_id, s.start_date, s.end_date,
       s.billing_interval, s.billing_interval_qty, s.cycles, s.billing_anchor, s.trial_ends_at, s.cancel_at, s.ends_at,
       s.last_charge, s.renews_at, s.retries, s.next_retry, s.currency, s.amount, s.metadata, s.cycles_processed,
       s.total_revenue, s.cancelled_at, s.created_at, s.updated_at,
       c.org_id, c.id, c.first_name, c.email, c.created_at, c.updated_at,
       count(*) OVER()
   FROM subscriptions s
   JOIN customers c ON s.org_id=c.org_id AND s.customer_id = c.id
			  WHERE s.org_id = @org_id
	ORDER BY
    -- Simplified to NULL if not sorting in ascending order.
    CASE
        WHEN @sort_dir = 'asc' THEN
            CASE @sort_col
                -- Check for each possible value of sort_col.
                WHEN 'created_at' THEN s.created_at
                --- etc.
                ELSE NULL
                END
        ELSE
            NULL
        END
        ASC ,

    -- Same as before, but for sort_dir = 'desc'
    CASE WHEN @sort_dir = 'desc' THEN
             CASE @sort_col
                 WHEN 'created_at' THEN s.created_at
                 ELSE NULL
                 END
         ELSE
             NULL
        END
        DESC
	LIMIT @lim OFFSET @off;`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Subscriptions`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var subscription models.Subscription
		var customer models.Customer

		err := rows.Scan(
			&subscription.OrgId,
			&subscription.Id,
			&subscription.OrderId,
			&subscription.OrderItemId,
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
			&subscription.NextRetryAt,
			&subscription.Currency,
			&subscription.Amount,
			&subscription.Metadata,
			&subscription.CyclesProcessed,
			&subscription.TotalRevenue,
			&subscription.CancelledAt,
			&subscription.CreatedAt,
			&subscription.UpdatedAt,

			&customer.OrgId,
			&customer.Id,
			&customer.FirstName,
			&customer.Email,
			&customer.CreatedAt,
			&customer.UpdatedAt,

			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan Subscription`, err.Error())
			return nil, 0, err
		}
		subscription.Customer = customer
		subscriptions = append(subscriptions, subscription.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return subscriptions, count, nil
}

func entityToNamedArgs(entity domain.Subscription) pgx.NamedArgs {
	metaJson, _ := json.Marshal(entity.Metadata)
	return pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"payment_method_id":    pgtype.Text{String: entity.PaymentMethodId, Valid: entity.PaymentMethodId != ""},
		"psp_id":               entity.PspId,
		"order_id":             entity.OrderId,
		"order_item_id":        entity.OrderItemId,
		"customer_id":          entity.CustomerId,
		"status":               entity.Status,
		"start_date":           pgtype.Date{Time: entity.StartDate, Valid: !entity.StartDate.IsZero()},
		"end_date":             pgtype.Date{Time: entity.EndDate, Valid: !entity.EndDate.IsZero()},
		"billing_interval":     entity.BillingInterval,
		"billing_interval_qty": entity.BillingIntervalQty,
		"cycles":               entity.Cycles,
		"billing_anchor":       entity.BillingAnchor,
		"trial_ends_at":        pgtype.Date{Time: entity.TrialEndsAt, Valid: !entity.TrialEndsAt.IsZero()},
		"cancel_at":            pgtype.Date{Time: entity.CancelAt, Valid: !entity.CancelAt.IsZero()},
		"ends_at":              pgtype.Date{Time: entity.EndsAt, Valid: !entity.EndsAt.IsZero()},
		"last_charge":          pgtype.Date{Time: entity.LastCharge, Valid: !entity.LastCharge.IsZero()},
		"renews_at":            pgtype.Date{Time: entity.RenewsAt, Valid: !entity.RenewsAt.IsZero()},
		"current_period_start": pgtype.Date{Time: entity.CurrentPeriodStart, Valid: !entity.CurrentPeriodStart.IsZero()},
		"current_period_end":   pgtype.Date{Time: entity.CurrentPeriodEnd, Valid: !entity.CurrentPeriodEnd.IsZero()},
		"retries":              entity.Retries,
		"next_retry":           pgtype.Date{Time: entity.NextRetryAt, Valid: !entity.NextRetryAt.IsZero()},
		"currency":             entity.Currency,
		"amount":               entity.Amount,
		"metadata":             metaJson,
		"cycles_processed":     entity.CyclesProcessed,
		"total_revenue":        entity.TotalRevenue,
		"cancelled_at":         pgtype.Date{Time: entity.CancelledAt, Valid: !entity.CancelledAt.IsZero()},
	}
}
