package orgs

type CreateOrgInput struct {
	Name        string            `json:"name" binding:"required"`
	Country     string            `json:"country" binding:"required"`
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata"`
}

type GetPaymentGatewayInput struct {
	OrgId string
	PspId string
}
