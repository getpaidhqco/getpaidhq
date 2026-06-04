package postgres

import (
	"time"

	"getpaidhq/internal/core/domain"
)

// sessionRow is the postgres on-the-wire shape of a Session. Package-internal.
type sessionRow struct {
	OrgId     string    `gorm:"column:org_id;primaryKey"`
	Id        string    `gorm:"column:id;primaryKey"`
	CartId    string    `gorm:"column:cart_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (sessionRow) TableName() string { return "sessions" }

func (r sessionRow) toDomain() domain.Session {
	return domain.Session{
		OrgId:     r.OrgId,
		Id:        r.Id,
		CartId:    r.CartId,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func sessionRowFromDomain(s domain.Session) sessionRow {
	return sessionRow{
		OrgId:     s.OrgId,
		Id:        s.Id,
		CartId:    s.CartId,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
