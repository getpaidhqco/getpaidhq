package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type ApiKeyRepo struct {
	db *gorm.DB
}

func NewApiKeyRepo(db *gorm.DB) port.ApiKeyRepository {
	return &ApiKeyRepo{db: db}
}

func (r *ApiKeyRepo) FindById(ctx context.Context, orgId string, id string) (domain.ApiKey, error) {
	var key domain.ApiKey
	err := r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&key).Error
	return key, err
}

func (r *ApiKeyRepo) FindByKey(ctx context.Context, key string) (domain.ApiKey, error) {
	var apiKey domain.ApiKey
	err := r.db.WithContext(ctx).
		Where("key = ?", key).
		First(&apiKey).Error
	return apiKey, err
}

func (r *ApiKeyRepo) Create(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	err := r.db.WithContext(ctx).Create(&entity).Error
	if err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Update(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	err := r.db.WithContext(ctx).Save(&entity).Error
	if err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Delete(ctx context.Context, orgId string, id string) error {
	return r.db.WithContext(ctx).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&domain.ApiKey{}).Error
}
