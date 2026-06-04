package port

// CreateVariantInput is the command input for VariantService.Create.
type CreateVariantInput struct {
	OrgId       string
	Id          string
	ProductId   string
	Name        string
	Description string
	Metadata    map[string]string
}

// UpdateVariantInput is the command input for VariantService.Update.
type UpdateVariantInput struct {
	Name        string
	Description string
	Metadata    map[string]string
}
