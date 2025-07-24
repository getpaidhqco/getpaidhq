package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/application/dto"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
	"time"
)

type DiscountRedemptionRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewDiscountRedemptionRepository(primaryDb lib.Database, logger logger.Logger) repositories.DiscountRedemptionRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return &DiscountRedemptionRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

// WithTrx enables repository with transaction
func (r DiscountRedemptionRepository) WithTrx(trxHandle interface{}) DiscountRedemptionRepository {
	if trxHandle == nil {
		r.logger.Warn("Transaction Database not found in gin context. ")
		return r
	}
	r.PgDatabase.Tx = trxHandle.(pgx.Tx)
	return r
}

func (r DiscountRedemptionRepository) FindById(ctx context.Context, orgId string, id string) (entities.DiscountRedemption, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT id, org_id, discount_id, customer_id, resource_type, resource_id, 
		       discount_amount, currency, created_at, metadata
		FROM discount_redemptions
		WHERE org_id = @org_id AND id = @id`

	var redemptionModel models.DiscountRedemption

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	}).Scan(
		&redemptionModel.Id,
		&redemptionModel.OrgId,
		&redemptionModel.DiscountId,
		&redemptionModel.CustomerId,
		&redemptionModel.ResourceType,
		&redemptionModel.ResourceId,
		&redemptionModel.DiscountAmount,
		&redemptionModel.Currency,
		&redemptionModel.CreatedAt,
		&redemptionModel.Metadata,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entities.DiscountRedemption{}, err
		}
		r.logger.Error(`failed to find DiscountRedemption by id`, err.Error())
		return entities.DiscountRedemption{}, err
	}

	// Convert model to entity
	redemption := redemptionModel.ToEntity()

	return redemption, nil
}

func (r DiscountRedemptionRepository) Create(ctx context.Context, redemption entities.DiscountRedemption) (entities.DiscountRedemption, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO discount_redemptions (
			id, org_id, discount_id, customer_id, resource_type, resource_id, 
			discount_amount, currency, created_at, metadata
		) VALUES (
			@id, @org_id, @discount_id, @customer_id, @resource_type, @resource_id, 
			@discount_amount, @currency, @created_at, @metadata
		) RETURNING id, created_at`

	// Create a model from the entity
	redemptionModel := models.DiscountRedemption{
		Id:             redemption.Id,
		OrgId:          redemption.OrgId,
		DiscountId:     redemption.DiscountId,
		CustomerId:     redemption.CustomerId,
		ResourceType:   redemption.ResourceType,
		ResourceId:     redemption.ResourceId,
		DiscountAmount: redemption.DiscountAmount,
		Currency:       redemption.Currency,
		Metadata:       redemption.Metadata,
	}

	now := time.Now()
	redemptionModel.CreatedAt = now

	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"id":              redemptionModel.Id,
		"org_id":          redemptionModel.OrgId,
		"discount_id":     redemptionModel.DiscountId,
		"customer_id":     redemptionModel.CustomerId,
		"resource_type":   redemptionModel.ResourceType,
		"resource_id":     redemptionModel.ResourceId,
		"discount_amount": redemptionModel.DiscountAmount,
		"currency":        redemptionModel.Currency,
		"created_at":      redemptionModel.CreatedAt,
		"metadata":        redemptionModel.Metadata,
	}).Scan(&redemptionModel.Id, &redemptionModel.CreatedAt)

	if err != nil {
		r.logger.Error(`failed to create DiscountRedemption`, err.Error())
		return entities.DiscountRedemption{}, err
	}

	// Convert model to entity
	createdRedemption := redemptionModel.ToEntity()

	return createdRedemption, nil
}

func (r DiscountRedemptionRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	query := `DELETE FROM discount_redemptions WHERE org_id = @org_id AND id = @id`

	result, err := tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id": orgId,
		"id":     id,
	})

	if err != nil {
		r.logger.Error(`failed to delete DiscountRedemption`, err.Error())
		return err
	}

	if result.RowsAffected() == 0 {
		return errors.New("discount redemption not found")
	}

	return nil
}

func (r DiscountRedemptionRepository) ListByDiscount(ctx context.Context, orgId string, discountId string, pagination dto.Pagination) ([]entities.DiscountRedemption, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var redemptions = make([]entities.DiscountRedemption, 0)
	var count int

	query := `SELECT id, org_id, discount_id, customer_id, resource_type, resource_id, 
		       discount_amount, currency, created_at, metadata,
		       count(*) OVER()
		FROM discount_redemptions
		WHERE org_id = @org_id AND discount_id = @discount_id
		ORDER BY
		CASE
			WHEN @sort_dir = 'asc' THEN
				CASE @sort_col
					WHEN 'created_at' THEN created_at
					ELSE NULL
				END
			ELSE
				NULL
			END
			ASC,
		CASE
			WHEN @sort_dir = 'desc' THEN
				CASE @sort_col
					WHEN 'created_at' THEN created_at
					ELSE NULL
				END
			ELSE
				NULL
			END
			DESC
		LIMIT @lim OFFSET @off`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":      orgId,
		"discount_id": discountId,
		"lim":         pagination.Limit,
		"off":         pagination.Offset,
		"sort_col":    pagination.SortBy,
		"sort_dir":    pagination.SortDirection,
	})

	if err != nil {
		r.logger.Error(`failed to list DiscountRedemptions by discount`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var redemptionModel models.DiscountRedemption

		err := rows.Scan(
			&redemptionModel.Id,
			&redemptionModel.OrgId,
			&redemptionModel.DiscountId,
			&redemptionModel.CustomerId,
			&redemptionModel.ResourceType,
			&redemptionModel.ResourceId,
			&redemptionModel.DiscountAmount,
			&redemptionModel.Currency,
			&redemptionModel.CreatedAt,
			&redemptionModel.Metadata,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan DiscountRedemption`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		redemption := redemptionModel.ToEntity()
		redemptions = append(redemptions, redemption)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return redemptions, count, nil
}

func (r DiscountRedemptionRepository) ListByCustomer(ctx context.Context, orgId string, customerId string, pagination dto.Pagination) ([]entities.DiscountRedemption, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var redemptions = make([]entities.DiscountRedemption, 0)
	var count int

	query := `SELECT id, org_id, discount_id, customer_id, resource_type, resource_id, 
		       discount_amount, currency, created_at, metadata,
		       count(*) OVER()
		FROM discount_redemptions
		WHERE org_id = @org_id AND customer_id = @customer_id
		ORDER BY
		CASE
			WHEN @sort_dir = 'asc' THEN
				CASE @sort_col
					WHEN 'created_at' THEN created_at
					ELSE NULL
				END
			ELSE
				NULL
			END
			ASC,
		CASE
			WHEN @sort_dir = 'desc' THEN
				CASE @sort_col
					WHEN 'created_at' THEN created_at
					ELSE NULL
				END
			ELSE
				NULL
			END
			DESC
		LIMIT @lim OFFSET @off`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":      orgId,
		"customer_id": customerId,
		"lim":         pagination.Limit,
		"off":         pagination.Offset,
		"sort_col":    pagination.SortBy,
		"sort_dir":    pagination.SortDirection,
	})

	if err != nil {
		r.logger.Error(`failed to list DiscountRedemptions by customer`, err.Error())
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var redemptionModel models.DiscountRedemption

		err := rows.Scan(
			&redemptionModel.Id,
			&redemptionModel.OrgId,
			&redemptionModel.DiscountId,
			&redemptionModel.CustomerId,
			&redemptionModel.ResourceType,
			&redemptionModel.ResourceId,
			&redemptionModel.DiscountAmount,
			&redemptionModel.Currency,
			&redemptionModel.CreatedAt,
			&redemptionModel.Metadata,
			&count,
		)

		if err != nil {
			r.logger.Error(`failed to scan DiscountRedemption`, err.Error())
			return nil, 0, err
		}

		// Convert model to entity
		redemption := redemptionModel.ToEntity()
		redemptions = append(redemptions, redemption)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, 0, rows.Err()
	}

	return redemptions, count, nil
}

func (r DiscountRedemptionRepository) ListByResource(ctx context.Context, orgId string, resourceType string, resourceId string) ([]entities.DiscountRedemption, error) {
	tx := r.getTransactionFromContext(ctx)

	var redemptions = make([]entities.DiscountRedemption, 0)

	query := `SELECT id, org_id, discount_id, customer_id, resource_type, resource_id, 
		       discount_amount, currency, created_at, metadata
		FROM discount_redemptions
		WHERE org_id = @org_id AND resource_type = @resource_type AND resource_id = @resource_id
		ORDER BY created_at DESC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":        orgId,
		"resource_type": resourceType,
		"resource_id":   resourceId,
	})

	if err != nil {
		r.logger.Error(`failed to list DiscountRedemptions by resource`, err.Error())
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var redemptionModel models.DiscountRedemption

		err := rows.Scan(
			&redemptionModel.Id,
			&redemptionModel.OrgId,
			&redemptionModel.DiscountId,
			&redemptionModel.CustomerId,
			&redemptionModel.ResourceType,
			&redemptionModel.ResourceId,
			&redemptionModel.DiscountAmount,
			&redemptionModel.Currency,
			&redemptionModel.CreatedAt,
			&redemptionModel.Metadata,
		)

		if err != nil {
			r.logger.Error(`failed to scan DiscountRedemption`, err.Error())
			return nil, err
		}

		// Convert model to entity
		redemption := redemptionModel.ToEntity()
		redemptions = append(redemptions, redemption)
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, rows.Err().Error())
		return nil, rows.Err()
	}

	return redemptions, nil
}

func (r DiscountRedemptionRepository) CountByDiscount(ctx context.Context, orgId string, discountId string) (int, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT COUNT(*) 
		FROM discount_redemptions 
		WHERE org_id = @org_id AND discount_id = @discount_id`

	var count int
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":      orgId,
		"discount_id": discountId,
	}).Scan(&count)

	if err != nil {
		r.logger.Error(`failed to count DiscountRedemptions by discount`, err.Error())
		return 0, err
	}

	return count, nil
}

func (r DiscountRedemptionRepository) CountByCustomerAndDiscount(ctx context.Context, orgId string, customerId string, discountId string) (int, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT COUNT(*) 
		FROM discount_redemptions 
		WHERE org_id = @org_id AND customer_id = @customer_id AND discount_id = @discount_id`

	var count int
	err := tx.QueryRow(ctx, query, pgx.NamedArgs{
		"org_id":      orgId,
		"customer_id": customerId,
		"discount_id": discountId,
	}).Scan(&count)

	if err != nil {
		r.logger.Error(`failed to count DiscountRedemptions by customer and discount`, err.Error())
		return 0, err
	}

	return count, nil
}
