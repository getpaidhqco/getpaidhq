package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"math"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/domain/values"
	"payloop/internal/lib"
	"time"
)

type ReportRepository struct {
	*PgDatabase
	primaryDb *PgDatabase
	logger    logger.Logger
}

func NewReportRepository(
	reportingDb lib.Database,
	primaryDb lib.Database,
	logger logger.Logger,
) repositories.ReportRepository {
	pgDatabase, ok := reportingDb.(*PgDatabase)
	if !ok {
		panic("reportingDb is not of type *tx.PgDatabase")
	}
	p, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("primaryDb is not of type *tx.PgDatabase")
	}
	return ReportRepository{
		PgDatabase: pgDatabase,
		primaryDb:  p,
		logger:     logger,
	}
}

func (r ReportRepository) UpsertSubscription(ctx context.Context, entity entities.Subscription) error {
	tx := r.getTransactionFromContext(ctx)

	r.logger.Debugf("upserting subscription for %s %s", entity.Id, entity.Status)

	query := `INSERT INTO subscriptions (org_id, id, psp_id, status, order_id,
                           order_item_id, order_item_name, customer_id, payment_method_id, payment_method_type,
                           start_date,end_date,billing_interval,billing_interval_qty,cycles,
                           billing_anchor,trial_ends_at,cancel_at,ends_at,last_charge,renews_at, 
                           current_period_start,current_period_end,retries,next_retry,currency,amount,
                           cycles_processed,total_revenue,cancelled_at,created_at,updated_at)
			  VALUES (@org_id, @id, @psp_id, @status, @order_id, @order_item_id, @order_item_name, @customer_id, @payment_method_id, @payment_method_type,
                      @start_date, @end_date, @billing_interval, @billing_interval_qty, @cycles, @billing_anchor, @trial_ends_at, @cancel_at, @ends_at, @last_charge, @renews_at,
                      @current_period_start, @current_period_end, @retries, @next_retry, @currency, @amount, @cycles_processed, @total_revenue, @cancelled_at, NOW(), NOW())
				ON CONFLICT (org_id, id) DO UPDATE SET
					psp_id = EXCLUDED.psp_id,
					status = EXCLUDED.status,
					order_id = EXCLUDED.order_id,
					order_item_id = EXCLUDED.order_item_id,
					order_item_name = EXCLUDED.order_item_name,
					customer_id = EXCLUDED.customer_id,
					payment_method_id = EXCLUDED.payment_method_id,
					payment_method_type = EXCLUDED.payment_method_type,
					start_date = EXCLUDED.start_date,
					end_date = EXCLUDED.end_date,
					billing_interval = EXCLUDED.billing_interval,
					billing_interval_qty = EXCLUDED.billing_interval_qty,
					cycles = EXCLUDED.cycles,
					billing_anchor = EXCLUDED.billing_anchor,
					trial_ends_at = EXCLUDED.trial_ends_at,
					cancel_at = EXCLUDED.cancel_at,
					ends_at = EXCLUDED.ends_at,
					last_charge = EXCLUDED.last_charge,
					renews_at = EXCLUDED.renews_at,
					current_period_start = EXCLUDED.current_period_start,
					current_period_end = EXCLUDED.current_period_end,
					retries = EXCLUDED.retries,
					next_retry = EXCLUDED.next_retry,
					currency = EXCLUDED.currency,
					amount = EXCLUDED.amount,
					cycles_processed = EXCLUDED.cycles_processed,
					total_revenue = EXCLUDED.total_revenue,
					cancelled_at = EXCLUDED.cancelled_at,
					updated_at = NOW()
`

	args := pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"psp_id":               entity.PspId,
		"status":               entity.Status,
		"order_id":             entity.OrderId,
		"order_item_id":        entity.OrderItemId,
		"order_item_name":      entity.OrderItemId,
		"customer_id":          entity.CustomerId,
		"payment_method_id":    pgtype.Text{String: entity.PaymentMethodId, Valid: entity.PaymentMethodId != ""},
		"payment_method_type":  pgtype.Text{String: entity.PaymentMethodId, Valid: entity.PaymentMethodId != ""},
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
		"cycles_processed":     entity.CyclesProcessed,
		"total_revenue":        entity.TotalRevenue,
		"cancelled_at":         pgtype.Date{Time: entity.CancelledAt, Valid: !entity.CancelledAt.IsZero()},
	}
	_, err := tx.Exec(ctx, query, args)
	if err != nil {
		r.logger.Errorf(`failed to upsert Subscription %s`, err.Error())
		return err
	}

	return nil
}

func (r ReportRepository) UpsertPayment(ctx context.Context, entity entities.Payment) error {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO payments (org_id, id, psp, psp_id, reference, recurring, order_id,
                      subscription_id, amount, currency, status, 
                      psp_fee, platform_fee, net_amount, completed_at, created_at, updated_at)
			  VALUES (@org_id, @id, @psp, @psp_id, @reference, @recurring, @order_id,
			          @subscription_id, @amount, @currency, @status,
			          @psp_fee, @platform_fee, @net_amount, @completed_at, NOW(), NOW())
			  ON CONFLICT (org_id, id) DO UPDATE SET
				psp = EXCLUDED.psp,
				psp_id = EXCLUDED.psp_id,
				reference = EXCLUDED.reference,
				recurring = EXCLUDED.recurring,
				order_id = EXCLUDED.order_id,
				subscription_id = EXCLUDED.subscription_id,
				amount = EXCLUDED.amount,
				currency = EXCLUDED.currency,
				status = EXCLUDED.status,
				psp_fee = EXCLUDED.psp_fee,
				platform_fee = EXCLUDED.platform_fee,
				net_amount = EXCLUDED.net_amount,
				completed_at = EXCLUDED.completed_at,
				updated_at = EXCLUDED.updated_at`

	args := pgx.NamedArgs{"org_id": entity.OrgId,
		"id":              entity.Id,
		"psp":             entity.Psp,
		"psp_id":          entity.PspId,
		"reference":       entity.Reference,
		"recurring":       entity.Recurring,
		"order_id":        entity.OrderId,
		"subscription_id": entity.SubscriptionId,
		"amount":          entity.Amount,
		"currency":        entity.Currency,
		"status":          entity.Status,
		"psp_fee":         entity.PspFee,
		"platform_fee":    entity.PlatformFee,
		"net_amount":      entity.NetAmount,
		"completed_at":    pgtype.Date{Time: entity.CompletedAt, Valid: !entity.CompletedAt.IsZero()},
	}

	_, err := tx.Exec(ctx, query, args)
	if err != nil {
		r.logger.Errorf(`failed to insert Payment %s`, err.Error())
		return err
	}

	return nil
}

func (r ReportRepository) UpsertCustomer(ctx context.Context, entity entities.Customer) error {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO customers (org_id, id, created_at, updated_at)
			  VALUES (@org_id, @id, NOW(), NOW())
			  ON CONFLICT (org_id, id) DO UPDATE SET
				updated_at = NOW()`

	args := pgx.NamedArgs{
		"org_id": entity.OrgId,
		"id":     entity.Id,
	}

	_, err := tx.Exec(ctx, query, args)
	if err != nil {
		r.logger.Errorf(`failed to insert Customer %s`, err.Error())
		return err
	}

	return nil
}

func (r ReportRepository) UpsertCustomerCohort(ctx context.Context, entity entities.CustomerCohort) error {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO customer_cohorts (org_id, customer_id, cohort_id, cohort_value, joined_at, created_at, updated_at)
			  VALUES (@org_id, @customer_id, @cohort_id, @cohort_value, @joined_at, NOW(), NOW())
			  ON CONFLICT (org_id, customer_id, cohort_id) DO UPDATE SET
				cohort_value = EXCLUDED.cohort_value,
				joined_at = EXCLUDED.joined_at,
				updated_at = NOW()`

	args := pgx.NamedArgs{
		"org_id":       entity.OrgId,
		"customer_id":  entity.CustomerId,
		"cohort_id":    entity.CohortId,
		"cohort_value": entity.CohortValue,
		"joined_at":    pgtype.Date{Time: entity.JoinedAt, Valid: !entity.JoinedAt.IsZero()},
	}

	_, err := tx.Exec(ctx, query, args)
	if err != nil {
		r.logger.Errorf(`failed to upsert CustomerCohort %s`, err.Error())
		return err
	}

	return nil
}

// GetMRR returns the Monthly Recurring Revenue (MRR) for a given organization and date range. It queries the
// daily_metrics table to calculate the MRR by summing the mrr values for each month within the specified date range.
func (r ReportRepository) GetMRR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {

	mrr := make([]values.RecurringRevenue, 0)
	query := `
		SELECT DATE_TRUNC('month', date) m, 
		       SUM(mrr) monthly_mrr,
		       'mrr'
		FROM daily_metrics 
		WHERE org_id = $1
		and date::date between $2::date and $3::date
		GROUP BY m
	`

	rows, err := r.Pool.Query(ctx, query, orgId, startDate, endDate)
	if err != nil {
		r.logger.Error("failed to execute query", "err", err)
		return nil, err
	}
	defer rows.Close()

	index := 0
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

		if index > 0 {
			revenue.GrowthMoM = ((revenue.Total - mrr[index-1].Total) / mrr[index-1].Total) * 100
		} else {
			revenue.GrowthMoM = 0 // No growth for the first month
		}
		mrr = append(mrr, revenue)
		index++
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return mrr, nil
}

func (r ReportRepository) GetARR(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {

	arr := make([]values.RecurringRevenue, 0)
	// Query to calculate the ARR
	query := `
        WITH daily_mrr AS (
            SELECT 
				org_id,
                date,
                mrr
            FROM 
                daily_metrics
        )
        SELECT 
            DATE_TRUNC('year', date) AS year,
            SUM(mrr) AS annual_recurring_revenue
        FROM 
            daily_mrr
		WHERE 
   			org_id = $1 AND
 			date BETWEEN $2::date AND $3::date
        GROUP BY 
            DATE_TRUNC('year', date)
        ORDER BY 
            year;
    `

	rows, err := r.Pool.Query(ctx, query, orgId, startDate, endDate)
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
		revenue.Type = "arr"
		arr = append(arr, revenue)
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return arr, nil
}

func (r ReportRepository) GetActiveSubscribers(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {

	activeSubs := make([]values.RecurringRevenue, 0)
	query := `
			SELECT 
				DATE_TRUNC('month', date) AS week_start,
				AVG(customer_count) AS weekly_avg_customer_count
			FROM 
				daily_metrics
				where org_id=$1
				 AND date::date between $2::date and $3::date
			GROUP BY 
				DATE_TRUNC('month', date)
			ORDER BY 
				week_start;
	`

	rows, err := r.Pool.Query(ctx, query, orgId, startDate, endDate)
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
		revenue.Type = "customers"
		revenue.Total = math.Round(revenue.Total*100) / 100
		activeSubs = append(activeSubs, revenue)
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return activeSubs, nil
}

func (r ReportRepository) GetRefundTotals(ctx context.Context, orgId string, startDate time.Time, endDate time.Time) ([]values.RecurringRevenue, error) {

	results := make([]values.RecurringRevenue, 0)
	query := `
			SELECT 
				DATE_TRUNC('month', date) AS month_start,
				AVG(refund_total) AS weekly_avg_refunds
			FROM 
				daily_metrics
				where org_id=$1
				 AND date::date between $2::date and $3::date
			GROUP BY 
				DATE_TRUNC('month', date)
			ORDER BY 
				month_start;
	`

	rows, err := r.Pool.Query(ctx, query, orgId, startDate, endDate)
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
		revenue.Type = "refund_totals"
		results = append(results, revenue)
	}

	if rows.Err() != nil {
		r.logger.Error("rows iteration error", rows.Err())
		return nil, rows.Err()
	}

	return results, nil
}

func (r ReportRepository) StoreDailyMetrics(ctx context.Context, orgId string, d time.Time) error {
	tx := r.getTransactionFromContext(ctx)

	date := d.Format("2006-01-02")
	r.logger.Debugf(`Calculating daily metrics for %s`, date)

	// Calculate successful payments (example calculation)
	var successfulPayments int
	paymentQuery := `SELECT COUNT(*)  
					FROM payments 
					WHERE org_id=@org_id 
					  AND status = 'succeeded' 
					  AND completed_at::date = @completed_at::date`
	err := tx.QueryRow(ctx, paymentQuery, pgx.NamedArgs{
		"org_id":       orgId,
		"completed_at": date,
	}).Scan(&successfulPayments)
	if err != nil {
		return err
	}
	r.logger.Debugf(`successful payments		%d`, successfulPayments)

	// Calculate failed payments (example calculation)
	var failedPayments int
	failedPaymentQuery := `SELECT COUNT(*)  
					FROM payments 
					WHERE org_id=@org_id 
					  AND status = 'failed' 
					  AND completed_at::date = @completed_at::date`
	err = tx.QueryRow(ctx, failedPaymentQuery, pgx.NamedArgs{
		"org_id":       orgId,
		"completed_at": date,
	}).Scan(&failedPayments)
	if err != nil {
		return err
	}
	r.logger.Debugf(`failed payments		[%d]`, failedPayments)

	// Calculate refunds
	var refundTotal int64
	var refundCount int64
	refundQuery := `SELECT count(*), COALESCE(SUM(amount), 0) 
					FROM refunds 
					WHERE org_id=@org_id 
					  AND date::date = @date::date `
	err = tx.QueryRow(ctx, refundQuery, pgx.NamedArgs{
		"org_id": orgId,
		"date":   date,
	}).Scan(&refundCount, &refundTotal)
	if err != nil {
		r.logger.Errorf(`refunds %v`, err)
		return err
	}
	r.logger.Debugf(`total refunds		%d`, refundTotal)

	// Calculate MRR
	var mrr int64
	query := `
        SELECT COALESCE(SUM(
            CASE
                WHEN billing_interval = 'month' THEN amount / @numdays
                WHEN billing_interval = 'year' THEN amount / 365
            END
        ), 0) AS daily_mrr
        FROM subscriptions
        WHERE org_id=@org_id
        AND status in ('trial','active','retry')
        AND start_date::date <= @date::date
        AND (cancelled_at::date IS NULL OR cancelled_at::date <> @date::date)
        AND (end_date::date IS NULL OR end_date::date >= @date::date)
    `
	err = tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":  orgId,
		"date":    date,
		"numdays": getDaysInMonth(d),
	}).Scan(&mrr)
	if err != nil {
		r.logger.Errorf(`mrr %v`, err)
		return err
	}
	r.logger.Debugf(`MRR		%d`, mrr)

	// Calculate ARR
	arr := mrr * 365

	// Calculate customer count
	var customerCount int
	customerQuery := `
				select COUNT(*)
				from (
						 select distinct on (customer_id) customer_id
						 from subscriptions
						 where org_id = $1
						   and status in ('active', 'past_due', 'trial')
					 ) as unique_customers`

	err = tx.QueryRow(ctx, customerQuery, orgId).
		Scan(&customerCount)
	if err != nil {
		r.logger.Errorf(`customers %v`, err)
		return err
	}

	// Calculate churn rate (example calculation)
	var churnedCustomers int
	churnQuery := `SELECT COUNT(*) FROM subscriptions 
                WHERE org_id=@org_id
                 AND (end_date::date = @date::date
                     OR cancelled_at::date = @date::date)`

	err = tx.QueryRow(ctx, churnQuery, pgx.NamedArgs{
		"org_id": orgId,
		"date":   date,
	}).Scan(&churnedCustomers)
	if err != nil {
		r.logger.Errorf(`churn %v`, err)
		return err
	}
	churnRate := 0.0
	arpu := 0.0
	if customerCount > 0 {
		churnRate = float64(churnedCustomers) / float64(customerCount) * 100
		arpu = float64(mrr*30) / float64(customerCount)
	}

	// Calculate CLTV
	cltv := arpu * 12

	// Insert daily metrics into the database
	dmQuery := `
				INSERT INTO daily_metrics (org_id,date, currency, mrr, arr, 
                           customer_count, churn_rate, ave_revenue_per_user, 
				           customer_lifetime_value, successful_payments, 
                           failed_payments, refund_total, refund_count) 
				VALUES (@org_id, @date, @currency, @mrr, @arr, 
				        @customer_count, @churn_rate, @arpu, @cltv, @successful_payments,
				        @failed_payments, @refund_total, @refund_count)
				ON CONFLICT (org_id, date) DO UPDATE SET
					currency = EXCLUDED.currency,
					mrr = EXCLUDED.mrr,
					arr = EXCLUDED.arr,
					customer_count = EXCLUDED.customer_count,
					churn_rate = EXCLUDED.churn_rate,
					ave_revenue_per_user = EXCLUDED.ave_revenue_per_user,
					customer_lifetime_value = EXCLUDED.customer_lifetime_value,
					successful_payments = EXCLUDED.successful_payments,
					failed_payments = EXCLUDED.failed_payments,
					refund_total = EXCLUDED.refund_total,
					refund_count = EXCLUDED.refund_count
`
	_, err = tx.Exec(ctx, dmQuery, pgx.NamedArgs{
		"org_id":              orgId,
		"date":                date,
		"currency":            "USD", // Assuming currency is USD, replace with actual value if different
		"mrr":                 mrr,
		"arr":                 arr,
		"customer_count":      customerCount,
		"churn_rate":          churnRate,
		"arpu":                arpu,
		"cltv":                cltv,
		"successful_payments": successfulPayments,
		"failed_payments":     failedPayments,
		"refund_total":        refundTotal,
		"refund_count":        refundCount,
	})
	if err != nil {
		r.logger.Errorf(`failed to insert daily_metrics %v`, err)
		return err
	}

	fmt.Println("Daily metrics calculated and stored successfully for", date)
	return nil
}

func (r ReportRepository) ProcessDailyMetrics(ctx context.Context, day time.Time) error {
	// Get all orgs
	orgs, err := r.primaryDb.Pool.Query(ctx, "SELECT id FROM orgs where status = 'active'")
	if err != nil {
		return err
	}

	for orgs.Next() {
		var orgId string
		if err := orgs.Scan(&orgId); err != nil {
			r.logger.Errorf("failed to get orgid %s: %v", orgId, err)
			return err
		}

		err = r.StoreDailyMetrics(ctx, orgId, day)
		if err != nil {
			r.logger.Errorf("failed to store daily metrics for org %s: %v", orgId, err)
			continue
		}
	}

	return nil
}

func getDaysInMonth(date time.Time) int {
	firstOfMonth := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	firstOfNextMonth := firstOfMonth.AddDate(0, 1, 0)
	return int(firstOfNextMonth.Sub(firstOfMonth).Hours() / 24)
}
