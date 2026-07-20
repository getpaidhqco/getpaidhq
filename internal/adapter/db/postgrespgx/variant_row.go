package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// variantRow is the postgres on-the-wire shape of a Variant. Prices are NOT
// embedded — composition is a service-layer concern. description is a nullable
// TEXT column scanned through a *string so a NULL row reads back as "" without
// a scan error; the domain's "" is stored directly (never NULL) by always
// passing a non-nil pointer on the write path. metadata is a nullable JSONB
// column mapped via jsonCol with no emptyIfNil applied, so a nil map marshals to
// JSON null.
type variantRow struct {
	OrgId       string
	Id          string
	ProductId   string
	Name        string
	Description *string
	Metadata    jsonCol[map[string]string]
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

const variantColumns = `org_id, id, product_id, name, description, metadata, created_at, updated_at`

func (r *variantRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.ProductId, &r.Name, &r.Description, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r variantRow) toDomain() domain.Variant {
	return domain.Variant{
		OrgId:       r.OrgId,
		Id:          r.Id,
		ProductId:   r.ProductId,
		Name:        r.Name,
		Description: strOrEmpty(r.Description),
		Metadata:    r.Metadata.V,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func variantRowFromDomain(v domain.Variant) variantRow {
	// Description is written as a non-nil pointer (storing "" when unset) so the
	// column behaves like a plain string, never NULL.
	desc := v.Description
	return variantRow{
		OrgId:       v.OrgId,
		Id:          v.Id,
		ProductId:   v.ProductId,
		Name:        v.Name,
		Description: &desc,
		Metadata:    newJSON(v.Metadata),
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}
