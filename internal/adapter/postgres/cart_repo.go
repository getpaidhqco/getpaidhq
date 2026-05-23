package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type CartRepo struct {
	db *gorm.DB
}

func NewCartRepo(db *gorm.DB) port.CartRepository {
	return &CartRepo{db: db}
}

func (r *CartRepo) FindById(ctx context.Context, orgId string, id string) (domain.Cart, error) {
	var cart domain.Cart
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&cart).Error
	return cart, err
}

func (r *CartRepo) Create(ctx context.Context, input domain.Cart) (domain.Cart, error) {
	err := dbFromCtx(ctx, r.db).Create(&input).Error
	if err != nil {
		return domain.Cart{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}

func (r *CartRepo) Update(ctx context.Context, input domain.Cart) (domain.Cart, error) {
	err := dbFromCtx(ctx, r.db).Save(&input).Error
	if err != nil {
		return domain.Cart{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
