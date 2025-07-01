package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"log/slog"
	"payloop/internal/api/dto/request"
	"payloop/internal/application/lib/logger"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/repositories"
	"payloop/internal/infrastructure/db/postgres/models"
	"payloop/internal/lib"
)

type PriceRepository struct {
	*PgDatabase
	logger logger.Logger
}

func NewPriceRepository(primaryDb lib.Database, logger logger.Logger) repositories.PriceRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return PriceRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r PriceRepository) Create(ctx context.Context, entity entities.Price) (entities.Price, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO prices (org_id, id, label, variant_id, category, scheme, cycles, currency, 
                    unit_price, min_price, suggested_price, billing_interval, billing_interval_qty, 
                    trial_interval, trial_interval_qty, tax_code, 
                    has_usage, usage_type, unit_type, aggregation_type, percentage_rate, fixed_fee, included_usage, usage_limit,
                    metadata, created_at, updated_at)
        VALUES (@org_id, @id, @label, @variant_id, @category, @scheme, @cycles, @currency, 
                @unit_price, @min_price, @suggested_price, @billing_interval, @billing_interval_qty, 
                @trial_interval, @trial_interval_qty, @tax_code, 
                @has_usage, @usage_type, @unit_type, @aggregation_type, @percentage_rate, @fixed_fee, @included_usage, @usage_limit,
                @metadata, NOW(), NOW())
       `

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		return entities.Price{}, err
	}

	_, err = tx.Exec(ctx, query, pgx.NamedArgs{
		"org_id":               entity.OrgId,
		"id":                   entity.Id,
		"label":                entity.Label,
		"variant_id":           entity.VariantId,
		"category":             entity.Category,
		"scheme":               entity.Scheme,
		"cycles":               entity.Cycles,
		"currency":             entity.Currency,
		"unit_price":           entity.UnitPrice,
		"min_price":            entity.MinPrice,
		"suggested_price":      entity.SuggestedPrice,
		"billing_interval":     entity.BillingInterval,
		"billing_interval_qty": entity.BillingIntervalQty,
		"trial_interval":       entity.TrialInterval,
		"trial_interval_qty":   entity.TrialIntervalQty,
		"tax_code":             pgtype.Text{String: entity.TaxCode, Valid: entity.TaxCode != ""},

		// Usage-based billing fields
		"has_usage":        entity.HasUsage,
		"usage_type":       pgtype.Text{String: string(entity.UsageType), Valid: string(entity.UsageType) != ""},
		"unit_type":        pgtype.Text{String: string(entity.UnitType), Valid: string(entity.UnitType) != ""},
		"aggregation_type": pgtype.Text{String: string(entity.AggregationType), Valid: string(entity.AggregationType) != ""},
		"percentage_rate":  entity.PercentageRate,
		"fixed_fee":        entity.FixedFee,
		"included_usage":   entity.IncludedUsage,
		"usage_limit":      entity.UsageLimit,

		"metadata": metadata,
	})

	if err != nil {
		r.logger.Error(`failed to create Price`, err.Error())
		return entities.Price{}, err
	}

	// Create price tiers if any
	if len(entity.Tiers) > 0 {
		err = r.CreatePriceTiers(ctx, entity.Tiers)
		if err != nil {
			r.logger.Error(`failed to create Price Tiers`, err.Error())
			return entities.Price{}, err
		}
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r PriceRepository) FindById(ctx context.Context, orgId string, id string) (entities.Price, error) {
	tx := r.getTransactionFromContext(ctx)
	var price models.Price
	err := tx.QueryRow(ctx, `SELECT org_id,id,variant_id,label,billing_interval,billing_interval_qty,
       category,scheme,cycles,currency,unit_price,min_price,suggested_price,
       trial_interval,trial_interval_qty,tax_code,
       has_usage,usage_type,unit_type,aggregation_type,percentage_rate,fixed_fee,included_usage,usage_limit,
       metadata,created_at,updated_at
							FROM prices WHERE org_id=@org_id AND id=@id`,
		pgx.NamedArgs{
			"org_id": orgId,
			"id":     id,
		}).Scan(
		&price.OrgId,
		&price.Id,
		&price.VariantId,
		&price.Label,
		&price.BillingInterval,
		&price.BillingIntervalQty,
		&price.Category,
		&price.Scheme,
		&price.Cycles,
		&price.Currency,
		&price.UnitPrice,
		&price.MinPrice,
		&price.SuggestedPrice,
		&price.TrialInterval,
		&price.TrialIntervalQty,
		&price.TaxCode,
		&price.HasUsage,
		&price.UsageType,
		&price.UnitType,
		&price.AggregationType,
		&price.PercentageRate,
		&price.FixedFee,
		&price.IncludedUsage,
		&price.UsageLimit,
		&price.Metadata,
		&price.CreatedAt,
		&price.UpdatedAt,
	)

	if err != nil {
		r.logger.Error(`failed to find Price`, slog.String("err", err.Error()))
		return entities.Price{}, errors.New("not found")
	}

	// Load price tiers
	tiers, err := r.GetPriceTiers(ctx, orgId, id)
	if err != nil {
		r.logger.Error(`failed to load price tiers`, slog.String("err", err.Error()))
		return entities.Price{}, err
	}

	entity := price.ToEntity()
	entity.Tiers = tiers
	return entity, nil
}

func (r PriceRepository) FindByVariantId(ctx context.Context, orgId string, variantId string, p request.Pagination) ([]entities.Price, int, error) {
	tx := r.getTransactionFromContext(ctx)

	var prices = make([]entities.Price, 0)
	var count int
	query := `SELECT org_id, id, variant_id, label, category, scheme, cycles, currency, unit_price, min_price, 
                     suggested_price, billing_interval, billing_interval_qty, trial_interval, trial_interval_qty,
                     tax_code, has_usage, usage_type, unit_type, aggregation_type, percentage_rate, fixed_fee, included_usage, usage_limit,
                     metadata, created_at, updated_at, count(*) OVER()
              FROM prices 
              WHERE org_id = @org_id AND variant_id = @variant_id
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
                CASE WHEN @sort_dir = 'desc' THEN
                         CASE @sort_col
                             WHEN 'created_at' THEN created_at
                             ELSE NULL
                             END
                     ELSE
                         NULL
                    END
                    DESC
              LIMIT @lim OFFSET @off;`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":     orgId,
		"variant_id": variantId,
		"lim":        p.Limit,
		"off":        p.Offset,
		"sort_col":   p.SortBy,
		"sort_dir":   p.SortDirection,
	})
	if err != nil {
		r.logger.Error(`failed to find Prices by VariantId`, slog.String("err", err.Error()))
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var price models.Price
		err := rows.Scan(
			&price.OrgId,
			&price.Id,
			&price.VariantId,
			&price.Label,
			&price.Category,
			&price.Scheme,
			&price.Cycles,
			&price.Currency,
			&price.UnitPrice,
			&price.MinPrice,
			&price.SuggestedPrice,
			&price.BillingInterval,
			&price.BillingIntervalQty,
			&price.TrialInterval,
			&price.TrialIntervalQty,
			&price.TaxCode,
			&price.HasUsage,
			&price.UsageType,
			&price.UnitType,
			&price.AggregationType,
			&price.PercentageRate,
			&price.FixedFee,
			&price.IncludedUsage,
			&price.UsageLimit,
			&price.Metadata,
			&price.CreatedAt,
			&price.UpdatedAt,
			&count,
		)
		if err != nil {
			r.logger.Error(`failed to scan Price`, slog.String("err", err.Error()))
			return nil, 0, err
		}
		prices = append(prices, price.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, slog.String("err", rows.Err().Error()))
		return nil, 0, rows.Err()
	}

	// Load tiers for each price
	for i, price := range prices {
		tiers, err := r.GetPriceTiers(ctx, price.OrgId, price.Id)
		if err != nil {
			r.logger.Error(`failed to load price tiers`, slog.String("err", err.Error()))
			return nil, 0, err
		}
		prices[i].Tiers = tiers
	}

	return prices, count, nil
}

func (r PriceRepository) Update(ctx context.Context, entity entities.Price) (entities.Price, error) {
	tx := r.getTransactionFromContext(ctx)

	metadata, err := json.Marshal(entity.Metadata)
	if err != nil {
		return entities.Price{}, err
	}

	_, err = tx.Exec(ctx, `UPDATE prices 
	SET label = @label, category = @category, scheme = @scheme, cycles = @cycles, currency = @currency, 
	    unit_price = @unit_price, min_price = @min_price, suggested_price = @suggested_price, 
	    billing_interval = @billing_interval, billing_interval_qty = @billing_interval_qty, 
	    trial_interval = @trial_interval, trial_interval_qty = @trial_interval_qty, 
	    tax_code = @tax_code, 
	    has_usage = @has_usage, usage_type = @usage_type, unit_type = @unit_type, aggregation_type = @aggregation_type,
	    percentage_rate = @percentage_rate, fixed_fee = @fixed_fee, included_usage = @included_usage, usage_limit = @usage_limit,
	    metadata = @metadata, updated_at = now()
	WHERE org_id = @org_id AND id = @id`,
		pgx.NamedArgs{
			"label":                entity.Label,
			"category":             entity.Category,
			"scheme":               entity.Scheme,
			"cycles":               entity.Cycles,
			"currency":             entity.Currency,
			"unit_price":           entity.UnitPrice,
			"min_price":            entity.MinPrice,
			"suggested_price":      entity.SuggestedPrice,
			"billing_interval":     entity.BillingInterval,
			"billing_interval_qty": entity.BillingIntervalQty,
			"trial_interval":       entity.TrialInterval,
			"trial_interval_qty":   entity.TrialIntervalQty,
			"tax_code":             entity.TaxCode,

			// Usage-based billing fields
			"has_usage":        entity.HasUsage,
			"usage_type":       pgtype.Text{String: string(entity.UsageType), Valid: string(entity.UsageType) != ""},
			"unit_type":        pgtype.Text{String: string(entity.UnitType), Valid: string(entity.UnitType) != ""},
			"aggregation_type": pgtype.Text{String: string(entity.AggregationType), Valid: string(entity.AggregationType) != ""},
			"percentage_rate":  entity.PercentageRate,
			"fixed_fee":        entity.FixedFee,
			"included_usage":   entity.IncludedUsage,
			"usage_limit":      entity.UsageLimit,

			"metadata": metadata,
			"org_id":   entity.OrgId,
			"id":       entity.Id,
		})

	if err != nil {
		r.logger.Error(`failed to update Price`, slog.String("err", err.Error()))
		return entities.Price{}, err
	}

	// Update price tiers
	err = r.UpdatePriceTiers(ctx, entity.OrgId, entity.Id, entity.Tiers)
	if err != nil {
		r.logger.Error(`failed to update Price Tiers`, slog.String("err", err.Error()))
		return entities.Price{}, err
	}

	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r PriceRepository) Delete(ctx context.Context, orgId string, id string) error {
	tx := r.getTransactionFromContext(ctx)

	// First delete all tiers for this price
	err := r.DeletePriceTiers(ctx, orgId, id)
	if err != nil {
		r.logger.Error(`failed to delete Price Tiers`, slog.String("err", err.Error()))
		return err
	}

	// Then delete the price
	_, err = tx.Exec(ctx, `DELETE FROM prices WHERE org_id = $1 AND id = $2`, orgId, id)
	if err != nil {
		r.logger.Error(`failed to delete Price`, slog.String("err", err.Error()))
		return err
	}

	return nil
}

// GetPriceTiers retrieves all tiers for a price
func (r PriceRepository) GetPriceTiers(ctx context.Context, orgId string, priceId string) ([]entities.PriceTier, error) {
	tx := r.getTransactionFromContext(ctx)

	query := `SELECT org_id, price_id, tier, from_qty, to_qty, unit_price, description, created_at, updated_at
              FROM price_tiers 
              WHERE org_id = @org_id AND price_id = @price_id
              ORDER BY tier ASC`

	rows, err := tx.Query(ctx, query, pgx.NamedArgs{
		"org_id":   orgId,
		"price_id": priceId,
	})
	if err != nil {
		r.logger.Error(`failed to get price tiers`, slog.String("err", err.Error()))
		return nil, err
	}
	defer rows.Close()

	var tiers []entities.PriceTier
	for rows.Next() {
		var tier models.PriceTier
		err := rows.Scan(
			&tier.OrgId,
			&tier.PriceId,
			&tier.Tier,
			&tier.FromQty,
			&tier.ToQty,
			&tier.UnitPrice,
			&tier.Description,
			&tier.CreatedAt,
			&tier.UpdatedAt,
		)
		if err != nil {
			r.logger.Error(`failed to scan price tier`, slog.String("err", err.Error()))
			return nil, err
		}
		tiers = append(tiers, tier.ToEntity())
	}

	if rows.Err() != nil {
		r.logger.Error(`rows iteration error`, slog.String("err", rows.Err().Error()))
		return nil, rows.Err()
	}

	return tiers, nil
}

// CreatePriceTiers creates multiple price tiers
func (r PriceRepository) CreatePriceTiers(ctx context.Context, tiers []entities.PriceTier) error {
	if len(tiers) == 0 {
		return nil
	}

	tx := r.getTransactionFromContext(ctx)

	query := `INSERT INTO price_tiers (org_id, price_id, tier, from_qty, to_qty, unit_price, description, created_at, updated_at)
              VALUES (@org_id, @price_id, @tier, @from_qty, @to_qty, @unit_price, @description, NOW(), NOW())`

	for _, tier := range tiers {
		var toQty pgtype.Int4
		if tier.ToQty != nil {
			toQty = pgtype.Int4{Int32: int32(*tier.ToQty), Valid: true}
		} else {
			toQty = pgtype.Int4{Valid: false}
		}

		_, err := tx.Exec(ctx, query, pgx.NamedArgs{
			"org_id":      tier.OrgId,
			"price_id":    tier.PriceId,
			"tier":        tier.Tier,
			"from_qty":    tier.FromQty,
			"to_qty":      toQty,
			"unit_price":  tier.UnitPrice,
			"description": pgtype.Text{String: tier.Description, Valid: tier.Description != ""},
		})

		if err != nil {
			r.logger.Error(`failed to create price tier`, slog.String("err", err.Error()))
			return err
		}
	}

	return nil
}

// UpdatePriceTiers updates all tiers for a price
func (r PriceRepository) UpdatePriceTiers(ctx context.Context, orgId string, priceId string, tiers []entities.PriceTier) error {
	// First delete all existing tiers
	err := r.DeletePriceTiers(ctx, orgId, priceId)
	if err != nil {
		return err
	}

	// Then create the new tiers
	if len(tiers) > 0 {
		return r.CreatePriceTiers(ctx, tiers)
	}

	return nil
}

// DeletePriceTiers deletes all tiers for a price
func (r PriceRepository) DeletePriceTiers(ctx context.Context, orgId string, priceId string) error {
	tx := r.getTransactionFromContext(ctx)

	_, err := tx.Exec(ctx, `DELETE FROM price_tiers WHERE org_id = @org_id AND price_id = @price_id`,
		pgx.NamedArgs{
			"org_id":   orgId,
			"price_id": priceId,
		})

	if err != nil {
		r.logger.Error(`failed to delete price tiers`, slog.String("err", err.Error()))
		return err
	}
	return nil
}
