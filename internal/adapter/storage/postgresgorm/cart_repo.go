package postgresgorm

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type CartRepo struct {
	db *gorm.DB
}

func NewCartRepo(db *gorm.DB) port.CartRepository {
	return &CartRepo{db: db}
}

func (r *CartRepo) FindById(ctx context.Context, orgId string, id string) (domain.Cart, error) {
	var row cartRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.Cart{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *CartRepo) Create(ctx context.Context, input domain.Cart) (domain.Cart, error) {
	row := cartRowFromDomain(input)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.Cart{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}

func (r *CartRepo) Update(ctx context.Context, input domain.Cart) (domain.Cart, error) {
	row := cartRowFromDomain(input)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.Cart{}, err
	}
	return r.FindById(ctx, input.OrgId, input.Id)
}
