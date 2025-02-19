package orgs

type CreateOrgInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GetPaymentGatewayInput struct {
	OrgId string
	PspId string
}
