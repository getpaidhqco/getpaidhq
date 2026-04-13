package domain

import "time"

type ApiKey struct {
	OrgId     string    `gorm:"column:org_id;primaryKey" json:"org_id" validate:"required"`
	Id        string    `gorm:"column:id;primaryKey" json:"id" validate:"required"`
	Key       string    `gorm:"column:key" json:"key" validate:"required"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at" validate:"required"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at" validate:"required"`
}

func (ApiKey) TableName() string { return "api_keys" }
