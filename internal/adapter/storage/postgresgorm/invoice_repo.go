package postgresgorm

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type InvoiceRepo struct {
	db *gorm.DB
}

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
