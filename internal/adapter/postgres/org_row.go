package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// orgRow is the postgres on-the-wire shape of an Org. Package-internal.
type orgRow struct {
	Id        string            `gorm:"column:id;primaryKey"`
	Name      string            `gorm:"column:name"`
	Country   string            `gorm:"column:country"`
	Timezone  string            `gorm:"column:timezone"`
	Status    domain.OrgStatus  `gorm:"column:status"`
	Metadata  map[string]string `gorm:"column:metadata;serializer:json"`
	CreatedAt time.Time         `gorm:"column:created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at"`
}

func (orgRow) TableName() string { return "orgs" }

func (r orgRow) toDomain() domain.Org {
	return domain.Org{
		Id:        r.Id,
		Name:      r.Name,
		Country:   r.Country,
		Timezone:  r.Timezone,
		Status:    r.Status,
		Metadata:  r.Metadata,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func orgRowFromDomain(o domain.Org) orgRow {
	return orgRow{
		Id:        o.Id,
		Name:      o.Name,
		Country:   o.Country,
		Timezone:  o.Timezone,
		Status:    o.Status,
		Metadata:  o.Metadata,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}
