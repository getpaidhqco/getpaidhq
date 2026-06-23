package port

// CreateSessionInput is the input for SessionService.Create.
type CreateSessionInput struct {
	OrgId    string
	Currency string
	Country  string
	Metadata map[string]string
}
