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

// Org is the multi-tenant root entity. Every other persisted entity is
// scoped to an Org via its OrgId. Single primary key (Id only).
type Org struct {
	Id        string
	Name      string
	Country   string
	Timezone  string
	Status    OrgStatus
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GetPaymentGatewayInput is a service-layer query parameter. Lives in domain
// historically; could move to port/ when its usage is audited.
type GetPaymentGatewayInput struct {
	OrgId string
	PspId string
}
