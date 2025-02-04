package postgres

import (
	"context"
	"errors"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/lib"
)

type PaymentRepository struct {
	*lib.PgDatabase
	logger lib.Logger
}

func NewPaymentRepository(database lib.Database, logger lib.Logger) repositories.PaymentRepository {
	pgDatabase, ok := database.(*lib.PgDatabase)
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
	query := `SELECT org_id, id, order_id, subscription_id, status, currency, amount, psp_fee, platform_fee, net_amount, metadata, created_at, updated_at
		          FROM payments
		          WHERE org_id = $1 AND id = $2`

	err := r.Pool.QueryRow(ctx, query, orgId, id).
		Scan(
			&payment.OrgId,
			&payment.Id,
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

func (r PaymentRepository) Create(ctx context.Context, entity entities.Payment) (entities.Payment, error) {
	query := `INSERT INTO payments (org_id, id, order_id, subscription_id, status, currency, amount, psp_fee, platform_fee, net_amount, metadata, created_at, updated_at)
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	          RETURNING org_id, id, order_id, subscription_id, status, currency, amount, psp_fee, platform_fee, net_amount, metadata, created_at, updated_at`

	err := r.Pool.QueryRow(ctx, query,
		entity.OrgId,
		entity.Id,
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
		r.logger.Error(`failed to create Payment`, err.Error())
		return entities.Payment{}, err
	}

	return entity, nil
}
