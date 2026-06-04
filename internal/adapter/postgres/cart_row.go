package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// cartRow is the postgres on-the-wire shape of a Cart. Status and Total are
// derived fields on the domain entity (populated by Cart.Calculate()) and
// are NOT persisted, so they have no column here.
type cartRow struct {
	OrgId     string            `gorm:"column:org_id;primaryKey"`
	Id        string            `gorm:"column:id;primaryKey"`
	Data      domain.CartData   `gorm:"column:data;serializer:json"`
	Metadata  map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt time.Time         `gorm:"column:created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at"`
}

func (cartRow) TableName() string { return "carts" }

func (r cartRow) toDomain() domain.Cart {
	c := domain.Cart{
		OrgId:     r.OrgId,
		Id:        r.Id,
		Data:      r.Data,
		Metadata:  r.Metadata,
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
		Data:      c.Data,
		Metadata:  c.Metadata,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
