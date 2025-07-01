package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
	"time"
)

type UsageRecordRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewUsageRecordRepository(primaryDb lib.Database, logger logger.Logger) repositories.UsageRecordRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return UsageRecordRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r UsageRecordRepository) FindById(ctx context.Context, orgId string, id string) (entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	var record models.UsageRecord
	query := `SELECT org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
              usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
              total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
              reference_id, reference_type, metadata, created_at, updated_at
              FROM usage_records
              WHERE org_id = @org_id AND id = @id`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&record.OrgId,
		&record.Id,
		&record.SubscriptionId,
		&record.SubscriptionItemId,
		&record.CustomerId,
		&record.PriceId,
		&record.UsageType,
		&record.Quantity,
		&record.UnitPrice,
		&record.TransactionValue,
		&record.PercentageRate,
		&record.CalculatedFee,
		&record.FixedFee,
		&record.TotalAmount,
		&record.UsageDate,
		&record.BillingPeriod,
		&record.Processed,
		&record.ProcessedAt,
		&record.InvoiceId,
		&record.ReferenceId,
		&record.ReferenceType,
		&record.Metadata,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find UsageRecord by id`, err.Error())
		return entities.UsageRecord{}, err
	}

	return record.ToEntity(), nil
}

func (r UsageRecordRepository) Create(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	metaJson, _ := json.Marshal(entity.Metadata)
	query := `INSERT INTO usage_records (
              org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
              usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
              total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
              reference_id, reference_type, metadata, created_at, updated_at)
              VALUES (
              @org_id, @id, @subscription_id, @subscription_item_id, @customer_id, @price_id, 
              @usage_type, @quantity, @unit_price, @transaction_value, @percentage_rate, @calculated_fee, @fixed_fee, 
              @total_amount, @usage_date, @billing_period, @processed, @processed_at, @invoice_id, 
              @reference_id, @reference_type, @metadata, NOW(), NOW())`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"subscription_id":      entity.SubscriptionId,
		"subscription_item_id": entity.SubscriptionItemId,
		"customer_id":          entity.CustomerId,
		"price_id":             entity.PriceId,
		"usage_type":           entity.UsageType,
		"quantity":             pgtype.Numeric{Valid: entity.Quantity != 0},
		"unit_price":           pgtype.Int8{Int64: entity.UnitPrice, Valid: entity.UnitPrice != 0},
		"transaction_value":    pgtype.Int8{Int64: entity.TransactionValue, Valid: entity.TransactionValue != 0},
		"percentage_rate":      pgtype.Numeric{Valid: entity.PercentageRate != 0},
		"calculated_fee":       pgtype.Int8{Int64: entity.CalculatedFee, Valid: entity.CalculatedFee != 0},
		"fixed_fee":            pgtype.Int8{Int64: entity.FixedFee, Valid: entity.FixedFee != 0},
		"total_amount":         entity.TotalAmount,
		"usage_date":           pgtype.Timestamp{Time: entity.UsageDate, Valid: !entity.UsageDate.IsZero()},
		"billing_period":       entity.BillingPeriod,
		"processed":            entity.Processed,
		"processed_at":         pgtype.Timestamp{Time: entity.ProcessedAt, Valid: !entity.ProcessedAt.IsZero()},
		"invoice_id":           pgtype.Text{String: entity.InvoiceId, Valid: entity.InvoiceId != ""},
		"reference_id":         pgtype.Text{String: entity.ReferenceId, Valid: entity.ReferenceId != ""},
		"reference_type":       pgtype.Text{String: entity.ReferenceType, Valid: entity.ReferenceType != ""},
		"metadata":             metaJson,
	})

	if err != nil {
		r.logger.Error(`failed to create UsageRecord`, err.Error())
		return entities.UsageRecord{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r UsageRecordRepository) Update(ctx context.Context, entity entities.UsageRecord) (entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	metaJson, _ := json.Marshal(entity.Metadata)
	query := `UPDATE usage_records SET
              processed = @processed,
              processed_at = @processed_at,
              invoice_id = @invoice_id,
              metadata = @metadata,
              updated_at = NOW()
              WHERE org_id = @org_id AND id = @id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":       entity.OrgId,
		"id":           entity.Id,
		"processed":    entity.Processed,
		"processed_at": pgtype.Timestamp{Time: entity.ProcessedAt, Valid: !entity.ProcessedAt.IsZero()},
		"invoice_id":   pgtype.Text{String: entity.InvoiceId, Valid: entity.InvoiceId != ""},
		"metadata":     metaJson,
	})

	if err != nil {
		r.logger.Error(`failed to update UsageRecord`, err.Error())
		return entities.UsageRecord{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r UsageRecordRepository) FindBySubscriptionItemId(ctx context.Context, orgId string, subscriptionItemId string) ([]entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
              usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
              total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
              reference_id, reference_type, metadata, created_at, updated_at
              FROM usage_records
              WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id
              ORDER BY usage_date DESC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":               orgId,
		"subscription_item_id": subscriptionItemId,
	})

	if err != nil {
		r.logger.Error(`failed to find UsageRecords by subscription_item_id`, err.Error())
		return nil, err
	}
	defer rows.Close()

	var records []entities.UsageRecord
	for rows.Next() {
		var record models.UsageRecord
		err := rows.Scan(
			&record.OrgId,
			&record.Id,
			&record.SubscriptionId,
			&record.SubscriptionItemId,
			&record.CustomerId,
			&record.PriceId,
			&record.UsageType,
			&record.Quantity,
			&record.UnitPrice,
			&record.TransactionValue,
			&record.PercentageRate,
			&record.CalculatedFee,
			&record.FixedFee,
			&record.TotalAmount,
			&record.UsageDate,
			&record.BillingPeriod,
			&record.Processed,
			&record.ProcessedAt,
			&record.InvoiceId,
			&record.ReferenceId,
			&record.ReferenceType,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan UsageRecord`, err.Error())
			return nil, err
		}
		records = append(records, record.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return records, nil
}

func (r UsageRecordRepository) FindBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
              usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
              total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
              reference_id, reference_type, metadata, created_at, updated_at
              FROM usage_records
              WHERE org_id = @org_id AND subscription_id = @subscription_id
              ORDER BY usage_date DESC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":          orgId,
		"subscription_id": subscriptionId,
	})

	if err != nil {
		r.logger.Error(`failed to find UsageRecords by subscription_id`, err.Error())
		return nil, err
	}
	defer rows.Close()

	var records []entities.UsageRecord
	for rows.Next() {
		var record models.UsageRecord
		err := rows.Scan(
			&record.OrgId,
			&record.Id,
			&record.SubscriptionId,
			&record.SubscriptionItemId,
			&record.CustomerId,
			&record.PriceId,
			&record.UsageType,
			&record.Quantity,
			&record.UnitPrice,
			&record.TransactionValue,
			&record.PercentageRate,
			&record.CalculatedFee,
			&record.FixedFee,
			&record.TotalAmount,
			&record.UsageDate,
			&record.BillingPeriod,
			&record.Processed,
			&record.ProcessedAt,
			&record.InvoiceId,
			&record.ReferenceId,
			&record.ReferenceType,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan UsageRecord`, err.Error())
			return nil, err
		}
		records = append(records, record.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return records, nil
}

func (r UsageRecordRepository) FindByBillingPeriod(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
              usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
              total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
              reference_id, reference_type, metadata, created_at, updated_at
              FROM usage_records
              WHERE org_id = @org_id AND subscription_id = @subscription_id AND billing_period = @billing_period
              ORDER BY usage_date DESC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":          orgId,
		"subscription_id": subscriptionId,
		"billing_period":  billingPeriod,
	})

	if err != nil {
		r.logger.Error(`failed to find UsageRecords by billing_period`, err.Error())
		return nil, err
	}
	defer rows.Close()

	var records []entities.UsageRecord
	for rows.Next() {
		var record models.UsageRecord
		err := rows.Scan(
			&record.OrgId,
			&record.Id,
			&record.SubscriptionId,
			&record.SubscriptionItemId,
			&record.CustomerId,
			&record.PriceId,
			&record.UsageType,
			&record.Quantity,
			&record.UnitPrice,
			&record.TransactionValue,
			&record.PercentageRate,
			&record.CalculatedFee,
			&record.FixedFee,
			&record.TotalAmount,
			&record.UsageDate,
			&record.BillingPeriod,
			&record.Processed,
			&record.ProcessedAt,
			&record.InvoiceId,
			&record.ReferenceId,
			&record.ReferenceType,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan UsageRecord`, err.Error())
			return nil, err
		}
		records = append(records, record.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return records, nil
}

func (r UsageRecordRepository) FindUnprocessed(ctx context.Context, orgId string, subscriptionId string, billingPeriod string) ([]entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
              usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
              total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
              reference_id, reference_type, metadata, created_at, updated_at
              FROM usage_records
              WHERE org_id = @org_id AND subscription_id = @subscription_id AND billing_period = @billing_period AND processed = false
              ORDER BY usage_date DESC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":          orgId,
		"subscription_id": subscriptionId,
		"billing_period":  billingPeriod,
	})

	if err != nil {
		r.logger.Error(`failed to find unprocessed UsageRecords`, err.Error())
		return nil, err
	}
	defer rows.Close()

	var records []entities.UsageRecord
	for rows.Next() {
		var record models.UsageRecord
		err := rows.Scan(
			&record.OrgId,
			&record.Id,
			&record.SubscriptionId,
			&record.SubscriptionItemId,
			&record.CustomerId,
			&record.PriceId,
			&record.UsageType,
			&record.Quantity,
			&record.UnitPrice,
			&record.TransactionValue,
			&record.PercentageRate,
			&record.CalculatedFee,
			&record.FixedFee,
			&record.TotalAmount,
			&record.UsageDate,
			&record.BillingPeriod,
			&record.Processed,
			&record.ProcessedAt,
			&record.InvoiceId,
			&record.ReferenceId,
			&record.ReferenceType,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan UsageRecord`, err.Error())
			return nil, err
		}
		records = append(records, record.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return records, nil
}

func (r UsageRecordRepository) MarkProcessed(ctx context.Context, orgId string, ids []string, invoiceId string) error {
	tx := r.getTransactionFromContext(ctx)

	// Convert the array of IDs to a string for the SQL IN clause
	query := `UPDATE usage_records SET
              processed = true,
              processed_at = NOW(),
              invoice_id = @invoice_id,
              updated_at = NOW()
              WHERE org_id = @org_id AND id = ANY(@ids)`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":     orgId,
		"ids":        ids,
		"invoice_id": invoiceId,
	})

	if err != nil {
		r.logger.Error(`failed to mark UsageRecords as processed`, err.Error())
		return err
	}

	return nil
}

func (r UsageRecordRepository) AggregateUsage(ctx context.Context, orgId string, subscriptionItemId string, billingPeriod string, aggregationType string) (float64, error) {
	tx := r.getTransactionFromContext(ctx)

	var query string
	switch aggregationType {
	case "sum":
		query = `SELECT COALESCE(SUM(quantity::float8), 0) FROM usage_records
                WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id AND billing_period = @billing_period AND processed = false`
	case "max":
		query = `SELECT COALESCE(MAX(quantity::float8), 0) FROM usage_records
                WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id AND billing_period = @billing_period AND processed = false`
	case "last_during_period":
		query = `SELECT COALESCE(quantity::float8, 0) FROM usage_records
                WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id AND billing_period = @billing_period AND processed = false
                ORDER BY usage_date DESC LIMIT 1`
	default:
		// Default to sum
		query = `SELECT COALESCE(SUM(quantity::float8), 0) FROM usage_records
                WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id AND billing_period = @billing_period AND processed = false`
	}

	var result float64
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":               orgId,
		"subscription_item_id": subscriptionItemId,
		"billing_period":       billingPeriod,
	}).Scan(&result)

	if err != nil {
		r.logger.Error(`failed to aggregate usage`, err.Error())
		return 0, err
	}

	return result, nil
}

func (r UsageRecordRepository) Find(ctx context.Context, orgId string, p request.Pagination) ([]entities.UsageRecord, int, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
              usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
              total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
              reference_id, reference_type, metadata, created_at, updated_at, count(*) OVER()
              FROM usage_records
              WHERE org_id = @org_id
              ORDER BY usage_date DESC
              LIMIT @limit OFFSET @offset`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"limit":  p.Limit,
		"offset": p.Offset,
	})

	if err != nil {
		r.logger.Error(`failed to find UsageRecords`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	var records []entities.UsageRecord
	var count int
	for rows.Next() {
		var record models.UsageRecord
		err := rows.Scan(
			&record.OrgId,
			&record.Id,
			&record.SubscriptionId,
			&record.SubscriptionItemId,
			&record.CustomerId,
			&record.PriceId,
			&record.UsageType,
			&record.Quantity,
			&record.UnitPrice,
			&record.TransactionValue,
			&record.PercentageRate,
			&record.CalculatedFee,
			&record.FixedFee,
			&record.TotalAmount,
			&record.UsageDate,
			&record.BillingPeriod,
			&record.Processed,
			&record.ProcessedAt,
			&record.InvoiceId,
			&record.ReferenceId,
			&record.ReferenceType,
			&record.Metadata,
			&record.CreatedAt,
			&record.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan UsageRecord`, err.Error())
			return nil, 0, err
		}
		records = append(records, record.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return records, count, nil
}

func (r UsageRecordRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM usage_records WHERE org_id = @org_id AND id = @id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	})

	if err != nil {
		r.logger.Error(`failed to delete UsageRecord`, err.Error())
		return err
	}

	return nil
}

func (r UsageRecordRepository) BatchCreate(ctx context.Context, entities []entities.UsageRecord) ([]entities.UsageRecord, error) {
	tx := r.getTransactionFromContext(ctx)

	for _, entity := range entities {
		metaJson, _ := json.Marshal(entity.Metadata)
		query := `INSERT INTO usage_records (
                  org_id, id, subscription_id, subscription_item_id, customer_id, price_id, 
                  usage_type, quantity, unit_price, transaction_value, percentage_rate, calculated_fee, fixed_fee, 
                  total_amount, usage_date, billing_period, processed, processed_at, invoice_id, 
                  reference_id, reference_type, metadata, created_at, updated_at)
                  VALUES (
                  @org_id, @id, @subscription_id, @subscription_item_id, @customer_id, @price_id, 
                  @usage_type, @quantity, @unit_price, @transaction_value, @percentage_rate, @calculated_fee, @fixed_fee, 
                  @total_amount, @usage_date, @billing_period, @processed, @processed_at, @invoice_id, 
                  @reference_id, @reference_type, @metadata, NOW(), NOW())`

		_, err := tx.Exec(ctx, query, pgx.NamedArgs{
			"org_id":               entity.OrgId,
			"id":                   entity.Id,
			"subscription_id":      entity.SubscriptionId,
			"subscription_item_id": entity.SubscriptionItemId,
			"customer_id":          entity.CustomerId,
			"price_id":             entity.PriceId,
			"usage_type":           entity.UsageType,
			"quantity":             pgtype.Numeric{Valid: entity.Quantity != 0},
			"unit_price":           pgtype.Int8{Int64: entity.UnitPrice, Valid: entity.UnitPrice != 0},
			"transaction_value":    pgtype.Int8{Int64: entity.TransactionValue, Valid: entity.TransactionValue != 0},
			"percentage_rate":      pgtype.Numeric{Valid: entity.PercentageRate != 0},
			"calculated_fee":       pgtype.Int8{Int64: entity.CalculatedFee, Valid: entity.CalculatedFee != 0},
			"fixed_fee":            pgtype.Int8{Int64: entity.FixedFee, Valid: entity.FixedFee != 0},
			"total_amount":         entity.TotalAmount,
			"usage_date":           pgtype.Timestamp{Time: entity.UsageDate, Valid: !entity.UsageDate.IsZero()},
			"billing_period":       entity.BillingPeriod,
			"processed":            entity.Processed,
			"processed_at":         pgtype.Timestamp{Time: entity.ProcessedAt, Valid: !entity.ProcessedAt.IsZero()},
			"invoice_id":           pgtype.Text{String: entity.InvoiceId, Valid: entity.InvoiceId != ""},
			"reference_id":         pgtype.Text{String: entity.ReferenceId, Valid: entity.ReferenceId != ""},
			"reference_type":       pgtype.Text{String: entity.ReferenceType, Valid: entity.ReferenceType != ""},
			"metadata":             metaJson,
		})

		if err != nil {
			r.logger.Error(`failed to create UsageRecord in batch`, err.Error())
			return nil, err
		}

	}

	return entities, nil
}

func (r UsageRecordRepository) GetUsageSummary(ctx context.Context, orgId string, subscriptionItemId string, startDate time.Time, endDate time.Time) (map[string]interface{}, error) {
	tx := r.getTransactionFromContext(ctx)

	// Get total quantity
	quantityQuery := `SELECT COALESCE(SUM(quantity::float8), 0) FROM usage_records
                     WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id 
                     AND usage_date >= @start_date AND usage_date <= @end_date`

	var totalQuantity float64
	err := tx.QueryRow(ctx, quantityQuery, pgx.NamedArgs{
		"org_id":               orgId,
		"subscription_item_id": subscriptionItemId,
		"start_date":           startDate,
		"end_date":             endDate,
	}).Scan(&totalQuantity)

	if err != nil {
		r.logger.Error(`failed to get total quantity`, err.Error())
		return nil, err
	}

	// Get total amount
	amountQuery := `SELECT COALESCE(SUM(total_amount), 0) FROM usage_records
                   WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id 
                   AND usage_date >= @start_date AND usage_date <= @end_date`

	var totalAmount int64
	err = tx.QueryRow(ctx, amountQuery, pgx.NamedArgs{
		"org_id":               orgId,
		"subscription_item_id": subscriptionItemId,
		"start_date":           startDate,
		"end_date":             endDate,
	}).Scan(&totalAmount)

	if err != nil {
		r.logger.Error(`failed to get total amount`, err.Error())
		return nil, err
	}

	// Get usage type
	typeQuery := `SELECT usage_type FROM usage_records
                WHERE org_id = @org_id AND subscription_item_id = @subscription_item_id
                LIMIT 1`

	var usageType string
	err = tx.QueryRow(ctx, typeQuery, pgx.NamedArgs{
		"org_id":               orgId,
		"subscription_item_id": subscriptionItemId,
	}).Scan(&usageType)

	if err != nil && err != pgx.ErrNoRows {
		r.logger.Error(`failed to get usage type`, err.Error())
		return nil, err
	}

	// Build the summary
	summary := map[string]interface{}{
		"subscription_item_id": subscriptionItemId,
		"start_date":           startDate,
		"end_date":             endDate,
		"total_quantity":       totalQuantity,
		"total_amount":         totalAmount,
		"usage_type":           usageType,
	}

	return summary, nil
}
