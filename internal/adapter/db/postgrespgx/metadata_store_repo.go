package postgrespgx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"getpaidhq/internal/core/domain"
	"getpaidhq/internal/core/port"
)

type MetadataStoreRepo struct {
	pool *pgxpool.Pool
}

func NewMetadataStoreRepo(pool *pgxpool.Pool) port.MetadataStoreRepository {
	return &MetadataStoreRepo{pool: pool}
}

func (r *MetadataStoreRepo) FindByKey(ctx context.Context, orgId string, parentId string, key string) (domain.MetadataStore, error) {
	q := dbFromCtx(ctx, r.pool)
	var row metadataStoreRow
	if err := row.scanInto(q.QueryRow(ctx,
		`SELECT `+metadataStoreColumns+` FROM metadata_store WHERE org_id = $1 AND parent_id = $2 AND key = $3`,
		orgId, parentId, key)); err != nil {
		return domain.MetadataStore{}, translateErr(err)
	}
	return row.toDomain(), nil
}

func (r *MetadataStoreRepo) FindByParent(ctx context.Context, orgId string, parentId string) ([]domain.MetadataStore, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+metadataStoreColumns+` FROM metadata_store WHERE org_id = $1 AND parent_id = $2`,
		orgId, parentId)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *MetadataStoreRepo) FindByParentType(ctx context.Context, orgId string, parentType string, key string) ([]domain.MetadataStore, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+metadataStoreColumns+` FROM metadata_store WHERE org_id = $1 AND parent_type = $2 AND key = $3`,
		orgId, parentType, key)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *MetadataStoreRepo) FindByValue(ctx context.Context, orgId string, key string, value string) ([]domain.MetadataStore, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+metadataStoreColumns+` FROM metadata_store WHERE org_id = $1 AND key = $2 AND value = $3`,
		orgId, key, value)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

// FindByValueWithoutOrg deliberately omits the org_id filter — it matches on
// key, value and parent_type across all orgs.
func (r *MetadataStoreRepo) FindByValueWithoutOrg(ctx context.Context, key string, value string, parentType string) ([]domain.MetadataStore, error) {
	q := dbFromCtx(ctx, r.pool)
	rows, err := q.Query(ctx,
		`SELECT `+metadataStoreColumns+` FROM metadata_store WHERE key = $1 AND value = $2 AND parent_type = $3`,
		key, value, parentType)
	if err != nil {
		return nil, err
	}
	return r.collect(rows)
}

func (r *MetadataStoreRepo) Create(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	row := metadataStoreRowFromDomain(metadata)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`INSERT INTO metadata_store (`+metadataStoreColumns+`) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		row.OrgId, row.ParentId, row.ParentType, row.Key, row.Value, row.Namespace, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.MetadataStore{}, err
	}
	return r.FindByKey(ctx, metadata.OrgId, metadata.ParentId, metadata.Key)
}

func (r *MetadataStoreRepo) Update(ctx context.Context, metadata domain.MetadataStore) (domain.MetadataStore, error) {
	row := metadataStoreRowFromDomain(metadata)
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`UPDATE metadata_store SET parent_type=$4, value=$5, namespace=$6, created_at=$7, updated_at=$8
		 WHERE org_id=$1 AND parent_id=$2 AND key=$3`,
		row.OrgId, row.ParentId, row.Key, row.ParentType, row.Value, row.Namespace, row.CreatedAt, row.UpdatedAt)
	if err != nil {
		return domain.MetadataStore{}, err
	}
	return r.FindByKey(ctx, metadata.OrgId, metadata.ParentId, metadata.Key)
}

func (r *MetadataStoreRepo) Delete(ctx context.Context, orgId string, parentId string, key string) error {
	q := dbFromCtx(ctx, r.pool)
	_, err := q.Exec(ctx,
		`DELETE FROM metadata_store WHERE org_id = $1 AND parent_id = $2 AND key = $3`,
		orgId, parentId, key)
	return err
}

// collect drains rows into domain metadata entries, closing rows. Returns a
// non-nil empty slice when no rows match.
func (r *MetadataStoreRepo) collect(rows pgx.Rows) ([]domain.MetadataStore, error) {
	defer rows.Close()
	out := make([]domain.MetadataStore, 0)
	for rows.Next() {
		var row metadataStoreRow
		if err := row.scanInto(rows); err != nil {
			return nil, err
		}
		out = append(out, row.toDomain())
	}
	return out, rows.Err()
}
