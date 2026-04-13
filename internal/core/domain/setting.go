package domain

import "time"

type Setting struct {
	OrgId     string    `gorm:"column:org_id;primaryKey" json:"org_id"`
	ParentId  string    `gorm:"column:parent_id;primaryKey" json:"parent_id"`
	Id        string    `gorm:"column:id;primaryKey" json:"id"`
	Type      string    `gorm:"column:value_type" json:"value_type"`
	Value     string    `gorm:"column:value;serializer:json" json:"value"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Setting) TableName() string { return "settings" }
