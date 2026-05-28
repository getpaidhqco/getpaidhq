package postgres

import (
	"context"

	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type MetadataStoreRepo struct {
	db *gorm.DB
}

func NewMetadataStoreRepo(db *gorm.DB) port.MetadataStoreRepository {
	return &MetadataStoreRepo{db: db}
}

func (r *MetadataStoreRepo) FindByKey(ctx context.Context, orgId string, parentId string, key string) (domain.MetadataStore, error) {
	var meta domain.MetadataStore
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ? AND key = ?", parentId, key).
		First(&meta).Error
	return meta, translateErr(err)
}

func (r *MetadataStoreRepo) FindByParent(ctx context.Context, orgId string, parentId string) ([]domain.MetadataStore, error) {
	var metas []domain.MetadataStore
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ?", parentId).
		Find(&metas).Error
	return metas, err
}

func (r *MetadataStoreRepo) FindByParentType(ctx context.Context, orgId string, parentType string, key string) ([]domain.MetadataStore, error) {
	var metas []domain.MetadataStore
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_type = ? AND key = ?", parentType, key).
		Find(&metas).Error
	return metas, err
}

func (r *MetadataStoreRepo) FindByValue(ctx context.Context, orgId string, key string, value string) ([]domain.MetadataStore, error) {
	var metas []domain.MetadataStore
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("key = ? AND value = ?", key, value).
		Find(&metas).Error
	return metas, err
}

func (r *MetadataStoreRepo) FindByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]domain.MetadataStore, error) {
	var metas []domain.MetadataStore
	err := dbFromCtx(ctx, r.db).
		Where("key = ? AND value = ? AND parent_type = ?", key, value, parentType).
		Find(&metas).Error
	return metas, err
}

func (r *MetadataStoreRepo) Create(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	err := dbFromCtx(ctx, r.db).Create(&metadata).Error
	if err != nil {
		return domain.MetadataStore{}, err
	}
	return r.FindByKey(ctx, metadata.OrgId, metadata.ParentId, metadata.Key)
}

func (r *MetadataStoreRepo) Update(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	err := dbFromCtx(ctx, r.db).Save(&metadata).Error
	if err != nil {
		return domain.MetadataStore{}, err
	}
	return r.FindByKey(ctx, metadata.OrgId, metadata.ParentId, metadata.Key)
}

func (r *MetadataStoreRepo) Delete(ctx context.Context, orgId string, parentId string, key string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ? AND key = ?", parentId, key).
		Delete(&domain.MetadataStore{}).Error
}
