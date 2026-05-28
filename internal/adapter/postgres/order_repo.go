package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type OrderRepo struct {
	db *gorm.DB
}

func NewOrderRepo(db *gorm.DB) port.OrderRepository {
	return &OrderRepo{db: db}
}

func (r *OrderRepo) FindById(ctx context.Context, orgId string, id string) (domain.Order, error) {
	var order domain.Order
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Preload("Customer").
		Preload("Items").
		First(&order).Error
	return order, translateErr(err)
}

func (r *OrderRepo) Create(ctx context.Context, entity domain.Order) (domain.Order, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.Order{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) Update(ctx context.Context, entity domain.Order) (domain.Order, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.Order{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) Find(ctx context.Context, orgId string, p domain.Pagination) ([]domain.Order, int, error) {
	var orders []domain.Order
	var count int64
	err := dbFromCtx(ctx, r.db).Model(&domain.Order{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error
	if err != nil {
		return nil, 0, err
	}
	err = dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(p)).
		Preload("Customer").
		Preload("Items").
		Find(&orders).Error
	return orders, int(count), err
}

func (r *OrderRepo) FindOrderItemById(ctx context.Context, orgId string, id string) (domain.OrderItem, error) {
	var item domain.OrderItem
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Preload("Price").
		First(&item).Error
	return item, translateErr(err)
}

func (r *OrderRepo) CreateOrderItem(ctx context.Context, entity domain.OrderItem) (domain.OrderItem, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.OrderItem{}, err
	}
	return r.FindOrderItemById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) UpdateOrderItem(ctx context.Context, entity domain.OrderItem) (domain.OrderItem, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.OrderItem{}, err
	}
	return r.FindOrderItemById(ctx, entity.OrgId, entity.Id)
}

func (r *OrderRepo) FindOrderItemsByOrderId(ctx context.Context, orgId string, orderId string) ([]domain.OrderItem, error) {
	var items []domain.OrderItem
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("order_id = ?", orderId).
		Preload("Price").
		Find(&items).Error
	return items, err
}
