package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// productRow is the postgres on-the-wire shape of a Product. Variants are NOT
// embedded — composition is a service-layer concern.
//
// Nullable-column handling:
//   - description is a nullable TEXT column, so it's held as *string and mapped
//     via strOrEmpty/nilIfEmpty rather than a bare string.
//   - status is the ProductStatus enum column, held as string and converted at
//     the domain boundary (never pass a defined enum type as a pgx arg).
//   - metadata is a nullable JSONB column that is NOT run through emptyIfNil, so
//     the value passes straight through jsonCol field-for-field.
//   - archived_at is a plain nullable timestamp, held as *time.Time and
//     passed/read straight.
type productRow struct {
	OrgId       string
	Id          string
	Name        string
	Description *string
	Status      string
	ArchivedAt  *time.Time
	Metadata    jsonCol[map[string]string]
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const productColumns = `org_id, id, name, description, status, archived_at, metadata, created_at, updated_at`

func (r *productRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Name, &r.Description, &r.Status,
		&r.ArchivedAt, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r productRow) toDomain() domain.Product {
	return domain.Product{
		OrgId:       r.OrgId,
		Id:          r.Id,
		Name:        r.Name,
		Description: strOrEmpty(r.Description),
		Status:      domain.ProductStatus(r.Status),
		ArchivedAt:  r.ArchivedAt,
		Metadata:    r.Metadata.V,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func productRowFromDomain(p domain.Product) productRow {
	return productRow{
		OrgId:       p.OrgId,
		Id:          p.Id,
		Name:        p.Name,
		Description: nilIfEmpty(p.Description),
		Status:      string(p.Status),
		ArchivedAt:  p.ArchivedAt,
		Metadata:    newJSON(p.Metadata),
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
