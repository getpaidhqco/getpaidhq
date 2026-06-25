package postgresgorm

import (
	"context"
	"time"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type InvoiceRepo struct {
	db *gorm.DB
}

type invoiceCounterRow struct {
	OrgId     string    `gorm:"column:org_id;primaryKey"`
	Value     int64     `gorm:"column:value"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (invoiceCounterRow) TableName() string { return "invoice_counters" }

func NewInvoiceRepo(db *gorm.DB) port.InvoiceRepository {
	return &InvoiceRepo{db: db}
}

func (r *InvoiceRepo) Create(ctx context.Context, entity domain.Invoice) (domain.Invoice, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	row := invoiceRowFromDomain(entity)
	// gorm Create cascades the associated LineItems in the same statement set;
	// callers that need atomicity with other writes wrap this in tx.RunInTx.
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Invoice{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *InvoiceRepo) NextInvoiceNumber(ctx context.Context, orgId string) (int64, error) {
	var next int64
	err := dbFromCtx(ctx, r.db).Raw(`
		INSERT INTO invoice_counters (org_id, value, created_at, updated_at)
		VALUES (?, 1, NOW(), NOW())
		ON CONFLICT (org_id) DO UPDATE
		SET value = invoice_counters.value + 1, updated_at = NOW()
		RETURNING value`, orgId).Scan(&next).Error
	if err != nil {
		return 0, err
	}
	return next, nil
}

func (r *InvoiceRepo) SetInvoiceCounter(ctx context.Context, orgId string, value int64) error {
	return dbFromCtx(ctx, r.db).Exec(`
		INSERT INTO invoice_counters (org_id, value, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
		ON CONFLICT (org_id) DO UPDATE
		SET value = EXCLUDED.value, updated_at = NOW()`, orgId, value).Error
}

func (r *InvoiceRepo) FindById(ctx context.Context, orgId string, id string) (domain.Invoice, error) {
	var row invoiceRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Preload("LineItems").
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Invoice{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *InvoiceRepo) FindBySubscriptionCycle(ctx context.Context, orgId, subscriptionId string, cycle int) (domain.Invoice, error) {
	var row invoiceRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Preload("LineItems").
		Where("subscription_id = ? AND cycle = ?", subscriptionId, cycle).
		First(&row).Error
	if err != nil {
		return domain.Invoice{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *InvoiceRepo) List(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Invoice, int, error) {
	var rows []invoiceRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&invoiceRow{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Preload("LineItems").
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return invoiceRowsToDomain(rows), int(count), nil
}

func (r *InvoiceRepo) FindBySubscriptionId(ctx context.Context, orgId, subscriptionId string, p domain.Pagination) ([]domain.Invoice, int, error) {
	var rows []invoiceRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&invoiceRow{}).
		Scopes(OrgScope(orgId)).
		Where("subscription_id = ?", subscriptionId).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Preload("LineItems").
		Where("subscription_id = ?", subscriptionId).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return invoiceRowsToDomain(rows), int(count), nil
}

func (r *InvoiceRepo) Update(ctx context.Context, entity domain.Invoice) (domain.Invoice, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	row := invoiceRowFromDomain(entity)
	// Update the invoice row only (status/total transitions); line items are
	// written at Create time and not mutated here.
	if err := dbFromCtx(ctx, r.db).Omit("LineItems").Save(&row).Error; err != nil {
		return domain.Invoice{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}
