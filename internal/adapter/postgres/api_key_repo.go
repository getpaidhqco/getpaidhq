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
	var row apiKeyRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		First(&row).Error
	if err != nil {
		return domain.ApiKey{}, translateErr(err)
	}
	return row.toDomain(), nil
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
	var row apiKeyRow
	err := dbFromCtx(ctx, r.db).
		Where("key_hash = ?", keyHash).
		First(&row).Error
	if err != nil {
		return domain.ApiKey{}, translateErr(err)
	}
	return row.toDomain(), nil
}

// List returns the org's API keys with stable pagination. Callers MUST NOT
// surface KeyHash to end-users.
func (r *ApiKeyRepo) List(ctx context.Context, orgId string, pagination domain.Pagination) ([]domain.ApiKey, int, error) {
	var rows []apiKeyRow
	var count int64
	if err := dbFromCtx(ctx, r.db).
		Model(&apiKeyRow{}).
		Scopes(OrgScope(orgId)).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId), Paginate(pagination)).
		Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]domain.ApiKey, len(rows))
	for i, row := range rows {
		out[i] = row.toDomain()
	}
	return out, int(count), nil
}

func (r *ApiKeyRepo) Create(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	row := apiKeyRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Update(ctx context.Context, entity domain.ApiKey) (domain.ApiKey, error) {
	row := apiKeyRowFromDomain(entity)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.ApiKey{}, err
	}
	return r.FindById(ctx, entity.OrgId, entity.Id)
}

func (r *ApiKeyRepo) Delete(ctx context.Context, orgId string, id string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("id = ?", id).
		Delete(&apiKeyRow{}).Error
}
