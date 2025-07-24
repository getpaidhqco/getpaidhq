package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type PaymentLinkRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPaymentLinkRepository(primaryDb lib.Database, logger logger.Logger) repositories.PaymentLinkRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PaymentLinkRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PaymentLinkRepository) FindById(ctx context.Context, orgId string, id string) (entities.PaymentLink, error) {
	tx := r.getTransactionFromContext(ctx)

	var paymentLink models.PaymentLink

	query := `SELECT org_id, id, slug, data, config, single_use, status, created_at, updated_at, used_at, expires_at
			  FROM payment_links
			  WHERE org_id = @org_id AND id = @id`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&paymentLink.OrgId,
		&paymentLink.Id,
		&paymentLink.Slug,
		&paymentLink.Data,
		&paymentLink.Config,
		&paymentLink.SingleUse,
		&paymentLink.Status,
		&paymentLink.CreatedAt,
		&paymentLink.UpdatedAt,
		&paymentLink.UsedAt,
		&paymentLink.ExpiresAt,
	)
	if err != nil {
		r.logger.Error("failed to find payment link", err)
		return entities.PaymentLink{}, err
	}

	return paymentLink.ToEntity(), nil
}

func (r PaymentLinkRepository) FindBySlug(ctx context.Context, slug string) (entities.PaymentLink, error) {
	tx := r.getTransactionFromContext(ctx)

	var paymentLink models.PaymentLink

	query := `SELECT org_id, id, slug, data, config, single_use, status, created_at, updated_at, used_at, expires_at
			  FROM payment_links
			  WHERE slug = @slug`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"slug": slug,
	}).Scan(
		&paymentLink.OrgId,
		&paymentLink.Id,
		&paymentLink.Slug,
		&paymentLink.Data,
		&paymentLink.Config,
		&paymentLink.SingleUse,
		&paymentLink.Status,
		&paymentLink.CreatedAt,
		&paymentLink.UpdatedAt,
		&paymentLink.UsedAt,
		&paymentLink.ExpiresAt,
	)
	if err != nil {
		r.logger.Error("failed to find payment link by slug", err)
		return entities.PaymentLink{}, err
	}

	return paymentLink.ToEntity(), nil
}

func (r PaymentLinkRepository) List(ctx context.Context, orgId string, p request.Pagination) ([]entities.PaymentLink, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var paymentLinks = make([]entities.PaymentLink, 0)
	var count int

	query := `SELECT org_id, id, slug, data, config, single_use, status, created_at, updated_at, used_at, expires_at, count(*) OVER()
			  FROM payment_links
			  WHERE org_id = @org_id
			  ORDER BY
			    -- Handle timestamp columns
			    CASE
			        WHEN @sort_col = 'created_at' AND @sort_dir = 'asc' THEN created_at
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'created_at' AND @sort_dir = 'desc' THEN created_at
			        ELSE NULL
			    END DESC,

			    -- Handle text columns
			    CASE
			        WHEN @sort_col = 'slug' AND @sort_dir = 'asc' THEN slug
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'slug' AND @sort_dir = 'desc' THEN slug
			        ELSE NULL
			    END DESC,

			    CASE
			        WHEN @sort_col = 'status' AND @sort_dir = 'asc' THEN status
			        ELSE NULL
			    END ASC,
			    CASE
			        WHEN @sort_col = 'status' AND @sort_dir = 'desc' THEN status
			        ELSE NULL
			    END DESC
			  LIMIT @lim OFFSET @off;`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"lim":      p.Limit,
		"off":      p.Offset,
		"sort_col": p.SortBy,
		"sort_dir": p.SortDirection,
	})
	if err != nil {
		r.logger.Error("failed to list payment links", err)
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var paymentLink models.PaymentLink
		err := rows.Scan(
			&paymentLink.OrgId,
			&paymentLink.Id,
			&paymentLink.Slug,
			&paymentLink.Data,
			&paymentLink.Config,
			&paymentLink.SingleUse,
			&paymentLink.Status,
			&paymentLink.CreatedAt,
			&paymentLink.UpdatedAt,
			&paymentLink.UsedAt,
			&paymentLink.ExpiresAt,
			&count,
		)
		if err != nil {
			r.logger.Error("failed to scan payment link", err)
			return nil, 0, err
		}
		paymentLinks = append(paymentLinks, paymentLink.ToEntity())
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating payment links", err)
		return nil, 0, err
	}

	return paymentLinks, count, nil
}

func (r PaymentLinkRepository) Create(ctx context.Context, input entities.PaymentLink) (entities.PaymentLink, error) {
	tx := r.getTransactionFromContext(ctx)

	var paymentLink models.PaymentLink

	query := `INSERT INTO payment_links (org_id, id, slug, data, config, single_use, status, created_at, updated_at, used_at, expires_at)
			  VALUES (@org_id, @id, @slug, @data, @config, @single_use, @status, NOW(), NOW(), @used_at, @expires_at)
			  RETURNING org_id, id, slug, data, config, single_use, status, created_at, updated_at, used_at, expires_at`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":     input.OrgId,
		"id":         input.Id,
		"slug":       input.Slug,
		"data":       input.Data,
		"config":     input.Config,
		"single_use": input.SingleUse,
		"status":     input.Status,
		"used_at":    input.UsedAt,
		"expires_at": input.ExpiresAt,
	}).Scan(
		&paymentLink.OrgId,
		&paymentLink.Id,
		&paymentLink.Slug,
		&paymentLink.Data,
		&paymentLink.Config,
		&paymentLink.SingleUse,
		&paymentLink.Status,
		&paymentLink.CreatedAt,
		&paymentLink.UpdatedAt,
		&paymentLink.UsedAt,
		&paymentLink.ExpiresAt,
	)

	if err != nil {
		r.logger.Error("failed to create payment link", err)
		return entities.PaymentLink{}, err
	}

	return paymentLink.ToEntity(), nil
}

func (r PaymentLinkRepository) Update(ctx context.Context, input entities.PaymentLink) (entities.PaymentLink, error) {
	tx := r.getTransactionFromContext(ctx)

	var paymentLink models.PaymentLink

	query := `UPDATE payment_links
			  SET slug = @slug, data = @data, config = @config, single_use = @single_use, status = @status, 
			      updated_at = NOW(), used_at = @used_at, expires_at = @expires_at
			  WHERE org_id = @org_id AND id = @id
			  RETURNING org_id, id, slug, data, config, single_use, status, created_at, updated_at, used_at, expires_at`

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":     input.OrgId,
		"id":         input.Id,
		"slug":       input.Slug,
		"data":       input.Data,
		"config":     input.Config,
		"single_use": input.SingleUse,
		"status":     input.Status,
		"used_at":    input.UsedAt,
		"expires_at": input.ExpiresAt,
	}).Scan(
		&paymentLink.OrgId,
		&paymentLink.Id,
		&paymentLink.Slug,
		&paymentLink.Data,
		&paymentLink.Config,
		&paymentLink.SingleUse,
		&paymentLink.Status,
		&paymentLink.CreatedAt,
		&paymentLink.UpdatedAt,
		&paymentLink.UsedAt,
		&paymentLink.ExpiresAt,
	)

	if err != nil {
		r.logger.Error("failed to update payment link", err)
		return entities.PaymentLink{}, err
	}

	return paymentLink.ToEntity(), nil
}

func (r PaymentLinkRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM payment_links WHERE org_id = @org_id AND id = @id`

	_, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	})

	if err != nil {
		r.logger.Error("failed to delete payment link", err)
		return fmt.Errorf("failed to delete payment link: %w", err)
	}

	return nil
}
