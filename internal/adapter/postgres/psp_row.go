package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// pspConfigRow is the postgres on-the-wire shape of a PspConfig. Note the
// table name is `gateways` (legacy schema name).
type pspConfigRow struct {
	OrgId     string         `gorm:"column:org_id;primaryKey"`
	Id        string         `gorm:"column:id;primaryKey"`
	PspId     domain.Gateway `gorm:"column:psp_id"`
	Name      string         `gorm:"column:name"`
	Active    bool           `gorm:"column:active"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
}

func (pspConfigRow) TableName() string { return "gateways" }

func (r pspConfigRow) toDomain() domain.PspConfig {
	return domain.PspConfig{
		OrgId:     r.OrgId,
		Id:        r.Id,
		PspId:     r.PspId,
		Name:      r.Name,
		Active:    r.Active,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func pspConfigRowFromDomain(p domain.PspConfig) pspConfigRow {
	return pspConfigRow{
		OrgId:     p.OrgId,
		Id:        p.Id,
		PspId:     p.PspId,
		Name:      p.Name,
		Active:    p.Active,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
