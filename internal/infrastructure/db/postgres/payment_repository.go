package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type PaymentRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPaymentRepository(primaryDb lib.Database, logger logger.Logger) repositories.PaymentRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PaymentRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PaymentRepository) FindById(ctx context.Context, orgId string, id string) (entities.Payment, error) {
	tx := r.getTransactionFromContext(ctx)

	var payment models.Payment
	query := `SELECT org_id, id,psp,psp_id, reference, order_id, subscription_id, invoice_id, status, currency, amount, psp_fee, platform_fee, net_amount, metadata,completed_at, created_at, updated_at
		          FROM payments
		          WHERE org_id = $1 AND id = $2`

	err := tx.QueryRow(ctx, query, orgId, id).
		Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Psp,
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.InvoiceId,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
	if err != nil {
		r.logger.Error(`failed to find Payment`, err.Error())
		return entities.Payment{}, errors.New("not found")
	}

	return payment.ToEntity(), nil
}
func (r PaymentRepository) FindBySubscriptionId(ctx context.Context, orgId string, id string, p entities.Pagination) ([]entities.Payment, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var payments []entities.Payment
	var total int
	query := `SELECT org_id, id, psp, psp_id, reference, order_id, subscription_id, invoice_id,
       status, currency, amount, psp_fee, platform_fee, net_amount, metadata, 
       completed_at, created_at, updated_at,
        count(*) OVER()
	          FROM payments
	          WHERE org_id = @org_id AND subscription_id =  @id
	          ORDER BY
    -- Simplified to NULL if not sorting in ascending order.
    CASE
        WHEN @sort_dir = 'asc' THEN
            CASE @sort_col
                -- Check for each possible value of sort_col.
                WHEN 'created_at' THEN created_at
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
                 WHEN 'created_at' THEN created_at
                 ELSE NULL
                 END
         ELSE
             NULL
        END
        DESC
	LIMIT @lim OFFSET @off;
	         `

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"id":       id,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Payments by SubscriptionId`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var payment models.Payment
		err := rows.Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Psp,
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.InvoiceId,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&total,
		)
		if err != nil {
			r.logger.Error(`failed to scan Payment`, err.Error())
			return nil, 0, err
		}
		payments = append(payments, payment.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return payments, total, nil
}

func (r PaymentRepository) FindByInvoiceId(ctx context.Context, orgId string, id string, p entities.Pagination) ([]entities.Payment, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var payments []entities.Payment
	var total int
	query := `SELECT org_id, id, psp, psp_id, reference, order_id, subscription_id, invoice_id,
       status, currency, amount, psp_fee, platform_fee, net_amount, metadata, 
       completed_at, created_at, updated_at,
        count(*) OVER()
	          FROM payments
	          WHERE org_id = @org_id AND invoice_id = @id
	          ORDER BY
    -- Simplified to NULL if not sorting in ascending order.
    CASE
        WHEN @sort_dir = 'asc' THEN
            CASE @sort_col
                -- Check for each possible value of sort_col.
                WHEN 'created_at' THEN created_at
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
                 WHEN 'created_at' THEN created_at
                 ELSE NULL
                 END
         ELSE
             NULL
        END
        DESC
	LIMIT @lim OFFSET @off;
	         `

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"id":       id,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Payments by InvoiceId`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var payment models.Payment
		err := rows.Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Psp,
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.InvoiceId,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&total,
		)
		if err != nil {
			r.logger.Error(`failed to scan Payment`, err.Error())
			return nil, 0, err
		}
		payments = append(payments, payment.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return payments, total, nil
}

func (r PaymentRepository) List(ctx context.Context, orgId string, p entities.Pagination) ([]entities.Payment, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var payments = make([]entities.Payment, 0)
	var total int
	query := `SELECT org_id, id, psp, psp_id, reference, order_id, subscription_id, invoice_id,
       status, currency, amount, psp_fee, platform_fee, net_amount, metadata, 
       completed_at, created_at, updated_at,
        count(*) OVER()
	          FROM payments
	          WHERE org_id = @org_id
	          ORDER BY
    -- Simplified to NULL if not sorting in ascending order.
    CASE
        WHEN @sort_dir = 'asc' THEN
            CASE @sort_col
                -- Check for each possible value of sort_col.
                WHEN 'created_at' THEN created_at
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
                 WHEN 'created_at' THEN created_at
                 ELSE NULL
                 END
         ELSE
             NULL
        END
        DESC
	LIMIT @lim OFFSET @off;
	         `

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to list Payments`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var payment models.Payment
		err := rows.Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Psp,
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.InvoiceId,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
			&total,
		)
		if err != nil {
			r.logger.Error(`failed to scan Payment`, err.Error())
			return nil, 0, err
		}
		payments = append(payments, payment.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return payments, total, nil
}

func (r PaymentRepository) Create(ctx context.Context, entity entities.Payment) (entities.Payment, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO payments (org_id, id, psp, psp_id, reference, 
                      order_id, subscription_id, invoice_id, recurring, status, currency, 
                      amount, psp_fee, platform_fee, net_amount, metadata, 
                      completed_at, created_at, updated_at)
          VALUES (@org_id, @id, @psp, @psp_id, @reference, 
                  @order_id, @subscription_id, @invoice_id, @recurring, @status, @currency, 
                  @amount, @psp_fee, @platform_fee, @net_amount, @metadata, 
                  @completed_at, @created_at, @updated_at)
          RETURNING org_id, id, psp, psp_id, reference, 
              order_id, subscription_id, invoice_id, recurring, status, currency, 
              amount, psp_fee, platform_fee, net_amount, metadata, 
              completed_at, created_at, updated_at`
	var payment models.Payment

	err := tx.QueryRow(ctx, query, paymentEntityToNamedArgs(entity)).
		Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Psp,
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.InvoiceId,
			&payment.Recurring,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)

	if err != nil {
		r.logger.Error(`failed to create Payment`, "err", err.Error())
		return entities.Payment{}, err
	}

	return payment.ToEntity(), nil
}

func (r PaymentRepository) Update(ctx context.Context, entity entities.Payment) (entities.Payment, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `UPDATE payments
	          SET psp = @psp, psp_id = @psp_id, reference = @reference, order_id = @order_id,
	              subscription_id = @subscription_id, invoice_id = @invoice_id, recurring = @recurring, status = @status,
	              currency = @currency, amount = @amount, psp_fee = @psp_fee, platform_fee = @platform_fee,
	              net_amount = @net_amount, metadata = @metadata, completed_at = @completed_at,
	              updated_at = @updated_at
	          WHERE org_id = @org_id AND id = @id
	          RETURNING org_id, id, psp, psp_id, reference, order_id, subscription_id, invoice_id, recurring,
	                    status, currency, amount, psp_fee, platform_fee, net_amount, metadata,
	                    completed_at, created_at, updated_at`

	var payment models.Payment

	err := tx.QueryRow(ctx, query, paymentEntityToNamedArgs(entity)).
		Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Psp,
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.InvoiceId,
			&payment.Recurring,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)

	if err != nil {
		r.logger.Error(`failed to update Payment`, "err", err.Error())
		return entities.Payment{}, err
	}

	return payment.ToEntity(), nil
}

func paymentEntityToNamedArgs(entity entities.Payment) pgx.NamedArgs {
	metaJson, _ := json.Marshal(entity.Metadata)
	return pgx.NamedArgs{
		"org_id":          entity.OrgId,
		"id":              entity.Id,
		"psp":             entity.Psp,
		"psp_id":          pgtype.Text{String: entity.PspId, Valid: entity.PspId != ""},
		"reference":       entity.Reference,
		"order_id":        pgtype.Text{String: entity.OrderId, Valid: entity.OrderId != ""},
		"subscription_id": pgtype.Text{String: entity.SubscriptionId, Valid: entity.SubscriptionId != ""},
		"invoice_id":      pgtype.Text{String: entity.InvoiceId, Valid: entity.InvoiceId != ""},
		"status":          entity.Status,
		"currency":        entity.Currency,
		"amount":          entity.Amount,
		"psp_fee":         entity.PspFee,
		"platform_fee":    entity.PlatformFee,
		"net_amount":      entity.NetAmount,
		"metadata":        metaJson,
		"recurring":       entity.Recurring,
		"completed_at":    pgtype.Date{Time: entity.CompletedAt, Valid: !entity.CompletedAt.IsZero()},
		"created_at":      entity.CreatedAt,
		"updated_at":      entity.UpdatedAt,
	}
}

func (r PaymentRepository) FindByPspId(ctx context.Context, orgId string, pspId string) (entities.Payment, error) {
	tx := r.getTransactionFromContext(ctx)

	var payment models.Payment
	query := `SELECT org_id, id
	          FROM payments
	          WHERE org_id = @org_id AND psp_id = @psp_id`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"psp_id": pspId,
	}).Scan(
		&payment.OrgId,
		&payment.Id,
	)
	if err != nil {
		r.logger.Error(`failed to find Payment by PspId`, "err", err.Error())
		return entities.Payment{}, errors.New("not found")
	}

	return r.FindById(ctx, orgId, payment.Id)
}

func (r PaymentRepository) ListByPspId(ctx context.Context, psp common.Gateway, pspId string) ([]entities.Payment, error) {
	tx := r.getTransactionFromContext(ctx)

	var payments = make([]entities.Payment, 0)

	query := `SELECT org_id, id, psp, psp_id, reference, order_id, subscription_id, invoice_id, recurring,
	                    status, currency, amount, psp_fee, platform_fee, net_amount, metadata,
	                    completed_at, created_at, updated_at
	          FROM payments
	          WHERE psp = @psp AND psp_id = @psp_id
`
	for rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"psp":    psp,
		"psp_id": pspId,
	}); err == nil && rows.Next(); {
		var payment models.Payment
		if scanErr := rows.Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Psp,
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.InvoiceId,
			&payment.Recurring,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CompletedAt,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		); scanErr != nil {
			r.logger.Error("failed to scan Payment", "err", scanErr.Error())
			return nil, scanErr
		}
		payments = append(payments, payment.ToEntity())
	}

	return payments, nil
}

func (r PaymentRepository) CreateRefund(ctx context.Context, refund entities.Refund) (entities.Refund, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO refunds (org_id, id, psp_refund_id, payment_id, amount, currency, reason, status, refunded_at, completed_at, created_at, updated_at)
	          VALUES (@org_id,@id, @psp_refund_id, @payment_id, @amount, @currency, @reason, @status, @refunded_at, @completed_at, @created_at, @updated_at)
	          RETURNING org_id, id, psp_refund_id, payment_id, amount, currency, reason, status, refunded_at, completed_at, created_at, updated_at`

	var refundModel models.Refund

	var completedAt pgtype.Timestamptz
	if refund.CompletedAt != nil {
		completedAt = pgtype.Timestamptz{Time: *refund.CompletedAt, Valid: true}
	}

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":        refund.OrgId,
		"id":            refund.Id,
		"psp_refund_id": pgtype.Text{String: refund.PspRefundId, Valid: refund.PspRefundId != ""},
		"payment_id":    refund.PaymentId,
		"amount":        refund.Amount,
		"currency":      refund.Currency,
		"reason":        refund.Reason,
		"status":        string(refund.Status),
		"refunded_at":   pgtype.Date{Time: refund.RefundedAt, Valid: !refund.RefundedAt.IsZero()},
		"completed_at":  completedAt,
		"created_at":    refund.CreatedAt,
		"updated_at":    refund.UpdatedAt,
	}).Scan(
		&refundModel.OrgId,
		&refundModel.Id,
		&refundModel.PspRefundId,
		&refundModel.PaymentId,
		&refundModel.Amount,
		&refundModel.Currency,
		&refundModel.Reason,
		&refundModel.Status,
		&refundModel.RefundedAt,
		&refundModel.CompletedAt,
		&refundModel.CreatedAt,
		&refundModel.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("failed to create PaymentRefund", "err", err.Error())
		return entities.Refund{}, err
	}

	return refundModel.ToEntity(), nil
}
