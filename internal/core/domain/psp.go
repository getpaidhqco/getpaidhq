package domain

import "time"

// PspConfig represents a payment service provider configuration for an organization.
// Named PspConfig (not Gateway) to avoid collision with the Gateway string type.
type PspConfig struct {
	OrgId     string    `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id        string    `gorm:"column:id;primaryKey" json:"id"`
	PspId     Gateway   `gorm:"column:psp_id" json:"psp_id"`
	Name      string    `gorm:"column:name" json:"name"`
	Active    bool      `gorm:"column:active" json:"active"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (PspConfig) TableName() string { return "gateways" }
