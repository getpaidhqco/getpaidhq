package postgrespgx

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// cartRow is the postgres on-the-wire shape of a Cart. Status and Total are
// derived fields on the domain entity (populated by Cart.Calculate()) and are
// NOT persisted, so they have no column here. data and metadata are nullable
// jsonb columns carried via jsonCol without emptyIfNil, so a nil map/zero struct
// serializes as JSON null.
type cartRow struct {
	OrgId     string
	Id        string
	Data      jsonCol[domain.CartData]
	Metadata  jsonCol[map[string]string]
	CreatedAt time.Time
	UpdatedAt time.Time
}

const cartColumns = `org_id, id, data, metadata, created_at, updated_at`

func (r *cartRow) scanInto(s scanner) error {
	return s.Scan(&r.OrgId, &r.Id, &r.Data, &r.Metadata, &r.CreatedAt, &r.UpdatedAt)
}

func (r cartRow) toDomain() domain.Cart {
	c := domain.Cart{
		OrgId:     r.OrgId,
		Id:        r.Id,
		Data:      r.Data.V,
		Metadata:  r.Metadata.V,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
	// Populate the derived Total / (Status stays default) from Data.
	c.Calculate()
	return c
}

func cartRowFromDomain(c domain.Cart) cartRow {
	return cartRow{
		OrgId:     c.OrgId,
		Id:        c.Id,
		Data:      newJSON(c.Data),
		Metadata:  newJSON(c.Metadata),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
