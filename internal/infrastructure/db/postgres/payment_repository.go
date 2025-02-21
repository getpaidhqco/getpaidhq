package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type PaymentRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPaymentRepository(database lib.Database, logger logger.Logger) repositories.PaymentRepository {
	pgDatabase, ok := database.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PaymentRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PaymentRepository) FindById(ctx context.Context, orgId string, id string) (entities.Payment, error) {
	var payment entities.Payment
	query := `SELECT org_id, id, reference, order_id, subscription_id, status, currency, amount, psp_fee, platform_fee, net_amount, metadata, created_at, updated_at
		          FROM payments
		          WHERE org_id = $1 AND id = $2`

	err := r.Pool.QueryRow(ctx, query, orgId, id).
		Scan(
			&payment.OrgId,
			&payment.Id,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
			&payment.CreatedAt,
			&payment.UpdatedAt,
		)
	if err != nil {
		r.logger.Error(`failed to find Payment`, err.Error())
		return entities.Payment{}, errors.New("not found")
	}

	return payment, nil
}
func (r PaymentRepository) FindBySubscriptionId(ctx context.Context, orgId string, id string, p entities.Pagination) ([]entities.Payment, int, error) {
	var payments []entities.Payment
	var total int
	query := `SELECT org_id, id, psp_id, reference, order_id, subscription_id,
       status, currency, amount, psp_fee, platform_fee, net_amount, metadata, 
       created_at, updated_at,
        count(*) OVER()
	          FROM payments
	          WHERE org_id = @org_id AND subscription_id =  @id
LIMIT @lim OFFSET @off;
	         `

	rows, err := r.Pool.Query(ctx, query, pgx.NamedArgs{
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
			&payment.PspId,
			&payment.Reference,
			&payment.OrderId,
			&payment.SubscriptionId,
			&payment.Status,
			&payment.Currency,
			&payment.Amount,
			&payment.PspFee,
			&payment.PlatformFee,
			&payment.NetAmount,
			&payment.Metadata,
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

	return payments, 0, nil
}

func (r PaymentRepository) Create(ctx context.Context, entity entities.Payment) (entities.Payment, error) {
	query := `INSERT INTO payments (org_id, id, psp_id,reference,order_id, subscription_id, status, currency, amount, psp_fee, platform_fee, net_amount, metadata, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	          RETURNING org_id, id, psp_id, reference, order_id, subscription_id, status, currency, amount, psp_fee, platform_fee, net_amount, metadata, created_at, updated_at`

	err := r.Pool.QueryRow(ctx, query,
		entity.OrgId,
		entity.Id,
		entity.PspId,
		entity.Reference,
		entity.OrderId,
		entity.SubscriptionId,
		entity.Status,
		entity.Currency,
		entity.Amount,
		entity.PspFee,
		entity.PlatformFee,
		entity.NetAmount,
		entity.Metadata,
		entity.CreatedAt,
		entity.UpdatedAt,
	).Scan(
		&entity.OrgId,
		&entity.Id,
		&entity.PspId,
		&entity.Reference,
		&entity.OrderId,
		&entity.SubscriptionId,
		&entity.Status,
		&entity.Currency,
		&entity.Amount,
		&entity.PspFee,
		&entity.PlatformFee,
		&entity.NetAmount,
		&entity.Metadata,
		&entity.CreatedAt,
		&entity.UpdatedAt,
	)
	if err != nil {
		r.logger.Error(`failed to create Payment`, "err", err.Error())
		return entities.Payment{}, err
	}

	return entity, nil
}
