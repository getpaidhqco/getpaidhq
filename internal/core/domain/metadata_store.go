package domain

import "time"

type MetadataStore struct {
	OrgId      string    `gorm:"column:org_id;primaryKey" json:"org_id"`
	ParentId   string    `gorm:"column:parent_id;primaryKey" json:"parent_id"`
	ParentType string    `gorm:"column:parent_type" json:"parent_type"`
	Key        string    `gorm:"column:key;primaryKey" json:"key"`
	Value      string    `gorm:"column:value" json:"value"`
	Namespace  string    `gorm:"column:namespace" json:"namespace"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (MetadataStore) TableName() string { return "metadata_store" }
