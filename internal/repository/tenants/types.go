package tenants

type CreateTenantInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
