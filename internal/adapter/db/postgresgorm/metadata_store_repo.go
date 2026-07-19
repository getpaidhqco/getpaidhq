package postgresgorm

import (
	"context"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"

	"gorm.io/gorm"
)

type MetadataStoreRepo struct {
	db *gorm.DB
}

func NewMetadataStoreRepo(db *gorm.DB) port.MetadataStoreRepository {
	return &MetadataStoreRepo{db: db}
}

func (r *MetadataStoreRepo) FindByKey(ctx context.Context, orgId string, parentId string, key string) (domain.MetadataStore, error) {
	var row metadataStoreRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ? AND key = ?", parentId, key).
		First(&row).Error
	if err != nil {
		return domain.MetadataStore{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *MetadataStoreRepo) FindByParent(ctx context.Context, orgId string, parentId string) ([]domain.MetadataStore, error) {
	var rows []metadataStoreRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ?", parentId).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return metadataStoreRowsToDomain(rows), nil
}

func (r *MetadataStoreRepo) FindByParentType(ctx context.Context, orgId string, parentType string, key string) ([]domain.MetadataStore, error) {
	var rows []metadataStoreRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_type = ? AND key = ?", parentType, key).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return metadataStoreRowsToDomain(rows), nil
}

func (r *MetadataStoreRepo) FindByValue(ctx context.Context, orgId string, key string, value string) ([]domain.MetadataStore, error) {
	var rows []metadataStoreRow
	err := dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("key = ? AND value = ?", key, value).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return metadataStoreRowsToDomain(rows), nil
}

func (r *MetadataStoreRepo) FindByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]domain.MetadataStore, error) {
	var rows []metadataStoreRow
	err := dbFromCtx(ctx, r.db).
		Where("key = ? AND value = ? AND parent_type = ?", key, value, parentType).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return metadataStoreRowsToDomain(rows), nil
}

func (r *MetadataStoreRepo) Create(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	row := metadataStoreRowFromDomain(metadata)
	if err := dbFromCtx(ctx, r.db).Create(&row).Error; err != nil {
		return domain.MetadataStore{}, err
	}
	return r.FindByKey(ctx, metadata.OrgId, metadata.ParentId, metadata.Key)
}

func (r *MetadataStoreRepo) Update(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	row := metadataStoreRowFromDomain(metadata)
	if err := dbFromCtx(ctx, r.db).Save(&row).Error; err != nil {
		return domain.MetadataStore{}, err
	}
	return r.FindByKey(ctx, metadata.OrgId, metadata.ParentId, metadata.Key)
}

func (r *MetadataStoreRepo) Delete(ctx context.Context, orgId string, parentId string, key string) error {
	return dbFromCtx(ctx, r.db).
		Scopes(OrgScope(orgId)).
		Where("parent_id = ? AND key = ?", parentId, key).
		Delete(&metadataStoreRow{}).Error
}
