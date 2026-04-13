package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5"
	"payloop/internal/core/domain"
	"payloop/internal/core/port"
	"payloop/internal/lib"
	"time"
)

type MetadataStoreRepository struct {
	*PgDatabase
	logger port.Logger
}

func NewMetadataStoreRepository(primaryDb lib.Database, logger port.Logger) port.MetadataStoreRepository {
	pgDatabase, ok := primaryDb.(*PgDatabase)
	if !ok {
		panic("database is not of type *db.PgDatabase")
	}
	return MetadataStoreRepository{
		PgDatabase: pgDatabase,
		logger:     logger,
	}
}

func (r MetadataStoreRepository) FindByKey(ctx context.Context, orgId string, parentId string, key string) (domain.MetadataStore, error) {
	tx := r.getTransactionFromContext(ctx)

	var metadata domain.MetadataStore
	err := tx.QueryRow(ctx,
		`SELECT org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at
		FROM metadata_store
		WHERE org_id = @org_id AND parent_id = @parent_id AND key = @key`,
		pgx.NamedArgs{
			"org_id":    orgId,
			"parent_id": parentId,
			"key":       key,
		}).Scan(
		&metadata.OrgId,
		&metadata.ParentId,
		&metadata.ParentType,
		&metadata.Key,
		&metadata.Value,
		&metadata.Namespace,
		&metadata.CreatedAt,
		&metadata.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.MetadataStore{}, errors.New("metadata not found")
		}
		r.logger.Error("failed to find metadata", "error", err.Error())
		return domain.MetadataStore{}, err
	}
	return metadata, nil
}

func (r MetadataStoreRepository) FindByParent(ctx context.Context, orgId string, parentId string) ([]domain.MetadataStore, error) {
	tx := r.getTransactionFromContext(ctx)

	rows, err := tx.Query(ctx,
		`SELECT org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at
		FROM metadata_store
		WHERE org_id = @org_id AND parent_id = @parent_id`,
		pgx.NamedArgs{
			"org_id":    orgId,
			"parent_id": parentId,
		})
	if err != nil {
		r.logger.Error("failed to find metadata by parent", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	var metadataList []domain.MetadataStore
	for rows.Next() {
		var metadata domain.MetadataStore
		err := rows.Scan(
			&metadata.OrgId,
			&metadata.ParentId,
			&metadata.ParentType,
			&metadata.Key,
			&metadata.Value,
			&metadata.Namespace,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan metadata", "error", err.Error())
			return nil, err
		}
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

func (r MetadataStoreRepository) FindByParentType(ctx context.Context, orgId string, parentType string, key string) ([]domain.MetadataStore, error) {
	tx := r.getTransactionFromContext(ctx)

	rows, err := tx.Query(ctx,
		`SELECT org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at
		FROM metadata_store
		WHERE org_id = @org_id AND parent_type = @parent_type AND key = @key`,
		pgx.NamedArgs{
			"org_id":      orgId,
			"parent_type": parentType,
			"key":         key,
		})
	if err != nil {
		r.logger.Error("failed to find metadata by parent type", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	var metadataList []domain.MetadataStore
	for rows.Next() {
		var metadata domain.MetadataStore
		err := rows.Scan(
			&metadata.OrgId,
			&metadata.ParentId,
			&metadata.ParentType,
			&metadata.Key,
			&metadata.Value,
			&metadata.Namespace,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan metadata", "error", err.Error())
			return nil, err
		}
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

func (r MetadataStoreRepository) FindByValue(ctx context.Context, orgId string, key string, value string) ([]domain.MetadataStore, error) {
	tx := r.getTransactionFromContext(ctx)

	rows, err := tx.Query(ctx,
		`SELECT org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at
		FROM metadata_store
		WHERE org_id = @org_id AND key = @key AND value = @value`,
		pgx.NamedArgs{
			"org_id": orgId,
			"key":    key,
			"value":  value,
		})
	if err != nil {
		r.logger.Error("failed to find metadata by value", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	var metadataList []domain.MetadataStore
	for rows.Next() {
		var metadata domain.MetadataStore
		err := rows.Scan(
			&metadata.OrgId,
			&metadata.ParentId,
			&metadata.ParentType,
			&metadata.Key,
			&metadata.Value,
			&metadata.Namespace,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan metadata", "error", err.Error())
			return nil, err
		}
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

func (r MetadataStoreRepository) FindByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]domain.MetadataStore, error) {
	tx := r.getTransactionFromContext(ctx)

	var rows pgx.Rows
	var err error

	if parentType != "" {
		rows, err = tx.Query(ctx,
			`SELECT org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at
			FROM metadata_store
			WHERE key = @key AND value = @value AND parent_type = @parent_type`,
			pgx.NamedArgs{
				"key":         key,
				"value":       value,
				"parent_type": parentType,
			})
	} else {
		rows, err = tx.Query(ctx,
			`SELECT org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at
			FROM metadata_store
			WHERE key = @key AND value = @value`,
			pgx.NamedArgs{
				"key":   key,
				"value": value,
			})
	}
	if err != nil {
		r.logger.Error("failed to find metadata by value without org", "error", err.Error())
		return nil, err
	}
	defer rows.Close()

	var metadataList []domain.MetadataStore
	for rows.Next() {
		var metadata domain.MetadataStore
		err := rows.Scan(
			&metadata.OrgId,
			&metadata.ParentId,
			&metadata.ParentType,
			&metadata.Key,
			&metadata.Value,
			&metadata.Namespace,
			&metadata.CreatedAt,
			&metadata.UpdatedAt,
		)
		if err != nil {
			r.logger.Error("failed to scan metadata", "error", err.Error())
			return nil, err
		}
		metadataList = append(metadataList, metadata)
	}

	return metadataList, nil
}

func (r MetadataStoreRepository) Create(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	tx := r.getTransactionFromContext(ctx)

	now := time.Now()
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = now
	}
	metadata.UpdatedAt = now

	var result domain.MetadataStore
	err := tx.QueryRow(ctx,
		`INSERT INTO metadata_store (org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at)
		VALUES (@org_id, @parent_id, @parent_type, @key, @value, @namespace, @created_at, @updated_at)
		RETURNING org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at`,
		pgx.NamedArgs{
			"org_id":      metadata.OrgId,
			"parent_id":   metadata.ParentId,
			"parent_type": metadata.ParentType,
			"key":         metadata.Key,
			"value":       metadata.Value,
			"namespace":   metadata.Namespace,
			"created_at":  metadata.CreatedAt,
			"updated_at":  metadata.UpdatedAt,
		}).Scan(
		&result.OrgId,
		&result.ParentId,
		&result.ParentType,
		&result.Key,
		&result.Value,
		&result.Namespace,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		r.logger.Error("failed to create metadata", "error", err.Error())
		return domain.MetadataStore{}, err
	}

	return result, nil
}

func (r MetadataStoreRepository) Update(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	tx := r.getTransactionFromContext(ctx)

	metadata.UpdatedAt = time.Now()

	var result domain.MetadataStore
	err := tx.QueryRow(ctx,
		`UPDATE metadata_store
		SET value = @value, namespace = @namespace, updated_at = @updated_at
		WHERE org_id = @org_id AND parent_id = @parent_id AND key = @key
		RETURNING org_id, parent_id, parent_type, key, value, namespace, created_at, updated_at`,
		pgx.NamedArgs{
			"org_id":     metadata.OrgId,
			"parent_id":  metadata.ParentId,
			"key":        metadata.Key,
			"value":      metadata.Value,
			"namespace":  metadata.Namespace,
			"updated_at": metadata.UpdatedAt,
		}).Scan(
		&result.OrgId,
		&result.ParentId,
		&result.ParentType,
		&result.Key,
		&result.Value,
		&result.Namespace,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.MetadataStore{}, errors.New("metadata not found")
		}
		r.logger.Error("failed to update metadata", "error", err.Error())
		return domain.MetadataStore{}, err
	}

	return result, nil
}

func (r MetadataStoreRepository) Delete(ctx context.Context, orgId string, parentId string, key string) error {
	tx := r.getTransactionFromContext(ctx)

	commandTag, err := tx.Exec(ctx,
		`DELETE FROM metadata_store
		WHERE org_id = @org_id AND parent_id = @parent_id AND key = @key`,
		pgx.NamedArgs{
			"org_id":    orgId,
			"parent_id": parentId,
			"key":       key,
		})

	if err != nil {
		r.logger.Error("failed to delete metadata", "error", err.Error())
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return errors.New("metadata not found")
	}

	return nil
}
