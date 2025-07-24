package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type PaymentLinkUsageRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPaymentLinkUsageRepository(primaryDb lib.Database, logger logger.Logger) repositories.PaymentLinkUsageRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PaymentLinkUsageRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PaymentLinkUsageRepository) FindById(ctx context.Context, orgId string, id string) (entities.PaymentLinkUsage, error) {
	tx := r.getTransactionFromContext(ctx)

	var usage models.PaymentLinkUsage

	query := `SELECT id, org_id, payment_link_id, session_id, customer_id, event_type, ip_address, user_agent, referer, country, metadata, timestamp
			  FROM payment_link_usage
			  WHERE org_id = @org_id AND id = @id`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&usage.Id,
		&usage.OrgId,
		&usage.PaymentLinkId,
		&usage.SessionId,
		&usage.CustomerId,
		&usage.EventType,
		&usage.IpAddress,
		&usage.UserAgent,
		&usage.Referer,
		&usage.Country,
		&usage.Metadata,
		&usage.Timestamp,
	)
	if err != nil {
		r.logger.Error("failed to find payment link usage", err)
		return entities.PaymentLinkUsage{}, err
	}

	return usage.ToEntity(), nil
}

func (r PaymentLinkUsageRepository) ListByPaymentLinkId(ctx context.Context, orgId string, paymentLinkId string) ([]entities.PaymentLinkUsage, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT id, org_id, payment_link_id, session_id, customer_id, event_type, ip_address, user_agent, referer, country, metadata, timestamp
			  FROM payment_link_usage
			  WHERE org_id = @org_id AND payment_link_id = @payment_link_id
			  ORDER BY timestamp DESC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":          orgId,
		"payment_link_id": paymentLinkId,
	})
	if err != nil {
		r.logger.Error("failed to list payment link usage", err)
		return nil, err
	}
	defer rows.Close()

	var usages []entities.PaymentLinkUsage
	for rows.Next() {
		var usage models.PaymentLinkUsage
		err := rows.Scan(
			&usage.Id,
			&usage.OrgId,
			&usage.PaymentLinkId,
			&usage.SessionId,
			&usage.CustomerId,
			&usage.EventType,
			&usage.IpAddress,
			&usage.UserAgent,
			&usage.Referer,
			&usage.Country,
			&usage.Metadata,
			&usage.Timestamp,
		)
		if err != nil {
			r.logger.Error("failed to scan payment link usage", err)
			return nil, err
		}
		usages = append(usages, usage.ToEntity())
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating payment link usages", err)
		return nil, err
	}

	return usages, nil
}

func (r PaymentLinkUsageRepository) Create(ctx context.Context, input entities.PaymentLinkUsage) (entities.PaymentLinkUsage, error) {
	tx := r.getTransactionFromContext(ctx)

	var usage models.PaymentLinkUsage

	query := `INSERT INTO payment_link_usage (id, org_id, payment_link_id, session_id, customer_id, event_type, ip_address, user_agent, referer, country, metadata, timestamp)
			  VALUES (@id, @org_id, @payment_link_id, @session_id, @customer_id, @event_type, @ip_address, @user_agent, @referer, @country, @metadata, @timestamp)
			  RETURNING id, org_id, payment_link_id, session_id, customer_id, event_type, ip_address, user_agent, referer, country, metadata, timestamp`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"id":              input.Id,
		"org_id":          input.OrgId,
		"payment_link_id": input.PaymentLinkId,
		"session_id":      input.SessionId,
		"customer_id":     input.CustomerId,
		"event_type":      input.EventType,
		"ip_address":      input.IpAddress,
		"user_agent":      input.UserAgent,
		"referer":         input.Referer,
		"country":         input.Country,
		"metadata":        input.Metadata,
		"timestamp":       input.Timestamp,
	}).Scan(
		&usage.Id,
		&usage.OrgId,
		&usage.PaymentLinkId,
		&usage.SessionId,
		&usage.CustomerId,
		&usage.EventType,
		&usage.IpAddress,
		&usage.UserAgent,
		&usage.Referer,
		&usage.Country,
		&usage.Metadata,
		&usage.Timestamp,
	)

	if err != nil {
		r.logger.Error("failed to create payment link usage", err)
		return entities.PaymentLinkUsage{}, err
	}

	return usage.ToEntity(), nil
}
