package domain

import "time"

type Product struct {
	OrgId       string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id          string            `gorm:"column:id;primaryKey" json:"id"`
	Name        string            `gorm:"column:name" json:"name"`
	Description string            `gorm:"column:description" json:"description"`
	Variants    []Variant         `gorm:"foreignKey:ProductId,OrgId;references:Id,OrgId" json:"variants"`
	Metadata    map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt   time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Product) TableName() string { return "products" }
