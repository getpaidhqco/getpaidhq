package domain

import "time"

type Variant struct {
	OrgId       string            `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id          string            `gorm:"column:id;primaryKey" json:"id"`
	ProductId   string            `gorm:"column:product_id" json:"product_id"`
	Name        string            `gorm:"column:name" json:"name"`
	Description string            `gorm:"column:description" json:"description"`
	Metadata    map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	Prices      []Price           `gorm:"foreignKey:VariantId,OrgId;references:Id,OrgId" json:"prices"`
	CreatedAt   time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Variant) TableName() string { return "variants" }

type CreateVariantInput struct {
	OrgId       string            `json:"org_id"`
	Id          string            `json:"id"`
	ProductId   string            `json:"product_id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}
