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
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&key).Error
	return key, translateErr(err)
}

// FindByKey looks up an API key by its HMAC hash. The caller (apikey
// authn middleware) is responsible for hashing the raw key with the
// configured pepper before calling — see lib.HashApiKey. The lookup
// hits the unique index on key_hash, so existence-vs-absence does NOT
// leak a timing difference from a row scan.
//
// The argument is named `keyHash` (not `key`) so call sites can't
// accidentally pass the raw secret here.
func (r *ApiKeyRepo) FindByKey(ctx context.Context, keyHash string) (domain.ApiKey, error) {
	var apiKey domain.ApiKey
	err := dbFromCtx(ctx, r.db).
		Where("key_hash = ?", keyHash).
		First(&apiKey).Error
	return apiKey, translateErr(err)
}

// List returns the org's API keys with stable pagination. The key_hash
// column is loaded — callers that surface results to the user MUST strip
// it (the domain JSON tag already hides it; just don't reflect it).
func (r *ApiKeyRepo) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.ApiKey, int, error) {
	var keys []domain.ApiKey
	var count int64
	if err := dbFromCtx(ctx, r.db).
		Model(&domain.ApiKey{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(pagination)).
		Find(&keys).Error
	return keys, int(count), err
}

func (r *ApiKeyRepo) Create(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	err := dbFromCtx(ctx, r.db).Create(&entity).Error
	if err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Update(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	err := dbFromCtx(ctx, r.db).Save(&entity).Error
	if err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&domain.ApiKey{}).Error
}
