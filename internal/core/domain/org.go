package domain

import "time"

type OrgStatus string

const (
	OrgStatusTrial    OrgStatus = "trial"
	OrgStatusActive   OrgStatus = "active"
	OrgStatusDemo     OrgStatus = "demo"
	OrgStatusInactive OrgStatus = "inactive"
	OrgStatusDeleted  OrgStatus = "deleted"
)

type Org struct {
	Id        string            `gorm:"column:id;primaryKey" json:"id"`
	Name      string            `gorm:"column:name" json:"name" validate:"required"`
	Country   string            `gorm:"column:country" json:"country" validate:"required"`
	Timezone  string            `gorm:"column:timezone" json:"timezone"`
	Status    OrgStatus         `gorm:"column:status" json:"status" validate:"required"`
	Metadata  map[string]string `gorm:"column:metadata;serializer:json" json:"metadata"`
	CreatedAt time.Time         `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at" json:"updated_at"`
}

func (Org) TableName() string { return "orgs" }

type GetPaymentGatewayInput struct {
	OrgId string
	PspId string
}
