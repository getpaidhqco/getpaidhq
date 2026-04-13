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
	Id        string            `json:"id"`
	Name      string            `json:"name" binding:"required"`
	Country   string            `json:"country" binding:"required"`
	Timezone  string            `json:"timezone"`
	Status    OrgStatus         `json:"status" binding:"required"`
	Metadata  map[string]string `json:"metadata"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

type GetPaymentGatewayInput struct {
	OrgId string
	PspId string
}
