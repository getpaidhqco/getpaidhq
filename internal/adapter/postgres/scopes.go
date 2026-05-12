package postgres

import (
	"gorm.io/gorm"
	"getpaidhq/internal/core/domain"
)

func OrgScope(orgId string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("org_id = ?", orgId)
	}
}

func Paginate(p domain.Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(p.SortBy + " " + p.SortDirection).Limit(p.Limit).Offset(p.Offset)
	}
}
