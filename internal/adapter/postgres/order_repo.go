package postgres

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) port.OrderRepository {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) FindById(ctx context.Context, orgId string, id string) (domain.Order, error) {
	var row orderRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Order{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *OrderRepo) Create(ctx context.Context, entity domain.Order) (domain.Order, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	row := orderRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Order{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) Update(ctx context.Context, entity domain.Order) (domain.Order, error) {
	row := orderRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Order{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Order, int, error) {
	var rows []orderRow
	var count int64
	if err := dbFromCtx(ctx, r.db).Model(&orderRow{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.Order, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, int(count), nil
}

func (r *OrderRepo) FindOrderItemById(ctx context.Context, orgId string, id string) (domain.OrderItem, error) {
	var row orderItemRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.OrderItem{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *OrderRepo) CreateOrderItem(ctx context.Context, entity domain.OrderItem) (domain.OrderItem, error) {
	entity.Metadata = emptyIfNil(entity.Metadata)
	row := orderItemRowFromDomain(entity)
	omits := []string{"Price"}
	// variant_id is nullable with a FK constraint; omit the column (→ NULL) when
	// no variant is set so that an empty string is not sent to postgres.
	if entity.VariantId == "" {
		omits = append(omits, "variant_id")
	}
	if err := dbFromCtx(ctx, r.db).Omit(omits...).Create(&row).Error; err != nil {
		return domain.OrderItem{}, err
	}
	return r.FindOrderItemById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) UpdateOrderItem(ctx context.Context, entity domain.OrderItem) (domain.OrderItem, error) {
	row := orderItemRowFromDomain(entity)
	omits := []string{"Price"}
	// variant_id is nullable with a FK constraint; omit the column (→ NULL) when
	// no variant is set so that an empty string is not sent to postgres.
	if entity.VariantId == "" {
		omits = append(omits, "variant_id")
	}
	if err := dbFromCtx(ctx, r.db).Omit(omits...).Save(&row).Error; err != nil {
		return domain.OrderItem{}, err
	}
	return r.FindOrderItemById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) FindOrderItemsByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.OrderItem, error) {
	var rows []orderItemRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("order_id = ?", orderId).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return orderItemRowsToDomain(rows), nil
}

// FindOrderItemsBySubscriptionId returns the order lines a subscription bills
// (the recurring lines stamped with this subscription's id).
func (r *OrderRepo) FindOrderItemsBySubscriptionId(ctx context.Context, orgId string, subscriptionId string) ([]domain.OrderItem, error) {
	var rows []orderItemRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("subscription_id = ?", subscriptionId).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return orderItemRowsToDomain(rows), nil
}
